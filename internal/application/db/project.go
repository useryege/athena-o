package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net"
	"net/url"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	log "github.com/sirupsen/logrus"
	"github.com/useryege/athena/internal/application"
	"github.com/useryege/athena/util/env"
)

const (
	postgresPingAttempts = 5
	postgresPingInterval = time.Second
)

type projectMetaStore struct {
	db *sql.DB
}

func NewProjectMetaStore(db *sql.DB) application.ProjectStore {
	return &projectMetaStore{db: db}
}

func NewProjectMetaStoreSource() func(context.Context) (application.ProjectStore, error) {
	return func(ctx context.Context) (application.ProjectStore, error) {
		log.Info("connecting to postgres database")
		db, err := sql.Open("pgx", postgresDSN())
		if err != nil {
			return nil, fmt.Errorf("failed to open postgres database: %w", err)
		}

		var pingErr error
		for attempt := 1; attempt <= postgresPingAttempts; attempt++ {
			pingCtx, cancel := context.WithTimeout(ctx, postgresPingInterval)
			pingErr = db.PingContext(pingCtx)
			cancel()
			if pingErr == nil {
				log.Info("successfully connected to postgres database")
				return NewProjectMetaStore(db), nil
			}

			log.Warnf("failed to ping postgres database, attempt %d/%d: %v", attempt, postgresPingAttempts, pingErr)
			if attempt < postgresPingAttempts {
				select {
				case <-ctx.Done():
					_ = db.Close()
					return nil, fmt.Errorf("postgres database ping interrupted: %w", ctx.Err())
				case <-time.After(postgresPingInterval):
				}
			}
		}

		if pingErr != nil {
			_ = db.Close()
			return nil, fmt.Errorf("failed to ping postgres database after %d attempts: %w", postgresPingAttempts, pingErr)
		}

		return NewProjectMetaStore(db), nil
	}
}

func postgresDSN() string {
	if dsn := env.StringFromEnv("ATHENA_APPLICATION_POSTGRES_DSN", ""); dsn != "" {
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
		Path:   env.StringFromEnv("POSTGRES_DB", "athena"),
	}
	if postgresPassword != "" {
		postgresURL.User = url.UserPassword(postgresUser, postgresPassword)
	}

	query := postgresURL.Query()
	query.Set("sslmode", "disable")
	postgresURL.RawQuery = query.Encode()

	return postgresURL.String()
}

func (s *projectMetaStore) Close() error {
	return s.db.Close()
}

func (s *projectMetaStore) SaveProjectMeta(ctx context.Context, meta application.ProjectMeta) error {
	if meta.Tx == nil {
		return errors.New("project meta transaction is nil")
	}

	_, err := s.db.ExecContext(ctx, `
INSERT INTO project (
  project_id,
  block_number,
  block_time,
  contract,
  creator,
  tx_hash,
  tx_index
) VALUES ($1, $2, $3, $4, $5, $6, $7)
ON CONFLICT DO NOTHING
`, meta.ProjectID, int64(meta.BlockNumber), int64(meta.BlockTime), meta.Contract.Bytes(), meta.Creator.Bytes(), meta.Tx.Hash().Bytes(), int64(meta.TxIndex))
	if err != nil {
		return fmt.Errorf("save project meta: %w", err)
	}
	return nil
}
