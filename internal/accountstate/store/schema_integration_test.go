//go:build integration

package store_test

import (
	"context"
	"sync"
	"testing"

	accountstatemigrations "github.com/useryege/athena/internal/accountstate/store/migrations"
	"github.com/useryege/athena/internal/testutil/pgtest"
	"github.com/useryege/athena/util/db/postgres"
)

func TestAthenaSchemaContainsAccountAndNotificationTables(t *testing.T) {
	db := pgtest.New(t, accountstatemigrations.FS, accountstatemigrations.Dir)
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
		if err := db.Pool.QueryRow(ctx, "SELECT to_regclass($1)::text", table).Scan(&relation); err != nil {
			t.Fatalf("look up %s: %v", table, err)
		}
		if relation != table {
			t.Fatalf("table %s relation=%q", table, relation)
		}
	}
}

func TestAthenaSchemaMigrateIsSafeAcrossTwoConnections(t *testing.T) {
	db := pgtest.New(t, accountstatemigrations.FS, accountstatemigrations.Dir)
	ctx := context.Background()

	var group sync.WaitGroup
	errs := make(chan error, 2)
	for range 2 {
		group.Add(1)
		go func() {
			defer group.Done()
			errs <- postgres.Migrate(ctx, db.DSN, accountstatemigrations.FS, accountstatemigrations.Dir)
		}()
	}
	group.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatalf("migrate athena schema concurrently: %v", err)
		}
	}

	var versions int
	if err := db.Pool.QueryRow(ctx, "SELECT count(*) FROM goose_db_version WHERE is_applied AND version_id = 1").Scan(&versions); err != nil {
		t.Fatalf("count applied athena migration versions: %v", err)
	}
	if versions != 1 {
		t.Fatalf("applied athena migration versions=%d, want 1", versions)
	}
}
