//go:build integration

package schema_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/useryege/athena/internal/accountstate/schema"
	accountstore "github.com/useryege/athena/internal/accountstate/store"
	notificationstore "github.com/useryege/athena/internal/notification/store"
	"github.com/useryege/athena/internal/testutil/pgtest"
)

func TestBusinessSourcesRequirePreexistingVerifiedSchema(t *testing.T) {
	for _, name := range []string{"account-state", "notification"} {
		t.Run(name, func(t *testing.T) {
			db := pgtest.NewUnmigrated(t)
			ctx := context.Background()
			t.Setenv(schema.DSNEnv, db.DSN)
			t.Setenv("ATHENA_SERVER_POSTGRES_DSN", db.DSN)
			t.Setenv("ATHENA_POSTGRES_AUTO_MIGRATE", "true")
			open := func() error {
				if name == "account-state" {
					s, err := accountstore.NewSQLStoreSource()(ctx)
					if s != nil {
						s.Close()
					}
					return err
				}
				s, err := notificationstore.NewSQLStoreSource()(ctx)
				if s != nil {
					s.Close()
				}
				return err
			}
			require.ErrorIs(t, open(), schema.ErrVersions)
			var exists bool
			require.NoError(t, db.Pool.QueryRow(ctx, `SELECT to_regclass('public.goose_db_version') IS NOT NULL`).Scan(&exists))
			require.False(t, exists)
			require.NoError(t, schema.Up(ctx, db.DSN))
			require.NoError(t, open())
		})
	}
}
