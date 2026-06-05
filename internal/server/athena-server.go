package server

import (
	"context"
	"errors"
	"fmt"
	goio "io"
	"io/fs"
	"math"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"os/signal"
	"path"
	"path/filepath"
	"regexp"
	go_runtime "runtime"
	"runtime/debug"
	"strings"
	"sync/atomic"
	"syscall"
	"time"

	gosync "sync"

	"github.com/golang-jwt/jwt/v5"
	golang_proto "github.com/golang/protobuf/proto" //nolint:staticcheck
	"github.com/gorilla/handlers"
	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/logging"
	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/recovery"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/improbable-eng/grpc-web/go/grpcweb"
	"github.com/redis/go-redis/v9"
	log "github.com/sirupsen/logrus"
	"github.com/soheilhy/cmux"
	"github.com/useryege/athena/common"
	applicationapiclient "github.com/useryege/athena/internal/application/apiclient"
	notificationapiclient "github.com/useryege/athena/internal/notification/apiclient"
	polymarketapiclient "github.com/useryege/athena/internal/polymarket/apiclient"
	"github.com/useryege/athena/internal/server/account"
	"github.com/useryege/athena/internal/server/application"
	servercache "github.com/useryege/athena/internal/server/cache"
	"github.com/useryege/athena/internal/server/logout"
	servernotification "github.com/useryege/athena/internal/server/notification"
	serverpolymarket "github.com/useryege/athena/internal/server/polymarket"
	"github.com/useryege/athena/internal/server/rbacpolicy"
	"github.com/useryege/athena/internal/server/session"
	"github.com/useryege/athena/internal/server/settings"
	"github.com/useryege/athena/internal/server/version"
	serverwallet "github.com/useryege/athena/internal/server/wallet"
	serverworm "github.com/useryege/athena/internal/server/worm"
	walletapiclient "github.com/useryege/athena/internal/wallet/apiclient"
	wormapiclient "github.com/useryege/athena/internal/worm/apiclient"
	"github.com/useryege/athena/pkg/apiclient"
	sessionpkg "github.com/useryege/athena/pkg/apiclient/session"
	settingspkg "github.com/useryege/athena/pkg/apiclient/settings"
	"github.com/useryege/athena/ui"
	"github.com/useryege/athena/util/assets"
	"github.com/useryege/athena/util/env"
	errorsutil "github.com/useryege/athena/util/errors"
	grpc_util "github.com/useryege/athena/util/grpc"
	"github.com/useryege/athena/util/healthz"
	httputil "github.com/useryege/athena/util/http"
	utilio "github.com/useryege/athena/util/io"
	"github.com/useryege/athena/util/io/files"
	jwtutil "github.com/useryege/athena/util/jwt"
	"github.com/useryege/athena/util/rbac"
	util_session "github.com/useryege/athena/util/session"
	settings_util "github.com/useryege/athena/util/settings"
	"github.com/useryege/athena/util/swagger"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/status"
	"k8s.io/apimachinery/pkg/util/wait"

	accountpkg "github.com/useryege/athena/pkg/apiclient/account"
	applicationpkg "github.com/useryege/athena/pkg/apiclient/application"
	notificationpkg "github.com/useryege/athena/pkg/apiclient/notification"
	polymarketpkg "github.com/useryege/athena/pkg/apiclient/polymarket"
	versionpkg "github.com/useryege/athena/pkg/apiclient/version"
	walletpkg "github.com/useryege/athena/pkg/apiclient/wallet"
	wormpkg "github.com/useryege/athena/pkg/apiclient/worm"
)

const (
	maxConcurrentLoginRequestsCountEnv = "ATHENA_MAX_CONCURRENT_LOGIN_REQUESTS_COUNT"
	replicasCountEnv                   = "ATHENA_API_SERVER_REPLICAS"
	renewTokenKey                      = "renew-token"
)

// ErrNoSession indicates no auth token was supplied as part of a request
var ErrNoSession = status.Errorf(codes.Unauthenticated, "no session information")

// noCacheHeaders is a map of headers that use to tell the browser not to cache the response
var noCacheHeaders = map[string]string{
	"Expires":         time.Unix(0, 0).Format(time.RFC1123),
	"Cache-Control":   "no-cache, private, max-age=0",
	"Pragma":          "no-cache",
	"X-Accel-Expires": "0",
}

// backoff is a backoff strategy for retrying operations
var backoff = wait.Backoff{
	Steps:    5,
	Duration: 500 * time.Millisecond,
	Factor:   1.0,
	Jitter:   0.1,
}

var (
	// clientConstraint = ">= " + common.MinClientVersion
	baseHRefRegex = regexp.MustCompile(`<base href="(.*?)">`)
	// limits number of concurrent login requests to prevent password brute forcing. If set to 0 then no limit is enforced.
	maxConcurrentLoginRequestsCount = 50
	replicasCount                   = 1
)

func init() {
	// parse the max concurrent login requests count from the environment variable
	maxConcurrentLoginRequestsCount = env.ParseNumFromEnv(maxConcurrentLoginRequestsCountEnv, maxConcurrentLoginRequestsCount, 0, math.MaxInt32)
	replicasCount = env.ParseNumFromEnv(replicasCountEnv, replicasCount, 0, math.MaxInt32)
	if replicasCount > 0 {
		maxConcurrentLoginRequestsCount = maxConcurrentLoginRequestsCount / replicasCount
	}
}

// AthenaServer is the API server for Athena
type AthenaServer struct {
	AthenaServerOpts
	settings    *settings_util.AthenaSettings
	log         *log.Entry
	sessionMgr  *util_session.SessionManager
	settingsMgr *settings_util.SettingsManager
	enf         *rbac.Enforcer
	// projInformer   cache.SharedIndexInformer
	policyEnforcer *rbacpolicy.RBACPolicyEnforcer
	// appInformer    cache.SharedIndexInformer
	// appLister      applisters.ApplicationLister
	// appsetInformer cache.SharedIndexInformer
	// appsetLister   applisters.ApplicationSetLister
	// db db.AthenaDB

	// stopCh is the channel which when closed, will shutdown the Athena server
	stopCh           chan os.Signal
	userStateStorage util_session.UserStateStorage
	indexDataInit    gosync.Once
	indexData        []byte
	indexDataErr     error
	staticAssets     http.FileSystem
	// apiFactory         api.Factory
	// secretInformer    cache.SharedIndexInformer
	// configMapInformer cache.SharedIndexInformer
	serviceSet *AthenaServiceSet
	// extensionManager   *extension.Manager
	Shutdown            func()
	terminateRequested  atomic.Bool
	available           atomic.Bool
	applicationClientMu gosync.Mutex
	applicationConn     *grpc.ClientConn
	applicationClient   applicationapiclient.ApplicationServiceClient
}

type AthenaServerOpts struct {
	DisableAuth     bool
	ContentTypes    []string
	EnableGZip      bool
	StaticAssetsDir string
	ListenPort      int
	ListenHost      string
	BaseHRef        string
	RootPath        string
	// DynamicClientset        dynamic.Interface
	// KubeControllerClientset client.Client
	// AppClientset            appclientset.Interface
	// RepoClientset           repoapiclient.Clientset
	Cache *servercache.Cache
	// RepoServerCache         *repocache.Cache
	RedisClient           *redis.Client
	XFrameOptions         string
	ContentSecurityPolicy string
	ApplicationClientset  applicationapiclient.Clientset
	NotificationClientset notificationapiclient.Clientset
	WalletClientset       walletapiclient.Clientset
	WormClientset         wormapiclient.Clientset
	PolymarketClientset   polymarketapiclient.Clientset
	// ApplicationNamespaces []string
	// EnableProxyExtension  bool
	// WebhookParallelism     int
	// EnableK8sEvent         []string
	// HydratorEnabled        bool
	// SyncWithReplaceAllowed bool
}

// NewServer returns a new instance of the Athena API server
func NewServer(ctx context.Context, opts AthenaServerOpts) *AthenaServer {
	settingsMgr, err := settings_util.NewSettingsManagerFromEnv(ctx)
	errorsutil.CheckError(err)
	settings, err := settingsMgr.InitializeSettings()
	errorsutil.CheckError(err)

	userStateStorage := util_session.NewUserStateStorage(opts.RedisClient)

	sessionMgr := util_session.NewSessionManager(settingsMgr, userStateStorage)

	enf := rbac.NewEnforcer(nil)
	enf.EnableEnforce(!opts.DisableAuth)
	err = enf.SetBuiltinPolicy(assets.BuiltinPolicyCSV)
	errorsutil.CheckError(err)
	enf.EnableLog(os.Getenv(common.EnvVarRBACDebug) == "1")
	policyEnf := rbacpolicy.NewRBACPolicyEnforcer(enf)
	enf.SetClaimsEnforcerFunc(policyEnf.EnforceClaims)

	// static assets
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

	logger := log.NewEntry(log.StandardLogger())

	noopShutdown := func() {
		log.Error("API Server Shutdown function called but server is not started yet.")
	}

	a := &AthenaServer{
		AthenaServerOpts: opts,
		log:              logger,
		settings:         settings,
		sessionMgr:       sessionMgr,
		settingsMgr:      settingsMgr,
		enf:              enf,
		policyEnforcer:   policyEnf,
		userStateStorage: userStateStorage,
		staticAssets:     http.FS(staticFS),
		Shutdown:         noopShutdown,
		stopCh:           make(chan os.Signal, 1),
	}

	return a

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
	log.Debugf("Listen started (host=%s, listenPort=%d)", server.ListenHost, server.ListenPort)
	mainLn, err := startListener(server.ListenHost, server.ListenPort)
	if err != nil {
		log.Debugf("Failed to start main listener on %s:%d: %v", server.ListenHost, server.ListenPort, err)
		return nil, err
	}
	log.Debugf("Started main listener on %s:%d", server.ListenHost, server.ListenPort)
	var dOpts []grpc.DialOption
	userAgent := fmt.Sprintf("%s/%s", common.AthenaUserAgentName, common.GetVersion().Version)
	log.Debugf("Configuring gRPC gateway dial options (maxRecvMsgSize=%d, userAgent=%s)", apiclient.MaxGRPCMessageSize, userAgent)
	dOpts = append(dOpts, grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(apiclient.MaxGRPCMessageSize)))
	dOpts = append(dOpts, grpc.WithUserAgent(userAgent))
	dOpts = append(dOpts, grpc.WithStatsHandler(otelgrpc.NewClientHandler()))
	dOpts = append(dOpts, grpc.WithTransportCredentials(insecure.NewCredentials()))

	gatewayAddr := fmt.Sprintf("localhost:%d", server.ListenPort)
	log.Debugf("Creating gRPC gateway client connection to %s", gatewayAddr)
	conn, err := grpc.NewClient(gatewayAddr, dOpts...)
	if err != nil {
		log.Debugf("Failed to create gRPC gateway client connection to %s: %v", gatewayAddr, err)
		utilio.Close(mainLn)
		return nil, err
	}
	log.Debug("Listen completed successfully")
	return &Listeners{Main: mainLn, GatewayConn: conn}, nil
}

// Init starts informers used by the API server
func (server *AthenaServer) Init(ctx context.Context) {
	// go server.projInformer.Run(ctx.Done())
	// go server.appInformer.Run(ctx.Done())
	// go server.appsetInformer.Run(ctx.Done())
	// go server.configMapInformer.Run(ctx.Done())
	// go server.secretInformer.Run(ctx.Done())
}

func (server *AthenaServer) newGRPCServer() *grpc.Server {
	sOpts := []grpc.ServerOption{
		// Set the both send and receive the bytes limit to be 100MB
		// The proper way to achieve high performance is to have pagination
		// while we work toward that, we can have high limit first
		grpc.MaxRecvMsgSize(apiclient.MaxGRPCMessageSize),
		grpc.MaxSendMsgSize(apiclient.MaxGRPCMessageSize),
		grpc.ConnectionTimeout(300 * time.Second),
		grpc.KeepaliveEnforcementPolicy(
			keepalive.EnforcementPolicy{
				MinTime: common.GetGRPCKeepAliveEnforcementMinimum(),
			},
		),
	}
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
	sOpts = append(sOpts, grpc.ChainStreamInterceptor(
		logging.StreamServerInterceptor(grpc_util.InterceptorLogger(server.log)),
		server.streamAuthInterceptor,
		// grpc_util.UserAgentStreamServerInterceptor(common.AthenaUserAgentName, clientConstraint),
		// grpc_util.PayloadStreamServerInterceptor(server.log, true, func(_ context.Context, c interceptors.CallMeta) bool {
		// 	return !sensitiveMethods[c.FullMethod()]
		// }),
		// grpc_util.ErrorCodeK8sStreamServerInterceptor(),
		// grpc_util.ErrorCodeGitStreamServerInterceptor(),
		recovery.StreamServerInterceptor(recovery.WithRecoveryHandler(grpc_util.LoggerRecoveryHandler(server.log))),
	))
	sOpts = append(sOpts, grpc.ChainUnaryInterceptor(
		// bug21955WorkaroundInterceptor,
		logging.UnaryServerInterceptor(grpc_util.InterceptorLogger(server.log)),
		server.unaryAuthInterceptor,
		// grpc_util.UserAgentUnaryServerInterceptor(common.AthenaUserAgentName, clientConstraint),
		// grpc_util.PayloadUnaryServerInterceptor(server.log, true, func(_ context.Context, c interceptors.CallMeta) bool {
		// 	return !sensitiveMethods[c.FullMethod()]
		// }),
		// grpc_util.ErrorCodeK8sUnaryServerInterceptor(),
		// grpc_util.ErrorCodeGitUnaryServerInterceptor(),
		recovery.UnaryServerInterceptor(recovery.WithRecoveryHandler(grpc_util.LoggerRecoveryHandler(server.log))),
	))
	sOpts = append(sOpts, grpc.StatsHandler(otelgrpc.NewServerHandler()))
	// create a new gRPC server
	grpcS := grpc.NewServer(sOpts...)

	// register all the services to the gRPC server
	grpc_health_v1.RegisterHealthServer(grpcS, server.serviceSet.HealthService)
	versionpkg.RegisterVersionServiceServer(grpcS, server.serviceSet.VersionService)
	sessionpkg.RegisterSessionServiceServer(grpcS, server.serviceSet.SessionService)
	settingspkg.RegisterSettingsServiceServer(grpcS, server.serviceSet.SettingsService)
	accountpkg.RegisterAccountServiceServer(grpcS, server.serviceSet.AccountService)
	applicationpkg.RegisterApplicationServiceServer(grpcS, server.serviceSet.ApplicationService)
	notificationpkg.RegisterNotificationServiceServer(grpcS, server.serviceSet.NotificationService)
	walletpkg.RegisterWalletServiceServer(grpcS, server.serviceSet.WalletService)
	wormpkg.RegisterWormServiceServer(grpcS, server.serviceSet.WormService)
	polymarketpkg.RegisterPolymarketServiceServer(grpcS, server.serviceSet.PolymarketService)

	// Register reflection service on gRPC server.
	reflection.Register(grpcS)
	// errorsutil.CheckError(server.serviceSet.ProjectService.NormalizeProjs())

	return grpcS
}

type AthenaServiceSet struct {
	HealthService       *health.Server
	SessionService      *session.Server
	SettingsService     *settings.Server
	AccountService      *account.Server
	VersionService      *version.Server
	ApplicationService  *application.Server
	NotificationService *servernotification.Server
	WalletService       *serverwallet.Server
	WormService         *serverworm.Server
	PolymarketService   *serverpolymarket.Server
}

func newAthenaServiceSet(server *AthenaServer) *AthenaServiceSet {
	// create a login rate limiter
	// used by the session service
	var loginRateLimiter func() (utilio.Closer, error)
	if maxConcurrentLoginRequestsCount > 0 {
		loginRateLimiter = session.NewLoginRateLimiter(maxConcurrentLoginRequestsCount)
	}

	// session service
	sessionService := session.NewServer(server.sessionMgr, server.settingsMgr, server, server.policyEnforcer, loginRateLimiter)

	// settings service
	settingsService := settings.NewServer(server.settingsMgr, server, server.DisableAuth)
	// account service
	accountService := account.NewServer(server.sessionMgr, server.settingsMgr, server.enf)
	// application service
	applicationService := application.NewServer(server.ApplicationClientset, server.enf)
	// notification service
	notificationService := servernotification.NewServer(server.NotificationClientset)
	// wallet service
	walletService := serverwallet.NewServer(server.WalletClientset)
	// worm service
	wormService := serverworm.NewServer(server.WormClientset)
	// polymarket service
	polymarketService := serverpolymarket.NewServer(server.PolymarketClientset)

	// certificateService := certificate.NewServer(a.db, a.enf)
	// gpgkeyService := gpgkey.NewServer(a.db, a.enf)
	versionService := version.NewServer(server, func() (bool, error) {
		return server.DisableAuth, nil
	})
	healthService := health.NewServer()

	return &AthenaServiceSet{
		HealthService:       healthService,
		SessionService:      sessionService,
		SettingsService:     settingsService,
		AccountService:      accountService,
		VersionService:      versionService,
		ApplicationService:  applicationService,
		NotificationService: notificationService,
		WalletService:       walletService,
		WormService:         wormService,
		PolymarketService:   polymarketService,
	}
}

type handlerSwitcher struct {
	handler              http.Handler
	urlToHandler         map[string]http.Handler
	contentTypeToHandler map[string]http.Handler
}

func (s *handlerSwitcher) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if urlHandler, ok := s.urlToHandler[r.URL.Path]; ok {
		urlHandler.ServeHTTP(w, r)
	} else if contentHandler, ok := s.contentTypeToHandler[r.Header.Get("content-type")]; ok {
		contentHandler.ServeHTTP(w, r)
	} else {
		s.handler.ServeHTTP(w, r)
	}
}

// translateGrpcCookieHeader conditionally sets a cookie on the response.
func (server *AthenaServer) translateGrpcCookieHeader(ctx context.Context, w http.ResponseWriter, resp golang_proto.Message) error {
	if sessionResp, ok := resp.(*sessionpkg.SessionResponse); ok {
		token := sessionResp.Token
		err := server.setTokenCookie(token, w)
		if err != nil {
			return fmt.Errorf("error setting token cookie from session response: %w", err)
		}
	} else if md, ok := runtime.ServerMetadataFromContext(ctx); ok {
		renewToken := md.HeaderMD[renewTokenKey]
		if len(renewToken) > 0 {
			return server.setTokenCookie(renewToken[0], w)
		}
	}
	return nil
}
func (server *AthenaServer) setTokenCookie(token string, w http.ResponseWriter) error {
	return httputil.SetTokenCookie(token, server.BaseHRef, false, w)
}

func compressHandler(handler http.Handler) http.Handler {
	compr := handlers.CompressHandler(handler)
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Header.Get("Accept") == "text/event-stream" {
			handler.ServeHTTP(writer, request)
		} else {
			compr.ServeHTTP(writer, request)
		}
	})
}
func enforceContentTypes(handler http.Handler, types []string) http.Handler {
	allowedTypes := map[string]bool{}
	for _, t := range types {
		allowedTypes[strings.ToLower(t)] = true
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet || allowedTypes[strings.ToLower(r.Header.Get("Content-Type"))] {
			handler.ServeHTTP(w, r)
		} else {
			http.Error(w, "Invalid content type", http.StatusUnsupportedMediaType)
		}
	})
}

type registerFunc func(ctx context.Context, mux *runtime.ServeMux, conn *grpc.ClientConn) error

// mustRegisterGWHandler is a convenience function to register a gateway handler
func mustRegisterGWHandler(ctx context.Context, register registerFunc, mux *runtime.ServeMux, conn *grpc.ClientConn) {
	err := register(ctx, mux, conn)
	if err != nil {
		panic(err)
	}
}

// registerDownloadHandlers registers HTTP handlers to support downloads directly from the API server
// (e.g. athena CLI)
func registerDownloadHandlers(mux *http.ServeMux, base string) {
	linuxPath, err := exec.LookPath("athena")
	if err != nil {
		log.Warnf("athena not in PATH")
	} else {
		mux.HandleFunc(base+"/athena-linux-"+go_runtime.GOARCH, func(w http.ResponseWriter, r *http.Request) {
			http.ServeFile(w, r, linuxPath)
		})
	}
}

var extensionsPattern = regexp.MustCompile(`^extension(.*)\.js$`)

func (server *AthenaServer) serveExtensions(extensionsSharedPath string, w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/javascript")

	err := filepath.Walk(extensionsSharedPath, func(filePath string, info os.FileInfo, err error) error {
		if err != nil {
			return fmt.Errorf("failed to iterate files in '%s': %w", extensionsSharedPath, err)
		}
		if !files.IsSymlink(info) && !info.IsDir() && extensionsPattern.MatchString(info.Name()) {
			processFile := func() error {
				if _, err = fmt.Fprintf(w, "// source: %s/%s \n", filePath, info.Name()); err != nil {
					return fmt.Errorf("failed to write to response: %w", err)
				}

				f, err := os.Open(filePath)
				if err != nil {
					return fmt.Errorf("failed to open file '%s': %w", filePath, err)
				}
				defer utilio.Close(f)

				if _, err := goio.Copy(w, f); err != nil {
					return fmt.Errorf("failed to copy file '%s': %w", filePath, err)
				}

				return nil
			}

			if processFile() != nil {
				return fmt.Errorf("failed to serve extension file '%s': %w", filePath, processFile())
			}
		}
		return nil
	})

	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		log.Errorf("Failed to walk extensions directory: %v", err)
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}
}

func (server *AthenaServer) uiAssetExists(filename string) bool {
	f, err := server.staticAssets.Open(strings.Trim(filename, "/"))
	if err != nil {
		return false
	}
	defer utilio.Close(f)
	stat, err := f.Stat()
	if err != nil {
		return false
	}
	return !stat.IsDir()
}

func replaceBaseHRef(data string, replaceWith string) string {
	return baseHRefRegex.ReplaceAllString(data, replaceWith)
}

func (server *AthenaServer) getIndexData() ([]byte, error) {
	server.indexDataInit.Do(func() {
		data, err := ui.Embedded.ReadFile("dist/app/index.html")
		if err != nil {
			server.indexDataErr = err
			return
		}
		if server.BaseHRef == "/" || server.BaseHRef == "" {
			server.indexData = data
		} else {
			server.indexData = []byte(replaceBaseHRef(string(data), fmt.Sprintf(`<base href="/%s/">`, strings.Trim(server.BaseHRef, "/"))))
		}
	})

	return server.indexData, server.indexDataErr
}

var mainJsBundleRegex = regexp.MustCompile(`^main\.[0-9a-f]{20}\.js$`)

func isMainJsBundle(url *url.URL) bool {
	filename := path.Base(url.Path)
	return mainJsBundleRegex.MatchString(filename)
}

// newStaticAssetsHandler returns an HTTP handler to serve UI static assets
func (server *AthenaServer) newStaticAssetsHandler() func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		acceptHTML := false
		for _, acceptType := range strings.Split(r.Header.Get("Accept"), ",") {
			if acceptType == "text/html" || acceptType == "html" {
				acceptHTML = true
				break
			}
		}

		fileRequest := r.URL.Path != "/index.html" && server.uiAssetExists(r.URL.Path)

		// Set X-Frame-Options according to configuration
		if server.XFrameOptions != "" {
			w.Header().Set("X-Frame-Options", server.XFrameOptions)
		}
		// Set Content-Security-Policy according to configuration
		if server.ContentSecurityPolicy != "" {
			w.Header().Set("Content-Security-Policy", server.ContentSecurityPolicy)
		}
		w.Header().Set("X-XSS-Protection", "1")

		// serve index.html for non file requests to support HTML5 History API
		if acceptHTML && !fileRequest && (r.Method == http.MethodGet || r.Method == http.MethodHead) {
			for k, v := range noCacheHeaders {
				w.Header().Set(k, v)
			}
			data, err := server.getIndexData()
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			modTime, err := time.Parse(common.GetVersion().BuildDate, time.RFC3339)
			if err != nil {
				modTime = time.Now()
			}
			http.ServeContent(w, r, "index.html", modTime, utilio.NewByteReadSeeker(data))
		} else {
			if isMainJsBundle(r.URL) {
				cacheControl := "public, max-age=31536000, immutable"
				if !fileRequest {
					cacheControl = "no-cache"
				}
				w.Header().Set("Cache-Control", cacheControl)
			}
			http.FileServer(server.staticAssets).ServeHTTP(w, r)
		}
	}
}

// newHTTPServer returns the HTTP server to serve HTTP/HTTPS requests. This is implemented
// using grpc-gateway as a proxy to the gRPC server.
func (server *AthenaServer) newHTTPServer(ctx context.Context, port int, grpcWebHandler http.Handler, conn *grpc.ClientConn) *http.Server {
	endpoint := fmt.Sprintf("localhost:%d", port)
	mux := http.NewServeMux()
	httpS := http.Server{
		Addr: endpoint,
		Handler: &handlerSwitcher{
			handler: mux,
			// do not need to be authenticated methods
			urlToHandler: map[string]http.Handler{
				// "/api/badge":          badge.NewHandler(server.AppClientset, server.settingsMgr, server.Namespace, server.ApplicationNamespaces),
				common.LogoutEndpoint: logout.NewHandler(server.settingsMgr, server.sessionMgr, server.RootPath, server.BaseHRef),
			},
			contentTypeToHandler: map[string]http.Handler{
				"application/grpc-web+proto": grpcWebHandler,
			},
		},
	}

	// HTTP 1.1+JSON Server
	// grpc-ecosystem/grpc-gateway is used to proxy HTTP requests to the corresponding gRPC call
	// NOTE: if a marshaller option is not supplied, grpc-gateway will default to the jsonpb from
	// golang/protobuf. Which does not support types such as time.Time. gogo/protobuf does support
	// time.Time, but does not support custom UnmarshalJSON() and MarshalJSON() methods. Therefore
	// we use our own Marshaler
	gwMuxOpts := runtime.WithMarshalerOption(runtime.MIMEWildcard, new(grpc_util.JSONMarshaler))
	gwCookieOpts := runtime.WithForwardResponseOption(server.translateGrpcCookieHeader)
	gwmux := runtime.NewServeMux(gwMuxOpts, gwCookieOpts)

	var handler http.Handler = gwmux
	if server.EnableGZip {
		handler = compressHandler(handler)
	}
	// withTracingHandler is a middleware that extracts OpenTelemetry trace context from HTTP headers
	// and injects it into the request context. This enables trace context propagation from HTTP clients
	// to gRPC services, allowing for better distributed tracing across the Athena server.
	withTracingHandler := func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			propagator := otel.GetTextMapPropagator()
			ctx := propagator.Extract(r.Context(), propagation.HeaderCarrier(r.Header))
			h.ServeHTTP(w, r.WithContext(ctx))
		})
	}
	handler = withTracingHandler(handler)
	if len(server.ContentTypes) > 0 {
		handler = enforceContentTypes(handler, server.ContentTypes)
	} else {
		log.WithField(common.SecurityField, common.SecurityHigh).Warnf("Content-Type enforcement is disabled, which may make your API vulnerable to CSRF attacks")
	}
	mux.Handle("/api/", handler)

	// // Proxy extension is currently an alpha feature and is disabled
	// // by default.
	// if server.EnableProxyExtension {
	// 	// API server won't panic if extensions fail to register. In
	// 	// this case an error log will be sent and no extension route
	// 	// will be added in mux.
	// 	registerExtensions(mux, server, metricsReg)
	// }

	mustRegisterGWHandler(ctx, versionpkg.RegisterVersionServiceHandler, gwmux, conn)
	mustRegisterGWHandler(ctx, applicationpkg.RegisterApplicationServiceHandler, gwmux, conn)
	mustRegisterGWHandler(ctx, notificationpkg.RegisterNotificationServiceHandler, gwmux, conn)
	mustRegisterGWHandler(ctx, walletpkg.RegisterWalletServiceHandler, gwmux, conn)
	mustRegisterGWHandler(ctx, wormpkg.RegisterWormServiceHandler, gwmux, conn)
	mustRegisterGWHandler(ctx, polymarketpkg.RegisterPolymarketServiceHandler, gwmux, conn)
	mustRegisterGWHandler(ctx, sessionpkg.RegisterSessionServiceHandler, gwmux, conn)
	mustRegisterGWHandler(ctx, settingspkg.RegisterSettingsServiceHandler, gwmux, conn)
	// mustRegisterGWHandler(ctx, projectpkg.RegisterProjectServiceHandler, gwmux, conn)
	mustRegisterGWHandler(ctx, accountpkg.RegisterAccountServiceHandler, gwmux, conn)
	// mustRegisterGWHandler(ctx, certificatepkg.RegisterCertificateServiceHandler, gwmux, conn)
	// mustRegisterGWHandler(ctx, gpgkeypkg.RegisterGPGKeyServiceHandler, gwmux, conn)

	// Swagger UI
	swagger.ServeSwaggerUI(mux, assets.SwaggerJSON, "/swagger-ui", server.RootPath)
	healthz.ServeHealthCheck(mux, server.healthCheck)

	// mux.HandleFunc("/api/webhook", acdWebhookHandler.Handler)

	// Serve cli binaries directly from API server
	registerDownloadHandlers(mux, "/download")

	// Serve extensions
	extensionsSharedPath := "/tmp/extensions/"

	var extensionsHandler http.Handler = http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		server.serveExtensions(extensionsSharedPath, writer)
	})
	if server.EnableGZip {
		extensionsHandler = compressHandler(extensionsHandler)
	}
	mux.Handle("/extensions.js", extensionsHandler)

	// Serve UI static assets
	var assetsHandler http.Handler = http.HandlerFunc(server.newStaticAssetsHandler())
	if server.EnableGZip {
		assetsHandler = compressHandler(assetsHandler)
	}
	mux.Handle("/", assetsHandler)
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

type Listeners struct {
	Main        net.Listener
	GatewayConn *grpc.ClientConn
}

func (l *Listeners) Close() error {
	if l.Main != nil {
		if err := l.Main.Close(); err != nil {
			return err
		}
		l.Main = nil
	}
	if l.GatewayConn != nil {
		if err := l.GatewayConn.Close(); err != nil {
			return err
		}
		l.GatewayConn = nil
	}
	return nil
}

// GracefulRestartSignal implements a signal to be used for a graceful restart trigger.
type GracefulRestartSignal struct{}

// String is a part of os.Signal interface to represent a signal as a string.
func (g GracefulRestartSignal) String() string {
	return "GracefulRestartSignal"
}

// Signal is a part of os.Signal interface doing nothing.
func (g GracefulRestartSignal) Signal() {}

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

	server.userStateStorage.Init(ctx)

	// Prepare all services for the athena server
	svcSet := newAthenaServiceSet(server)

	// set the service set to the server
	server.serviceSet = svcSet
	// create a new gRPC server
	grpcS := server.newGRPCServer()
	// wrap the gRPC server l(grpc server => http handler)
	grpcWebS := grpcweb.WrapServer(grpcS)
	httpS := server.newHTTPServer(ctx, server.ListenPort, grpcWebS, listeners.GatewayConn)
	if server.RootPath != "" {
		httpS.Handler = withRootPath(httpS.Handler, server)
	}
	// CMux is used to support servicing gRPC and HTTP1.1+JSON on the same port
	tcpm := cmux.New(listeners.Main)
	var grpcL net.Listener
	var httpL net.Listener
	httpL = tcpm.Match(cmux.HTTP1Fast("PATCH"))
	grpcL = tcpm.MatchWithWriters(cmux.HTTP2MatchHeaderFieldSendSettings("content-type", "application/grpc"))

	// Start the muxed listeners for our servers
	log.Infof("athena %s serving on port %d (url: %s)", common.GetVersion(), server.ListenPort, server.settings.URL)
	// log.Infof("Enabled application namespace patterns: %s", server.allowedApplicationNamespacesAsString())

	go func() { server.checkServeErr("gRPC server", grpcS.Serve(grpcL)) }()
	go func() { server.checkServeErr("HTTP server", httpS.Serve(httpL)) }()
	// go server.rbacPolicyLoader(ctx)
	go func() { server.checkServeErr("TCP mux", tcpm.Serve()) }()
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

		// Shutdown gRPC server
		wg.Add(1)
		go func() {
			defer wg.Done()
			grpcS.GracefulStop()
		}()

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

	log.Info("API Server started")

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

// Authenticate checks for the presence of a valid token when accessing server-side resources.
func (server *AthenaServer) Authenticate(ctx context.Context) (context.Context, error) {
	// if authentication is disabled, present the request as a local admin session
	if server.DisableAuth {
		return withDisabledAuthClaims(ctx), nil
	}

	claims, newToken, claimsErr := server.getClaims(ctx)
	if claims != nil {
		// Add claims to the context to inspect for RBAC
		//nolint:staticcheck
		ctx = context.WithValue(ctx, "claims", claims) // ctx {data:data, claims:claims}
		if newToken != "" {
			// Session tokens that are expiring soon should be regenerated if user stays active.
			// The renewed token is stored in outgoing ServerMetadata. Metadata is available to grpc-gateway
			// response forwarder that will translate it into Set-Cookie header.
			if err := grpc.SendHeader(ctx, metadata.New(map[string]string{renewTokenKey: newToken})); err != nil {
				log.Warnf("Failed to set %s header", renewTokenKey)
			}
		}
	}
	if claimsErr != nil {
		//nolint:staticcheck
		ctx = context.WithValue(ctx, util_session.AuthErrorCtxKey, claimsErr) // ctx {data:data, auth-error:claimsErr}
	}

	return ctx, claimsErr
}

// getClaims extracts, validates and refreshes a JWT token from an incoming request context.
func (server *AthenaServer) getClaims(ctx context.Context) (jwt.Claims, string, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, "", ErrNoSession
	}
	tokenString := getToken(md)
	if tokenString == "" {
		return nil, "", ErrNoSession
	}
	claims, newToken, err := server.sessionMgr.VerifyToken(ctx, tokenString)
	if err != nil {
		return claims, "", status.Errorf(codes.Unauthenticated, "invalid session: %v", err)
	}

	return claims, newToken, nil
}

// getToken extracts the token from gRPC metadata or cookie headers
func getToken(md metadata.MD) string {
	// check the "token" metadata
	{
		tokens, ok := md[apiclient.MetaDataTokenKey]
		if ok && len(tokens) > 0 {
			return tokens[0]
		}
	}

	// looks for the HTTP header `Authorization: Bearer ...`
	// athena prefers bearer token over cookie
	for _, t := range md["authorization"] {
		token := strings.TrimPrefix(t, "Bearer ")
		if strings.HasPrefix(t, "Bearer ") && jwtutil.IsValid(token) {
			return token
		}
	}

	// check the HTTP cookie
	for _, t := range md["grpcgateway-cookie"] {
		header := http.Header{}
		header.Add("Cookie", t)
		request := http.Request{Header: header}
		token, err := httputil.JoinCookies(common.AuthCookieName, request.Cookies())
		if err == nil && jwtutil.IsValid(token) {
			return token
		}
	}

	return ""
}
