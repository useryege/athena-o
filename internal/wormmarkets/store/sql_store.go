package store

import (
	"context"
	"embed"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	log "github.com/sirupsen/logrus"
	wormmarketssqlc "github.com/useryege/athena/internal/wormmarkets/store/sqlc"
	"github.com/useryege/athena/util/db/postgres"
)

//go:embed migrations/*.sql
var migrations embed.FS

func Migrations() embed.FS {
	return migrations
}

type SQLStore struct {
	pool    *pgxpool.Pool
	queries wormmarketssqlc.Querier
}

func NewSQLStore(db any) *SQLStore {
	switch value := db.(type) {
	case *pgxpool.Pool:
		if value == nil {
			return &SQLStore{}
		}
		return &SQLStore{pool: value, queries: wormmarketssqlc.New(value)}
	case nil:
		return &SQLStore{}
	default:
		panic(fmt.Sprintf("unsupported Worm Markets postgres store db %T", db))
	}
}

func NewSQLStoreWithQuerier(querier wormmarketssqlc.Querier) *SQLStore {
	return &SQLStore{queries: querier}
}

func NewSQLStoreSource() func(context.Context) (*SQLStore, error) {
	return func(ctx context.Context) (*SQLStore, error) {
		pool, err := postgres.ConnectAndMigrate(ctx, postgres.Options{
			Module:       "worm-markets",
			DSNEnv:       "ATHENA_WORM_MARKETS_POSTGRES_DSN",
			Database:     "worm_markets",
			Migrations:   migrations,
			MigrationDir: "migrations",
		})
		if err != nil {
			return nil, fmt.Errorf("connect Worm Markets postgres: %w", err)
		}
		log.Info("Worm Markets postgres migrations are up to date")
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

func (s *SQLStore) querier() (wormmarketssqlc.Querier, error) {
	if s == nil || s.queries == nil {
		return nil, fmt.Errorf("Worm Markets postgres database is not configured")
	}
	return s.queries, nil
}
