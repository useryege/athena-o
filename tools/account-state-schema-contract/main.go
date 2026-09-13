// account-state-schema-contract snapshots only a new randomly named database.
package main

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/useryege/athena/internal/accountstate/schema/catalog"
	"github.com/useryege/athena/internal/accountstate/store/migrations"
	"github.com/useryege/athena/util/db/postgres"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
func run() (resultErr error) {
	dsn := strings.TrimSpace(os.Getenv("ATHENA_TEST_PG_ADMIN_DSN"))
	if dsn == "" {
		return fmt.Errorf("ATHENA_TEST_PG_ADMIN_DSN is required")
	}
	if _, err := pgx.ParseConfig(dsn); err != nil {
		return fmt.Errorf("ATHENA_TEST_PG_ADMIN_DSN is malformed")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()
	admin, err := pgx.Connect(ctx, dsn)
	if err != nil {
		return err
	}
	defer admin.Close(context.Background())
	name := "athena_contract_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	ident := pgx.Identifier{name}.Sanitize()
	if _, err = admin.Exec(ctx, "CREATE DATABASE "+ident); err != nil {
		return err
	}
	defer func() {
		cleanup, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		if _, err := admin.Exec(cleanup, "DROP DATABASE "+ident+" WITH (FORCE)"); err != nil {
			resultErr = fmt.Errorf("drop contract database: %w", err)
		}
	}()
	target, err := databaseDSN(dsn, name)
	if err != nil {
		return err
	}
	config, err := pgxpool.ParseConfig(target)
	if err != nil {
		return err
	}
	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return err
	}
	defer pool.Close()
	var actualDatabase string
	if err := pool.QueryRow(ctx, "SELECT current_database()").Scan(&actualDatabase); err != nil {
		return err
	}
	if actualDatabase != name {
		return fmt.Errorf("refusing migration: target is not the generated database")
	}
	if err := postgres.Migrate(ctx, target, migrations.FS, migrations.Dir); err != nil {
		return err
	}
	tx, err := pool.BeginTx(ctx, pgx.TxOptions{AccessMode: pgx.ReadOnly})
	if err != nil {
		return err
	}
	defer tx.Rollback(context.Background())
	data, err := catalog.Read(ctx, tx)
	if err != nil {
		return err
	}
	if len(data) < 3 || !strings.Contains(string(data), `"name": "trader_sync_runtime_control"`) || !strings.Contains(string(data), `"name": "trader_sync_guard_attempt_terminal()"`) {
		return fmt.Errorf("generated schema catalog is incomplete")
	}
	_, err = os.Stdout.Write(data)
	return err
}

func databaseDSN(source, name string) (string, error) {
	if parsed, err := url.Parse(source); err == nil && (parsed.Scheme == "postgres" || parsed.Scheme == "postgresql") {
		parsed.Path = "/" + name
		query := parsed.Query()
		query.Del("dbname")
		parsed.RawQuery = query.Encode()
		return parsed.String(), nil
	}
	// libpq keyword parameters use the last occurrence; quote the generated name.
	return source + " dbname='" + strings.ReplaceAll(strings.ReplaceAll(name, `\`, `\\`), "'", `\'`) + "'", nil
}
