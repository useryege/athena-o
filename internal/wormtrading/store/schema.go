package store

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const DSNEnv = "ATHENA_WORM_TRADING_POSTGRES_DSN"
const DefaultTimeout = 120 * time.Second

var ErrVersions = errors.New("Worm Trading migration versions mismatch")
var ErrSchemaRelations = errors.New("Worm Trading required relation missing")

// Required relations are the current CREATE TABLE names in the embedded migrations.
// Verification checks their existence and effective migration versions, not a full catalog fingerprint.
var requiredTables = []string{
	"worm_wallet_connections",
	"worm_wallet_credentials",
	"worm_wallet_connection_attempts",
	"worm_market_combinations",
	"worm_market_combination_items",
	"worm_execution_plans",
	"worm_execution_plan_wallets",
	"worm_execution_plan_items",
	"worm_execution_plan_steps",
	"worm_execution_runs",
	"worm_execution_run_wallets",
	"worm_execution_run_items",
	"worm_execution_run_steps",
	"worm_execution_authorizations",
	"worm_execution_coordinators",
	"worm_execution_commands",
	"worm_execution_mutation_attempts",
	"worm_execution_combination_locks",
	"worm_execution_wallet_locks",
	"worm_execution_step_isolations",
	"worm_position_cash_outs",
	"worm_position_cash_out_authorizations",
	"worm_position_cash_out_commands",
	"worm_position_cash_out_attempts",
	"worm_position_cash_out_batches",
	"worm_position_cash_out_batch_wallets",
	"worm_position_cash_out_batch_wallet_locks",
	"worm_position_cash_out_batch_items",
	"worm_position_cash_out_batch_authorizations",
	"worm_position_cash_out_batch_commands",
	"worm_trading_wallet_selections",
	"worm_trading_wallet_selection_items",
	"worm_trading_wallet_retirements",
}

// VerifySchema never initializes goose metadata, including on an empty database.
func VerifySchema(ctx context.Context, pool *pgxpool.Pool) error {
	ctx, cancel := context.WithTimeout(ctx, DefaultTimeout)
	defer cancel()
	tx, err := pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if err != nil {
		return fmt.Errorf("begin Worm Trading schema verification: %w", err)
	}
	// Cleanup shares the remaining verification budget. pgx closes a connection
	// on rollback failure, and pgxpool discards it instead of returning it idle.
	defer tx.Rollback(ctx)
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
	for _, table := range requiredTables {
		if err := tx.QueryRow(ctx, `SELECT to_regclass($1) IS NOT NULL`, "public."+table).Scan(&exists); err != nil {
			return err
		}
		if !exists {
			return fmt.Errorf("%w: %s", ErrSchemaRelations, table)
		}
	}
	return tx.Commit(ctx)
}
func migrationVersions() ([]int64, error) {
	files, err := fs.ReadDir(migrations, "migrations")
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
