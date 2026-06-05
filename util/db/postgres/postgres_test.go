package postgres

import (
	"context"
	"io/fs"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"testing/fstest"
)

func TestAutoMigrateEnabledDefaultsTrue(t *testing.T) {
	t.Setenv(AutoMigrateEnv, "")

	if !AutoMigrateEnabled() {
		t.Fatalf("AutoMigrateEnabled() = false, want true")
	}
}

func TestConnectAndMigrateRunsMigrationWhenEnabled(t *testing.T) {
	t.Setenv(AutoMigrateEnv, "true")

	var migrated bool
	restore := replaceConnectHooks(
		func(context.Context, string, fs.FS, string) error {
			migrated = true
			return nil
		},
		func(context.Context, string, string) (*pgxpool.Pool, error) {
			return nil, nil
		},
	)
	defer restore()

	_, err := ConnectAndMigrate(context.Background(), Options{
		Module:     "test",
		Database:   "test",
		Migrations: fstest.MapFS{"migrations/000001_init.sql": &fstest.MapFile{Data: []byte("-- +goose Up\nSELECT 1;")}},
	})
	if err != nil {
		t.Fatalf("ConnectAndMigrate() error = %v", err)
	}
	if !migrated {
		t.Fatalf("migration was not run")
	}
}

func TestConnectAndMigrateSkipsMigrationWhenDisabled(t *testing.T) {
	t.Setenv(AutoMigrateEnv, "false")

	var migrated bool
	restore := replaceConnectHooks(
		func(context.Context, string, fs.FS, string) error {
			migrated = true
			return nil
		},
		func(context.Context, string, string) (*pgxpool.Pool, error) {
			return nil, nil
		},
	)
	defer restore()

	_, err := ConnectAndMigrate(context.Background(), Options{
		Module:     "test",
		Database:   "test",
		Migrations: fstest.MapFS{"migrations/000001_init.sql": &fstest.MapFile{Data: []byte("-- +goose Up\nSELECT 1;")}},
	})
	if err != nil {
		t.Fatalf("ConnectAndMigrate() error = %v", err)
	}
	if migrated {
		t.Fatalf("migration was run while %s=false", AutoMigrateEnv)
	}
}

func TestMigrationLockNameOmitsCredentials(t *testing.T) {
	got := migrationLockName("postgres://athena:secret@127.0.0.1:5432/application?sslmode=disable", "migrations")
	want := "postgres://127.0.0.1:5432/application:migrations"
	if got != want {
		t.Fatalf("migrationLockName() = %q, want %q", got, want)
	}
}

func replaceConnectHooks(
	migrate func(context.Context, string, fs.FS, string) error,
	open func(context.Context, string, string) (*pgxpool.Pool, error),
) func() {
	previousMigrate := runMigrations
	previousOpen := openPool
	runMigrations = migrate
	openPool = open
	return func() {
		runMigrations = previousMigrate
		openPool = previousOpen
	}
}
