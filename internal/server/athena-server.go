package server

import (
	"context"
	"errors"
	"fmt"
	goio "io"
	"io/fs"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
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
	"github.com/useryege/athena/internal/accountaccess"
	"github.com/useryege/athena/internal/accountcenter"
	"github.com/useryege/athena/internal/accountcredentials"
	accountstatestore "github.com/useryege/athena/internal/accountstate/store"
	"github.com/useryege/athena/internal/authregistration"
	"github.com/useryege/athena/internal/googleoidc"
	managedooapiclient "github.com/useryege/athena/internal/managedoo/apiclient"
	marketradarapiclient "github.com/useryege/athena/internal/marketradar/apiclient"
	"github.com/useryege/athena/internal/moduleaccess"
	notificationapiclient "github.com/useryege/athena/internal/notification/apiclient"
	"github.com/useryege/athena/internal/phantomauth"
	profitsharingapiclient "github.com/useryege/athena/internal/profitsharing/apiclient"
	"github.com/useryege/athena/internal/server/account"
	"github.com/useryege/athena/internal/server/accountavatarhttp"
	serverappbootstrap "github.com/useryege/athena/internal/server/appbootstrap"
	servercache "github.com/useryege/athena/internal/server/cache"
	"github.com/useryege/athena/internal/server/logout"
	servermanagedoo "github.com/useryege/athena/internal/server/managedoo"
	servermarketradar "github.com/useryege/athena/internal/server/marketradar"
	servermoduleaccess "github.com/useryege/athena/internal/server/moduleaccess"
	servernotification "github.com/useryege/athena/internal/server/notification"
	serverprofitsharing "github.com/useryege/athena/internal/server/profitsharing"
	serverservicestatus "github.com/useryege/athena/internal/server/servicestatus"
	"github.com/useryege/athena/internal/server/session"
	"github.com/useryege/athena/internal/server/settings"
	serversolana "github.com/useryege/athena/internal/server/solana"
	servertokenapi "github.com/useryege/athena/internal/server/tokenapi"
	servertradersync "github.com/useryege/athena/internal/server/tradersync"
	"github.com/useryege/athena/internal/server/version"
	serverwallet "github.com/useryege/athena/internal/server/wallet"
	"github.com/useryege/athena/internal/server/walletavatarhttp"
	"github.com/useryege/athena/internal/server/walletsecrethttp"
	serverwormtrading "github.com/useryege/athena/internal/server/wormtrading"
	solanaapiclient "github.com/useryege/athena/internal/solanadiscovery/apiclient"
	tokenapiapiclient "github.com/useryege/athena/internal/tokenapi/apiclient"
	trpc "github.com/useryege/athena/internal/tradersync/apiclient"
	tradersyncstore "github.com/useryege/athena/internal/tradersync/store"
	walletapiclient "github.com/useryege/athena/internal/wallet/apiclient"
	"github.com/useryege/athena/internal/walletsecret"
	wormtradingapiclient "github.com/useryege/athena/internal/wormtrading/apiclient"
	"github.com/useryege/athena/pkg/apiclient"
	appbootstrappkg "github.com/useryege/athena/pkg/apiclient/appbootstrap"
	moduleaccesspkg "github.com/useryege/athena/pkg/apiclient/moduleaccess"
	servicestatuspkg "github.com/useryege/athena/pkg/apiclient/servicestatus"
	sessionpkg "github.com/useryege/athena/pkg/apiclient/session"
	solanapkg "github.com/useryege/athena/pkg/apiclient/solana"
	tspkg "github.com/useryege/athena/pkg/apiclient/tradersync"
	"github.com/useryege/athena/ui"
	grpc_util "github.com/useryege/athena/util/grpc"
	"github.com/useryege/athena/util/healthz"
	httputil "github.com/useryege/athena/util/http"
	utilio "github.com/useryege/athena/util/io"
	"github.com/useryege/athena/util/io/files"
	jwtutil "github.com/useryege/athena/util/jwt"
	util_session "github.com/useryege/athena/util/session"
	settings_util "github.com/useryege/athena/util/settings"
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
	managedoopkg "github.com/useryege/athena/pkg/apiclient/managedoo"
	marketradarpkg "github.com/useryege/athena/pkg/apiclient/marketradar"
	notificationpkg "github.com/useryege/athena/pkg/apiclient/notification"
	profitsharingpkg "github.com/useryege/athena/pkg/apiclient/profitsharing"
	tokenapipkg "github.com/useryege/athena/pkg/apiclient/tokenapi"
	versionpkg "github.com/useryege/athena/pkg/apiclient/version"
	walletpkg "github.com/useryege/athena/pkg/apiclient/wallet"
	wormtradingpkg "github.com/useryege/athena/pkg/apiclient/wormtrading"
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
	baseHRefRegex           = regexp.MustCompile(`<base href="[^"]*">`)
	deploymentBaseHRefRegex = regexp.MustCompile(`<meta name="athena-deployment-base-href" content="[^"]*">`)
)

type uiApplication string

const (
	memberApplication uiApplication = "member"
	adminApplication  uiApplication = "admin"
)

type indexDataCache struct {
	init gosync.Once
	data []byte
	err  error
}

// AthenaServer is the API server for Athena
type AthenaServer struct {
	AthenaServerOpts
	settings                 *settings_util.AthenaSettings
	log                      *log.Entry
	sessionMgr               *util_session.SessionManager
	settingsMgr              *settings_util.SettingsManager
	credentialMgr            *accountcredentials.CredentialManager
	accountStateStore        *accountstatestore.SQLStore
	moduleAccessStore        moduleaccess.Store
	accountCenter            *accountcenter.Manager
	accountAvatarHTTP        *accountavatarhttp.Handler
	walletAvatarHTTP         *walletavatarhttp.Handler
	accessController         *accountaccess.Controller
	authRegistration         *authregistration.Handler
	googleOIDC               *googleoidc.Handler
	phantomAuth              *phantomauth.Handler
	walletSecretMgr          *walletsecret.Manager
	wormCredentialMgr        *walletsecret.Manager
	walletSecretHTTP         *walletsecrethttp.Handler
	walletSecretPublicOrigin string
	developmentAccountIDs    map[accountcredentials.ApplicationRealm]string
	// db db.AthenaDB

	// stopCh is the channel which when closed, will shutdown the Athena server
	stopCh           chan os.Signal
	userStateStorage util_session.UserStateStorage
	memberIndexData  indexDataCache
	adminIndexData   indexDataCache
	staticAssets     http.FileSystem
	staticRoot       *os.Root
	// apiFactory         api.Factory
	// secretInformer    cache.SharedIndexInformer
	// configMapInformer cache.SharedIndexInformer
	serviceSet       *AthenaServiceSet
	traderSyncCloser goio.Closer
	processCancel    context.CancelFunc
	processWorkers   gosync.WaitGroup
	// extensionManager   *extension.Manager
	Shutdown           func()
	terminateRequested atomic.Bool
	available          atomic.Bool
}

type AthenaServerOpts struct {
	// TraderSyncClient is borrowed. When nil, the API creates and owns its connection.
	TraderSyncClient trpc.TraderSyncServiceClient
	DisableAuth      bool
	ContentTypes     []string
	EnableGZip       bool
	StaticAssetsDir  string
	ListenPort       int
	ListenHost       string
	BaseHRef         string
	RootPath         string
	// DynamicClientset        dynamic.Interface
	// KubeControllerClientset client.Client
	// AppClientset            appclientset.Interface
	// RepoClientset           repoapiclient.Clientset
	Cache *servercache.Cache
	// RepoServerCache         *repocache.Cache
	RedisClient                       *redis.Client
	XFrameOptions                     string
	ContentSecurityPolicy             string
	NotificationClientset             notificationapiclient.Clientset
	WalletClientset                   walletapiclient.Clientset
	MarketRadarClientset              marketradarapiclient.Clientset
	ManagedOOClientset                managedooapiclient.Clientset
	WormTradingClientset              wormtradingapiclient.Clientset
	ProfitSharingClientset            profitsharingapiclient.Clientset
	TokenAPIClientset                 tokenapiapiclient.Clientset
	SolanaClientset                   solanaapiclient.Clientset
	EtherscanGatewayIPs               string
	EtherscanGatewayToken             string
	EtherscanAPIKeys                  string
	EtherscanGatewayProbeQueryAddress string
	// EnableProxyExtension  bool
	// WebhookParallelism     int
	// EnableK8sEvent         []string
	// HydratorEnabled        bool
	// SyncWithReplaceAllowed bool
}

// NewServer returns a new instance of the Athena API server
func NewServer(ctx context.Context, opts AthenaServerOpts) (*AthenaServer, error) {
	ctx, processCancel := context.WithCancel(ctx)
	initialized := false
	defer func() {
		if !initialized {
			processCancel()
		}
	}()
	if opts.DisableAuth && !isLoopbackListenHost(opts.ListenHost) {
		return nil, fmt.Errorf("disabled authentication is allowed only on a loopback listen address")
	}
	settingsMgr, err := settings_util.NewSettingsManagerFromEnv(ctx)
	if err != nil {
		return nil, err
	}
	settings, err := settingsMgr.GetSettings()
	if err != nil {
		return nil, err
	}
	accountStateStore, err := accountstatestore.NewSQLStoreSource()(ctx)
	if err != nil {
		return nil, err
	}
	defer func() {
		if !initialized {
			_ = accountStateStore.Close()
		}
	}()
	accountStateStore.SetAccessChangeHook(tradersyncstore.NewAccessRevocationAdapter().ApplyAccessChangeTx)
	if initErr := accountStateStore.RequireAccessChangeHook(); initErr != nil {
		return nil, initErr
	}
	developmentAccountIDs := make(map[accountcredentials.ApplicationRealm]string, 2)
	if opts.DisableAuth {
		developmentIdentities := []struct {
			role  accountcredentials.DevelopmentRole
			realm accountcredentials.ApplicationRealm
		}{
			{role: accountcredentials.DevelopmentRoleMember, realm: accountcredentials.ApplicationRealmMember},
			{role: accountcredentials.DevelopmentRoleAdministrator, realm: accountcredentials.ApplicationRealmAdmin},
		}
		for _, identity := range developmentIdentities {
			developmentAccount, ensureErr := accountStateStore.EnsureDevelopmentAccount(ctx, identity.role)
			if ensureErr != nil {
				return nil, ensureErr
			}
			if actualRealm := developmentAccount.ApplicationRealm(); actualRealm != identity.realm {

				return nil, fmt.Errorf("development %s identity resolved to application realm %q", identity.role, actualRealm)
			}
			developmentAccountIDs[identity.realm] = developmentAccount.ID
		}
	}
	jwtSigningKey, err := accountcredentials.LoadJWTSigningKey()
	if err != nil {
		return nil, err
	}
	jwtCodec, err := accountcredentials.NewJWTCodec(jwtSigningKey)
	if err != nil {
		return nil, err
	}
	credentialMgr, err := accountcredentials.NewCredentialManager(ctx, accountStateStore, jwtCodec)
	if err != nil {
		return nil, err
	}
	if !opts.DisableAuth {
		for _, configuredAccount := range credentialMgr.List() {
			if configuredAccount.IdentityProvider == accountcredentials.IdentityProviderDevelopment {

				return nil, fmt.Errorf("development identity exists while authentication is enabled; reset account state before using external authentication")
			}
		}
	}
	accessController, err := accountaccess.NewController(ctx, accountStateStore)
	if err != nil {
		return nil, err
	}
	accountCenter, err := accountcenter.NewManager(accountStateStore)
	if err != nil {
		return nil, err
	}

	userStateStorage := util_session.NewUserStateStorage(opts.RedisClient)

	sessionMgr := util_session.NewSessionManager(credentialMgr, jwtCodec, userStateStorage, accessController)
	var registrationHandler *authregistration.Handler
	var googleOIDCHandler *googleoidc.Handler
	var phantomAuthHandler *phantomauth.Handler
	walletSecretSecureCookie := false
	walletSecretPublicOrigin := ""
	if !opts.DisableAuth {
		googleOIDCConfig, err := googleoidc.LoadConfigFromEnv(opts.BaseHRef)
		if initErr := err; initErr != nil {
			return nil, initErr
		}
		walletSecretSecureCookie = googleOIDCConfig.SecureCookie()
		walletSecretPublicOrigin = googleOIDCConfig.PublicOrigin()
		externalAuth, err := newExternalAuthBackend(credentialMgr, sessionMgr, settings.UserSessionDuration)
		if initErr := err; initErr != nil {
			return nil, initErr
		}
		registrationStore, err := authregistration.NewStore(opts.RedisClient)
		if initErr := err; initErr != nil {
			return nil, initErr
		}
		registrationHandler, err = authregistration.NewHandler(registrationStore, externalAuth, opts.BaseHRef, googleOIDCConfig.SecureCookie())
		if initErr := err; initErr != nil {
			return nil, initErr
		}
		googleOIDCHandler, err = googleoidc.NewHandler(
			googleOIDCConfig,
			opts.RedisClient,
			externalAuth,
			registrationHandler,
			opts.BaseHRef,
		)
		if initErr := err; initErr != nil {
			return nil, initErr
		}
		phantomAuthHandler, err = phantomauth.NewHandler(opts.RedisClient, externalAuth, registrationHandler, googleOIDCConfig.PublicOrigin(), opts.BaseHRef)
		if initErr := err; initErr != nil {
			return nil, initErr
		}
	}
	walletSecretMgr, err := walletsecret.NewManager(opts.RedisClient, opts.BaseHRef, walletSecretSecureCookie)
	if err != nil {
		return nil, err
	}
	wormCredentialMgr, err := walletsecret.NewWormCredentialManager(opts.RedisClient, opts.BaseHRef, walletSecretSecureCookie)
	if err != nil {
		return nil, err
	}

	// static assets
	staticFS, err := fs.Sub(ui.Embedded, "dist/app")
	if err != nil {
		return nil, err
	}

	root, err := os.OpenRoot(opts.StaticAssetsDir)
	if err != nil {
		if os.IsNotExist(err) {
			log.Warnf("Static assets directory %q does not exist, using only embedded assets", opts.StaticAssetsDir)
		} else {
			if initErr := err; initErr != nil {
				return nil, initErr
			}
		}
	} else {
		defer func() {
			if !initialized {
				_ = root.Close()
			}
		}()
		staticFS = utilio.NewComposableFS(staticFS, root.FS())
	}

	logger := log.NewEntry(log.StandardLogger())

	noopShutdown := func() {
		log.Error("API Server Shutdown function called but server is not started yet.")
	}

	a := &AthenaServer{
		processCancel:            processCancel,
		staticRoot:               root,
		AthenaServerOpts:         opts,
		log:                      logger,
		settings:                 settings,
		sessionMgr:               sessionMgr,
		settingsMgr:              settingsMgr,
		credentialMgr:            credentialMgr,
		accountStateStore:        accountStateStore,
		moduleAccessStore:        accountStateStore,
		accountCenter:            accountCenter,
		accessController:         accessController,
		authRegistration:         registrationHandler,
		googleOIDC:               googleOIDCHandler,
		phantomAuth:              phantomAuthHandler,
		walletSecretMgr:          walletSecretMgr,
		wormCredentialMgr:        wormCredentialMgr,
		walletSecretPublicOrigin: walletSecretPublicOrigin,
		developmentAccountIDs:    developmentAccountIDs,
		userStateStorage:         userStateStorage,
		staticAssets:             http.FS(staticFS),
		Shutdown:                 noopShutdown,
		stopCh:                   make(chan os.Signal, 1),
	}
	if googleOIDCHandler != nil {
		if initErr := googleOIDCHandler.EnableWalletSecretReauthentication(opts.RedisClient, a.authenticateWalletSecretHTTP, credentialMgr, walletSecretMgr); initErr != nil {
			return nil, initErr
		}
		if initErr := googleOIDCHandler.EnableWormCredentialReauthentication(opts.RedisClient, a.authenticateWormVerificationHTTP, a.admitWormAccess, credentialMgr, wormCredentialMgr); initErr != nil {
			return nil, initErr
		}
	}
	if phantomAuthHandler != nil {
		if initErr := phantomAuthHandler.EnableWalletSecretReauthentication(opts.RedisClient, a.authenticateWalletSecretHTTP, credentialMgr, walletSecretMgr); initErr != nil {
			return nil, initErr
		}
		if initErr := phantomAuthHandler.EnableWormCredentialReauthentication(opts.RedisClient, a.authenticateWormVerificationHTTP, a.admitWormAccess, credentialMgr, wormCredentialMgr); initErr != nil {
			return nil, initErr
		}
	}
	if initErr := a.enableWormExecutionAuthorization(); initErr != nil {
		return nil, initErr
	}
	if initErr := a.enableWormPositionCashOutAuthorization(); initErr != nil {
		return nil, initErr
	}
	if initErr := a.enableWormPositionCashOutBatchAuthorization(); initErr != nil {
		return nil, initErr
	}
	walletSecretHTTP, err := newWalletSecretHTTPHandler(a)
	if err != nil {
		return nil, err
	}
	a.walletSecretHTTP = walletSecretHTTP
	accountAvatarHTTP, err := newAccountAvatarHandler(ctx, a)
	if err != nil {
		return nil, err
	}
	a.accountAvatarHTTP = accountAvatarHTTP

	walletAvatarHTTP, err := newWalletAvatarHandler(ctx, a)
	if err != nil {
		return nil, err
	}
	a.walletAvatarHTTP = walletAvatarHTTP

	if a.TraderSyncClient == nil {
		a.TraderSyncClient, a.traderSyncCloser = newTraderSyncClient(os.LookupEnv)
	}
	a.processWorkers.Add(2)
	go func() { defer a.processWorkers.Done(); accountAvatarHTTP.RunGarbageCollector(ctx, 0, 0) }()
	go func() { defer a.processWorkers.Done(); walletAvatarHTTP.RunGarbageCollector(ctx, 0, 0) }()
	if !opts.DisableAuth {
		a.userStateStorage.Init(ctx)
	}
	initialized = true
	return a, nil

}

func isLoopbackListenHost(host string) bool {
	host = strings.Trim(strings.TrimSpace(host), "[]")
	if strings.EqualFold(host, "localhost") {
		return true
	}
	address := net.ParseIP(host)
	return address != nil && address.IsLoopback()
}

// Close releases process-lifetime resources owned by the API server. It is
// intentionally separate from Run shutdown because Run may be invoked again
// during an in-process graceful restart.
func (server *AthenaServer) Close() error {
	if server == nil {
		return nil
	}
	if server.processCancel != nil {
		server.processCancel()
	}
	var result error
	if server.traderSyncCloser != nil {
		result = server.traderSyncCloser.Close()
		server.traderSyncCloser = nil
	}
	server.processWorkers.Wait()
	if server.staticRoot != nil {
		result = errors.Join(result, server.staticRoot.Close())
		server.staticRoot = nil
	}
	if server.accountStateStore != nil {
		result = errors.Join(result, server.accountStateStore.Close())
	}
	return result
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
	sOpts = append(sOpts, grpc.ChainStreamInterceptor(
		logging.StreamServerInterceptor(grpc_util.InterceptorLogger(server.log)),
		server.streamAuthInterceptor,
		// grpc_util.UserAgentStreamServerInterceptor(common.AthenaUserAgentName, clientConstraint),
		// grpc_util.PayloadStreamServerInterceptor(server.log, true, func(_ context.Context, c interceptors.CallMeta) bool {
		// 	return !sensitiveMethods[c.FullMethod()]
		// }),
		// grpc_util.ErrorCodeK8sStreamServerInterceptor(),
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
		recovery.UnaryServerInterceptor(recovery.WithRecoveryHandler(grpc_util.LoggerRecoveryHandler(server.log))),
	))
	sOpts = append(sOpts, grpc.StatsHandler(otelgrpc.NewServerHandler()))
	// create a new gRPC server
	grpcS := grpc.NewServer(sOpts...)

	// register all the services to the gRPC server
	grpc_health_v1.RegisterHealthServer(grpcS, server.serviceSet.HealthService)
	versionpkg.RegisterVersionServiceServer(grpcS, server.serviceSet.VersionService)
	sessionpkg.RegisterSessionServiceServer(grpcS, server.serviceSet.SessionService)
	appbootstrappkg.RegisterAppBootstrapServiceServer(grpcS, server.serviceSet.AppBootstrapService)
	accountpkg.RegisterAccountServiceServer(grpcS, server.serviceSet.AccountService)
	moduleaccesspkg.RegisterModuleAccessServiceServer(grpcS, server.serviceSet.ModuleAccessService)
	notificationpkg.RegisterNotificationServiceServer(grpcS, server.serviceSet.NotificationService)
	tspkg.RegisterTraderSyncServiceServer(grpcS, server.serviceSet.TraderSyncService)
	walletpkg.RegisterWalletServiceServer(grpcS, server.serviceSet.WalletService)
	marketradarpkg.RegisterMarketRadarServiceServer(grpcS, server.serviceSet.MarketRadarService)
	managedoopkg.RegisterManagedOOServiceServer(grpcS, server.serviceSet.ManagedOOService)
	wormtradingpkg.RegisterWormTradingServiceServer(grpcS, server.serviceSet.WormTradingService)
	profitsharingpkg.RegisterProfitSharingServiceServer(grpcS, server.serviceSet.ProfitSharingService)
	tokenapipkg.RegisterTokenCatalogServiceServer(grpcS, server.serviceSet.TokenServices)
	tokenapipkg.RegisterTokenCollectionServiceServer(grpcS, server.serviceSet.TokenServices)
	tokenapipkg.RegisterTokenPolicyServiceServer(grpcS, server.serviceSet.TokenServices)
	tokenapipkg.RegisterTokenOperationsServiceServer(grpcS, server.serviceSet.TokenServices)
	solanapkg.RegisterSolanaServiceServer(grpcS, server.serviceSet.SolanaService)
	servicestatuspkg.RegisterServiceStatusServiceServer(grpcS, server.serviceSet.ServiceStatusService)

	// Register reflection service on gRPC server.
	reflection.Register(grpcS)
	// errorsutil.CheckError(server.serviceSet.ProjectService.NormalizeProjs())

	return grpcS
}

type AthenaServiceSet struct {
	ModuleAccessService  *servermoduleaccess.Server
	TraderSyncService    *servertradersync.Server
	HealthService        *health.Server
	SessionService       *session.Server
	AppBootstrapService  *serverappbootstrap.Server
	AccountService       *account.Server
	VersionService       *version.Server
	NotificationService  *servernotification.Server
	WalletService        *serverwallet.Server
	MarketRadarService   *servermarketradar.Server
	ManagedOOService     *servermanagedoo.Server
	WormTradingService   *serverwormtrading.Server
	ProfitSharingService *serverprofitsharing.Server
	TokenServices        *servertokenapi.Server
	SolanaService        *serversolana.Server
	ServiceStatusService *serverservicestatus.Server
}

func newAthenaServiceSet(server *AthenaServer) *AthenaServiceSet {
	// session service
	sessionService := session.NewServer(server, server.accessController, server.accountCenter, server.credentialMgr)

	settingsProjector := settings.NewProjector(server.settingsMgr)
	appBootstrapService := serverappbootstrap.NewServer(settingsProjector, server.accessController, server.accountCenter, server.credentialMgr, server)
	// account service
	accountService := account.NewServer(server.credentialMgr, server.accessController, server.accountCenter, server.accountStateStore)
	// notification service
	notificationService := servernotification.NewServer(server.NotificationClientset)
	// wallet service
	walletService := serverwallet.NewServer(server.WalletClientset, server.walletAvatarHTTP.DeleteObjectBestEffort)
	marketRadarService := servermarketradar.NewServer(server.MarketRadarClientset)
	managedOOService := servermanagedoo.NewServer(server.ManagedOOClientset)
	wormTradingService := serverwormtrading.NewServer(server.WalletClientset, server.WormTradingClientset)
	profitSharingService := serverprofitsharing.NewServer(server.ProfitSharingClientset, server.credentialMgr, server.accessController, server.accountCenter)
	// token api service
	tokenAPIService := servertokenapi.NewServer(server.TokenAPIClientset)
	var solanaClient solanapkg.SolanaServiceClient
	if server.SolanaClientset != nil {
		solanaClient = server.SolanaClientset.Solana()
	}
	solanaService := serversolana.NewServer(solanaClient)
	serviceStatusService := serverservicestatus.NewServer(
		server.NotificationClientset,
		server.WalletClientset,
		server.MarketRadarClientset,
		server.ManagedOOClientset,
		server.WormTradingClientset,
		server.ProfitSharingClientset,
		server.TokenAPIClientset,
		server.EtherscanGatewayIPs,
		server.EtherscanGatewayToken,
		server.EtherscanAPIKeys,
		server.EtherscanGatewayProbeQueryAddress,
	)

	// certificateService := certificate.NewServer(a.db, a.enf)
	// gpgkeyService := gpgkey.NewServer(a.db, a.enf)
	versionService := version.NewServer(server, func() (bool, error) {
		return server.DisableAuth, nil
	})
	healthService := health.NewServer()

	return &AthenaServiceSet{
		TraderSyncService:    servertradersync.New(server.TraderSyncClient, server.resolveTraderSyncActor),
		HealthService:        healthService,
		SessionService:       sessionService,
		AppBootstrapService:  appBootstrapService,
		AccountService:       accountService,
		ModuleAccessService:  servermoduleaccess.NewServer(server.moduleAccessStore),
		VersionService:       versionService,
		NotificationService:  notificationService,
		WalletService:        walletService,
		MarketRadarService:   marketRadarService,
		ManagedOOService:     managedOOService,
		WormTradingService:   wormTradingService,
		ProfitSharingService: profitSharingService,
		TokenServices:        tokenAPIService,
		SolanaService:        solanaService,
		ServiceStatusService: serviceStatusService,
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

// translateGRPCResponseHeaders applies HTTP-only response headers at the gateway boundary.
func (server *AthenaServer) translateGRPCResponseHeaders(_ context.Context, w http.ResponseWriter, resp golang_proto.Message) error {
	switch resp.(type) {
	case *appbootstrappkg.GetAppBootstrapResponse, *moduleaccesspkg.ListModuleAccessStatesResponse, *moduleaccesspkg.ListModuleAccessSettingsResponse, *moduleaccesspkg.UpdateModuleAccessSettingResponse:
		w.Header().Set("Cache-Control", "no-store, private")
		w.Header().Set("Vary", "Cookie, Authorization, "+common.ApplicationRealmHeader)
	case *walletpkg.BatchCreateWalletsResponse:
		w.Header().Set("Cache-Control", "no-store, private")
		w.Header().Set("Pragma", "no-cache")
		w.Header().Set("Vary", "Cookie, Authorization, "+common.ApplicationRealmHeader)
	case *wormtradingpkg.ListWalletBalancesResponse, *wormtradingpkg.ListWalletTradingActivityResponse:
		w.Header().Set("Cache-Control", "no-store, private")
		w.Header().Set("Vary", "Cookie, Authorization, "+common.ApplicationRealmHeader)
	}
	return nil
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

func normalizeBaseHRef(value string) string {
	trimmed := strings.Trim(strings.TrimSpace(value), "/")
	if trimmed == "" {
		return "/"
	}
	return "/" + trimmed + "/"
}

func applicationBaseHRef(deploymentBaseHRef string, application uiApplication) string {
	if application == adminApplication {
		return normalizeBaseHRef(deploymentBaseHRef) + "admin/"
	}
	return normalizeBaseHRef(deploymentBaseHRef)
}

func replaceRuntimeBaseHRefs(data string, applicationBase string, deploymentBase string) string {
	data = baseHRefRegex.ReplaceAllLiteralString(data, fmt.Sprintf(`<base href="%s">`, applicationBase))
	return deploymentBaseHRefRegex.ReplaceAllLiteralString(data, fmt.Sprintf(`<meta name="athena-deployment-base-href" content="%s">`, deploymentBase))
}

func applicationForPath(requestPath string) uiApplication {
	if requestPath == "/admin" || strings.HasPrefix(requestPath, "/admin/") {
		return adminApplication
	}
	return memberApplication
}

func (server *AthenaServer) getIndexData(application uiApplication) ([]byte, error) {
	cache := &server.memberIndexData
	indexPath := "dist/app/index.html"
	if application == adminApplication {
		cache = &server.adminIndexData
		indexPath = "dist/app/admin/index.html"
	}

	cache.init.Do(func() {
		data, err := ui.Embedded.ReadFile(indexPath)
		if err != nil {
			cache.err = err
			return
		}
		deploymentBase := normalizeBaseHRef(server.BaseHRef)
		applicationBase := applicationBaseHRef(deploymentBase, application)
		cache.data = []byte(replaceRuntimeBaseHRefs(string(data), applicationBase, deploymentBase))
	})

	return cache.data, cache.err
}

func isUIStaticPath(requestPath string) bool {
	return requestPath == "/fonts.css" ||
		strings.HasPrefix(requestPath, "/assets/") ||
		strings.HasPrefix(requestPath, "/images/")
}

func isReservedHTTPNamespace(requestPath string) bool {
	return requestPath == "/api" || strings.HasPrefix(requestPath, "/api/") ||
		requestPath == "/auth" || strings.HasPrefix(requestPath, "/auth/")
}

func isRetiredDocumentationPath(requestPath string) bool {
	return requestPath == "/swagger-ui" || strings.HasPrefix(requestPath, "/swagger-ui/") ||
		requestPath == "/swagger.json" ||
		requestPath == "/llms.txt" ||
		requestPath == "/docs/ai" || strings.HasPrefix(requestPath, "/docs/ai/") ||
		requestPath == "/assets/scripts/redoc.standalone.js" ||
		requestPath == "/assets/scripts/redoc-LICENSE.txt" ||
		requestPath == "/assets/scripts/README.md"
}

func isRetiredDocumentationRequestPath(requestPath string) bool {
	if isRetiredDocumentationPath(requestPath) {
		return true
	}
	if requestPath == "/athena" {
		return false
	}
	return strings.HasPrefix(requestPath, "/athena/") && isRetiredDocumentationPath(strings.TrimPrefix(requestPath, "/athena"))
}

// newStaticAssetsHandler returns an HTTP handler to serve UI static assets
func (server *AthenaServer) newStaticAssetsHandler() func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		if isRetiredDocumentationRequestPath(r.URL.Path) {
			http.NotFound(w, r)
			return
		}

		acceptHTML := false
		for _, acceptType := range strings.Split(r.Header.Get("Accept"), ",") {
			mediaType := strings.TrimSpace(strings.SplitN(acceptType, ";", 2)[0])
			if mediaType == "text/html" || mediaType == "html" {
				acceptHTML = true
				break
			}
		}

		application := applicationForPath(r.URL.Path)
		indexRequest := r.URL.Path == "/index.html" || r.URL.Path == "/admin/index.html"
		fileRequest := !indexRequest && server.uiAssetExists(r.URL.Path)
		fallbackExcludedRequest := isUIStaticPath(r.URL.Path) || isReservedHTTPNamespace(r.URL.Path)

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
		if acceptHTML && !fileRequest && !fallbackExcludedRequest && (r.Method == http.MethodGet || r.Method == http.MethodHead) {
			for k, v := range noCacheHeaders {
				w.Header().Set(k, v)
			}
			data, err := server.getIndexData(application)
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
			if strings.HasPrefix(r.URL.Path, "/assets/") {
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
	publicHandlers := map[string]http.Handler{
		common.LogoutEndpoint: logout.NewHandler(server.settingsMgr, server.sessionMgr, server.walletSecretMgr, server.wormCredentialMgr, server.RootPath, server.BaseHRef),
	}
	if server.googleOIDC != nil {
		publicHandlers["/auth/google/login"] = http.HandlerFunc(server.googleOIDC.Login)
		publicHandlers["/auth/google/callback"] = http.HandlerFunc(server.googleOIDC.Callback)
		publicHandlers["/auth/wallet-secrets/google"] = http.HandlerFunc(server.googleOIDC.WalletSecretReauthentication)
		publicHandlers["/auth/worm-trading/google"] = http.HandlerFunc(server.googleOIDC.WormCredentialReauthentication)
		publicHandlers["/auth/worm-trading/executions/google"] = http.HandlerFunc(server.googleOIDC.WormExecutionAuthorization)
		publicHandlers["/auth/worm-trading/position-cash-outs/google"] = http.HandlerFunc(server.googleOIDC.WormPositionCashOutAuthorization)
		publicHandlers["/auth/worm-trading/position-cash-out-batches/google"] = http.HandlerFunc(server.googleOIDC.WormPositionCashOutBatchAuthorization)
	}
	if server.authRegistration != nil {
		publicHandlers["/auth/registration"] = http.HandlerFunc(server.authRegistration.Registration)
		publicHandlers["/auth/registration/username-availability"] = http.HandlerFunc(server.authRegistration.UsernameAvailability)
	}
	if server.phantomAuth != nil {
		publicHandlers["/auth/phantom/challenge"] = http.HandlerFunc(server.phantomAuth.Challenge)
		publicHandlers["/auth/phantom/verify"] = http.HandlerFunc(server.phantomAuth.Verify)
		publicHandlers["/auth/wallet-secrets/solana/challenge"] = http.HandlerFunc(server.phantomAuth.WalletSecretChallenge)
		publicHandlers["/auth/wallet-secrets/solana/verify"] = http.HandlerFunc(server.phantomAuth.WalletSecretVerify)
		publicHandlers["/auth/worm-trading/solana/challenge"] = http.HandlerFunc(server.phantomAuth.WormCredentialChallenge)
		publicHandlers["/auth/worm-trading/solana/verify"] = http.HandlerFunc(server.phantomAuth.WormCredentialVerify)
	}
	if server.DisableAuth {
		publicHandlers["/auth/wallet-secrets/development"] = http.HandlerFunc(server.developmentWalletSecretLease)
		publicHandlers["/auth/worm-trading/development"] = http.HandlerFunc(server.developmentWormCredentialLease)
	}
	httpS := http.Server{
		Addr: endpoint,
		Handler: &handlerSwitcher{
			handler:      mux,
			urlToHandler: publicHandlers,
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
	gwMuxOpts := runtime.WithMarshalerOption(runtime.MIMEWildcard, new(moduleAccessJSONMarshaler))
	gwResponseHeaderOpts := runtime.WithForwardResponseOption(server.translateGRPCResponseHeaders)
	gwApplicationRealmOpts := runtime.WithIncomingHeaderMatcher(applicationRealmHeaderMatcher)
	gwmux := runtime.NewServeMux(gwMuxOpts, gwResponseHeaderOpts, gwApplicationRealmOpts, runtime.WithOutgoingHeaderMatcher(moduleAccessOutgoingHeader))

	var handler http.Handler = adaptApplicationRealmQuery(gwmux)
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
	registerAccountAvatarHandlers(mux, server.accountAvatarHTTP)
	registerWalletAvatarHandlers(mux, server.walletAvatarHTTP)
	registerWalletSecretHandlers(mux, server.walletSecretHTTP)
	registerWormWalletSelectionHandlers(mux, server)
	registerWormConnectionHandlers(mux, server)
	registerWormCombinationHandlers(mux, server)
	registerWormExecutionPlanHandlers(mux, server)
	registerWormExecutionHandlers(mux, server)
	registerWormPositionCashOutHandlers(mux, server)
	registerWormPositionCashOutBatchHandlers(mux, server)
	if server.phantomAuth != nil {
		mux.Handle("POST /auth/worm-trading/executions/{runId}/solana/challenge", traceHTTP(http.HandlerFunc(server.phantomAuth.WormExecutionChallenge)))
		mux.Handle("POST /auth/worm-trading/executions/{runId}/solana/verify", traceHTTP(http.HandlerFunc(server.phantomAuth.WormExecutionVerify)))
		mux.Handle("POST /auth/worm-trading/position-cash-outs/{cashOutId}/solana/challenge", traceHTTP(http.HandlerFunc(server.phantomAuth.WormPositionCashOutChallenge)))
		mux.Handle("POST /auth/worm-trading/position-cash-outs/{cashOutId}/solana/verify", traceHTTP(http.HandlerFunc(server.phantomAuth.WormPositionCashOutVerify)))
		mux.Handle("POST /auth/worm-trading/position-cash-out-batches/{batchId}/solana/challenge", traceHTTP(http.HandlerFunc(server.phantomAuth.WormPositionCashOutBatchChallenge)))
		mux.Handle("POST /auth/worm-trading/position-cash-out-batches/{batchId}/solana/verify", traceHTTP(http.HandlerFunc(server.phantomAuth.WormPositionCashOutBatchVerify)))
	}
	if server.DisableAuth {
		mux.Handle("POST /auth/worm-trading/executions/{runId}/development", traceHTTP(http.HandlerFunc(server.developmentWormExecutionAuthorization)))
		mux.Handle("POST /auth/worm-trading/position-cash-outs/{cashOutId}/development", traceHTTP(http.HandlerFunc(server.developmentWormPositionCashOutAuthorization)))
		mux.Handle("POST /auth/worm-trading/position-cash-out-batches/{batchId}/development", traceHTTP(http.HandlerFunc(server.developmentWormPositionCashOutBatchAuthorization)))
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
	mustRegisterGWHandler(ctx, notificationpkg.RegisterNotificationServiceHandler, gwmux, conn)
	mustRegisterGWHandler(ctx, tspkg.RegisterTraderSyncServiceHandler, gwmux, conn)
	mustRegisterGWHandler(ctx, walletpkg.RegisterWalletServiceHandler, gwmux, conn)
	mustRegisterGWHandler(ctx, marketradarpkg.RegisterMarketRadarServiceHandler, gwmux, conn)
	mustRegisterGWHandler(ctx, managedoopkg.RegisterManagedOOServiceHandler, gwmux, conn)
	mustRegisterGWHandler(ctx, wormtradingpkg.RegisterWormTradingServiceHandler, gwmux, conn)
	mustRegisterGWHandler(ctx, profitsharingpkg.RegisterProfitSharingServiceHandler, gwmux, conn)
	mustRegisterGWHandler(ctx, tokenapipkg.RegisterTokenCatalogServiceHandler, gwmux, conn)
	mustRegisterGWHandler(ctx, tokenapipkg.RegisterTokenCollectionServiceHandler, gwmux, conn)
	mustRegisterGWHandler(ctx, tokenapipkg.RegisterTokenPolicyServiceHandler, gwmux, conn)
	mustRegisterGWHandler(ctx, tokenapipkg.RegisterTokenOperationsServiceHandler, gwmux, conn)
	mustRegisterGWHandler(ctx, solanapkg.RegisterSolanaServiceHandler, gwmux, conn)
	mustRegisterGWHandler(ctx, moduleaccesspkg.RegisterModuleAccessServiceHandler, gwmux, conn)
	mustRegisterGWHandler(ctx, servicestatuspkg.RegisterServiceStatusServiceHandler, gwmux, conn)
	mustRegisterGWHandler(ctx, sessionpkg.RegisterSessionServiceHandler, gwmux, conn)
	mustRegisterGWHandler(ctx, appbootstrappkg.RegisterAppBootstrapServiceHandler, gwmux, conn)
	// mustRegisterGWHandler(ctx, projectpkg.RegisterProjectServiceHandler, gwmux, conn)
	mustRegisterGWHandler(ctx, accountpkg.RegisterAccountServiceHandler, gwmux, conn)
	// mustRegisterGWHandler(ctx, certificatepkg.RegisterCertificateServiceHandler, gwmux, conn)
	// mustRegisterGWHandler(ctx, gpgkeypkg.RegisterGPGKeyServiceHandler, gwmux, conn)

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
	if l == nil {
		return nil
	}
	var result error
	if l.Main != nil {
		if err := l.Main.Close(); err != nil && !errors.Is(err, net.ErrClosed) {
			result = errors.Join(result, err)
		}
		l.Main = nil
	}
	if l.GatewayConn != nil {
		if err := l.GatewayConn.Close(); err != nil && !errors.Is(err, grpc.ErrClientConnClosing) {
			result = errors.Join(result, err)
		}
		l.GatewayConn = nil
	}
	return result
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
func (server *AthenaServer) Run(ctx context.Context, listeners *Listeners) (runErr error) {
	defer func() { runErr = errors.Join(runErr, listeners.Close()) }()
	defer func() {
		if r := recover(); r != nil {
			log.WithField("trace", string(debug.Stack())).Error("Recovered from panic: ", r)
			server.terminateRequested.Store(true)
			runErr = fmt.Errorf("API server panic: %v", r)
			server.Shutdown()
		}
	}()

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

	go func() { server.checkServeErr("gRPC server", grpcS.Serve(grpcL)) }()
	go func() { server.checkServeErr("HTTP server", httpS.Serve(httpL)) }()
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
	return runErr
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
	// If authentication is disabled, present the request as the local
	// development identity for its explicit application realm. Authorization
	// still evaluates that identity's persisted access.
	if server.DisableAuth {
		developmentAccountID, err := server.developmentAccountIDFromIncomingContext(ctx)
		if err != nil {
			return ctx, err
		}
		ctx = withDisabledAuthClaims(ctx, developmentAccountID)
		access, err := server.accessController.Get(developmentAccountID)
		if err != nil {
			return ctx, err
		}
		if !access.LoginEnabled {
			ctx = context.WithValue(ctx, util_session.AuthErrorCtxKey, util_session.AccountMaintenanceErr) //nolint:staticcheck
			return ctx, util_session.AccountMaintenanceErr
		}
		credential, ok := util_session.AuthenticatedCredentialFromContext(ctx)
		if !ok {
			return ctx, status.Error(codes.Internal, "development credential is not configured")
		}
		credential.AccessRevision = access.Revision
		return util_session.WithAuthenticatedCredential(ctx, credential), nil
	}

	claims, credential, _, claimsErr := server.getClaims(ctx)
	if claims != nil {
		// Add claims to the context for account authorization.
		//nolint:staticcheck
		ctx = context.WithValue(ctx, "claims", claims) // ctx {data:data, claims:claims}
	}
	if credential.AccountID != "" {
		ctx = util_session.WithAuthenticatedCredential(ctx, credential)
	}
	if claimsErr != nil {
		//nolint:staticcheck
		ctx = context.WithValue(ctx, util_session.AuthErrorCtxKey, claimsErr) // ctx {data:data, auth-error:claimsErr}
	}

	return ctx, claimsErr
}

// getClaims extracts and validates a JWT token from an incoming request context.
func (server *AthenaServer) getClaims(ctx context.Context) (jwt.Claims, accountcredentials.AuthenticatedCredential, string, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, accountcredentials.AuthenticatedCredential{}, "", ErrNoSession
	}
	tokenString, cookieRealm, realmBound, tokenErr := getToken(md)
	if tokenErr != nil {
		return nil, accountcredentials.AuthenticatedCredential{}, "", tokenErr
	}
	if tokenString == "" {
		return nil, accountcredentials.AuthenticatedCredential{}, "", ErrNoSession
	}
	claims, credential, err := server.sessionMgr.AuthenticateToken(tokenString)
	if err != nil {
		if util_session.IsAccountMaintenanceError(err) {
			return claims, credential, "", err
		}
		return claims, credential, "", status.Errorf(codes.Unauthenticated, "invalid session: %v", err)
	}
	if realmBound {
		account, err := server.credentialMgr.Get(credential.AccountID)
		if err != nil || account.ApplicationRealm() != cookieRealm {
			return nil, accountcredentials.AuthenticatedCredential{}, "", status.Error(codes.Unauthenticated, "session does not match the application realm")
		}
	}

	return claims, credential, "", nil
}

// getToken extracts the token from gRPC metadata or cookie headers
func getToken(md metadata.MD) (string, accountcredentials.ApplicationRealm, bool, error) {
	// check the "token" metadata
	{
		tokens, ok := md[apiclient.MetaDataTokenKey]
		if ok && len(tokens) > 0 {
			return tokens[0], "", false, nil
		}
	}

	// looks for the HTTP header `Authorization: Bearer ...`
	// athena prefers bearer token over cookie
	for _, t := range md["authorization"] {
		token := strings.TrimPrefix(t, "Bearer ")
		if strings.HasPrefix(t, "Bearer ") && jwtutil.IsValid(token) {
			return token, "", false, nil
		}
	}

	cookieHeaders := md["grpcgateway-cookie"]
	if len(cookieHeaders) == 0 {
		return "", "", false, nil
	}
	realm, present, err := parseApplicationRealmValues(md.Get(common.ApplicationRealmHeader))
	if err != nil {
		return "", "", false, err
	}
	if !present {
		return "", "", false, applicationRealmRequiredError()
	}
	cookieName, err := httputil.RealmAuthCookieName(realm)
	if err != nil {
		return "", "", false, status.Error(codes.Unauthenticated, "application realm is invalid")
	}

	// check the HTTP cookie
	for _, t := range cookieHeaders {
		header := http.Header{}
		header.Add("Cookie", t)
		request := http.Request{Header: header}
		token, err := httputil.JoinCookies(cookieName, request.Cookies())
		if err == nil && jwtutil.IsValid(token) {
			return token, realm, true, nil
		}
	}

	return "", realm, true, nil
}
