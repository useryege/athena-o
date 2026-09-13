package commands

import (
	"context"
	"fmt"
	"runtime/debug"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/spf13/cobra"

	log "github.com/sirupsen/logrus"

	cmdutil "github.com/useryege/athena/cmd/util"
	"github.com/useryege/athena/common"
	managedooapiclient "github.com/useryege/athena/internal/managedoo/apiclient"
	marketradarapiclient "github.com/useryege/athena/internal/marketradar/apiclient"
	notificationapiclient "github.com/useryege/athena/internal/notification/apiclient"
	profitsharingapiclient "github.com/useryege/athena/internal/profitsharing/apiclient"
	"github.com/useryege/athena/internal/server"
	servercache "github.com/useryege/athena/internal/server/cache"
	sportshistoryapiclient "github.com/useryege/athena/internal/sportshistory/apiclient"
	sportsliveapiclient "github.com/useryege/athena/internal/sportslive/apiclient"
	tokenapiapiclient "github.com/useryege/athena/internal/tokenapi/apiclient"
	walletapiclient "github.com/useryege/athena/internal/wallet/apiclient"
	wormmarketsapiclient "github.com/useryege/athena/internal/wormmarkets/apiclient"
	wormtradingapiclient "github.com/useryege/athena/internal/wormtrading/apiclient"
	"github.com/useryege/athena/pkg/stats"
	cacheutil "github.com/useryege/athena/util/cache"
	"github.com/useryege/athena/util/cli"
	"github.com/useryege/athena/util/env"
	"github.com/useryege/athena/util/errors"
	utilio "github.com/useryege/athena/util/io"
	"github.com/useryege/athena/util/templates"
	traceutil "github.com/useryege/athena/util/trace"
)

const (
	// cliName is the name of the CLI
	cliName = "athena-server"
)

// NewCommand returns a new instance of an athena command
func NewCommand() *cobra.Command {
	var (
		staticAssetsDir            string
		baseHRef                   string
		rootPath                   string
		glogLevel                  int
		disableAuth                bool
		contentTypes               string
		enableGZip                 bool
		listenHost                 string
		listenPort                 int
		otlpAddress                string
		otlpInsecure               bool
		otlpHeaders                map[string]string
		otlpAttrs                  []string
		frameOptions               string
		contentSecurityPolicy      string
		notificationServerAddress  string
		walletServerAddress        string
		marketRadarServerAddress   string
		sportsLiveServerAddress    string
		sportsHistoryServerAddress string
		managedOOServerAddress     string
		wormMarketsServerAddress   string
		wormTradingServerAddress   string
		profitSharingServerAddress string
		tokenAPIServerAddress      string
		etherscanGatewayIPs        string
		etherscanGatewayAuthToken  string
		etherscanAPIKeys           string
		etherscanProbeQueryAddress string
		// hydratorEnabled        bool
		// syncWithReplaceAllowed bool

		redisClient *redis.Client
		cacheSrc    func() (*servercache.Cache, error)
	)
	command := &cobra.Command{
		Use:   cliName,
		Short: "Run the Athena API server",
		Long: "The API server is a gRPC/REST server which exposes the API consumed by the Web UI, CLI, and CI/CD systems. " +
			"This command runs API server in the foreground. It can be configured by following options.\n\n" +
			"ATHENA_WALLET_INTERNAL_AUTH_TOKEN and ATHENA_WORM_TRADING_INTERNAL_AUTH_TOKEN must each contain at least 32 bytes without whitespace " +
			"and match their target service. They are independent internal credentials, not Athena user API Keys, and the server refuses startup when either is absent or invalid.",
		DisableAutoGenTag: true,
		RunE: func(c *cobra.Command, _ []string) error {
			ctx := c.Context()

			// Log the startup information
			vers := common.GetVersion()

			vers.LogStartupInfo(
				"Athena API Server",
				map[string]any{
					"port": listenPort,
				},
			)

			// Set the log format and level
			cli.SetLogFormat(cmdutil.LogFormat)
			cli.SetLogLevel(cmdutil.LogLevel)
			cli.SetGLogLevel(glogLevel)

			// Recover from panic and log the error using the configured logger instead of the default.
			defer func() {
				if r := recover(); r != nil {
					log.WithField("trace", string(debug.Stack())).Fatal("Recovered from panic: ", r)
				}
			}()

			cache, err := cacheSrc()
			errors.CheckError(err)

			log.Infof("athena-server/%s (%s)", vers.Version, vers.Platform)

			var contentTypesList []string
			if contentTypes != "" {
				contentTypesList = strings.Split(contentTypes, ";")
			}

			notificationclientset, err := notificationapiclient.NewNotificationClientset(
				notificationServerAddress,
				env.StringFromEnv(notificationapiclient.InternalAuthTokenEnv, ""),
			)
			if err != nil {
				return fmt.Errorf("create notification clientset: %w", err)
			}
			defer utilio.Close(notificationclientset)
			walletclientset, err := walletapiclient.NewWalletClientset(
				walletServerAddress,
				env.StringFromEnv(walletapiclient.InternalAuthTokenEnv, ""),
			)
			if err != nil {
				return fmt.Errorf("create Wallet clientset: %w", err)
			}
			defer utilio.Close(walletclientset)
			marketRadarClientset, err := marketradarapiclient.NewMarketRadarClientset(marketRadarServerAddress)
			if err != nil {
				return fmt.Errorf("create Market Radar clientset: %w", err)
			}
			defer utilio.Close(marketRadarClientset)
			sportsLiveClientset, err := sportsliveapiclient.NewSportsLiveClientset(sportsLiveServerAddress)
			if err != nil {
				return fmt.Errorf("create Sports Live clientset: %w", err)
			}
			defer utilio.Close(sportsLiveClientset)
			sportsHistoryClientset, err := sportshistoryapiclient.NewSportsHistoryClientset(sportsHistoryServerAddress)
			if err != nil {
				return fmt.Errorf("create Sports History clientset: %w", err)
			}
			defer utilio.Close(sportsHistoryClientset)
			managedOOClientset, err := managedooapiclient.NewManagedOOClientset(managedOOServerAddress)
			if err != nil {
				return fmt.Errorf("create Managed OO clientset: %w", err)
			}
			defer utilio.Close(managedOOClientset)
			wormMarketsClientset, err := wormmarketsapiclient.NewWormMarketsClientset(wormMarketsServerAddress)
			if err != nil {
				return fmt.Errorf("create Worm Markets clientset: %w", err)
			}
			defer utilio.Close(wormMarketsClientset)
			wormTradingClientset, err := wormtradingapiclient.NewWormTradingClientset(
				wormTradingServerAddress,
				env.StringFromEnv(wormtradingapiclient.InternalAuthTokenEnv, ""),
			)
			if err != nil {
				return fmt.Errorf("create Worm Trading clientset: %w", err)
			}
			defer utilio.Close(wormTradingClientset)
			profitSharingClientset, err := profitsharingapiclient.NewProfitSharingClientset(profitSharingServerAddress)
			if err != nil {
				return fmt.Errorf("create Profit Sharing clientset: %w", err)
			}
			defer utilio.Close(profitSharingClientset)
			tokenAPIClientset, err := tokenapiapiclient.NewTokenAPIClientset(tokenAPIServerAddress)
			if err != nil {
				return fmt.Errorf("create Token API clientset: %w", err)
			}
			defer utilio.Close(tokenAPIClientset)

			athenaOpts := server.AthenaServerOpts{
				ContentTypes:                      contentTypesList,
				ListenPort:                        listenPort,
				ListenHost:                        listenHost,
				StaticAssetsDir:                   staticAssetsDir,
				BaseHRef:                          baseHRef,
				RootPath:                          rootPath,
				DisableAuth:                       disableAuth,
				EnableGZip:                        enableGZip,
				XFrameOptions:                     frameOptions,
				ContentSecurityPolicy:             contentSecurityPolicy,
				RedisClient:                       redisClient,
				Cache:                             cache,
				NotificationClientset:             notificationclientset,
				WalletClientset:                   walletclientset,
				MarketRadarClientset:              marketRadarClientset,
				SportsLiveClientset:               sportsLiveClientset,
				SportsHistoryClientset:            sportsHistoryClientset,
				ManagedOOClientset:                managedOOClientset,
				WormMarketsClientset:              wormMarketsClientset,
				WormTradingClientset:              wormTradingClientset,
				ProfitSharingClientset:            profitSharingClientset,
				TokenAPIClientset:                 tokenAPIClientset,
				EtherscanGatewayIPs:               etherscanGatewayIPs,
				EtherscanGatewayToken:             etherscanGatewayAuthToken,
				EtherscanAPIKeys:                  etherscanAPIKeys,
				EtherscanGatewayProbeQueryAddress: etherscanProbeQueryAddress,
				// HydratorEnabled:        hydratorEnabled,
				// SyncWithReplaceAllowed: syncWithReplaceAllowed,
			}

			// Register stack dumper and start stats ticker and heap dumper
			stats.RegisterStackDumper()
			stats.StartStatsTicker(10 * time.Minute)
			stats.RegisterHeapDumper("memprofile")

			// Initialize the Athena server
			athena, err := server.NewServer(ctx, athenaOpts)
			if err != nil {
				return err
			}
			defer utilio.Close(athena)
			athena.Init(ctx)

			for {
				var closer func()
				serverCtx, cancel := context.WithCancel(ctx)
				lns, err := athena.Listen()
				if err != nil {
					cancel()
					return err
				}
				if otlpAddress != "" {
					closer, err = traceutil.InitTracer(serverCtx, "athena-server", otlpAddress, otlpInsecure, otlpHeaders, otlpAttrs)
					if err != nil {
						cancel()
						_ = lns.Close()
						return fmt.Errorf("failed to initialize tracing: %w", err)
					}
				}
				runErr := athena.Run(serverCtx, lns)
				if closer != nil {
					closer()
				}
				cancel()
				if runErr != nil {
					return runErr
				}
				if athena.TerminateRequested() {
					break
				}
			}
			return nil
		},
		Example: templates.Examples(`
			# Start the Athena API server with default settings
			$ athena-server

			# Start the Athena API server on a custom port and enable tracing
			$ athena-server --port 8888 --otlp-address localhost:4317
		`),
	}

	command.Flags().StringVar(&staticAssetsDir, "staticassets", env.StringFromEnv("ATHENA_SERVER_STATIC_ASSETS", "/shared/app"), "Directory path that contains additional static assets")
	command.Flags().StringVar(&baseHRef, "basehref", env.StringFromEnv("ATHENA_SERVER_BASEHREF", "/"), "Value for base href in index.html. Used if Athena is running behind reverse proxy under subpath different from /")
	command.Flags().StringVar(&rootPath, "rootpath", env.StringFromEnv("ATHENA_SERVER_ROOTPATH", ""), "Used if Athena is running behind reverse proxy under subpath different from /")
	command.Flags().StringVar(&cmdutil.LogFormat, "logformat", env.StringFromEnv(common.EnvLogFormat, "json"), "Set the logging format. One of: json|text")
	command.Flags().StringVar(&cmdutil.LogLevel, "loglevel", env.StringFromEnv(common.EnvLogLevel, "info"), "Set the logging level. One of: debug|info|warn|error")
	command.Flags().IntVar(&glogLevel, "gloglevel", 0, "Set the glog logging level")
	command.Flags().BoolVar(&disableAuth, "disable-auth", env.ParseBoolFromEnv("ATHENA_SERVER_DISABLE_AUTH", false), "Disable client authentication")
	command.Flags().StringVar(&contentTypes, "api-content-types", env.StringFromEnv("ATHENA_API_CONTENT_TYPES", "application/json", env.StringFromEnvOpts{AllowEmpty: true}), "Semicolon separated list of allowed content types for non GET api requests. Any content type is allowed if empty.")
	command.Flags().BoolVar(&enableGZip, "enable-gzip", env.ParseBoolFromEnv("ATHENA_SERVER_ENABLE_GZIP", true), "Enable GZIP compression")
	command.AddCommand(cli.NewVersionCmd(cliName))
	command.Flags().StringVar(&listenHost, "address", env.StringFromEnv("ATHENA_SERVER_LISTEN_ADDRESS", common.DefaultAddressAPIServer), "Listen on given address")
	command.Flags().IntVar(&listenPort, "port", common.DefaultPortAthenaAPIServer, "Listen on given port")
	command.Flags().StringVar(&otlpAddress, "otlp-address", env.StringFromEnv("ATHENA_SERVER_OTLP_ADDRESS", ""), "OpenTelemetry collector address to send traces to")
	command.Flags().BoolVar(&otlpInsecure, "otlp-insecure", env.ParseBoolFromEnv("ATHENA_SERVER_OTLP_INSECURE", true), "OpenTelemetry collector insecure mode")
	command.Flags().StringToStringVar(&otlpHeaders, "otlp-headers", env.ParseStringToStringFromEnv("ATHENA_SERVER_OTLP_HEADERS", map[string]string{}, ","), "List of OpenTelemetry collector extra headers sent with traces, headers are comma-separated key-value pairs(e.g. key1=value1,key2=value2)")
	command.Flags().StringSliceVar(&otlpAttrs, "otlp-attrs", env.StringsFromEnv("ATHENA_SERVER_OTLP_ATTRS", []string{}, ","), "List of OpenTelemetry collector extra attrs when send traces, each attribute is separated by a colon(e.g. key:value)")
	command.Flags().StringVar(&frameOptions, "x-frame-options", env.StringFromEnv("ATHENA_SERVER_X_FRAME_OPTIONS", "sameorigin"), "Set X-Frame-Options header in HTTP responses to `value`. To disable, set to \"\".")
	command.Flags().StringVar(&contentSecurityPolicy, "content-security-policy", env.StringFromEnv("ATHENA_SERVER_CONTENT_SECURITY_POLICY", "frame-ancestors 'self';"), "Set Content-Security-Policy header in HTTP responses to `value`. To disable, set to \"\".")
	command.Flags().StringVar(&notificationServerAddress, "notification-server-address", env.StringFromEnv("ATHENA_NOTIFICATION_SERVER_ADDRESS", fmt.Sprintf("%s:%d", common.DefaultLocalGRPCHost, common.DefaultPortNotification)), "Athena notification server address")
	command.Flags().StringVar(&walletServerAddress, "wallet-server-address", env.StringFromEnv("ATHENA_WALLET_SERVER_ADDRESS", fmt.Sprintf("%s:%d", common.DefaultLocalGRPCHost, common.DefaultPortWallet)), "Athena wallet server address")
	command.Flags().StringVar(&marketRadarServerAddress, "market-radar-server-address", env.StringFromEnv("ATHENA_MARKET_RADAR_SERVER_ADDRESS", fmt.Sprintf("%s:%d", common.DefaultLocalGRPCHost, common.DefaultPortMarketRadar)), "Athena Market Radar server address")
	command.Flags().StringVar(&sportsLiveServerAddress, "sports-live-server-address", env.StringFromEnv("ATHENA_SPORTS_LIVE_SERVER_ADDRESS", fmt.Sprintf("%s:%d", common.DefaultLocalGRPCHost, common.DefaultPortSportsLive)), "Athena Sports Live server address")
	command.Flags().StringVar(&sportsHistoryServerAddress, "sports-history-server-address", env.StringFromEnv("ATHENA_SPORTS_HISTORY_SERVER_ADDRESS", fmt.Sprintf("%s:%d", common.DefaultLocalGRPCHost, common.DefaultPortSportsHistory)), "Athena Sports History server address")
	command.Flags().StringVar(&managedOOServerAddress, "managed-oo-server-address", env.StringFromEnv("ATHENA_MANAGED_OO_SERVER_ADDRESS", fmt.Sprintf("%s:%d", common.DefaultLocalGRPCHost, common.DefaultPortManagedOO)), "Athena Managed OO server address")
	command.Flags().StringVar(&wormMarketsServerAddress, "worm-markets-server-address", env.StringFromEnv("ATHENA_WORM_MARKETS_SERVER_ADDRESS", fmt.Sprintf("%s:%d", common.DefaultLocalGRPCHost, common.DefaultPortWormMarkets)), "Athena Worm Markets server address")
	command.Flags().StringVar(&wormTradingServerAddress, "worm-trading-server-address", env.StringFromEnv("ATHENA_WORM_TRADING_SERVER_ADDRESS", fmt.Sprintf("%s:%d", common.DefaultLocalGRPCHost, common.DefaultPortWormTrading)), "Athena Worm Trading server address")
	command.Flags().StringVar(&profitSharingServerAddress, "profit-sharing-server-address", env.StringFromEnv("ATHENA_PROFIT_SHARING_SERVER_ADDRESS", fmt.Sprintf("%s:%d", common.DefaultLocalGRPCHost, common.DefaultPortProfitSharing)), "Athena Profit Sharing server address")
	command.Flags().StringVar(&tokenAPIServerAddress, "token-api-server-address", env.StringFromEnv("ATHENA_TOKEN_API_SERVER_ADDRESS", fmt.Sprintf("%s:%d", common.DefaultLocalGRPCHost, common.DefaultPortTokenAPI)), "Athena token API server address")
	command.Flags().StringVar(&etherscanGatewayIPs, "etherscan-gateway-ips", env.StringFromEnv("ETHERSCAN_GATEWAY_IPS", ""), "Comma, space, or newline-separated Etherscan Gateway IP addresses")
	command.Flags().StringVar(&etherscanGatewayAuthToken, "etherscan-gateway-auth-token", env.StringFromEnv("ATHENA_ETHERSCAN_GATEWAY_AUTH_TOKEN", ""), "Bearer token for Etherscan Gateway gRPC status calls")
	command.Flags().StringVar(&etherscanAPIKeys, "etherscan-api-keys", env.StringFromEnv("ATHENA_ETHERSCAN_MANAGER_API_KEYS", ""), "Comma, space, or newline-separated Etherscan API keys used by Etherscan Gateway probe runs")
	command.Flags().StringVar(&etherscanProbeQueryAddress, "etherscan-gateway-probe-query-address", env.StringFromEnv("ATHENA_ETHERSCAN_GATEWAY_PROBE_QUERY_ADDRESS", ""), "Ethereum address used by Etherscan Gateway probe runs")
	// command.Flags().BoolVar(&hydratorEnabled, "hydrator-enabled", env.ParseBoolFromEnv("ATHENA_SERVER_HYDRATOR_ENABLED", false), "Feature flag to enable Hydrator. Default (\"false\")")
	// command.Flags().BoolVar(&syncWithReplaceAllowed, "sync-with-replace-allowed", env.ParseBoolFromEnv("ATHENA_SERVER_SYNC_WITH_REPLACE_ALLOWED", true), "Whether to allow users to select replace for syncs from UI/CLI")

	cacheSrc = servercache.AddCacheFlagsToCmd(command, cacheutil.Options{
		OnClientCreated: func(client *redis.Client) {
			redisClient = client
		},
	})

	return command
}
