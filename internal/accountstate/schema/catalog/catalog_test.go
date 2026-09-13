//go:build integration

package catalog

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"
	"github.com/useryege/athena/internal/testutil/pgtest"
)

func TestReadCapturesStructureWithoutDataAndIgnoresSearchPath(t *testing.T) {
	db := pgtest.NewUnmigrated(t)
	ctx := context.Background()
	_, err := db.Pool.Exec(ctx, `CREATE TABLE public.example (id bigint PRIMARY KEY, value text NOT NULL DEFAULT 'initial'); CREATE INDEX example_value ON public.example(value)`)
	require.NoError(t, err)
	snapshot := func() []byte {
		tx, err := db.Pool.BeginTx(ctx, pgx.TxOptions{AccessMode: pgx.ReadOnly})
		require.NoError(t, err)
		defer tx.Rollback(ctx)
		_, err = tx.Exec(ctx, `SET LOCAL search_path = pg_catalog`)
		require.NoError(t, err)
		data, err := Read(ctx, tx)
		require.NoError(t, err)
		require.True(t, json.Valid(data))
		return data
	}
	before := snapshot()
	require.Contains(t, string(before), "example_value")
	require.Contains(t, string(before), "bigint")
	require.Contains(t, string(before), "NOT NULL")
	_, err = db.Pool.Exec(ctx, `INSERT INTO public.example VALUES (1, 'untracked data')`)
	require.NoError(t, err)
	require.Equal(t, before, snapshot())
	_, err = db.Pool.Exec(ctx, `ALTER TABLE public.example ALTER COLUMN value DROP NOT NULL`)
	require.NoError(t, err)
	require.NotEqual(t, before, snapshot())
}
