package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/useryege/athena/internal/application"
	"github.com/useryege/athena/util/env"
)

type projectMetaStore struct {
	db *sql.DB
}

func AddProjectMetaStoreFlagsToCmd(cmd *cobra.Command) func(context.Context) (application.ProjectMetaStore, error) {
	var postgresDSN string
	defaultDSN := fmt.Sprintf("host=127.0.0.1 port=%s user=%s dbname=%s sslmode=disable",
		env.StringFromEnv("ATHENA_POSTGRES_PORT", "5432"),
		env.StringFromEnv("POSTGRES_USER", "athena"),
		env.StringFromEnv("POSTGRES_DB", "athena"),
	)
	cmd.Flags().StringVar(&postgresDSN, "postgres-dsn", env.StringFromEnv("ATHENA_APPLICATION_POSTGRES_DSN", defaultDSN), "PostgreSQL DSN")

	return func(ctx context.Context) (application.ProjectMetaStore, error) {
		if postgresDSN == "" {
			return application.NewNoopProjectMetaStore(), nil
		}

		log.Info("connecting to postgres database")
		db, err := sql.Open("pgx", postgresDSN)
		if err != nil {
			return nil, fmt.Errorf("failed to open postgres database: %w", err)
		}

		pingCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()
		if err := db.PingContext(pingCtx); err != nil {
			_ = db.Close()
			return nil, fmt.Errorf("failed to ping postgres database: %w", err)
		}

		log.Info("successfully connected to postgres database")
		return NewProjectMetaStore(db), nil
	}
}

func NewProjectMetaStore(db *sql.DB) application.ProjectMetaStore {
	if db == nil {
		return application.NewNoopProjectMetaStore()
	}
	return &projectMetaStore{db: db}
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
