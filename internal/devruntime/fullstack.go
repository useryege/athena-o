package devruntime

import (
	"context"
	"errors"
	"fmt"
	"github.com/jackc/pgx/v5"
	"net"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

// FullStackServices is the explicit development graph. Commented-out modules in
// the former Procfile are intentionally absent; infrastructure is in each spec.
func FullStackServices() []string {
	return []string{"trader-sync", "profit-sharing", "notification", "wallet", "ui", "api-server"}
}

func fullStackSpecs() []ServiceSpec {
	registry := serviceRegistry()
	registry["wallet"] = ServiceSpec{Name: "wallet", BuildPackage: "./cmd/main.go", Binary: "athena-wallet", Infrastructure: []string{"postgres"}, EnvironmentKeys: keys(toolEnvironment, loggingEnvironment, []string{"ATHENA_GRPC_MAX_SIZE_MB", "ATHENA_WALLET_LISTEN_ADDRESS", "ATHENA_WALLET_POSTGRES_DSN", "ATHENA_WALLET_ENCRYPTION_KEY", "ATHENA_WALLET_INTERNAL_AUTH_TOKEN", "ATHENA_WALLET_WORM_EXECUTION_SIGNER_TOKEN"}), StartupTimeout: 60 * time.Second, ShutdownTimeout: 30 * time.Second}
	registry["profit-sharing"] = ServiceSpec{Name: "profit-sharing", BuildPackage: "./cmd/main.go", Binary: "athena-profit-sharing", Infrastructure: []string{"postgres"}, EnvironmentKeys: keys(toolEnvironment, loggingEnvironment, []string{"ATHENA_GRPC_MAX_SIZE_MB", "ATHENA_PROFIT_SHARING_LISTEN_ADDRESS", "ATHENA_PROFIT_SHARING_POSTGRES_DSN"}), StartupTimeout: 60 * time.Second, ShutdownTimeout: 30 * time.Second}
	specs := make([]ServiceSpec, 0, len(FullStackServices()))
	for _, name := range FullStackServices() {
		specs = append(specs, registry[name])
	}
	return specs
}

// These are the pre-existing full-stack schema consumers, not service roots.
type fullStackModule struct{ Name, Database, DSNEnv string }

func fullStackModules() []fullStackModule {
	return []fullStackModule{
		{"worm-markets", "worm_markets", "ATHENA_WORM_MARKETS_POSTGRES_DSN"},
		{"worm-trading", "worm_trading", "ATHENA_WORM_TRADING_POSTGRES_DSN"},
		{"wallet", "wallet", "ATHENA_WALLET_POSTGRES_DSN"},
		{"managed-oo", "managed_oo", "ATHENA_MANAGED_OO_POSTGRES_DSN"},
		{"profit-sharing", "profit_sharing", "ATHENA_PROFIT_SHARING_POSTGRES_DSN"},
		{"token", "token", "ATHENA_TOKEN_POSTGRES_DSN"},
	}
}
func prepareFullStackEnvironment(env map[string]string) {
	for key, value := range map[string]string{
		"ATHENA_WALLET_LISTEN_ADDRESS":              "127.0.0.1",
		"ATHENA_PROFIT_SHARING_LISTEN_ADDRESS":      "127.0.0.1",
		"ATHENA_WALLET_INTERNAL_AUTH_TOKEN":         "athena-local-wallet-internal-auth-token-2026",
		"ATHENA_WALLET_WORM_EXECUTION_SIGNER_TOKEN": "athena-local-wallet-worm-execution-signer-token-2026",
		"ATHENA_WALLET_ENCRYPTION_KEY":              "athena-local-wallet-encryption-key",
		"ATHENA_GOOGLE_OIDC_REDIRECT_URI":           "http://localhost:4000/auth/google/callback",
		"ATHENA_ACCOUNT_AVATAR_MAX_BYTES":           "2097152",
	} {
		if _, ok := env[key]; !ok {
			env[key] = value
		}
	}
	env["ATHENA_WALLET_SERVER_ADDRESS"] = net.JoinHostPort(env["ATHENA_WALLET_LISTEN_ADDRESS"], envDefault(env, "ATHENA_WALLET_PORT", "8088"))
	env["ATHENA_PROFIT_SHARING_SERVER_ADDRESS"] = net.JoinHostPort(env["ATHENA_PROFIT_SHARING_LISTEN_ADDRESS"], envDefault(env, "ATHENA_PROFIT_SHARING_PORT", "8108"))
}

func (m *Manager) prepareFullStackDatabases(ctx context.Context, env map[string]string) error {
	state, err := m.Status()
	if err != nil {
		return err
	}
	if state.DBMode != "managed" || state.Phase != "starting" {
		return errors.New("full stack schema requires a starting managed instance")
	}
	deadline, cancel := context.WithTimeout(ctx, 120*time.Second)
	defer cancel()
	dsn, err := url.Parse(env["ATHENA_ACCOUNT_STATE_POSTGRES_DSN"])
	if err != nil || dsn.Host == "" {
		return errors.New("invalid managed full stack database configuration")
	}
	conn, err := pgx.Connect(deadline, dsn.String())
	if err != nil {
		return errors.New("cannot connect to managed full stack database")
	}
	defer conn.Close(context.Background())
	databases := []string{"temporal", "temporal_visibility"}
	for _, module := range fullStackModules() {
		databases = append(databases, module.Database)
	}
	for _, database := range databases {
		var exists bool
		if err = conn.QueryRow(deadline, "SELECT EXISTS(SELECT FROM pg_database WHERE datname=$1)", database).Scan(&exists); err != nil {
			return fmt.Errorf("inspect full stack database %s: %w", database, err)
		}
		if !exists {
			if _, err = conn.Exec(deadline, "CREATE DATABASE "+pgx.Identifier{database}.Sanitize()); err != nil {
				return fmt.Errorf("create full stack database %s: %w", database, err)
			}
		}
	}
	dir, err := os.MkdirTemp(m.Key.Dir(), "full-stack-migration-")
	if err != nil {
		return err
	}
	binary := filepath.Join(dir, "athena-migrate")
	build := exec.Command("go", "build", "-o", binary, "./cmd/main.go")
	build.Dir = m.Key.Checkout
	build.Env = EnvironmentFor(environmentMap(os.Environ()), []string{"PATH", "HOME", "TMPDIR", "GOCACHE", "GOMODCACHE", "GOPATH", "GOROOT", "GOTOOLCHAIN", "CGO_ENABLED", "CC", "CXX", "PKG_CONFIG_PATH"})
	if err = m.RunHelper(ctx, "build-full-stack-schema", build, 30*time.Second); err != nil {
		return err
	}
	for _, module := range fullStackModules() {
		location := *dsn
		location.Path = "/" + module.Database
		location.RawPath = ""
		env[module.DSNEnv] = location.String()
		cmd := exec.Command(binary, "up", "--module", module.Name)
		cmd.Dir = m.Key.Checkout
		cmd.Env = EnvironmentFor(env, keys(toolEnvironment, loggingEnvironment, []string{module.DSNEnv}))
		migration, cancel := context.WithTimeout(ctx, 120*time.Second)
		err = m.RunHelper(migration, "full-stack-schema-"+module.Name, cmd, 30*time.Second)
		cancel()
		if err != nil {
			return err
		}
	}
	return nil
}

// RunFullStack uses the same resource owner and startup sequence as a selected
// independent service run. Legacy module databases exist only in this graph.
func RunFullStack(ctx context.Context, o RunOptions) error {
	if o.DBMode != "" && o.DBMode != "managed" {
		return errors.New("full stack requires DB_MODE=managed; use run-services for external account-state databases")
	}
	if len(o.Services) != 0 {
		return errors.New("full stack has an explicit service graph; use run-services for a selection")
	}
	o.DBMode = "managed"
	o.Services = FullStackServices()
	return runResolved(ctx, o, fullStackSpecs(), true)
}
