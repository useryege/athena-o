package pgtest

import (
	"context"
	"io/fs"
	"os"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/useryege/athena/util/db/postgres"
)

const adminDSNEnv = "ATHENA_TEST_PG_ADMIN_DSN"

type DB struct {
	Pool *pgxpool.Pool
	DSN  string
}

func New(t *testing.T, schema fs.FS, dir string) *DB {
	t.Helper()
	adminDSN := strings.TrimSpace(os.Getenv(adminDSNEnv))
	if adminDSN == "" {
		t.Fatalf("%s is required", adminDSNEnv)
	}
	ctx := context.Background()
	admin, err := pgx.Connect(ctx, adminDSN)
	if err != nil {
		t.Fatalf("connect test postgres admin: %v", err)
	}
	database := "athena_test_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	if _, err := admin.Exec(ctx, "CREATE DATABASE "+pgx.Identifier{database}.Sanitize()); err != nil {
		admin.Close(ctx)
		t.Fatalf("create test database: %v", err)
	}
	t.Cleanup(func() {
		if _, err := admin.Exec(context.Background(), "DROP DATABASE "+pgx.Identifier{database}.Sanitize()+" WITH (FORCE)"); err != nil {
			t.Errorf("drop test database %q: %v", database, err)
		}
		admin.Close(context.Background())
	})

	config, err := pgxpool.ParseConfig(adminDSN)
	if err != nil {
		t.Fatalf("parse test postgres admin DSN: %v", err)
	}
	config.ConnConfig.Database = database
	dsn := config.ConnString()
	if err := postgres.Migrate(ctx, dsn, schema, dir); err != nil {
		t.Fatalf("migrate test database: %v", err)
	}
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("open test database pool: %v", err)
	}
	t.Cleanup(pool.Close)
	return &DB{Pool: pool, DSN: dsn}
}
