//go:build integration

package store

import (
	"context"
	"io/fs"
	"net/url"
	"regexp"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/useryege/athena/internal/testutil/pgtest"
)

func TestTradingSourceOnlyVerifiesExplicitDatabase(t *testing.T) {
	db := pgtest.NewUnmigrated(t)
	t.Setenv("ATHENA_WORM_TRADING_POSTGRES_DSN", db.DSN)
	t.Setenv("ATHENA_POSTGRES_AUTO_MIGRATE", "true")
	source, err := NewSQLStoreSource()(context.Background())
	if source != nil {
		source.Close()
	}
	require.Error(t, err, "service must reject an empty schema instead of migrating it")
	var exists bool
	require.NoError(t, db.Pool.QueryRow(context.Background(), `SELECT to_regclass('public.goose_db_version') IS NOT NULL`).Scan(&exists))
	require.False(t, exists)
}

func TestTradingVerifyRejectsVersionHolesAndMissingRelations(t *testing.T) {
	for _, tc := range []struct{ name, sql string }{
		{"older", `DELETE FROM goose_db_version WHERE version_id >= 10`},
		{"future", `INSERT INTO goose_db_version(version_id,is_applied) VALUES(999,true)`},
		{"hole", `DELETE FROM goose_db_version WHERE version_id=1`},
		{"reverted", `INSERT INTO goose_db_version(version_id,is_applied) VALUES(1,false)`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			db := pgtest.New(t, Migrations(), "migrations")
			require.NoError(t, VerifySchema(context.Background(), db.Pool))
			_, err := db.Pool.Exec(context.Background(), tc.sql)
			require.NoError(t, err)
			require.ErrorIs(t, VerifySchema(context.Background(), db.Pool), ErrVersions)
		})
	}
	db := pgtest.New(t, Migrations(), "migrations")
	files, err := fs.Glob(Migrations(), "migrations/*.sql")
	require.NoError(t, err)
	var relations []string
	pattern := regexp.MustCompile(`CREATE TABLE (?:IF NOT EXISTS )?([a-z_]+)`)
	for _, file := range files {
		data, err := fs.ReadFile(Migrations(), file)
		require.NoError(t, err)
		for _, match := range pattern.FindAllStringSubmatch(string(data), -1) {
			relations = append(relations, match[1])
		}
	}
	require.NotEmpty(t, relations)
	for _, table := range relations {
		t.Run(table, func(t *testing.T) {
			// Rename preserves foreign keys and allows one fixture to test every required relation.
			_, err := db.Pool.Exec(context.Background(), `ALTER TABLE `+table+` RENAME TO absent_relation`)
			require.NoError(t, err)
			require.ErrorIs(t, VerifySchema(context.Background(), db.Pool), ErrSchemaRelations)
			_, err = db.Pool.Exec(context.Background(), `ALTER TABLE absent_relation RENAME TO `+table)
			require.NoError(t, err)
		})
	}
}

func TestTradingSourceSupportsReadOnlyRoleAndClosesIndependentPools(t *testing.T) {
	db := pgtest.New(t, Migrations(), "migrations")
	ctx := context.Background()
	role := "trading_verify_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	_, err := db.Pool.Exec(ctx, `CREATE ROLE `+role+` LOGIN PASSWORD 'verification-only'`)
	require.NoError(t, err)
	defer func() {
		_, err := db.Pool.Exec(ctx, `DROP OWNED BY `+role+`; DROP ROLE `+role)
		require.NoError(t, err)
	}()
	_, err = db.Pool.Exec(ctx, `GRANT USAGE ON SCHEMA public TO `+role+`; GRANT SELECT ON ALL TABLES IN SCHEMA public TO `+role)
	require.NoError(t, err)
	u, err := url.Parse(db.DSN)
	require.NoError(t, err)
	u.User = url.UserPassword(role, "verification-only")
	t.Setenv(DSNEnv, u.String())
	a, err := NewSQLStoreSource()(ctx)
	require.NoError(t, err)
	defer a.Close()
	b, err := NewSQLStoreSource()(ctx)
	require.NoError(t, err)
	defer b.Close()
	require.NotSame(t, a.pool, b.pool)
	require.NoError(t, a.Close())
	require.NoError(t, b.Ping(ctx))
	_, err = b.pool.Exec(ctx, `CREATE TABLE forbidden(id int)`)
	require.Error(t, err)
	require.NoError(t, b.Close())
	require.Zero(t, a.pool.Stat().TotalConns())
	require.Zero(t, b.pool.Stat().TotalConns())
}
