//go:build integration

package store_test

import (
	"context"
	"github.com/useryege/athena/internal/accountcredentials"
	accountstore "github.com/useryege/athena/internal/accountstate/store"
	"github.com/useryege/athena/internal/moduleaccess"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/useryege/athena/internal/accountstate/store/migrations"
	"github.com/useryege/athena/internal/testutil/pgtest"
)

func TestModuleAccessSchemaDefaultsAndRejectsUnknownKeys(t *testing.T) {
	db := pgtest.New(t, migrations.FS, migrations.Dir)
	ctx := context.Background()
	var count int
	err := db.Pool.QueryRow(ctx, "SELECT count(*) FROM athena_module_access_setting").Scan(&count)
	require.NoError(t, err)
	require.Zero(t, count, "startup must not seed or reset settings")
	for _, key := range []string{"trader_sync", "solana", "market_radar", "managed_oo", "profit_sharing", "worm"} {
		var open bool
		require.NoError(t, db.Pool.QueryRow(ctx, "INSERT INTO athena_module_access_setting(module_key) VALUES($1) RETURNING is_open", key).Scan(&open))
		require.False(t, open)
	}
	for _, key := range []string{"token", "unknown", ""} {
		_, err := db.Pool.Exec(ctx, "INSERT INTO athena_module_access_setting(module_key) VALUES($1)", key)
		require.Error(t, err)
	}
}

func TestModuleAccessDurableUpdatesDoNotChangeAccountRevision(t *testing.T) {
	db := pgtest.New(t, migrations.FS, migrations.Dir)
	ctx := context.Background()
	s := accountstore.NewSQLStore(db.Pool)
	admin, err := s.EnsureDevelopmentAccount(ctx, accountcredentials.DevelopmentRoleAdministrator)
	require.NoError(t, err)
	before, err := s.GetAccountAccess(ctx, admin.ID)
	require.NoError(t, err)
	initial, err := s.ListModuleAccessSettings(ctx)
	require.NoError(t, err)
	require.Len(t, initial, 6)
	for _, v := range initial {
		require.False(t, v.Open)
		require.True(t, v.UpdatedAt.IsZero())
	}
	first, err := s.UpdateModuleAccessSetting(ctx, moduleaccess.Worm, true, admin.ID)
	require.NoError(t, err)
	require.True(t, first.Open)
	require.Equal(t, "local-admin", first.UpdatedByUsername)
	second, err := s.UpdateModuleAccessSetting(ctx, moduleaccess.Worm, true, admin.ID)
	require.NoError(t, err)
	require.True(t, second.UpdatedAt.After(first.UpdatedAt))
	require.Equal(t, admin.ID, second.UpdatedByAccountID)
	restarted := accountstore.NewSQLStore(db.Pool)
	saved, err := restarted.GetModuleAccessSetting(ctx, moduleaccess.Worm)
	require.NoError(t, err)
	require.True(t, saved.Open)
	_, err = s.UpdateModuleAccessSetting(ctx, moduleaccess.Solana, false, admin.ID)
	require.NoError(t, err)
	closed, err := restarted.GetModuleAccessSetting(ctx, moduleaccess.Solana)
	require.NoError(t, err)
	require.False(t, closed.Open)
	require.False(t, closed.UpdatedAt.IsZero())
	after, err := s.GetAccountAccess(ctx, admin.ID)
	require.NoError(t, err)
	require.Equal(t, before.Revision, after.Revision)
	for _, key := range []moduleaccess.Key{"token", "unknown", ""} {
		_, err = s.UpdateModuleAccessSetting(ctx, key, true, admin.ID)
		require.Equal(t, codes.InvalidArgument, status.Code(err))
	}
}

func TestModuleAccessConcurrentLastCommitWins(t *testing.T) {
	db := pgtest.New(t, migrations.FS, migrations.Dir)
	ctx := context.Background()
	s := accountstore.NewSQLStore(db.Pool)
	admin, err := s.EnsureDevelopmentAccount(ctx, accountcredentials.DevelopmentRoleAdministrator)
	require.NoError(t, err)
	first, err := db.Pool.Begin(ctx)
	require.NoError(t, err)
	defer first.Rollback(ctx)
	_, err = first.Exec(ctx, "INSERT INTO athena_module_access_setting(module_key,is_open,updated_by_account_id,updated_at) VALUES('worm',true,$1,clock_timestamp())", admin.ID)
	require.NoError(t, err)
	done := make(chan error, 1)
	go func() { _, err := s.UpdateModuleAccessSetting(ctx, moduleaccess.Worm, false, admin.ID); done <- err }()
	require.Eventually(t, func() bool {
		var n int
		err := db.Pool.QueryRow(ctx, "SELECT count(*) FROM pg_stat_activity WHERE datname=current_database() AND wait_event_type='Lock' AND query LIKE '%UpsertModuleAccessSetting%'").Scan(&n)
		return err == nil && n == 1
	}, 5*time.Second, 10*time.Millisecond)
	require.NoError(t, first.Commit(ctx))
	require.NoError(t, <-done)
	saved, err := s.GetModuleAccessSetting(ctx, moduleaccess.Worm)
	require.NoError(t, err)
	require.False(t, saved.Open)
}
