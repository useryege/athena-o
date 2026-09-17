package serviceschema_test

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
	managedcmd "github.com/useryege/athena/cmd/athena-managed-oo/commands"
	profitcmd "github.com/useryege/athena/cmd/athena-profit-sharing/commands"
	solanacmd "github.com/useryege/athena/cmd/athena-solana-discovery/commands"
	walletcmd "github.com/useryege/athena/cmd/athena-wallet/commands"
	managedstore "github.com/useryege/athena/internal/managedoo/store"
	profitstore "github.com/useryege/athena/internal/profitsharing/store"
	walletstore "github.com/useryege/athena/internal/wallet/store"
)

func database(t *testing.T) (*pgxpool.Pool, string) {
	t.Helper()
	dsn := os.Getenv("SERVICE_SCHEMA_TEST_POSTGRES_DSN")
	if dsn == "" {
		t.Skip("set SERVICE_SCHEMA_TEST_POSTGRES_DSN for isolated PostgreSQL tests")
	}
	ctx := context.Background()
	admin, err := pgx.Connect(ctx, dsn)
	require.NoError(t, err)
	name := fmt.Sprintf("service_schema_%d", time.Now().UnixNano())
	_, err = admin.Exec(ctx, "CREATE DATABASE "+pgx.Identifier{name}.Sanitize())
	require.NoError(t, err)
	u, err := url.Parse(dsn)
	require.NoError(t, err)
	u.Path = "/" + name
	pool, err := pgxpool.New(ctx, u.String())
	require.NoError(t, err)
	t.Cleanup(func() {
		pool.Close()
		_, _ = admin.Exec(ctx, "DROP DATABASE "+pgx.Identifier{name}.Sanitize()+" WITH (FORCE)")
		_ = admin.Close(ctx)
	})
	return pool, u.String()
}
func catalog(t *testing.T, pool *pgxpool.Pool) string {
	t.Helper()
	var result string
	err := pool.QueryRow(context.Background(), `SELECT COALESCE(jsonb_agg(x ORDER BY x.table_schema,x.table_name,x.ordinal_position)::text,'[]') FROM (SELECT table_schema,table_name,column_name,ordinal_position,data_type FROM information_schema.columns WHERE table_schema NOT IN ('pg_catalog','information_schema')) x`).Scan(&result)
	require.NoError(t, err)
	return result
}
func execute(factory func() *cobra.Command, args ...string) error {
	c := factory()
	c.SetOut(io.Discard)
	c.SetErr(io.Discard)
	c.SetArgs(args)
	return c.Execute()
}

// Runtime startup must never create or repair a borrowed database, even when
// the legacy auto-migrate environment defaults to true.
func TestStoreSourcesRejectEmptyDatabaseWithoutDDL(t *testing.T) {
	for _, tc := range []struct {
		name, env string
		open      func(context.Context) (io.Closer, error)
	}{
		{"wallet", "ATHENA_WALLET_POSTGRES_DSN", func(ctx context.Context) (io.Closer, error) { return walletstore.NewSQLStoreSource()(ctx) }},
		{"managed", "ATHENA_MANAGED_OO_POSTGRES_DSN", func(ctx context.Context) (io.Closer, error) { return managedstore.NewSQLStoreSource()(ctx) }},
		{"profit", "ATHENA_PROFIT_SHARING_POSTGRES_DSN", func(ctx context.Context) (io.Closer, error) { return profitstore.NewSQLStoreSource()(ctx) }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			pool, dsn := database(t)
			t.Setenv(tc.env, dsn)
			t.Setenv("ATHENA_POSTGRES_AUTO_MIGRATE", "true")
			before := catalog(t, pool)
			store, err := tc.open(context.Background())
			if err == nil {
				require.NoError(t, store.Close())
			}
			require.Error(t, err)
			require.Equal(t, before, catalog(t, pool))
		})
	}
}

// Real subcommands must bypass business credentials and verify versions and
// required relations/columns without invoking migration or changing catalog.
func TestSchemaCommandsReadOnlyVerification(t *testing.T) {
	for _, tc := range []struct {
		name, env, table, column string
		factory                  func() *cobra.Command
		goose                    bool
	}{
		{"wallet", "ATHENA_WALLET_POSTGRES_DSN", "public.wallets", "remark", walletcmd.NewCommand, true},
		{"managed", "ATHENA_MANAGED_OO_POSTGRES_DSN", "public.managed_oo_market", "slug", managedcmd.NewCommand, true},
		{"profit", "ATHENA_PROFIT_SHARING_POSTGRES_DSN", "public.profit_sharing_round", "title", profitcmd.NewCommand, true},
		{"solana", "ATHENA_SOLANA_DISCOVERY_POSTGRES_DSN", "solana_discovery.projects", "symbol", solanacmd.NewCommand, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			pool, dsn := database(t)
			t.Setenv(tc.env, dsn)
			t.Setenv("ATHENA_SOLANA_DISCOVERY_PORT", "invalid-business-port")
			for _, key := range []string{"ATHENA_WALLET_ENCRYPTION_KEY", "ATHENA_WALLET_INTERNAL_AUTH_TOKEN", "ATHENA_WALLET_WORM_EXECUTION_SIGNER_TOKEN", "ATHENA_SOLANA_DISCOVERY_INTERNAL_AUTH_TOKEN", "ATHENA_NOTIFICATION_INTERNAL_AUTH_TOKEN"} {
				t.Setenv(key, "")
			}
			before := catalog(t, pool)
			require.Error(t, execute(tc.factory, "schema", "verify", "--timeout=5s"))
			require.Equal(t, before, catalog(t, pool))
			require.NoError(t, execute(tc.factory, "schema", "up", "--timeout=5s"))
			require.NoError(t, execute(tc.factory, "schema", "verify", "--timeout=5s"))
			before = catalog(t, pool)
			t.Setenv(tc.env, dsn+"&default_transaction_read_only=on")
			require.NoError(t, execute(tc.factory, "schema", "verify", "--timeout=5s"))
			require.Equal(t, before, catalog(t, pool))
			t.Setenv(tc.env, dsn)
			ctx := context.Background()
			if tc.goose {
				_, err := pool.Exec(ctx, `INSERT INTO public.goose_db_version(version_id,is_applied) VALUES (1,false)`)
				require.NoError(t, err)
				require.Error(t, execute(tc.factory, "schema", "verify"))
				_, err = pool.Exec(ctx, `INSERT INTO public.goose_db_version(version_id,is_applied) VALUES (1,true)`)
				require.NoError(t, err)
				require.NoError(t, execute(tc.factory, "schema", "verify"))
				_, err = pool.Exec(ctx, `INSERT INTO public.goose_db_version(version_id,is_applied) VALUES (999,true)`)
				require.NoError(t, err)
				require.Error(t, execute(tc.factory, "schema", "verify"))
				_, err = pool.Exec(ctx, `DELETE FROM public.goose_db_version WHERE version_id=999`)
				require.NoError(t, err)
			}
			_, err := pool.Exec(ctx, "ALTER TABLE "+tc.table+" DROP COLUMN "+tc.column+" CASCADE")
			require.NoError(t, err)
			before = catalog(t, pool)
			require.Error(t, execute(tc.factory, "schema", "verify"))
			require.Equal(t, before, catalog(t, pool))
			_, err = pool.Exec(ctx, "DROP TABLE "+tc.table+" CASCADE")
			require.NoError(t, err)
			before = catalog(t, pool)
			require.Error(t, execute(tc.factory, "schema", "verify"))
			require.Equal(t, before, catalog(t, pool))
		})
	}
}

func TestSchemaCommandTimeoutDoesNotRequireBusinessConfig(t *testing.T) {
	for _, tc := range []struct {
		name, env string
		factory   func() *cobra.Command
	}{
		{"wallet", "ATHENA_WALLET_POSTGRES_DSN", walletcmd.NewCommand},
		{"managed", "ATHENA_MANAGED_OO_POSTGRES_DSN", managedcmd.NewCommand},
		{"profit", "ATHENA_PROFIT_SHARING_POSTGRES_DSN", profitcmd.NewCommand},
		{"solana", "ATHENA_SOLANA_DISCOVERY_POSTGRES_DSN", solanacmd.NewCommand},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv(tc.env, "postgres://postgres:unused@127.0.0.1:1/unreachable?sslmode=disable")
			for _, action := range []string{"up", "verify"} {
				start := time.Now()
				err := execute(tc.factory, "schema", action, "--timeout=20ms")
				require.Error(t, err)
				require.Less(t, time.Since(start), time.Second)
			}
		})
	}
}
