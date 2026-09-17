package store

import (
	"context"
	"embed"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	log "github.com/sirupsen/logrus"
	profitsharingsqlc "github.com/useryege/athena/internal/profitsharing/store/sqlc"
)

//go:embed migrations/*.sql
var migrations embed.FS

func Migrations() embed.FS {
	return migrations
}

type SQLStore struct {
	pool    *pgxpool.Pool
	queries profitsharingsqlc.Querier
}

func NewSQLStore(pool *pgxpool.Pool) *SQLStore {
	if pool == nil {
		return &SQLStore{}
	}
	return &SQLStore{pool: pool, queries: profitsharingsqlc.New(pool)}
}

func NewSQLStoreWithQuerier(querier profitsharingsqlc.Querier) *SQLStore {
	return &SQLStore{queries: querier}
}

func NewSQLStoreSource() func(context.Context) (*SQLStore, error) {
	return func(ctx context.Context) (*SQLStore, error) {
		pool, err := Schema().ConnectVerified(ctx)
		if err != nil {
			return nil, fmt.Errorf("connect profit sharing postgres: %w", err)
		}
		log.Info("profit sharing postgres schema verified")
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

func (s *SQLStore) requireQueries() (profitsharingsqlc.Querier, error) {
	if s == nil || s.queries == nil {
		return nil, ErrNotConfigured
	}
	return s.queries, nil
}

func (s *SQLStore) requirePool() (*pgxpool.Pool, error) {
	if s == nil || s.pool == nil {
		return nil, ErrNotConfigured
	}
	return s.pool, nil
}
