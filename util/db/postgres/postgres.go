package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"io/fs"
	"net"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
	log "github.com/sirupsen/logrus"
	"github.com/useryege/athena/util/env"
)

const (
	pingAttempts = 5
	pingInterval = time.Second

	AutoMigrateEnv = "ATHENA_POSTGRES_AUTO_MIGRATE"
)

var gooseMu sync.Mutex

var (
	runMigrations = Migrate
	openPool      = OpenPool
)

type Options struct {
	Module       string
	DSNEnv       string
	Database     string
	Migrations   fs.FS
	MigrationDir string
}

func ConnectAndMigrate(ctx context.Context, opts Options) (*pgxpool.Pool, error) {
	if opts.MigrationDir == "" {
		opts.MigrationDir = "migrations"
	}
	dsn := DSN(opts.DSNEnv, opts.Database)
	if opts.Migrations != nil {
		if AutoMigrateEnabled() {
			if err := runMigrations(ctx, dsn, opts.Migrations, opts.MigrationDir); err != nil {
				return nil, err
			}
		} else {
			log.Infof("automatic %s postgres migrations are disabled by %s", opts.Module, AutoMigrateEnv)
		}
	}
	return openPool(ctx, opts.Module, dsn)
}

func AutoMigrateEnabled() bool {
	return env.ParseBoolFromEnv(AutoMigrateEnv, true)
}

func OpenPool(ctx context.Context, module string, dsn string) (*pgxpool.Pool, error) {
	log.Infof("connecting to %s postgres database", module)
	config, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("parse %s postgres dsn: %w", module, err)
	}
	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("open %s postgres pool: %w", module, err)
	}

	var pingErr error
	for attempt := 1; attempt <= pingAttempts; attempt++ {
		pingCtx, cancel := context.WithTimeout(ctx, pingInterval)
		pingErr = pool.Ping(pingCtx)
		cancel()
		if pingErr == nil {
			log.Infof("successfully connected to %s postgres database", module)
			return pool, nil
		}

		log.Warnf("failed to ping %s postgres database, attempt %d/%d: %v", module, attempt, pingAttempts, pingErr)
		if attempt < pingAttempts {
			select {
			case <-ctx.Done():
				pool.Close()
				return nil, fmt.Errorf("%s postgres database ping interrupted: %w", module, ctx.Err())
			case <-time.After(pingInterval):
			}
		}
	}

	pool.Close()
	return nil, fmt.Errorf("failed to ping %s postgres database after %d attempts: %w", module, pingAttempts, pingErr)
}

func Migrate(ctx context.Context, dsn string, migrations fs.FS, dir string) error {
	return withMigrationDB(ctx, dsn, func(db *sql.DB) error {
		gooseMu.Lock()
		defer gooseMu.Unlock()

		return withMigrationAdvisoryLock(ctx, db, migrationLockName(dsn, dir), func() error {
			goose.SetBaseFS(migrations)
			defer goose.SetBaseFS(nil)
			if err := goose.SetDialect("postgres"); err != nil {
				return fmt.Errorf("set goose postgres dialect: %w", err)
			}
			if err := goose.UpContext(ctx, db, dir); err != nil {
				return fmt.Errorf("run postgres migrations: %w", err)
			}
			return nil
		})
	})
}

func MigrationStatus(ctx context.Context, dsn string, migrations fs.FS, dir string) error {
	return withMigrationDB(ctx, dsn, func(db *sql.DB) error {
		gooseMu.Lock()
		defer gooseMu.Unlock()

		return withMigrationAdvisoryLock(ctx, db, migrationLockName(dsn, dir), func() error {
			goose.SetBaseFS(migrations)
			defer goose.SetBaseFS(nil)
			if err := goose.SetDialect("postgres"); err != nil {
				return fmt.Errorf("set goose postgres dialect: %w", err)
			}
			if err := goose.StatusContext(ctx, db, dir); err != nil {
				return fmt.Errorf("show postgres migration status: %w", err)
			}
			return nil
		})
	})
}

func withMigrationAdvisoryLock(ctx context.Context, db *sql.DB, lockName string, fn func() error) error {
	conn, err := db.Conn(ctx)
	if err != nil {
		return fmt.Errorf("open postgres migration lock connection: %w", err)
	}
	defer conn.Close()

	if _, err := conn.ExecContext(ctx, "SELECT pg_advisory_lock(hashtext($1::text)::bigint)", lockName); err != nil {
		return fmt.Errorf("acquire postgres migration lock: %w", err)
	}
	defer func() {
		if _, err := conn.ExecContext(context.Background(), "SELECT pg_advisory_unlock(hashtext($1::text)::bigint)", lockName); err != nil {
			log.Warnf("failed to release postgres migration lock %q: %v", lockName, err)
		}
	}()

	return fn()
}

func migrationLockName(dsn string, dir string) string {
	if parsed, err := url.Parse(dsn); err == nil && parsed.Host != "" {
		database := strings.TrimPrefix(parsed.Path, "/")
		if database == "" {
			database = parsed.Query().Get("dbname")
		}
		return fmt.Sprintf("%s://%s/%s:%s", parsed.Scheme, parsed.Host, database, dir)
	}
	return fmt.Sprintf("%s:%s", dsn, dir)
}

func withMigrationDB(ctx context.Context, dsn string, fn func(*sql.DB) error) error {
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return fmt.Errorf("open postgres migration database: %w", err)
	}
	defer db.Close()

	var pingErr error
	for attempt := 1; attempt <= pingAttempts; attempt++ {
		pingCtx, cancel := context.WithTimeout(ctx, pingInterval)
		pingErr = db.PingContext(pingCtx)
		cancel()
		if pingErr == nil {
			break
		}

		log.Warnf("failed to ping postgres migration database, attempt %d/%d: %v", attempt, pingAttempts, pingErr)
		if attempt < pingAttempts {
			select {
			case <-ctx.Done():
				return fmt.Errorf("postgres migration database ping interrupted: %w", ctx.Err())
			case <-time.After(pingInterval):
			}
		}
	}
	if pingErr != nil {
		return fmt.Errorf("failed to ping postgres migration database after %d attempts: %w", pingAttempts, pingErr)
	}
	return fn(db)
}

func DSN(envName, database string) string {
	if dsn := env.StringFromEnv(envName, ""); dsn != "" {
		return dsn
	}
	return defaultDSN(database)
}

func defaultDSN(database string) string {
	postgresUser := env.StringFromEnv("POSTGRES_USER", "athena")
	postgresPassword := env.StringFromEnv("POSTGRES_PASSWORD", "")

	postgresURL := url.URL{
		Scheme: "postgres",
		User:   url.User(postgresUser),
		Host:   net.JoinHostPort("127.0.0.1", env.StringFromEnv("ATHENA_POSTGRES_PORT", "5432")),
		Path:   database,
	}
	if postgresPassword != "" {
		postgresURL.User = url.UserPassword(postgresUser, postgresPassword)
	}

	query := postgresURL.Query()
	query.Set("sslmode", "disable")
	postgresURL.RawQuery = query.Encode()

	return postgresURL.String()
}
