package devruntime

import (
	"fmt"
	"net"
	"time"
)

type ServiceSpec struct {
	Name, BuildPackage, Binary      string
	Infrastructure, EnvironmentKeys []string
	Args                            []string
	Schemas                         []string
	ListenKey, PortKey, DefaultPort string
	AddressKeys                     []string
	Readiness, HealthService        string
	Core                            bool
	StopLayer                       int
	StartupTimeout, ShutdownTimeout time.Duration
}

var toolEnvironment = []string{"PATH", "HOME", "TMPDIR", "LANG", "LC_ALL", "SSL_CERT_FILE", "SSL_CERT_DIR", "ATHENA_LOCAL_RUNTIME_INSTANCE"}
var dbEnvironment = []string{"ATHENA_ACCOUNT_STATE_POSTGRES_DSN", "ATHENA_URL", "ATHENA_GRPC_MAX_SIZE_MB"}

var loggingEnvironment = []string{"ATHENA_LOGLEVEL", "ATHENA_LOGFORMAT", "ATHENA_LOG_FORMAT_ENABLE_FULL_TIMESTAMP", "ATHENA_LOG_FORMAT_TIMESTAMP", "FORCE_LOG_COLORS", "ATHENA_GRPC_KEEP_ALIVE_MIN"}

var proxyEnvironment = []string{"HTTP_PROXY", "HTTPS_PROXY", "NO_PROXY", "ALL_PROXY", "http_proxy", "https_proxy", "no_proxy", "all_proxy"}

var apiConsumerEnvironment = []string{"ETHERSCAN_GATEWAY_IPS", "ATHENA_ETHERSCAN_GATEWAY_AUTH_TOKEN", "ATHENA_ETHERSCAN_MANAGER_API_KEYS", "ATHENA_ETHERSCAN_GATEWAY_PROBE_QUERY_ADDRESS", "ATHENA_API_CONTENT_TYPES", "ATHENA_SERVER_OTLP_ADDRESS", "ATHENA_SERVER_OTLP_INSECURE", "ATHENA_SERVER_OTLP_HEADERS", "ATHENA_SERVER_OTLP_ATTRS", "ATHENA_SERVER_X_FRAME_OPTIONS", "ATHENA_SERVER_CONTENT_SECURITY_POLICY", "ATHENA_SERVER_CONNECTION_STATUS_CACHE_EXPIRATION", "ATHENA_DEFAULT_CACHE_EXPIRATION", "REDISDB", "REDIS_USERNAME", "REDIS_COMPRESSION", "REDIS_RETRY_COUNT", "REDIS_CREDS_DIR_PATH", "REDIS_SENTINEL_USERNAME", "REDIS_SENTINEL_PASSWORD", "ATHENA_MAX_COOKIE_NUMBER", "ATHENA_ADDITIONAL_URLS", "ATHENA_SESSION_DURATION", "ATHENA_HELP_CHAT_URL", "ATHENA_HELP_CHAT_TEXT", "ATHENA_STATUS_BADGE_ENABLED", "ATHENA_STATUS_BADGE_ROOT_URL", "ATHENA_ANONYMOUS_USER_ENABLED", "ATHENA_UI_CSS_URL", "ATHENA_UI_BANNER_CONTENT", "ATHENA_UI_BANNER_PERMANENT", "ATHENA_UI_BANNER_POSITION", "ATHENA_UI_BANNER_URL", "ATHENA_HELP_DOWNLOAD_DARWIN_AMD64", "ATHENA_HELP_DOWNLOAD_DARWIN_ARM64", "ATHENA_HELP_DOWNLOAD_WINDOWS_AMD64", "ATHENA_HELP_DOWNLOAD_LINUX_AMD64", "ATHENA_HELP_DOWNLOAD_LINUX_ARM64", "ATHENA_HELP_DOWNLOAD_LINUX_PPC64LE", "ATHENA_HELP_DOWNLOAD_LINUX_S390X", "ATHENA_ACCOUNT_AVATAR_MAX_BYTES"}

func keys(groups ...[]string) []string {
	var out []string
	for _, g := range groups {
		out = append(out, g...)
	}
	return out
}
func serviceRegistry() map[string]ServiceSpec {
	rpc := []string{"ATHENA_TRADER_SYNC_INTERNAL_AUTH_TOKEN", "ATHENA_TRADER_SYNC_INTERNAL_AUTH_TOKEN_FILE", "ATHENA_TRADER_SYNC_GRPC_TRANSPORT"}
	registry := map[string]ServiceSpec{}
	for _, spec := range []ServiceSpec{
		{Name: "worm-trading", BuildPackage: "./cmd/athena-worm-trading", Binary: "athena-worm-trading", Infrastructure: []string{"postgres"}, EnvironmentKeys: keys(toolEnvironment, loggingEnvironment, proxyEnvironment, dbEnvironment, []string{"ATHENA_WORM_TRADING_POSTGRES_DSN", "ATHENA_WORM_TRADING_LISTEN_ADDRESS", "ATHENA_WORM_TRADING_PORT", "ATHENA_WORM_TRADING_SOLANA_RPC_URL", "ATHENA_WORM_TRADING_RPC_ATTEMPT_TIMEOUT", "ATHENA_WORM_TRADING_BALANCE_BUDGET", "ATHENA_WORM_TRADING_RPC_RATE_LIMIT", "ATHENA_WORM_TRADING_RPC_RATE_BURST", "ATHENA_WORM_TRADING_WORM_API_ATTEMPT_TIMEOUT", "ATHENA_WORM_TRADING_CATALOG_BUDGET", "ATHENA_WORM_TRADING_POSITION_BUDGET", "ATHENA_WORM_TRADING_POSITION_CONCURRENCY", "ATHENA_WORM_TRADING_CREDENTIAL_ENCRYPTION_KEY", "ATHENA_WORM_TRADING_INTERNAL_AUTH_TOKEN", "ATHENA_WALLET_SERVER_ADDRESS", "ATHENA_WALLET_WORM_EXECUTION_SIGNER_TOKEN"})},
		{Name: "trader-sync", BuildPackage: "./cmd/athena-trader-sync", Binary: "athena-trader-sync", Infrastructure: []string{"postgres"}, EnvironmentKeys: keys(toolEnvironment, loggingEnvironment, dbEnvironment, rpc, []string{"ATHENA_TRADER_SYNC_LISTEN_ADDRESS", "ATHENA_TRADER_SYNC_HTTP_URL", "ATHENA_TRADER_SYNC_WSS_URL", "ATHENA_TRADER_SYNC_PROXY_URL", "ATHENA_TRADER_SYNC_CURSOR_HMAC_KEY", "ATHENA_TRADER_SYNC_CURSOR_HMAC_KEY_FILE", "ATHENA_TRADER_SYNC_MAX_IN_FLIGHT_SOURCES", "ATHENA_TRADER_SYNC_TLS_CERT_FILE", "ATHENA_TRADER_SYNC_TLS_KEY_FILE", "ATHENA_TRADER_SYNC_SHUTDOWN_TIMEOUT"})},
		{Name: "api-server", BuildPackage: "./cmd/athena-server", Binary: "athena-server", Infrastructure: []string{"postgres", "redis", "minio"}, EnvironmentKeys: keys(toolEnvironment, loggingEnvironment, proxyEnvironment, apiConsumerEnvironment, dbEnvironment, rpc, []string{"ATHENA_TRADER_SYNC_SERVER_ADDRESS", "ATHENA_TRADER_SYNC_TLS_CA_FILE", "ATHENA_TRADER_SYNC_TLS_SERVER_NAME", "ATHENA_SERVER_LISTEN_ADDRESS", "ATHENA_SERVER_DISABLE_AUTH", "ATHENA_SERVER_STATIC_ASSETS", "ATHENA_SERVER_ROOTPATH", "ATHENA_SERVER_BASEHREF", "ATHENA_SERVER_ENABLE_GZIP", "ATHENA_SERVER_HYDRATOR_ENABLED", "ATHENA_NOTIFICATION_SERVER_ADDRESS", "ATHENA_NOTIFICATION_INTERNAL_AUTH_TOKEN", "ATHENA_WALLET_SERVER_ADDRESS", "ATHENA_TOKEN_API_SERVER_ADDRESS", "ATHENA_SOLANA_DISCOVERY_SERVER_ADDRESS", "ATHENA_SOLANA_DISCOVERY_INTERNAL_AUTH_TOKEN", "ATHENA_MARKET_RADAR_SERVER_ADDRESS", "ATHENA_MANAGED_OO_SERVER_ADDRESS", "ATHENA_PROFIT_SHARING_SERVER_ADDRESS", "ATHENA_WORM_TRADING_SERVER_ADDRESS", "ATHENA_OPERATION_LOG_SERVER_ADDRESS", "ATHENA_OPERATION_LOG_GRPC_TRANSPORT", "ATHENA_OPERATION_LOG_INTERNAL_AUTH_TOKEN", "ATHENA_OPERATION_LOG_INTERNAL_AUTH_TOKEN_FILE", "ATHENA_OPERATION_LOG_POSTGRES_DSN", "ATHENA_OPERATION_LOG_TLS_CA_FILE", "ATHENA_OPERATION_LOG_TLS_SERVER_NAME", "ATHENA_GOOGLE_OIDC_CLIENT_ID", "ATHENA_GOOGLE_OIDC_CLIENT_SECRET", "ATHENA_GOOGLE_OIDC_CLIENT_SECRET_FILE", "ATHENA_GOOGLE_OIDC_REDIRECT_URI", "ATHENA_ADMIN_GOOGLE_EMAIL", "ATHENA_JWT_SECRET", "ATHENA_JWT_SECRET_FILE", "REDIS_SERVER", "REDIS_PASSWORD", "ATHENA_ACCOUNT_AVATAR_S3_ENDPOINT", "ATHENA_ACCOUNT_AVATAR_S3_REGION", "ATHENA_ACCOUNT_AVATAR_S3_BUCKET", "ATHENA_ACCOUNT_AVATAR_S3_ACCESS_KEY_ID", "ATHENA_ACCOUNT_AVATAR_S3_SECRET_ACCESS_KEY", "ATHENA_ACCOUNT_AVATAR_S3_PATH_STYLE", "ATHENA_WALLET_INTERNAL_AUTH_TOKEN", "ATHENA_WORM_TRADING_INTERNAL_AUTH_TOKEN"})},
		{Name: "operation-log", BuildPackage: "./cmd/athena-operation-log", Binary: "athena-operation-log", Infrastructure: []string{"postgres"}, EnvironmentKeys: keys(toolEnvironment, loggingEnvironment, dbEnvironment, []string{"ATHENA_OPERATION_LOG_POSTGRES_DSN", "ATHENA_OPERATION_LOG_LISTEN_ADDRESS", "ATHENA_OPERATION_LOG_GRPC_TRANSPORT", "ATHENA_OPERATION_LOG_INTERNAL_AUTH_TOKEN", "ATHENA_OPERATION_LOG_INTERNAL_AUTH_TOKEN_FILE", "ATHENA_OPERATION_LOG_CURSOR_HMAC_KEY", "ATHENA_OPERATION_LOG_CURSOR_HMAC_KEY_FILE", "ATHENA_OPERATION_LOG_TLS_CERT_FILE", "ATHENA_OPERATION_LOG_TLS_KEY_FILE", "ATHENA_OPERATION_LOG_TLS_CA_FILE", "ATHENA_OPERATION_LOG_TLS_SERVER_NAME"})},
		{Name: "notification", BuildPackage: "./cmd/athena-notification", Binary: "athena-notification", Infrastructure: []string{"postgres"}, EnvironmentKeys: keys(toolEnvironment, loggingEnvironment, proxyEnvironment, dbEnvironment, []string{"ATHENA_NOTIFICATION_LISTEN_ADDRESS", "ATHENA_NOTIFICATION_INTERNAL_AUTH_TOKEN", "ATHENA_NOTIFICATION_TELEGRAM_BOT_TOKEN", "ATHENA_NOTIFICATION_TELEGRAM_API_URL", "ATHENA_NOTIFICATION_TELEGRAM_TIMEOUT_SECONDS", "ATHENA_NOTIFICATION_TELEGRAM_BOT_NAME", "ATHENA_NOTIFICATION_TELEGRAM_BOT_SHORT_DESCRIPTION", "ATHENA_NOTIFICATION_TELEGRAM_BOT_DESCRIPTION", "ATHENA_NOTIFICATION_TEST_TELEGRAM_CHAT_ID", "ATHENA_NOTIFICATION_PROD_TELEGRAM_CHAT_ID", "ATHENA_NOTIFICATION_WORKER_CONCURRENCY"})},
		{Name: "ui", Binary: "node", EnvironmentKeys: keys(toolEnvironment, []string{"NODE_ENV", "ATHENA_API_URL", "ATHENA_NOTIFICATION_API_URL", "NODE_EXTRA_CA_CERTS"})},
		{Name: "wallet", BuildPackage: "./cmd/athena-wallet", Binary: "athena-wallet", Infrastructure: []string{"postgres"}, EnvironmentKeys: keys(toolEnvironment, loggingEnvironment, []string{"ATHENA_GRPC_MAX_SIZE_MB", "ATHENA_WALLET_POSTGRES_DSN", "ATHENA_WALLET_ENCRYPTION_KEY", "ATHENA_WALLET_INTERNAL_AUTH_TOKEN", "ATHENA_WALLET_WORM_EXECUTION_SIGNER_TOKEN"})},
		{Name: "etherscan-manager", BuildPackage: "./cmd/athena-etherscan-manager", Binary: "athena-etherscan-manager", EnvironmentKeys: keys(toolEnvironment, loggingEnvironment, proxyEnvironment, []string{"ATHENA_GRPC_MAX_SIZE_MB", "ATHENA_ETHERSCAN_MANAGER_GATEWAY_ADDRS", "ATHENA_ETHERSCAN_MANAGER_API_KEYS", "ATHENA_ETHERSCAN_GATEWAY_AUTH_TOKEN"})},
		{Name: "market-radar", BuildPackage: "./cmd/athena-market-radar", Binary: "athena-market-radar", EnvironmentKeys: keys(toolEnvironment, loggingEnvironment, proxyEnvironment, []string{"ATHENA_GRPC_MAX_SIZE_MB", "ATHENA_MARKET_RADAR_NOTIFICATION_ENABLED", "ATHENA_MARKET_RADAR_NOTIFICATION_SERVER_ADDRESS", "ATHENA_MARKET_RADAR_NOTIFICATION_INVITE_CODE", "ATHENA_NOTIFICATION_INTERNAL_AUTH_TOKEN"})},
		{Name: "managed-oo", BuildPackage: "./cmd/athena-managed-oo", Binary: "athena-managed-oo", Infrastructure: []string{"postgres"}, EnvironmentKeys: keys(toolEnvironment, loggingEnvironment, proxyEnvironment, []string{"ATHENA_GRPC_MAX_SIZE_MB", "ATHENA_MANAGED_OO_POSTGRES_DSN", "ATHENA_MANAGED_OO_NOTIFICATION_ENABLED", "ATHENA_MANAGED_OO_NOTIFICATION_SERVER_ADDRESS", "ATHENA_MANAGED_OO_NOTIFICATION_INVITE_CODE", "ATHENA_MANAGED_OO_POLYGON_RPC_URL", "ATHENA_NOTIFICATION_INTERNAL_AUTH_TOKEN"})},
		{Name: "profit-sharing", BuildPackage: "./cmd/athena-profit-sharing", Binary: "athena-profit-sharing", Infrastructure: []string{"postgres"}, EnvironmentKeys: keys(toolEnvironment, loggingEnvironment, []string{"ATHENA_GRPC_MAX_SIZE_MB", "ATHENA_PROFIT_SHARING_POSTGRES_DSN"})},
		{Name: "solana-discovery", BuildPackage: "./cmd/athena-solana-discovery", Binary: "athena-solana-discovery", Infrastructure: []string{"postgres"}, EnvironmentKeys: keys(toolEnvironment, loggingEnvironment, proxyEnvironment, dbEnvironment, []string{"ATHENA_SOLANA_DISCOVERY_POSTGRES_DSN", "ATHENA_SOLANA_DISCOVERY_INTERNAL_AUTH_TOKEN", "ATHENA_SOLANA_DISCOVERY_RPC_URL", "ATHENA_SOLANA_DISCOVERY_START_SLOT", "ATHENA_SOLANA_DISCOVERY_RANGE_SIZE", "ATHENA_SOLANA_DISCOVERY_CONCURRENCY", "ATHENA_SOLANA_DISCOVERY_REQUESTS_PER_SECOND", "ATHENA_SOLANA_DISCOVERY_REQUEST_TIMEOUT", "ATHENA_SOLANA_DISCOVERY_POLL_INTERVAL"})},
	} {
		registry[spec.Name] = spec
	}
	for name, spec := range registry {
		spec.Readiness = "grpc"
		spec.StopLayer = 1
		spec.StartupTimeout, spec.ShutdownTimeout = 60*time.Second, 30*time.Second
		prefix := ""
		switch name {
		case "api-server":
			prefix = "SERVER"
			spec.DefaultPort = "8080"
			spec.Schemas = []string{"account", "operation-log"}
			spec.Core = true
			spec.StopLayer = 0
			spec.Readiness = "http"
		case "ui":
			prefix = "UI"
			spec.DefaultPort = "4000"
			spec.Core = true
			spec.StopLayer = 0
			spec.Readiness = "http"
			spec.StartupTimeout = 120 * time.Second
			spec.EnvironmentKeys = append(spec.EnvironmentKeys, "ATHENA_SERVER_BASEHREF")
		case "notification":
			prefix = "NOTIFICATION"
			spec.DefaultPort = "8086"
			spec.Schemas = []string{"account"}
			spec.Core = true
			spec.StopLayer = 2
			spec.StartupTimeout = 180 * time.Second
			spec.AddressKeys = []string{"ATHENA_NOTIFICATION_SERVER_ADDRESS", "ATHENA_MARKET_RADAR_NOTIFICATION_SERVER_ADDRESS", "ATHENA_MANAGED_OO_NOTIFICATION_SERVER_ADDRESS"}
		case "wallet":
			prefix = "WALLET"
			spec.DefaultPort = "8088"
			spec.Schemas = []string{"wallet"}
			spec.Core = true
			spec.StopLayer = 2
		case "etherscan-manager":
			prefix = "ETHERSCAN_MANAGER"
			spec.DefaultPort = "8100"
			spec.Core = true
			spec.StopLayer = 2
		case "trader-sync":
			prefix = "TRADER_SYNC"
			spec.DefaultPort = "8122"
			spec.Schemas = []string{"account"}
			spec.HealthService = "tradersync.internal.v1.TraderSyncService"
		case "solana-discovery":
			prefix = "SOLANA_DISCOVERY"
			spec.DefaultPort = "8112"
			spec.Schemas = []string{"account", "solana-discovery"}
		case "market-radar":
			prefix = "MARKET_RADAR"
			spec.DefaultPort = "8092"
		case "managed-oo":
			prefix = "MANAGED_OO"
			spec.DefaultPort = "8106"
			spec.Schemas = []string{"managed-oo"}
		case "profit-sharing":
			prefix = "PROFIT_SHARING"
			spec.DefaultPort = "8108"
			spec.Schemas = []string{"profit-sharing"}
		case "worm-trading":
			prefix = "WORM_TRADING"
			spec.DefaultPort = "8090"
			spec.Schemas = []string{"account", "worm-trading"}
		case "operation-log":
			prefix = "OPERATION_LOG"
			spec.DefaultPort = "8124"
			spec.Schemas = []string{"account", "operation-log"}
			spec.ListenKey = "ATHENA_OPERATION_LOG_LISTEN_ADDRESS"
		}
		if spec.ListenKey == "" {
			spec.ListenKey = "ATHENA_" + prefix + "_LISTEN_ADDRESS"
		}
		spec.PortKey = "ATHENA_" + prefix + "_PORT"
		if name == "trader-sync" || name == "operation-log" {
			spec.PortKey = ""
		}
		if name == "market-radar" || name == "managed-oo" || name == "profit-sharing" {
			spec.PortKey = "ATHENA_" + prefix + "_LISTEN_PORT"
		}
		if len(spec.AddressKeys) == 0 && name != "ui" && name != "api-server" && name != "etherscan-manager" {
			spec.AddressKeys = []string{"ATHENA_" + prefix + "_SERVER_ADDRESS"}
		}
		spec.EnvironmentKeys = append(spec.EnvironmentKeys, spec.ListenKey)
		if spec.PortKey != "" {
			spec.EnvironmentKeys = append(spec.EnvironmentKeys, spec.PortKey)
		}
		registry[name] = spec
	}
	return registry
}

func (s ServiceSpec) Address(env map[string]string) string {
	if s.Name == "trader-sync" || s.Name == "operation-log" {
		return envDefault(env, s.ListenKey, "127.0.0.1:"+s.DefaultPort)
	}
	return net.JoinHostPort(envDefault(env, s.ListenKey, "127.0.0.1"), envDefault(env, s.PortKey, s.DefaultPort))
}
func needsSchema(specs []ServiceSpec, name string) bool {
	for _, spec := range specs {
		for _, owner := range spec.Schemas {
			if owner == name {
				return true
			}
		}
	}
	return false
}
func ResolveServices(names []string) ([]ServiceSpec, error) {
	if len(names) == 0 {
		return nil, fmt.Errorf("select at least one service")
	}
	registry := serviceRegistry()
	seen := map[string]bool{}
	var result []ServiceSpec
	for _, n := range names {
		s, ok := registry[n]
		if !ok {
			return nil, fmt.Errorf("unknown independent service %q", n)
		}
		if seen[n] {
			return nil, fmt.Errorf("duplicate service %q", n)
		}
		seen[n] = true
		result = append(result, s)
	}
	var roots []string
	for _, s := range result {
		roots = append(roots, s.Infrastructure...)
	}
	if _, err := resolveInfrastructure(roots, map[string][]string{"postgres": nil, "redis": nil, "minio": nil}); err != nil {
		return nil, err
	}
	return result, nil
}
func resolveInfrastructure(roots []string, graph map[string][]string) ([]string, error) {
	visited := map[string]int{}
	var out []string
	var visit func(string) error
	visit = func(n string) error {
		if visited[n] == 1 {
			return fmt.Errorf("infrastructure dependency cycle")
		}
		if visited[n] == 2 {
			return nil
		}
		deps, ok := graph[n]
		if !ok {
			return fmt.Errorf("unknown infrastructure %q", n)
		}
		visited[n] = 1
		for _, d := range deps {
			if err := visit(d); err != nil {
				return err
			}
		}
		visited[n] = 2
		out = append(out, n)
		return nil
	}
	for _, r := range roots {
		if err := visit(r); err != nil {
			return nil, err
		}
	}
	return out, nil
}
func selected(specs []ServiceSpec, name string) bool {
	for _, s := range specs {
		if s.Name == name {
			return true
		}
	}
	return false
}
func needsDatabase(specs []ServiceSpec) bool {
	for _, s := range specs {
		for _, i := range s.Infrastructure {
			if i == "postgres" {
				return true
			}
		}
	}
	return false
}
