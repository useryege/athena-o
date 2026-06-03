package store

import (
	"context"
	"embed"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	log "github.com/sirupsen/logrus"
	"github.com/useryege/athena/internal/polymarket/store/sqlc"
	"github.com/useryege/athena/util/db/postgres"
)

//go:embed migrations/*.sql
var migrations embed.FS

func Migrations() embed.FS {
	return migrations
}

type SQLStore struct {
	pool    *pgxpool.Pool
	queries *sqlc.Queries
}

func NewSQLStore(pool *pgxpool.Pool) *SQLStore {
	var queries *sqlc.Queries
	if pool != nil {
		queries = sqlc.New(pool)
	}
	return &SQLStore{pool: pool, queries: queries}
}

func NewSQLStoreSource() func(context.Context) (*SQLStore, error) {
	return func(ctx context.Context) (*SQLStore, error) {
		pool, err := postgres.ConnectAndMigrate(ctx, postgres.Options{
			Module:       "polymarket",
			DSNEnv:       "ATHENA_POLYMARKET_POSTGRES_DSN",
			Database:     "polymarket",
			Migrations:   migrations,
			MigrationDir: "migrations",
		})
		if err != nil {
			return nil, fmt.Errorf("connect polymarket postgres: %w", err)
		}
		log.Info("polymarket postgres migrations are up to date")
		return NewSQLStore(pool), nil
	}
}

func (s *SQLStore) Close() error {
	if s.pool == nil {
		return nil
	}
	s.pool.Close()
	return nil
}
