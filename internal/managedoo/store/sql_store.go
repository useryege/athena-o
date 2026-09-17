package store

import (
	"context"
	"embed"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	log "github.com/sirupsen/logrus"
	"github.com/useryege/athena/internal/managedoo/store/sqlc"
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
		pool, err := Schema().ConnectVerified(ctx)
		if err != nil {
			return nil, fmt.Errorf("connect managed oo postgres: %w", err)
		}
		log.Info("managed oo postgres schema verified")
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
