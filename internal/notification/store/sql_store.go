package store

import (
	"context"
	"database/sql"
	"fmt"
	"net"
	"net/url"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	log "github.com/sirupsen/logrus"
	"github.com/useryege/athena/util/env"
)

const (
	postgresPingAttempts = 5
	postgresPingInterval = time.Second
)

type SQLStore struct {
	db *sql.DB
}

func NewSQLStore(db *sql.DB) *SQLStore {
	return &SQLStore{db: db}
}

func NewSQLStoreSource() func(context.Context) (*SQLStore, error) {
	return func(ctx context.Context) (*SQLStore, error) {
		log.Info("connecting to notification postgres database")
		db, err := sql.Open("pgx", postgresDSN())
		if err != nil {
			return nil, fmt.Errorf("failed to open notification postgres database: %w", err)
		}

		var pingErr error
		for attempt := 1; attempt <= postgresPingAttempts; attempt++ {
			pingCtx, cancel := context.WithTimeout(ctx, postgresPingInterval)
			pingErr = db.PingContext(pingCtx)
			cancel()
			if pingErr == nil {
				log.Info("successfully connected to notification postgres database")
				return NewSQLStore(db), nil
			}

			log.Warnf("failed to ping notification postgres database, attempt %d/%d: %v", attempt, postgresPingAttempts, pingErr)
			if attempt < postgresPingAttempts {
				select {
				case <-ctx.Done():
					_ = db.Close()
					return nil, fmt.Errorf("notification postgres database ping interrupted: %w", ctx.Err())
				case <-time.After(postgresPingInterval):
				}
			}
		}

		_ = db.Close()
		return nil, fmt.Errorf("failed to ping notification postgres database after %d attempts: %w", postgresPingAttempts, pingErr)
	}
}

func postgresDSN() string {
	if dsn := env.StringFromEnv("ATHENA_NOTIFICATION_POSTGRES_DSN", ""); dsn != "" {
		return dsn
	}
	return defaultPostgresDSN()
}

func defaultPostgresDSN() string {
	postgresUser := env.StringFromEnv("POSTGRES_USER", "athena")
	postgresPassword := env.StringFromEnv("POSTGRES_PASSWORD", "")

	postgresURL := url.URL{
		Scheme: "postgres",
		User:   url.User(postgresUser),
		Host:   net.JoinHostPort("127.0.0.1", env.StringFromEnv("ATHENA_POSTGRES_PORT", "5432")),
		Path:   "notification",
	}
	if postgresPassword != "" {
		postgresURL.User = url.UserPassword(postgresUser, postgresPassword)
	}

	query := postgresURL.Query()
	query.Set("sslmode", "disable")
	postgresURL.RawQuery = query.Encode()

	return postgresURL.String()
}

func (s *SQLStore) Close() error {
	if s.db == nil {
		return nil
	}
	return s.db.Close()
}
