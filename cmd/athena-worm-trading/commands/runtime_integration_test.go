//go:build integration

package commands

import (
	"context"
	"io"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/useryege/athena/internal/accountstate/store/migrations"
	"github.com/useryege/athena/internal/testutil/pgtest"
	wormstore "github.com/useryege/athena/internal/wormtrading/store"
)

func TestStartupRejectsAccountSchemaAndMissingSecretsAndReleasesPools(t *testing.T) {
	trading := pgtest.New(t, wormstore.Migrations(), "migrations")
	account := pgtest.New(t, migrations.FS, migrations.Dir)
	empty := pgtest.NewUnmigrated(t)
	t.Setenv(wormstore.DSNEnv, trading.DSN)
	t.Setenv("ATHENA_ACCOUNT_STATE_POSTGRES_DSN", account.DSN)
	t.Setenv("ATHENA_WALLET_WORM_EXECUTION_SIGNER_TOKEN", "test-owned-signer-token-01234567890123456789")
	t.Setenv(credentialKeyEnv, "test-owned-credential-key")
	for _, tc := range []struct{ name, key, value, want string }{
		{"account schema", "ATHENA_ACCOUNT_STATE_POSTGRES_DSN", empty.DSN, "migration versions mismatch"},
		{"signer token", "ATHENA_WALLET_WORM_EXECUTION_SIGNER_TOKEN", "", "token"},
		{"credential key", credentialKeyEnv, "", "encryption"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv(tc.key, tc.value)
			cmd := NewCommand()
			cmd.SetOut(io.Discard)
			cmd.SetErr(io.Discard)
			cmd.SetArgs(nil)
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			require.ErrorContains(t, cmd.ExecuteContext(ctx), tc.want)
			for _, db := range []*pgtest.DB{trading, account, empty} {
				require.Eventually(t, func() bool {
					var count int
					err := db.Pool.QueryRow(context.Background(), `SELECT count(*) FROM pg_stat_activity WHERE datname=current_database() AND pid<>pg_backend_pid()`).Scan(&count)
					return err == nil && count == 0
				}, time.Second, 10*time.Millisecond, "command must close both owned pools after startup failure")
			}
		})
	}
}
