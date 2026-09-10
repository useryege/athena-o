package pgtest

import (
	"context"
	"fmt"
	"io/fs"
	"net"
	"net/url"
	"os"
	"strconv"
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
	ctx := context.Background()
	dsn := newDatabase(t)
	if err := postgres.Migrate(ctx, dsn, schema, dir); err != nil {
		t.Fatalf("migrate test database: %v", err)
	}
	return &DB{Pool: newPool(t, dsn), DSN: dsn}
}

func NewUnmigrated(t *testing.T) *DB {
	t.Helper()
	dsn := newDatabase(t)
	return &DB{Pool: newPool(t, dsn), DSN: dsn}
}

func newDatabase(t *testing.T) string {
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

	dsn, err := databaseDSN(adminDSN, database)
	if err != nil {
		t.Fatalf("replace test postgres database in DSN: %v", err)
	}
	return dsn
}

func newPool(t *testing.T, dsn string) *pgxpool.Pool {
	t.Helper()
	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		t.Fatalf("open test database pool: %v", err)
	}
	t.Cleanup(pool.Close)
	return pool
}

func databaseDSN(source string, database string) (string, error) {
	config, err := pgx.ParseConfig(source)
	if err != nil {
		return "", err
	}
	if config.Host == "" || config.User == "" {
		return "", fmt.Errorf("test postgres admin DSN must include host and user")
	}
	if !hasDisabledSSLMode(source) {
		return "", fmt.Errorf("test postgres admin DSN must set sslmode=disable")
	}
	target := &url.URL{
		Scheme: "postgres",
		User:   url.UserPassword(config.User, config.Password),
		Host:   net.JoinHostPort(config.Host, strconv.Itoa(int(config.Port))),
		Path:   database,
	}
	query := target.Query()
	query.Set("sslmode", "disable")
	target.RawQuery = query.Encode()
	return target.String(), nil
}

func hasDisabledSSLMode(source string) bool {
	if parsed, err := url.Parse(source); err == nil && parsed.Scheme != "" {
		return parsed.Query().Get("sslmode") == "disable"
	}
	for _, field := range strings.Fields(source) {
		if field == "sslmode=disable" {
			return true
		}
	}
	return false
}
