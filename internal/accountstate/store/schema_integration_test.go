//go:build integration

package store_test

import (
	"bytes"
	"context"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	accountstatemigrations "github.com/useryege/athena/internal/accountstate/store/migrations"
	"github.com/useryege/athena/internal/testutil/pgtest"
	"github.com/useryege/athena/util/db/postgres"
)

func TestAthenaSchemaContainsAccountAndNotificationTables(t *testing.T) {
	db := pgtest.New(t, accountstatemigrations.FS, accountstatemigrations.Dir)
	assertAthenaTables(t, db.Pool)
}

func TestAthenaSchemaFixtureMigratesOnlyCreatedDatabase(t *testing.T) {
	adminDSN := os.Getenv("ATHENA_TEST_PG_ADMIN_DSN")
	db := pgtest.New(t, accountstatemigrations.FS, accountstatemigrations.Dir)
	ctx := context.Background()

	var currentDatabase string
	if err := db.Pool.QueryRow(ctx, "SELECT current_database()").Scan(&currentDatabase); err != nil {
		t.Fatalf("read test database name: %v", err)
	}
	targetDatabase := databaseName(t, db.DSN)
	adminDatabase := databaseName(t, adminDSN)
	if currentDatabase != targetDatabase {
		t.Fatalf("current database=%q, target database=%q", currentDatabase, targetDatabase)
	}
	if currentDatabase == adminDatabase {
		t.Fatalf("fixture migrated admin database %q", adminDatabase)
	}

	admin, err := pgx.Connect(ctx, adminDSN)
	if err != nil {
		t.Fatalf("connect admin database: %v", err)
	}
	defer admin.Close(ctx)
	var publicTables int
	if err := admin.QueryRow(ctx, "SELECT count(*) FROM pg_tables WHERE schemaname = 'public'").Scan(&publicTables); err != nil {
		t.Fatalf("count admin public tables: %v", err)
	}
	if publicTables != 0 {
		t.Fatalf("admin database has %d public tables after fixture migration", publicTables)
	}
}

func TestAthenaSchemaMigrateIsSafeAcrossTwoProcesses(t *testing.T) {
	if os.Getenv("ATHENA_PGTEST_MIGRATION_HELPER") == "1" {
		if err := postgres.Migrate(context.Background(), os.Getenv("ATHENA_PGTEST_DSN"), accountstatemigrations.FS, accountstatemigrations.Dir); err != nil {
			t.Fatalf("migrate in helper process: %v", err)
		}
		return
	}
	db := pgtest.NewUnmigrated(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	lock, err := pgx.Connect(ctx, db.DSN)
	if err != nil {
		t.Fatalf("connect migration lock holder: %v", err)
	}
	defer lock.Close(context.Background())
	lockName := migrationLockName(t, db.DSN, accountstatemigrations.Dir)
	if _, err := lock.Exec(ctx, "SELECT pg_advisory_lock(hashtext($1::text)::bigint)", lockName); err != nil {
		t.Fatalf("acquire migration lock: %v", err)
	}
	locked := true
	defer func() {
		if locked {
			_, _ = lock.Exec(context.Background(), "SELECT pg_advisory_unlock(hashtext($1::text)::bigint)", lockName)
		}
	}()

	commands := []*exec.Cmd{
		migrationProcess(ctx, db.DSN),
		migrationProcess(ctx, db.DSN),
	}
	for _, command := range commands {
		if err := command.Start(); err != nil {
			t.Fatalf("start migration helper: %v", err)
		}
	}
	if err := waitForMigrationWaiters(ctx, db.Pool, 2); err != nil {
		t.Fatalf("wait for independent migration lock contenders: %v", err)
	}
	if _, err := lock.Exec(ctx, "SELECT pg_advisory_unlock(hashtext($1::text)::bigint)", lockName); err != nil {
		t.Fatalf("release migration lock: %v", err)
	}
	locked = false
	for _, command := range commands {
		if err := command.Wait(); err != nil {
			t.Fatalf("migration helper failed: %v\n%s", err, commandOutput(command))
		}
	}

	assertAthenaTables(t, db.Pool)
	assertOneAppliedMigration(t, db.Pool)
}

func assertAthenaTables(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	ctx := context.Background()

	for _, table := range []string{
		"athena_account",
		"account_access",
		"telegram_bindings",
		"account_notification_deliveries",
		"system_notification_deliveries",
		"telegram_polling_state",
	} {
		var relation string
		if err := pool.QueryRow(ctx, "SELECT to_regclass($1)::text", table).Scan(&relation); err != nil {
			t.Fatalf("look up %s: %v", table, err)
		}
		if relation != table {
			t.Fatalf("table %s relation=%q", table, relation)
		}
	}
}

func assertOneAppliedMigration(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	var versions int
	if err := pool.QueryRow(context.Background(), "SELECT count(*) FROM goose_db_version WHERE is_applied AND version_id = 1").Scan(&versions); err != nil {
		t.Fatalf("count applied athena migration versions: %v", err)
	}
	if versions != 1 {
		t.Fatalf("applied athena migration versions=%d, want 1", versions)
	}
}

func databaseName(t *testing.T, dsn string) string {
	t.Helper()
	parsed, err := pgx.ParseConfig(dsn)
	if err != nil {
		t.Fatalf("parse DSN: %v", err)
	}
	return parsed.Database
}

func migrationProcess(ctx context.Context, dsn string) *exec.Cmd {
	command := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestAthenaSchemaMigrateIsSafeAcrossTwoProcesses$", "-test.count=1")
	command.Env = append(os.Environ(), "ATHENA_PGTEST_MIGRATION_HELPER=1", "ATHENA_PGTEST_DSN="+dsn)
	output := new(bytes.Buffer)
	command.Stdout = output
	command.Stderr = output
	return command
}

func waitForMigrationWaiters(ctx context.Context, pool *pgxpool.Pool, want int) error {
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()
	for {
		var waiters int
		if err := pool.QueryRow(ctx, "SELECT count(*) FROM pg_stat_activity WHERE datname = current_database() AND wait_event_type = 'Lock' AND query LIKE 'SELECT pg_advisory_lock%'").Scan(&waiters); err != nil {
			return err
		}
		if waiters >= want {
			return nil
		}
		select {
		case <-ctx.Done():
			return fmt.Errorf("migration lock waiters=%d, want at least %d: %w", waiters, want, ctx.Err())
		case <-ticker.C:
		}
	}
}

func migrationLockName(t *testing.T, dsn string, dir string) string {
	t.Helper()
	parsed, err := url.Parse(dsn)
	if err != nil {
		t.Fatalf("parse migration DSN: %v", err)
	}
	return fmt.Sprintf("%s://%s/%s:%s", parsed.Scheme, parsed.Host, parsed.Path[1:], dir)
}

func commandOutput(command *exec.Cmd) string {
	if output, ok := command.Stdout.(*bytes.Buffer); ok {
		return output.String()
	}
	return ""
}
