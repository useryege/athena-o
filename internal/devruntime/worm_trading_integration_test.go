//go:build integration

package devruntime

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/useryege/athena/internal/testutil/pgtest"
	wormstore "github.com/useryege/athena/internal/wormtrading/store"
)

func TestWormTradingExternalSchemaPreparationDoesNotMigrate(t *testing.T) {
	o := runnerOptions(t, "external", []string{"worm-trading"}, "")
	m := NewManager(o.Key)
	s := initialState(t, o.Key)
	s.Phase = "starting"
	s.DBMode = "external"
	s.Endpoints = map[string]string{}
	require.NoError(t, SaveState(o.Key.StatePath(), s))
	defer m.Stop(context.Background())
	db := pgtest.NewUnmigrated(t)
	env := map[string]string{wormstore.DSNEnv: db.DSN}
	require.Error(t, m.prepareDatabaseSchemas(context.Background(), env, []ServiceSpec{{Schemas: []string{"worm-trading"}}}, nil, ""))
	var exists bool
	require.NoError(t, db.Pool.QueryRow(context.Background(), `SELECT to_regclass('public.goose_db_version') IS NOT NULL`).Scan(&exists))
	require.False(t, exists)
}
