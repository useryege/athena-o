package store

import (
	"context"
	"embed"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	log "github.com/sirupsen/logrus"
	"github.com/useryege/athena/util/db/postgres"
)

//go:embed migrations/*.sql
var migrations embed.FS

func Migrations() embed.FS {
	return migrations
}

type SQLStore struct {
	pool *pgxpool.Pool
}

func NewSQLStore(pool *pgxpool.Pool) *SQLStore {
	return &SQLStore{pool: pool}
}

func NewSQLStoreSource() func(context.Context) (*SQLStore, error) {
	return func(ctx context.Context) (*SQLStore, error) {
		pool, err := postgres.ConnectAndMigrate(ctx, postgres.Options{
			Module:       "ethereumapi",
			DSNEnv:       "ATHENA_ETHEREUM_API_POSTGRES_DSN",
			Database:     "ethereumapi",
			Migrations:   migrations,
			MigrationDir: "migrations",
		})
		if err != nil {
			return nil, fmt.Errorf("connect ethereumapi postgres: %w", err)
		}
		log.Info("ethereumapi postgres migrations are up to date")
		return NewSQLStore(pool), nil
	}
}

func (s *SQLStore) Close() error {
	if s == nil || s.pool == nil {
		return nil
	}
	s.pool.Close()
	return nil
}
