// Package schema owns operation_log migration and strictly read-only verification.
package schema

import (
	"bytes"
	"context"
	"database/sql"
	_ "embed"
	"errors"
	"fmt"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
	"github.com/pressly/goose/v3/database"
	"github.com/useryege/athena/internal/operationlog/schema/catalog"
	"github.com/useryege/athena/internal/operationlog/store/migrations"
	"github.com/useryege/athena/util/db/postgres"
	"strings"
	"time"
)

const DefaultTimeout = 120 * time.Second

var ErrVersions = errors.New("operation-log migration versions mismatch")
var ErrCatalog = errors.New("operation-log schema catalog mismatch")

//go:embed contract.json
var contract []byte

func Up(ctx context.Context, dsn string) error {
	if strings.TrimSpace(dsn) == "" {
		return errors.New("operation-log postgres DSN required")
	}
	ctx, cancel := context.WithTimeout(ctx, DefaultTimeout)
	defer cancel()
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return err
	}
	defer db.Close()
	conn, err := db.Conn(ctx)
	if err != nil {
		return err
	}
	defer conn.Close()
	if _, err = conn.ExecContext(ctx, `SELECT pg_advisory_lock(hashtext($1::text)::bigint)`, postgres.MigrationLockName); err != nil {
		return err
	}
	defer conn.ExecContext(ctx, `SELECT pg_advisory_unlock(hashtext($1::text)::bigint)`, postgres.MigrationLockName)
	if _, err = conn.ExecContext(ctx, `CREATE SCHEMA IF NOT EXISTS operation_log`); err != nil {
		return err
	}
	versionStore, err := database.NewStore(database.DialectPostgres, "operation_log.goose_db_version")
	if err != nil {
		return err
	}
	provider, err := goose.NewProvider(goose.DialectCustom, db, migrations.FS, goose.WithStore(versionStore), goose.WithDisableGlobalRegistry(true))
	if err != nil {
		return err
	}
	_, err = provider.Up(ctx)
	return err
}
func Verify(ctx context.Context, pool *pgxpool.Pool) error {
	ctx, cancel := context.WithTimeout(ctx, DefaultTimeout)
	defer cancel()
	tx, err := pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	var exists bool
	if err = tx.QueryRow(ctx, `SELECT to_regclass('operation_log.goose_db_version') IS NOT NULL`).Scan(&exists); err != nil {
		return err
	}
	if !exists {
		return ErrVersions
	}
	var versions []int64
	if err = tx.QueryRow(ctx, `SELECT coalesce(array_agg(version_id ORDER BY version_id),'{}'::bigint[]) FROM (SELECT DISTINCT ON(version_id) version_id,is_applied FROM operation_log.goose_db_version ORDER BY version_id,id DESC) e WHERE is_applied AND version_id<>0`).Scan(&versions); err != nil {
		return fmt.Errorf("%w: %v", ErrVersions, err)
	}
	if len(versions) != 1 || versions[0] != 1 {
		return ErrVersions
	}
	actual, err := catalog.Read(ctx, tx)
	if err != nil {
		return err
	}
	if !bytes.Equal(contract, actual) {
		return ErrCatalog
	}
	return tx.Commit(ctx)
}
