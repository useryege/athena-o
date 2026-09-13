// Package schema owns explicit account-state migration and read-only verification.
package schema

import (
	"bytes"
	"context"
	_ "embed"
	"errors"
	"fmt"
	"io/fs"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/useryege/athena/internal/accountstate/schema/catalog"
	"github.com/useryege/athena/internal/accountstate/store/migrations"
	"github.com/useryege/athena/util/db/postgres"
)

const MigrationLockName = postgres.MigrationLockName
const DefaultTimeout = 120 * time.Second

var (
	ErrVersions = errors.New("account-state migration versions mismatch")
	ErrCatalog  = errors.New("account-state schema catalog mismatch")
)

//go:embed contract.json
var contract []byte

func Up(ctx context.Context, dsn string) error {
	if err := validateDSN(dsn); err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(ctx, DefaultTimeout)
	defer cancel()
	return postgres.Migrate(ctx, dsn, migrations.FS, migrations.Dir)
}

// Verify runs exclusively inside a read-only transaction and never calls goose
// status/version APIs, which initialize version metadata on an empty database.
func Verify(ctx context.Context, pool *pgxpool.Pool) error {
	ctx, cancel := context.WithTimeout(ctx, DefaultTimeout)
	defer cancel()
	tx, err := pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if err != nil {
		return fmt.Errorf("begin account-state schema verification: %w", err)
	}
	defer tx.Rollback(context.Background())
	var exists bool
	if err := tx.QueryRow(ctx, `SELECT to_regclass('public.goose_db_version') IS NOT NULL`).Scan(&exists); err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("%w: metadata table missing", ErrVersions)
	}
	rows, err := tx.Query(ctx, `SELECT version_id FROM (SELECT DISTINCT ON (version_id) version_id,is_applied FROM public.goose_db_version ORDER BY version_id,id DESC) effective WHERE is_applied AND version_id<>0 ORDER BY version_id`)
	if err != nil {
		return fmt.Errorf("%w: cannot read version records: %v", ErrVersions, err)
	}
	actual := []int64{}
	for rows.Next() {
		var version int64
		if err := rows.Scan(&version); err != nil {
			rows.Close()
			return fmt.Errorf("%w: %v", ErrVersions, err)
		}
		actual = append(actual, version)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return fmt.Errorf("%w: %v", ErrVersions, err)
	}
	expected, err := migrationVersions()
	if err != nil {
		return err
	}
	if !slices.Equal(actual, expected) {
		return fmt.Errorf("%w: expected %v, found %v", ErrVersions, expected, actual)
	}
	snapshot, err := catalog.Read(ctx, tx)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrCatalog, err)
	}
	if !bytes.Equal(snapshot, contract) {
		return ErrCatalog
	}
	return tx.Commit(ctx)
}
func migrationVersions() ([]int64, error) {
	files, err := fs.ReadDir(migrations.FS, migrations.Dir)
	if err != nil {
		return nil, err
	}
	versions := []int64{}
	for _, file := range files {
		if file.IsDir() || !strings.HasSuffix(file.Name(), ".sql") {
			continue
		}
		prefix, _, ok := strings.Cut(file.Name(), "_")
		if !ok {
			return nil, fmt.Errorf("invalid migration filename %q", file.Name())
		}
		version, err := strconv.ParseInt(prefix, 10, 64)
		if err != nil {
			return nil, err
		}
		versions = append(versions, version)
	}
	slices.Sort(versions)
	return versions, nil
}

// ConnectVerified returns a separately owned pool; callers close it on shutdown.
func ConnectVerified(ctx context.Context, dsn string) (*pgxpool.Pool, error) {
	if err := validateDSN(dsn); err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(ctx, DefaultTimeout)
	defer cancel()
	pool, err := postgres.OpenPool(ctx, "account-state", dsn)
	if err != nil {
		return nil, err
	}
	if err := Verify(ctx, pool); err != nil {
		pool.Close()
		return nil, err
	}
	return pool, nil
}
