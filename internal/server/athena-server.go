package server

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io/fs"
	"net"
	"net/http"
	"os"
	"os/signal"
	"regexp"
	"runtime/debug"
	"strings"
	"sync/atomic"
	"syscall"
	"time"

	gosync "sync"

	"github.com/improbable-eng/grpc-web/go/grpcweb"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/redis/go-redis/v9"
	log "github.com/sirupsen/logrus"
	"github.com/soheilhy/cmux"
	"github.com/stretchr/testify/assert/yaml"
	"github.com/useryege/athena/common"
	"github.com/useryege/athena/internal/server/application"
	servercache "github.com/useryege/athena/internal/server/cache"
	"github.com/useryege/athena/internal/server/metrics"
	"github.com/useryege/athena/internal/server/rbacpolicy"
	"github.com/useryege/athena/pkg/apiclient"
	"github.com/useryege/athena/pkg/apis/application/v1alpha1"
	"github.com/useryege/athena/ui"
	"github.com/useryege/athena/util/assets"
	cacheutil "github.com/useryege/athena/util/cache"
	"github.com/useryege/athena/util/db"
	dexutil "github.com/useryege/athena/util/dex"
	errorsutil "github.com/useryege/athena/util/errors"
	"github.com/useryege/athena/util/healthz"
	utilio "github.com/useryege/athena/util/io"
	"github.com/useryege/athena/util/notification/k8s"
	"github.com/useryege/athena/util/oidc"
	"github.com/useryege/athena/util/rbac"
	util_session "github.com/useryege/athena/util/session"
	settings_util "github.com/useryege/athena/util/settings"
	tlsutil "github.com/useryege/athena/util/tls"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
)

// AthenaServer is the API server for Argo CD
type AthenaServer struct {
	AthenaServerOpts
	ssoClientApp *oidc.ClientApp
	settings     *settings_util.ArgoCDSettings
	log          *log.Entry
	sessionMgr   *util_session.SessionManager
	settingsMgr  *settings_util.SettingsManager
	enf          *rbac.Enforcer
	// projInformer   cache.SharedIndexInformer
	policyEnforcer *rbacpolicy.RBACPolicyEnforcer
	// appInformer    cache.SharedIndexInformer
	// appLister      applisters.ApplicationLister
	// appsetInformer cache.SharedIndexInformer
	// appsetLister   applisters.ApplicationSetLister
	db db.ArgoDB

	// stopCh is the channel which when closed, will shutdown the Argo CD server
	stopCh           chan os.Signal
	userStateStorage util_session.UserStateStorage
	// indexDataInit    gosync.Once
	// indexData        []byte
	// indexDataErr     error
	staticAssets http.FileSystem
	// apiFactory         api.Factory
	secretInformer    cache.SharedIndexInformer
	configMapInformer cache.SharedIndexInformer
	serviceSet        *AthenaServiceSet
	// extensionManager   *extension.Manager
	Shutdown           func()
	terminateRequested atomic.Bool
	available          atomic.Bool
}

type AthenaServerOpts struct {
	DisableAuth     bool
	ContentTypes    []string
	EnableGZip      bool
	Insecure        bool
	StaticAssetsDir string
	ListenPort      int
	ListenHost      string
	MetricsPort     int
	MetricsHost     string
	Namespace       string
	DexServerAddr   string
	DexTLSConfig    *dexutil.DexTLSConfig
	BaseHRef        string
	RootPath        string
	// DynamicClientset        dynamic.Interface
	// KubeControllerClientset client.Client
	KubeClientset kubernetes.Interface
	// AppClientset            appclientset.Interface
	// RepoClientset           repoapiclient.Clientset
	Cache *servercache.Cache
	// RepoServerCache         *repocache.Cache
	RedisClient           *redis.Client
	TLSConfigCustomizer   tlsutil.ConfigCustomizer
	XFrameOptions         string
	ContentSecurityPolicy string
	ApplicationNamespaces []string
	// EnableProxyExtension   bool
	// WebhookParallelism     int
	// EnableK8sEvent         []string
	// HydratorEnabled        bool
	// SyncWithReplaceAllowed bool
}

// initializeDefaultProject creates the default project if it does not already exist
func initializeDefaultProject(opts AthenaServerOpts) error {
	// TODO: implement
	return nil
}

// NewServer returns a new instance of the Argo CD API server
func NewServer(ctx context.Context, opts AthenaServerOpts) *AthenaServer {
	settingsMgr := settings_util.NewSettingsManager(ctx, opts.KubeClientset, opts.Namespace)
	settings, err := settingsMgr.InitializeSettings(opts.Insecure)
	errorsutil.CheckError(err)

	err = initializeDefaultProject(opts)
	errorsutil.CheckError(err)

	userStateStorage := util_session.NewUserStateStorage(opts.RedisClient)
	ssoClientApp, err := oidc.NewClientApp(settings, opts.DexServerAddr, opts.DexTLSConfig, opts.BaseHRef, cacheutil.NewRedisCache(opts.RedisClient, settings.UserInfoCacheExpiration(), cacheutil.RedisCompressionNone))
	errorsutil.CheckError(err)
	sessionMgr := util_session.NewSessionManager(settingsMgr, opts.DexServerAddr, opts.DexTLSConfig, userStateStorage)
	enf := rbac.NewEnforcer(opts.KubeClientset, opts.Namespace, common.ArgoCDRBACConfigMapName, nil)
	enf.EnableEnforce(!opts.DisableAuth)
	err = enf.SetBuiltinPolicy(assets.BuiltinPolicyCSV)
	errorsutil.CheckError(err)
	enf.EnableLog(os.Getenv(common.EnvVarRBACDebug) == "1")
	policyEnf := rbacpolicy.NewRBACPolicyEnforcer(enf)
	enf.SetClaimsEnforcerFunc(policyEnf.EnforceClaims)

	staticFS, err := fs.Sub(ui.Embedded, "dist/app")
	errorsutil.CheckError(err)

	root, err := os.OpenRoot(opts.StaticAssetsDir)
	if err != nil {
		if os.IsNotExist(err) {
			log.Warnf("Static assets directory %q does not exist, using only embedded assets", opts.StaticAssetsDir)
		} else {
			errorsutil.CheckError(err)
		}
	} else {
		staticFS = utilio.NewComposableFS(staticFS, root.FS())
	}

	secretInformer := k8s.NewSecretInformer(opts.KubeClientset, opts.Namespace, "athena-notifications-secret")
	configMapInformer := k8s.NewConfigMapInformer(opts.KubeClientset, opts.Namespace, "athena-notifications-cm")

	dbInstance := db.NewDB(opts.Namespace, settingsMgr, opts.KubeClientset)
	logger := log.NewEntry(log.StandardLogger())

	noopShutdown := func() {
		log.Error("API Server Shutdown function called but server is not started yet.")
	}

	a := &AthenaServer{
		AthenaServerOpts: opts,
		// ApplicationSetOpts: appsetOpts,
		ssoClientApp: ssoClientApp,
		log:          logger,
		settings:     settings,
		sessionMgr:   sessionMgr,
		settingsMgr:  settingsMgr,
		enf:          enf,
		// projInformer:      projInformer,
		// appInformer:       appInformer,
		// appLister:         appLister,
		// appsetInformer:    appsetInformer,
		// appsetLister:      appsetLister,
		policyEnforcer:   policyEnf,
		userStateStorage: userStateStorage,
		staticAssets:     http.FS(staticFS),
		db:               dbInstance,
		// apiFactory:        apiFactory,
		secretInformer:    secretInformer,
		configMapInformer: configMapInformer,
		// extensionManager: em,
		Shutdown: noopShutdown,
		stopCh:   make(chan os.Signal, 1),
	}

	err = a.logInClusterWarnings()
	if err != nil {
		// Just log. It's not critical.
		log.Warnf("Failed to log in-cluster warnings: %v", err)
	}

	return a

}

// logInClusterWarnings checks the in-cluster configuration and prints out any warnings.
func (server *AthenaServer) logInClusterWarnings() error {
	labelSelector := labels.NewSelector()
	req, err := labels.NewRequirement(common.LabelKeySecretType, selection.Equals, []string{common.LabelValueSecretTypeCluster})
	if err != nil {
		return fmt.Errorf("failed to construct cluster-type label selector: %w", err)
	}
	labelSelector = labelSelector.Add(*req)
	secretsLister, err := server.settingsMgr.GetSecretsLister()
	if err != nil {
		return fmt.Errorf("failed to get secrets lister: %w", err)
	}
	clusterSecrets, err := secretsLister.Secrets(server.AthenaServerOpts.Namespace).List(labelSelector)
	if err != nil {
		return fmt.Errorf("failed to list cluster secrets: %w", err)
	}
	var inClusterSecrets []string
	for _, clusterSecret := range clusterSecrets {
		cluster, err := db.SecretToCluster(clusterSecret)
		if err != nil {
			return fmt.Errorf("could not unmarshal cluster secret %q: %w", clusterSecret.Name, err)
		}
		if cluster.Server == v1alpha1.KubernetesInternalAPIServerAddr {
			inClusterSecrets = append(inClusterSecrets, clusterSecret.Name)
		}
	}
	if len(inClusterSecrets) > 0 {
		// Don't make this call unless we actually have in-cluster secrets, to save time.
		dbSettings, err := server.settingsMgr.GetSettings()
		if err != nil {
			return fmt.Errorf("could not get DB settings: %w", err)
		}
		if !dbSettings.InClusterEnabled {
			for _, clusterName := range inClusterSecrets {
				log.Warnf("cluster %q uses in-cluster server address but it's disabled in Argo CD settings", clusterName)
			}
		}
	}
	return nil
}

func (server *AthenaServer) healthCheck(r *http.Request) error {
	if server.terminateRequested.Load() {
		return errors.New("API Server is terminating and unable to serve requests")
	}
	if !server.available.Load() {
		return errors.New("API Server is not available: it either hasn't started or is restarting")
	}
	// TODO: deep dependency check
	return nil
}

func startListener(host string, port int) (net.Listener, error) {
	var conn net.Listener
	var realErr error
	lc := net.ListenConfig{}
	_ = wait.ExponentialBackoff(backoff, func() (bool, error) {
		conn, realErr = lc.Listen(context.Background(), "tcp", fmt.Sprintf("%s:%d", host, port))
		if realErr != nil {
			return false, nil
		}
		return true, nil
	})
	return conn, realErr
}

func (server *AthenaServer) Listen() (*Listeners, error) {
	mainLn, err := startListener(server.ListenHost, server.ListenPort)
	if err != nil {
		return nil, err
	}
	metricsLn, err := startListener(server.ListenHost, server.MetricsPort)
	if err != nil {
		utilio.Close(mainLn)
		return nil, err
	}
	var dOpts []grpc.DialOption
	dOpts = append(dOpts, grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(apiclient.MaxGRPCMessageSize)))
	dOpts = append(dOpts, grpc.WithUserAgent(fmt.Sprintf("%s/%s", common.ArgoCDUserAgentName, common.GetVersion().Version)))
	dOpts = append(dOpts, grpc.WithStatsHandler(otelgrpc.NewClientHandler()))
	if server.useTLS() {
		// The following sets up the dial Options for grpc-gateway to talk to gRPC server over TLS.
		// grpc-gateway is just translating HTTP/HTTPS requests as gRPC requests over localhost,
		// so we need to supply the same certificates to establish the connections that a normal,
		// external gRPC client would need.
		tlsConfig := server.settings.TLSConfig()
		if server.TLSConfigCustomizer != nil {
			server.TLSConfigCustomizer(tlsConfig)
		}
		tlsConfig.InsecureSkipVerify = true
		dCreds := credentials.NewTLS(tlsConfig)
		dOpts = append(dOpts, grpc.WithTransportCredentials(dCreds))
	} else {
		dOpts = append(dOpts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	}

	conn, err := grpc.NewClient(fmt.Sprintf("localhost:%d", server.ListenPort), dOpts...)
	if err != nil {
		utilio.Close(mainLn)
		utilio.Close(metricsLn)
		return nil, err
	}
	return &Listeners{Main: mainLn, Metrics: metricsLn, GatewayConn: conn}, nil
}

// Init starts informers used by the API server
func (server *AthenaServer) Init(ctx context.Context) {
	// go server.projInformer.Run(ctx.Done())
	// go server.appInformer.Run(ctx.Done())
	// go server.appsetInformer.Run(ctx.Done())
	go server.configMapInformer.Run(ctx.Done())
	go server.secretInformer.Run(ctx.Done())
}

func (server *AthenaServer) newGRPCServer(prometheusRegistry *prometheus.Registry) (*grpc.Server, application.AppResourceTreeFn) {
	// var serverMetricsOptions []grpc_prometheus.ServerMetricsOption
	// if enableGRPCTimeHistogram {
	// 	serverMetricsOptions = append(serverMetricsOptions, grpc_prometheus.WithServerHandlingTimeHistogram())
	// }
	// serverMetrics := grpc_prometheus.NewServerMetrics(serverMetricsOptions...)
	// prometheusRegistry.MustRegister(serverMetrics)

	// sOpts := []grpc.ServerOption{
	// 	// Set the both send and receive the bytes limit to be 100MB
	// 	// The proper way to achieve high performance is to have pagination
	// 	// while we work toward that, we can have high limit first
	// 	grpc.MaxRecvMsgSize(apiclient.MaxGRPCMessageSize),
	// 	grpc.MaxSendMsgSize(apiclient.MaxGRPCMessageSize),
	// 	grpc.ConnectionTimeout(300 * time.Second),
	// 	grpc.KeepaliveEnforcementPolicy(
	// 		keepalive.EnforcementPolicy{
	// 			MinTime: common.GetGRPCKeepAliveEnforcementMinimum(),
	// 		},
	// 	),
	// }
	// sensitiveMethods := map[string]bool{
	// 	"/cluster.ClusterService/Create":                               true,
	// 	"/cluster.ClusterService/Update":                               true,
	// 	"/session.SessionService/Create":                               true,
	// 	"/account.AccountService/UpdatePassword":                       true,
	// 	"/gpgkey.GPGKeyService/CreateGnuPGPublicKey":                   true,
	// 	"/repository.RepositoryService/Create":                         true,
	// 	"/repository.RepositoryService/Update":                         true,
	// 	"/repository.RepositoryService/CreateRepository":               true,
	// 	"/repository.RepositoryService/UpdateRepository":               true,
	// 	"/repository.RepositoryService/ValidateAccess":                 true,
	// 	"/repocreds.RepoCredsService/CreateRepositoryCredentials":      true,
	// 	"/repocreds.RepoCredsService/UpdateRepositoryCredentials":      true,
	// 	"/repository.RepositoryService/CreateWriteRepository":          true,
	// 	"/repository.RepositoryService/UpdateWriteRepository":          true,
	// 	"/repository.RepositoryService/ValidateWriteAccess":            true,
	// 	"/repocreds.RepoCredsService/CreateWriteRepositoryCredentials": true,
	// 	"/repocreds.RepoCredsService/UpdateWriteRepositoryCredentials": true,
	// 	"/application.ApplicationService/PatchResource":                true,
	// 	// Remove from logs both because the contents are sensitive and because they may be very large.
	// 	"/application.ApplicationService/GetManifestsWithFiles": true,
	// }
	// // NOTE: notice we do not configure the gRPC server here with TLS (e.g. grpc.Creds(creds))
	// // This is because TLS handshaking occurs in cmux handling
	// sOpts = append(sOpts, grpc.ChainStreamInterceptor(
	// 	logging.StreamServerInterceptor(grpc_util.InterceptorLogger(server.log)),
	// 	serverMetrics.StreamServerInterceptor(),
	// 	grpc_auth.StreamServerInterceptor(server.Authenticate),
	// 	grpc_util.UserAgentStreamServerInterceptor(common.ArgoCDUserAgentName, clientConstraint),
	// 	grpc_util.PayloadStreamServerInterceptor(server.log, true, func(_ context.Context, c interceptors.CallMeta) bool {
	// 		return !sensitiveMethods[c.FullMethod()]
	// 	}),
	// 	grpc_util.ErrorCodeK8sStreamServerInterceptor(),
	// 	grpc_util.ErrorCodeGitStreamServerInterceptor(),
	// 	recovery.StreamServerInterceptor(recovery.WithRecoveryHandler(grpc_util.LoggerRecoveryHandler(server.log))),
	// ))
	// sOpts = append(sOpts, grpc.ChainUnaryInterceptor(
	// 	bug21955WorkaroundInterceptor,
	// 	logging.UnaryServerInterceptor(grpc_util.InterceptorLogger(server.log)),
	// 	serverMetrics.UnaryServerInterceptor(),
	// 	grpc_auth.UnaryServerInterceptor(server.Authenticate),
	// 	grpc_util.UserAgentUnaryServerInterceptor(common.ArgoCDUserAgentName, clientConstraint),
	// 	grpc_util.PayloadUnaryServerInterceptor(server.log, true, func(_ context.Context, c interceptors.CallMeta) bool {
	// 		return !sensitiveMethods[c.FullMethod()]
	// 	}),
	// 	grpc_util.ErrorCodeK8sUnaryServerInterceptor(),
	// 	grpc_util.ErrorCodeGitUnaryServerInterceptor(),
	// 	recovery.UnaryServerInterceptor(recovery.WithRecoveryHandler(grpc_util.LoggerRecoveryHandler(server.log))),
	// ))
	// sOpts = append(sOpts, grpc.StatsHandler(otelgrpc.NewServerHandler()))
	// grpcS := grpc.NewServer(sOpts...)

	// healthService := health.NewServer()
	// grpc_health_v1.RegisterHealthServer(grpcS, healthService)

	// versionpkg.RegisterVersionServiceServer(grpcS, server.serviceSet.VersionService)
	// clusterpkg.RegisterClusterServiceServer(grpcS, server.serviceSet.ClusterService)
	// applicationpkg.RegisterApplicationServiceServer(grpcS, server.serviceSet.ApplicationService)
	// applicationsetpkg.RegisterApplicationSetServiceServer(grpcS, server.serviceSet.ApplicationSetService)
	// notificationpkg.RegisterNotificationServiceServer(grpcS, server.serviceSet.NotificationService)
	// repositorypkg.RegisterRepositoryServiceServer(grpcS, server.serviceSet.RepoService)
	// repocredspkg.RegisterRepoCredsServiceServer(grpcS, server.serviceSet.RepoCredsService)
	// sessionpkg.RegisterSessionServiceServer(grpcS, server.serviceSet.SessionService)
	// settingspkg.RegisterSettingsServiceServer(grpcS, server.serviceSet.SettingsService)
	// projectpkg.RegisterProjectServiceServer(grpcS, server.serviceSet.ProjectService)
	// accountpkg.RegisterAccountServiceServer(grpcS, server.serviceSet.AccountService)
	// certificatepkg.RegisterCertificateServiceServer(grpcS, server.serviceSet.CertificateService)
	// gpgkeypkg.RegisterGPGKeyServiceServer(grpcS, server.serviceSet.GpgkeyService)
	// // Register reflection service on gRPC server.
	// reflection.Register(grpcS)
	// serverMetrics.InitializeMetrics(grpcS)
	// errorsutil.CheckError(server.serviceSet.ProjectService.NormalizeProjs())

	// TODO: delete this line when implementing the gRPC server
	grpcS := grpc.NewServer()
	return grpcS, server.serviceSet.AppResourceTreeFn
}

type AthenaServiceSet struct {
	AppResourceTreeFn application.AppResourceTreeFn
}

func newAthenaServiceSet(a *AthenaServer) *AthenaServiceSet {
	// kubectl := kubeutil.NewKubectl()
	// clusterService := cluster.NewServer(a.db, a.enf, a.Cache, kubectl)
	// repoService := repository.NewServer(a.RepoClientset, a.db, a.enf, a.Cache, a.appLister, a.projInformer, a.Namespace, a.settingsMgr, a.HydratorEnabled)
	// repoCredsService := repocreds.NewServer(a.db, a.enf)
	// var loginRateLimiter func() (utilio.Closer, error)
	// if maxConcurrentLoginRequestsCount > 0 {
	// 	loginRateLimiter = session.NewLoginRateLimiter(maxConcurrentLoginRequestsCount)
	// }
	// sessionService := session.NewServer(a.sessionMgr, a.settingsMgr, a, a.policyEnforcer, loginRateLimiter)
	// projectLock := sync.NewKeyLock()
	// applicationService, appResourceTreeFn := application.NewServer(
	// 	a.Namespace,
	// 	a.KubeClientset,
	// 	a.AppClientset,
	// 	a.appLister,
	// 	a.appInformer,
	// 	nil,
	// 	a.RepoClientset,
	// 	a.Cache,
	// 	kubectl,
	// 	a.db,
	// 	a.enf,
	// 	projectLock,
	// 	a.settingsMgr,
	// 	a.projInformer,
	// 	a.ApplicationNamespaces,
	// 	a.EnableK8sEvent,
	// 	a.SyncWithReplaceAllowed,
	// )

	// applicationSetService := applicationset.NewServer(
	// 	a.db,
	// 	a.KubeClientset,
	// 	a.DynamicClientset,
	// 	a.KubeControllerClientset,
	// 	a.enf,
	// 	a.RepoClientset,
	// 	a.AppClientset,
	// 	a.appsetInformer,
	// 	a.appsetLister,
	// 	a.Namespace,
	// 	projectLock,
	// 	a.ApplicationNamespaces,
	// 	a.GitSubmoduleEnabled,
	// 	a.EnableNewGitFileGlobbing,
	// 	a.ScmRootCAPath,
	// 	a.AllowedScmProviders,
	// 	a.EnableScmProviders,
	// 	a.EnableGitHubAPIMetrics,
	// 	a.EnableK8sEvent,
	// )

	// projectService := project.NewServer(a.Namespace, a.KubeClientset, a.AppClientset, a.enf, projectLock, a.sessionMgr, a.policyEnforcer, a.projInformer, a.settingsMgr, a.db, a.EnableK8sEvent)
	// appsInAnyNamespaceEnabled := len(a.ApplicationNamespaces) > 0
	// settingsService := settings.NewServer(a.settingsMgr, a.RepoClientset, a, a.DisableAuth, appsInAnyNamespaceEnabled, a.HydratorEnabled, a.SyncWithReplaceAllowed)
	// accountService := account.NewServer(a.sessionMgr, a.settingsMgr, a.enf)

	// notificationService := notification.NewServer(a.apiFactory)
	// certificateService := certificate.NewServer(a.db, a.enf)
	// gpgkeyService := gpgkey.NewServer(a.db, a.enf)
	// versionService := version.NewServer(a, func() (bool, error) {
	// 	if a.DisableAuth {
	// 		return true, nil
	// 	}
	// 	sett, err := a.settingsMgr.GetSettings()
	// 	if err != nil {
	// 		return false, err
	// 	}
	// 	return sett.AnonymousUserEnabled, err
	// })

	// return &ArgoCDServiceSet{
	// 	ClusterService:        clusterService,
	// 	RepoService:           repoService,
	// 	RepoCredsService:      repoCredsService,
	// 	SessionService:        sessionService,
	// 	ApplicationService:    applicationService,
	// 	AppResourceTreeFn:     appResourceTreeFn,
	// 	ApplicationSetService: applicationSetService,
	// 	ProjectService:        projectService,
	// 	SettingsService:       settingsService,
	// 	AccountService:        accountService,
	// 	NotificationService:   notificationService,
	// 	CertificateService:    certificateService,
	// 	GpgkeyService:         gpgkeyService,
	// 	VersionService:        versionService,
	// }
	return &AthenaServiceSet{}
}

// newRedirectServer returns an HTTP server which does a 307 redirect to the HTTPS server
func newRedirectServer(port int, rootPath string) *http.Server {
	var addr string
	if rootPath == "" {
		addr = fmt.Sprintf("localhost:%d", port)
	} else {
		addr = fmt.Sprintf("localhost:%d/%s", port, strings.Trim(rootPath, "/"))
	}

	return &http.Server{
		Addr: addr,
		Handler: http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			target := "https://" + req.Host

			if rootPath != "" {
				root := strings.Trim(rootPath, "/")
				prefix := "/" + root

				// If the request path already starts with rootPath, no need to add rootPath again
				if strings.HasPrefix(req.URL.Path, prefix) {
					target += req.URL.Path
				} else {
					target += prefix + req.URL.Path
				}
			} else {
				target += req.URL.Path
			}

			if req.URL.RawQuery != "" {
				target += "?" + req.URL.RawQuery
			}
			http.Redirect(w, req, target, http.StatusTemporaryRedirect)
		}),
	}
}

// newHTTPServer returns the HTTP server to serve HTTP/HTTPS requests. This is implemented
// using grpc-gateway as a proxy to the gRPC server.
func (server *AthenaServer) newHTTPServer(ctx context.Context, port int, grpcWebHandler http.Handler, appResourceTreeFn application.AppResourceTreeFn, conn *grpc.ClientConn, metricsReg HTTPMetricsRegistry) *http.Server {
	endpoint := fmt.Sprintf("localhost:%d", port)
	// mux := http.NewServeMux()
	httpS := http.Server{
		Addr: endpoint,
		// Handler: &handlerSwitcher{
		// 	handler: mux,
		// 	urlToHandler: map[string]http.Handler{
		// 		"/api/badge":          badge.NewHandler(server.AppClientset, server.settingsMgr, server.Namespace, server.ApplicationNamespaces),
		// 		common.LogoutEndpoint: logout.NewHandler(server.settingsMgr, server.sessionMgr, server.RootPath, server.BaseHRef),
		// 	},
		// 	contentTypeToHandler: map[string]http.Handler{
		// 		"application/grpc-web+proto": grpcWebHandler,
		// 	},
		// },
	}

	// // HTTP 1.1+JSON Server
	// // grpc-ecosystem/grpc-gateway is used to proxy HTTP requests to the corresponding gRPC call
	// // NOTE: if a marshaller option is not supplied, grpc-gateway will default to the jsonpb from
	// // golang/protobuf. Which does not support types such as time.Time. gogo/protobuf does support
	// // time.Time, but does not support custom UnmarshalJSON() and MarshalJSON() methods. Therefore
	// // we use our own Marshaler
	// gwMuxOpts := runtime.WithMarshalerOption(runtime.MIMEWildcard, new(grpc_util.JSONMarshaler))
	// gwCookieOpts := runtime.WithForwardResponseOption(server.translateGrpcCookieHeader)
	// gwmux := runtime.NewServeMux(gwMuxOpts, gwCookieOpts)

	// var handler http.Handler = gwmux
	// if server.EnableGZip {
	// 	handler = compressHandler(handler)
	// }
	// // withTracingHandler is a middleware that extracts OpenTelemetry trace context from HTTP headers
	// // and injects it into the request context. This enables trace context propagation from HTTP clients
	// // to gRPC services, allowing for better distributed tracing across the Athena server.
	// withTracingHandler := func(h http.Handler) http.Handler {
	// 	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	// 		propagator := otel.GetTextMapPropagator()
	// 		ctx := propagator.Extract(r.Context(), propagation.HeaderCarrier(r.Header))
	// 		h.ServeHTTP(w, r.WithContext(ctx))
	// 	})
	// }
	// handler = withTracingHandler(handler)
	// if len(server.ContentTypes) > 0 {
	// 	handler = enforceContentTypes(handler, server.ContentTypes)
	// } else {
	// 	log.WithField(common.SecurityField, common.SecurityHigh).Warnf("Content-Type enforcement is disabled, which may make your API vulnerable to CSRF attacks")
	// }
	// mux.Handle("/api/", handler)

	// terminalOpts := application.TerminalOptions{DisableAuth: server.DisableAuth, Enf: server.enf}

	// terminal := application.NewHandler(server.appLister, server.Namespace, server.ApplicationNamespaces, server.db, appResourceTreeFn, server.settings.ExecShells, server.sessionMgr, &terminalOpts).
	// 	WithFeatureFlagMiddleware(server.settingsMgr.GetSettings)
	// th := util_session.WithAuthMiddleware(server.DisableAuth, server.settings.IsSSOConfigured(), server.ssoClientApp, server.sessionMgr, terminal)
	// mux.Handle("/terminal", th)

	// // Proxy extension is currently an alpha feature and is disabled
	// // by default.
	// if server.EnableProxyExtension {
	// 	// API server won't panic if extensions fail to register. In
	// 	// this case an error log will be sent and no extension route
	// 	// will be added in mux.
	// 	registerExtensions(mux, server, metricsReg)
	// }

	// mustRegisterGWHandler(ctx, versionpkg.RegisterVersionServiceHandler, gwmux, conn)
	// mustRegisterGWHandler(ctx, clusterpkg.RegisterClusterServiceHandler, gwmux, conn)
	// mustRegisterGWHandler(ctx, applicationpkg.RegisterApplicationServiceHandler, gwmux, conn)
	// mustRegisterGWHandler(ctx, applicationsetpkg.RegisterApplicationSetServiceHandler, gwmux, conn)
	// mustRegisterGWHandler(ctx, notificationpkg.RegisterNotificationServiceHandler, gwmux, conn)
	// mustRegisterGWHandler(ctx, repositorypkg.RegisterRepositoryServiceHandler, gwmux, conn)
	// mustRegisterGWHandler(ctx, repocredspkg.RegisterRepoCredsServiceHandler, gwmux, conn)
	// mustRegisterGWHandler(ctx, sessionpkg.RegisterSessionServiceHandler, gwmux, conn)
	// mustRegisterGWHandler(ctx, settingspkg.RegisterSettingsServiceHandler, gwmux, conn)
	// mustRegisterGWHandler(ctx, projectpkg.RegisterProjectServiceHandler, gwmux, conn)
	// mustRegisterGWHandler(ctx, accountpkg.RegisterAccountServiceHandler, gwmux, conn)
	// mustRegisterGWHandler(ctx, certificatepkg.RegisterCertificateServiceHandler, gwmux, conn)
	// mustRegisterGWHandler(ctx, gpgkeypkg.RegisterGPGKeyServiceHandler, gwmux, conn)

	// // Swagger UI
	// swagger.ServeSwaggerUI(mux, assets.SwaggerJSON, "/swagger-ui", server.RootPath)
	// healthz.ServeHealthCheck(mux, server.healthCheck)

	// // Dex reverse proxy and OAuth2 login/callback
	// server.registerDexHandlers(mux)

	// // Webhook handler for git events (Note: cache timeouts are hardcoded because API server does not write to cache and not really using them)
	// argoDB := db.NewDB(server.Namespace, server.settingsMgr, server.KubeClientset)
	// acdWebhookHandler := webhook.NewHandler(server.Namespace, server.ApplicationNamespaces, server.WebhookParallelism, server.AppClientset, server.appLister, server.settings, server.settingsMgr, server.RepoServerCache, server.Cache, argoDB, server.settingsMgr.GetMaxWebhookPayloadSize())

	// mux.HandleFunc("/api/webhook", acdWebhookHandler.Handler)

	// // Serve cli binaries directly from API server
	// registerDownloadHandlers(mux, "/download")

	// // Serve extensions
	// extensionsSharedPath := "/tmp/extensions/"

	// var extensionsHandler http.Handler = http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
	// 	server.serveExtensions(extensionsSharedPath, writer)
	// })
	// if server.EnableGZip {
	// 	extensionsHandler = compressHandler(extensionsHandler)
	// }
	// mux.Handle("/extensions.js", extensionsHandler)

	// // Serve UI static assets
	// var assetsHandler http.Handler = http.HandlerFunc(server.newStaticAssetsHandler())
	// if server.EnableGZip {
	// 	assetsHandler = compressHandler(assetsHandler)
	// }
	// mux.Handle("/", assetsHandler)
	return &httpS
}
func withRootPath(handler http.Handler, a *AthenaServer) http.Handler {
	// If RootPath is empty, directly return the original handler
	if a.RootPath == "" {
		return handler
	}

	// get rid of slashes
	root := strings.Trim(a.RootPath, "/")

	mux := http.NewServeMux()
	mux.Handle("/"+root+"/", http.StripPrefix("/"+root, handler))

	healthz.ServeHealthCheck(mux, a.healthCheck)

	return mux
}

// Workaround for https://github.com/golang/go/issues/21955 to support escaped URLs in URL path.
type bug21955Workaround struct {
	handler http.Handler
}

var pathPatters = []*regexp.Regexp{
	regexp.MustCompile(`/api/v1/clusters/[^/]+`),
	regexp.MustCompile(`/api/v1/repositories/[^/]+`),
	regexp.MustCompile(`/api/v1/repocreds/[^/]+`),
	regexp.MustCompile(`/api/v1/repositories/[^/]+/apps`),
	regexp.MustCompile(`/api/v1/repositories/[^/]+/apps/[^/]+`),
	regexp.MustCompile(`/settings/clusters/[^/]+`),
}

func (bf *bug21955Workaround) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	for _, pattern := range pathPatters {
		if pattern.MatchString(r.URL.RawPath) {
			r.URL.Path = r.URL.RawPath
			break
		}
	}
	bf.handler.ServeHTTP(w, r)
}

// Run runs the API Server
// We use k8s.io/code-generator/cmd/go-to-protobuf to generate the .proto files from the API types.
// k8s.io/ go-to-protobuf uses protoc-gen-gogo, which comes from gogo/protobuf (a fork of
// golang/protobuf).
func (server *AthenaServer) Run(ctx context.Context, listeners *Listeners) {
	defer func() {
		if r := recover(); r != nil {
			log.WithField("trace", string(debug.Stack())).Error("Recovered from panic: ", r)
			server.terminateRequested.Store(true)
			server.Shutdown()
		}
	}()
	metricsServ := metrics.NewMetricsServer(server.MetricsHost, server.MetricsPort)
	if server.RedisClient != nil {
		cacheutil.CollectMetrics(server.RedisClient, metricsServ, server.userStateStorage.GetLockObject())
	}

	// Don't init storage until after CollectMetrics. CollectMetrics adds hooks to the Redis client, and Init
	// reads those hooks. If this is called first, there may be a data race.
	server.userStateStorage.Init(ctx)

	svcSet := newAthenaServiceSet(server)
	if server.sessionMgr != nil {
		server.sessionMgr.CollectMetrics(metricsServ)
	}
	server.serviceSet = svcSet
	grpcS, appResourceTreeFn := server.newGRPCServer(metricsServ.PrometheusRegistry)
	grpcWebS := grpcweb.WrapServer(grpcS)
	var httpS *http.Server
	var httpsS *http.Server
	if server.useTLS() {
		httpS = newRedirectServer(server.ListenPort, server.RootPath)
		httpsS = server.newHTTPServer(ctx, server.ListenPort, grpcWebS, appResourceTreeFn, listeners.GatewayConn, metricsServ)
	} else {
		httpS = server.newHTTPServer(ctx, server.ListenPort, grpcWebS, appResourceTreeFn, listeners.GatewayConn, metricsServ)
	}
	if server.RootPath != "" {
		httpS.Handler = withRootPath(httpS.Handler, server)

		if httpsS != nil {
			httpsS.Handler = withRootPath(httpsS.Handler, server)
		}
	}
	httpS.Handler = &bug21955Workaround{handler: httpS.Handler}
	if httpsS != nil {
		httpsS.Handler = &bug21955Workaround{handler: httpsS.Handler}
	}

	// CMux is used to support servicing gRPC and HTTP1.1+JSON on the same port
	tcpm := cmux.New(listeners.Main)
	var tlsm cmux.CMux
	var grpcL net.Listener
	var httpL net.Listener
	var httpsL net.Listener
	if !server.useTLS() {
		httpL = tcpm.Match(cmux.HTTP1Fast("PATCH"))
		grpcL = tcpm.MatchWithWriters(cmux.HTTP2MatchHeaderFieldSendSettings("content-type", "application/grpc"))
	} else {
		// We first match on HTTP 1.1 methods.
		httpL = tcpm.Match(cmux.HTTP1Fast("PATCH"))

		// If not matched, we assume that its TLS.
		tlsl := tcpm.Match(cmux.Any())
		tlsConfig := tls.Config{
			// Advertise that we support both http/1.1 and http2 for application level communication.
			// By putting http/1.1 first, we ensure that HTTPS clients will use http/1.1, which is the only
			// protocol our server supports for HTTPS clients. By including h2 in the list, we ensure that
			// gRPC clients know we support http2 for their communication.
			NextProtos: []string{"http/1.1", "h2"},
		}
		tlsConfig.GetCertificate = func(_ *tls.ClientHelloInfo) (*tls.Certificate, error) {
			return server.settings.Certificate, nil
		}
		if server.TLSConfigCustomizer != nil {
			server.TLSConfigCustomizer(&tlsConfig)
		}
		tlsl = tls.NewListener(tlsl, &tlsConfig)

		// Now, we build another mux recursively to match HTTPS and gRPC.
		tlsm = cmux.New(tlsl)
		httpsL = tlsm.Match(cmux.HTTP1Fast("PATCH"))
		grpcL = tlsm.MatchWithWriters(cmux.HTTP2MatchHeaderFieldSendSettings("content-type", "application/grpc"))
	}

	// Start the muxed listeners for our servers
	log.Infof("athena %s serving on port %d (url: %s, tls: %v, namespace: %s, sso: %v)",
		common.GetVersion(), server.ListenPort, server.settings.URL, server.useTLS(), server.Namespace, server.settings.IsSSOConfigured())
	log.Infof("Enabled application namespace patterns: %s", server.allowedApplicationNamespacesAsString())

	go func() { server.checkServeErr("grpcS", grpcS.Serve(grpcL)) }()
	go func() { server.checkServeErr("httpS", httpS.Serve(httpL)) }()
	if server.useTLS() {
		go func() { server.checkServeErr("httpsS", httpsS.Serve(httpsL)) }()
		go func() { server.checkServeErr("tlsm", tlsm.Serve()) }()
	}
	go server.watchSettings()
	go server.rbacPolicyLoader(ctx)
	go func() { server.checkServeErr("tcpm", tcpm.Serve()) }()
	go func() { server.checkServeErr("metrics", metricsServ.Serve(listeners.Metrics)) }()
	// if !cache.WaitForCacheSync(ctx.Done(), server.projInformer.HasSynced, server.appInformer.HasSynced) {
	// 	log.Fatal("Timed out waiting for project cache to sync")
	// }

	shutdownFunc := func() {
		log.Info("API Server shutdown initiated. Shutting down servers...")
		server.available.Store(false)
		shutdownCtx, cancel := context.WithTimeout(ctx, 20*time.Second)
		defer cancel()
		var wg gosync.WaitGroup

		// Shutdown http server
		wg.Add(1)
		go func() {
			defer wg.Done()
			err := httpS.Shutdown(shutdownCtx)
			if err != nil {
				log.Errorf("Error shutting down http server: %s", err)
			}
		}()

		if server.useTLS() {
			// Shutdown https server
			wg.Add(1)
			go func() {
				defer wg.Done()
				err := httpsS.Shutdown(shutdownCtx)
				if err != nil {
					log.Errorf("Error shutting down https server: %s", err)
				}
			}()
		}

		// Shutdown gRPC server
		wg.Add(1)
		go func() {
			defer wg.Done()
			grpcS.GracefulStop()
		}()

		// Shutdown metrics server
		wg.Add(1)
		go func() {
			defer wg.Done()
			err := metricsServ.Shutdown(shutdownCtx)
			if err != nil {
				log.Errorf("Error shutting down metrics server: %s", err)
			}
		}()

		if server.useTLS() {
			// Shutdown tls server
			wg.Add(1)
			go func() {
				defer wg.Done()
				tlsm.Close()
			}()
		}

		// Shutdown tcp server
		wg.Add(1)
		go func() {
			defer wg.Done()
			tcpm.Close()
		}()

		c := make(chan struct{})
		// This goroutine will wait for all servers to conclude the shutdown
		// process
		go func() {
			defer close(c)
			wg.Wait()
		}()

		select {
		case <-c:
			log.Info("All servers were gracefully shutdown. Exiting...")
		case <-shutdownCtx.Done():
			log.Warn("Graceful shutdown timeout. Exiting...")
		}
	}
	server.Shutdown = shutdownFunc
	signal.Notify(server.stopCh, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	server.available.Store(true)

	select {
	case signal := <-server.stopCh:
		log.Infof("API Server received signal: %s", signal.String())
		gracefulRestartSignal := GracefulRestartSignal{}
		if signal != gracefulRestartSignal {
			server.terminateRequested.Store(true)
		}
		server.Shutdown()
	case <-ctx.Done():
		log.Infof("API Server: %s", ctx.Err())
		server.terminateRequested.Store(true)
		server.Shutdown()
	}
}

func (server *AthenaServer) Initialized() bool {
	// return server.projInformer.HasSynced() && server.appInformer.HasSynced()
	return true
}

// TerminateRequested returns whether a shutdown was initiated by a signal or context cancel
// as opposed to a watch.
func (server *AthenaServer) TerminateRequested() bool {
	return server.terminateRequested.Load()
}

// checkServeErr checks the error from a .Serve() call to decide if it was a graceful shutdown
func (server *AthenaServer) checkServeErr(name string, err error) {
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Errorf("Error received from server %s: %v", name, err)
	} else {
		log.Infof("Graceful shutdown of %s initiated", name)
	}
}

// func checkOIDCConfigChange(currentOIDCConfig *settings_util.OIDCConfig, newArgoCDSettings *settings_util.ArgoCDSettings) bool {
// 	newOIDCConfig := newArgoCDSettings.OIDCConfig()

// 	if (currentOIDCConfig != nil && newOIDCConfig == nil) || (currentOIDCConfig == nil && newOIDCConfig != nil) {
// 		return true
// 	}

// 	if currentOIDCConfig != nil && newOIDCConfig != nil {
// 		if !reflect.DeepEqual(*currentOIDCConfig, *newOIDCConfig) {
// 			return true
// 		}
// 	}

// 	return false
// }

// watchSettings watches the configmap and secret for any setting updates that would warrant a
// restart of the API server.
func (server *AthenaServer) watchSettings() {
	// updateCh := make(chan *settings_util.ArgoCDSettings, 1)
	// server.settingsMgr.Subscribe(updateCh)

	// prevURL := server.settings.URL
	// prevAdditionalURLs := server.settings.AdditionalURLs
	// prevOIDCConfig := server.settings.OIDCConfig()
	// prevDexCfgBytes, err := dexutil.GenerateDexConfigYAML(server.settings, server.DexTLSConfig == nil || server.DexTLSConfig.DisableTLS)
	// errorsutil.CheckError(err)
	// prevGitHubSecret := server.settings.GetWebhookGitHubSecret()
	// prevGitLabSecret := server.settings.GetWebhookGitLabSecret()
	// prevBitbucketUUID := server.settings.GetWebhookBitbucketUUID()
	// prevBitbucketServerSecret := server.settings.GetWebhookBitbucketServerSecret()
	// prevGogsSecret := server.settings.GetWebhookGogsSecret()
	// prevExtConfig := server.settings.ExtensionConfig
	// var prevCert, prevCertKey string
	// if server.settings.Certificate != nil && !server.Insecure {
	// 	prevCert, prevCertKey = tlsutil.EncodeX509KeyPairString(*server.settings.Certificate)
	// }

	// for {
	// 	newSettings := <-updateCh
	// 	server.settings = newSettings
	// 	newDexCfgBytes, err := dexutil.GenerateDexConfigYAML(server.settings, server.DexTLSConfig == nil || server.DexTLSConfig.DisableTLS)
	// 	errorsutil.CheckError(err)
	// 	if !bytes.Equal(newDexCfgBytes, prevDexCfgBytes) {
	// 		log.Infof("dex config modified. restarting")
	// 		break
	// 	}
	// 	if checkOIDCConfigChange(prevOIDCConfig, server.settings) {
	// 		log.Infof("oidc config modified. restarting")
	// 		break
	// 	}
	// 	if prevURL != server.settings.URL {
	// 		log.Infof("url modified. restarting")
	// 		break
	// 	}
	// 	if !reflect.DeepEqual(prevAdditionalURLs, server.settings.AdditionalURLs) {
	// 		log.Infof("additionalURLs modified. restarting")
	// 		break
	// 	}
	// 	if prevGitHubSecret != server.settings.GetWebhookGitHubSecret() {
	// 		log.Infof("github secret modified. restarting")
	// 		break
	// 	}
	// 	if prevGitLabSecret != server.settings.GetWebhookGitLabSecret() {
	// 		log.Infof("gitlab secret modified. restarting")
	// 		break
	// 	}
	// 	if prevBitbucketUUID != server.settings.GetWebhookBitbucketUUID() {
	// 		log.Infof("bitbucket uuid modified. restarting")
	// 		break
	// 	}
	// 	if prevBitbucketServerSecret != server.settings.GetWebhookBitbucketServerSecret() {
	// 		log.Infof("bitbucket server secret modified. restarting")
	// 		break
	// 	}
	// 	if prevGogsSecret != server.settings.GetWebhookGogsSecret() {
	// 		log.Infof("gogs secret modified. restarting")
	// 		break
	// 	}
	// 	if !reflect.DeepEqual(prevExtConfig, server.settings.ExtensionConfig) {
	// 		prevExtConfig = server.settings.ExtensionConfig
	// 		log.Infof("extensions configs modified. Updating proxy registry...")
	// 		err := server.extensionManager.UpdateExtensionRegistry(server.settings)
	// 		if err != nil {
	// 			log.Errorf("error updating extensions configs: %s", err)
	// 		} else {
	// 			log.Info("extensions configs updated successfully")
	// 		}
	// 	}
	// 	if !server.Insecure {
	// 		var newCert, newCertKey string
	// 		if server.settings.Certificate != nil {
	// 			newCert, newCertKey = tlsutil.EncodeX509KeyPairString(*server.settings.Certificate)
	// 		}
	// 		if newCert != prevCert || newCertKey != prevCertKey {
	// 			log.Infof("tls certificate modified. reloading certificate")
	// 			// No need to break out of this loop since TlsConfig.GetCertificate will automagically reload the cert.
	// 		}
	// 	}
	// }
	// log.Info("shutting down settings watch")
	// server.settingsMgr.Unsubscribe(updateCh)
	// close(updateCh)
	// // Triggers server restart
	// server.stopCh <- GracefulRestartSignal{}

}

func (server *AthenaServer) rbacPolicyLoader(ctx context.Context) {
	err := server.enf.RunPolicyLoader(ctx, func(cm *corev1.ConfigMap) error {
		var scopes []string
		if scopesStr, ok := cm.Data[rbac.ConfigMapScopesKey]; scopesStr != "" && ok {
			scopes = make([]string, 0)
			err := yaml.Unmarshal([]byte(scopesStr), &scopes)
			if err != nil {
				return fmt.Errorf("error unmarshalling scopes: %w", err)
			}
		}

		server.policyEnforcer.SetScopes(scopes)
		return nil
	})
	errorsutil.CheckError(err)
}

func (server *AthenaServer) useTLS() bool {
	if server.Insecure || server.settings.Certificate == nil {
		return false
	}
	return true
}

// allowedApplicationNamespacesAsString returns a string containing comma-separated list
// of allowed application namespaces
func (server *AthenaServer) allowedApplicationNamespacesAsString() string {
	ns := server.Namespace
	if len(server.ApplicationNamespaces) > 0 {
		ns += ", "
		ns += strings.Join(server.ApplicationNamespaces, ", ")
	}
	return ns
}
