package devruntime

import (
	"fmt"
	"time"
)

type ServiceSpec struct {
	Name, BuildPackage, Binary      string
	Infrastructure, EnvironmentKeys []string
	Args                            []string
	StartupTimeout, ShutdownTimeout time.Duration
}

var toolEnvironment = []string{"PATH", "HOME", "TMPDIR", "LANG", "LC_ALL", "SSL_CERT_FILE", "SSL_CERT_DIR", "ATHENA_LOCAL_RUNTIME_INSTANCE"}
var dbEnvironment = []string{"ATHENA_ACCOUNT_STATE_POSTGRES_DSN", "ATHENA_URL", "ATHENA_LOG_LEVEL", "ATHENA_GRPC_MAX_SIZE_MB"}

func keys(groups ...[]string) []string {
	var out []string
	for _, g := range groups {
		out = append(out, g...)
	}
	return out
}
func serviceRegistry() map[string]ServiceSpec {
	rpc := []string{"ATHENA_TRADER_SYNC_INTERNAL_AUTH_TOKEN", "ATHENA_TRADER_SYNC_INTERNAL_AUTH_TOKEN_FILE", "ATHENA_TRADER_SYNC_GRPC_TRANSPORT"}
	return map[string]ServiceSpec{
		"trader-sync":  {Name: "trader-sync", BuildPackage: "./cmd/athena-trader-sync", Binary: "athena-trader-sync", Infrastructure: []string{"postgres"}, EnvironmentKeys: keys(toolEnvironment, dbEnvironment, rpc, []string{"ATHENA_TRADER_SYNC_LISTEN_ADDRESS", "ATHENA_TRADER_SYNC_HTTP_URL", "ATHENA_TRADER_SYNC_WSS_URL", "ATHENA_TRADER_SYNC_PROXY_URL", "ATHENA_TRADER_SYNC_CURSOR_HMAC_KEY", "ATHENA_TRADER_SYNC_CURSOR_HMAC_KEY_FILE", "ATHENA_TRADER_SYNC_MAX_IN_FLIGHT_SOURCES", "ATHENA_TRADER_SYNC_TLS_CERT_FILE", "ATHENA_TRADER_SYNC_TLS_KEY_FILE", "ATHENA_TRADER_SYNC_SHUTDOWN_TIMEOUT"}), StartupTimeout: 60 * time.Second, ShutdownTimeout: 30 * time.Second},
		"api-server":   {Name: "api-server", BuildPackage: "./cmd/athena-server", Binary: "athena-server", Infrastructure: []string{"postgres", "redis", "minio"}, EnvironmentKeys: keys(toolEnvironment, dbEnvironment, rpc, []string{"ATHENA_TRADER_SYNC_SERVER_ADDRESS", "ATHENA_TRADER_SYNC_TLS_CA_FILE", "ATHENA_TRADER_SYNC_TLS_SERVER_NAME", "ATHENA_SERVER_LISTEN_ADDRESS", "ATHENA_SERVER_DISABLE_AUTH", "ATHENA_SERVER_STATIC_ASSETS", "ATHENA_SERVER_ROOTPATH", "ATHENA_SERVER_BASEHREF", "ATHENA_SERVER_ENABLE_GZIP", "ATHENA_SERVER_HYDRATOR_ENABLED", "ATHENA_NOTIFICATION_SERVER_ADDRESS", "ATHENA_NOTIFICATION_INTERNAL_AUTH_TOKEN", "ATHENA_WALLET_SERVER_ADDRESS", "ATHENA_TOKEN_API_SERVER_ADDRESS", "ATHENA_MARKET_RADAR_SERVER_ADDRESS", "ATHENA_MANAGED_OO_SERVER_ADDRESS", "ATHENA_PROFIT_SHARING_SERVER_ADDRESS", "ATHENA_WORM_MARKETS_SERVER_ADDRESS", "ATHENA_WORM_TRADING_SERVER_ADDRESS", "ATHENA_SPORTS_HISTORY_SERVER_ADDRESS", "ATHENA_SPORTS_LIVE_SERVER_ADDRESS", "ATHENA_GOOGLE_OIDC_CLIENT_ID", "ATHENA_GOOGLE_OIDC_CLIENT_SECRET", "ATHENA_GOOGLE_OIDC_CLIENT_SECRET_FILE", "ATHENA_GOOGLE_OIDC_REDIRECT_URI", "ATHENA_ADMIN_GOOGLE_EMAIL", "ATHENA_JWT_SECRET", "ATHENA_JWT_SECRET_FILE", "REDIS_SERVER", "REDIS_PASSWORD", "ATHENA_ACCOUNT_AVATAR_S3_ENDPOINT", "ATHENA_ACCOUNT_AVATAR_S3_REGION", "ATHENA_ACCOUNT_AVATAR_S3_BUCKET", "ATHENA_ACCOUNT_AVATAR_S3_ACCESS_KEY_ID", "ATHENA_ACCOUNT_AVATAR_S3_SECRET_ACCESS_KEY", "ATHENA_ACCOUNT_AVATAR_S3_PATH_STYLE", "ATHENA_WALLET_INTERNAL_AUTH_TOKEN", "ATHENA_WORM_TRADING_INTERNAL_AUTH_TOKEN"}), StartupTimeout: 60 * time.Second, ShutdownTimeout: 30 * time.Second},
		"notification": {Name: "notification", BuildPackage: "./cmd/athena-notification", Binary: "athena-notification", Infrastructure: []string{"postgres"}, EnvironmentKeys: keys(toolEnvironment, dbEnvironment, []string{"ATHENA_NOTIFICATION_LISTEN_ADDRESS", "ATHENA_NOTIFICATION_INTERNAL_AUTH_TOKEN", "ATHENA_NOTIFICATION_TELEGRAM_BOT_TOKEN", "ATHENA_NOTIFICATION_TELEGRAM_API_URL", "ATHENA_NOTIFICATION_TELEGRAM_TIMEOUT_SECONDS", "ATHENA_NOTIFICATION_TELEGRAM_BOT_NAME", "ATHENA_NOTIFICATION_TELEGRAM_BOT_SHORT_DESCRIPTION", "ATHENA_NOTIFICATION_TELEGRAM_BOT_DESCRIPTION", "ATHENA_NOTIFICATION_TEST_TELEGRAM_CHAT_ID", "ATHENA_NOTIFICATION_PROD_TELEGRAM_CHAT_ID", "ATHENA_NOTIFICATION_WORKER_CONCURRENCY"}), StartupTimeout: 60 * time.Second, ShutdownTimeout: 30 * time.Second},
		"ui":           {Name: "ui", Binary: "node", EnvironmentKeys: keys(toolEnvironment, []string{"NODE_ENV", "ATHENA_API_URL", "ATHENA_NOTIFICATION_API_URL", "NODE_EXTRA_CA_CERTS"}), StartupTimeout: 120 * time.Second, ShutdownTimeout: 30 * time.Second},
	}
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
