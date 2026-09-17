// Package serviceschema provides explicit schema preparation and read-only
// startup checks for services with existing embedded migrations.
package serviceschema

import (
	"context"
	"fmt"
	"io/fs"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/useryege/athena/util/db/postgres"
)

const DefaultTimeout = 120 * time.Second

// Spec verifies the effective Goose versions and required relations/columns.
// It intentionally does not fingerprint unrelated schemas or data.
// CustomUp preserves a service's existing non-Goose migration implementation.
type Spec struct {
	Module     string
	DSN        func() string
	Migrations fs.FS
	Relations  map[string][]string
	CustomUp   func(context.Context, *pgxpool.Pool) error
}

func (s Spec) Up(ctx context.Context) error {
	if strings.TrimSpace(s.DSN()) == "" {
		return fmt.Errorf("%s schema requires a database DSN", s.Module)
	}
	ctx, cancel := context.WithTimeout(ctx, DefaultTimeout)
	defer cancel()
	if s.CustomUp == nil {
		return postgres.Migrate(ctx, s.DSN(), s.Migrations, "migrations")
	}
	pool, err := postgres.OpenPool(ctx, s.Module, s.DSN())
	if err != nil {
		return err
	}
	defer pool.Close()
	return s.CustomUp(ctx, pool)
}
func (s Spec) ConnectVerified(ctx context.Context) (*pgxpool.Pool, error) {
	if strings.TrimSpace(s.DSN()) == "" {
		return nil, fmt.Errorf("%s schema requires a database DSN", s.Module)
	}
	ctx, cancel := context.WithTimeout(ctx, DefaultTimeout)
	defer cancel()
	pool, err := postgres.OpenPool(ctx, s.Module, s.DSN())
	if err != nil {
		return nil, err
	}
	if err = s.Verify(ctx, pool); err != nil {
		pool.Close()
		return nil, err
	}
	return pool, nil
}
func (s Spec) Verify(ctx context.Context, pool *pgxpool.Pool) error {
	ctx, cancel := context.WithTimeout(ctx, DefaultTimeout)
	defer cancel()
	tx, err := pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if s.Migrations != nil {
		expected, err := migrationVersions(s.Migrations)
		if err != nil {
			return err
		}
		rows, err := tx.Query(ctx, `SELECT version_id FROM (SELECT DISTINCT ON (version_id) version_id,is_applied FROM public.goose_db_version ORDER BY version_id,id DESC) effective WHERE is_applied AND version_id<>0 ORDER BY version_id`)
		if err != nil {
			return fmt.Errorf("%s schema version: %w", s.Module, err)
		}
		actual := []int64{}
		for rows.Next() {
			var version int64
			if err := rows.Scan(&version); err != nil {
				rows.Close()
				return err
			}
			actual = append(actual, version)
		}
		rows.Close()
		if err := rows.Err(); err != nil {
			return err
		}
		if !slices.Equal(actual, expected) {
			return fmt.Errorf("%s schema versions: expected %v, found %v", s.Module, expected, actual)
		}
	}
	names := make([]string, 0, len(s.Relations))
	for name := range s.Relations {
		names = append(names, name)
	}
	slices.Sort(names)
	for _, name := range names {
		cols := s.Relations[name]
		quoted := make([]string, len(cols))
		for i, col := range cols {
			quoted[i] = pgx.Identifier{col}.Sanitize()
		}
		rows, err := tx.Query(ctx, "SELECT "+strings.Join(quoted, ",")+" FROM "+pgx.Identifier(strings.Split(name, ".")).Sanitize()+" LIMIT 0")
		if err != nil {
			return fmt.Errorf("%s schema relation %s: %w", s.Module, name, err)
		}
		rows.Close()
		if err := rows.Err(); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}
func migrationVersions(files fs.FS) ([]int64, error) {
	entries, err := fs.ReadDir(files, "migrations")
	if err != nil {
		return nil, err
	}
	versions := []int64{}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}
		prefix, _, ok := strings.Cut(entry.Name(), "_")
		if !ok {
			return nil, fmt.Errorf("invalid migration %s", entry.Name())
		}
		v, err := strconv.ParseInt(prefix, 10, 64)
		if err != nil {
			return nil, err
		}
		versions = append(versions, v)
	}
	slices.Sort(versions)
	return versions, nil
}
