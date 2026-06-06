package store

import (
	"context"
	"embed"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	log "github.com/sirupsen/logrus"
	tokensqlc "github.com/useryege/athena/internal/token/store/sqlc"
	"github.com/useryege/athena/util/db/postgres"
)

//go:embed migrations/*.sql
var migrations embed.FS

func Migrations() embed.FS {
	return migrations
}

type SQLStore struct {
	pool    *pgxpool.Pool
	queries tokensqlc.Querier
}

func NewSQLStore(db any) *SQLStore {
	switch value := db.(type) {
	case *pgxpool.Pool:
		if value == nil {
			return &SQLStore{}
		}
		return &SQLStore{pool: value, queries: tokensqlc.New(value)}
	case nil:
		return &SQLStore{}
	default:
		panic(fmt.Sprintf("unsupported token postgres store db %T", db))
	}
}

func NewSQLStoreWithQuerier(querier tokensqlc.Querier) *SQLStore {
	return &SQLStore{queries: querier}
}

func NewSQLStoreSource() func(context.Context) (*SQLStore, error) {
	return func(ctx context.Context) (*SQLStore, error) {
		pool, err := postgres.ConnectAndMigrate(ctx, postgres.Options{
			Module:       "token",
			DSNEnv:       "ATHENA_TOKEN_POSTGRES_DSN",
			Database:     "token",
			Migrations:   migrations,
			MigrationDir: "migrations",
		})
		if err != nil {
			return nil, fmt.Errorf("connect token postgres: %w", err)
		}
		log.Info("token postgres migrations are up to date")
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

func (s *SQLStore) querier() (tokensqlc.Querier, error) {
	if s == nil || s.queries == nil {
		return nil, fmt.Errorf("token postgres database is not configured")
	}
	return s.queries, nil
}
