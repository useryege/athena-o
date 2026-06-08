package store

import (
	"context"
	"embed"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	log "github.com/sirupsen/logrus"
	wormsqlc "github.com/useryege/athena/internal/worm/store/sqlc"
	"github.com/useryege/athena/util/db/postgres"
)

//go:embed migrations/*.sql
var migrations embed.FS

func Migrations() embed.FS {
	return migrations
}

type SQLStore struct {
	pool    *pgxpool.Pool
	queries wormsqlc.Querier
}

func NewSQLStore(db any) *SQLStore {
	switch value := db.(type) {
	case *pgxpool.Pool:
		if value == nil {
			return &SQLStore{}
		}
		return &SQLStore{pool: value, queries: wormsqlc.New(value)}
	case nil:
		return &SQLStore{}
	default:
		panic(fmt.Sprintf("unsupported worm postgres store db %T", db))
	}
}

func NewSQLStoreWithQuerier(querier wormsqlc.Querier) *SQLStore {
	return &SQLStore{queries: querier}
}

func NewSQLStoreSource() func(context.Context) (*SQLStore, error) {
	return func(ctx context.Context) (*SQLStore, error) {
		pool, err := postgres.ConnectAndMigrate(ctx, postgres.Options{
			Module:       "worm",
			DSNEnv:       "ATHENA_WORM_POSTGRES_DSN",
			Database:     "worm",
			Migrations:   migrations,
			MigrationDir: "migrations",
		})
		if err != nil {
			return nil, fmt.Errorf("connect worm postgres: %w", err)
		}
		log.Info("worm postgres migrations are up to date")
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

func (s *SQLStore) querier() (wormsqlc.Querier, error) {
	if s == nil || s.queries == nil {
		return nil, fmt.Errorf("worm postgres database is not configured")
	}
	return s.queries, nil
}
