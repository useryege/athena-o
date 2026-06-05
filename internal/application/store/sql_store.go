package store

import (
	"context"
	"embed"
	"fmt"
	"math"

	"github.com/jackc/pgx/v5/pgxpool"
	log "github.com/sirupsen/logrus"
	appsqlc "github.com/useryege/athena/internal/application/store/sqlc"
	"github.com/useryege/athena/util/db/postgres"
	"github.com/useryege/athena/util/env"
)

//go:embed migrations/*.sql
var migrations embed.FS

const DefaultChainID int64 = 56

func Migrations() embed.FS {
	return migrations
}

type SQLStore struct {
	pool           *pgxpool.Pool
	queries        appsqlc.Querier
	defaultChainID int64
}

func NewSQLStore(db any) *SQLStore {
	switch value := db.(type) {
	case *pgxpool.Pool:
		if value == nil {
			return &SQLStore{}
		}
		return &SQLStore{pool: value, queries: appsqlc.New(value), defaultChainID: DefaultChainID}
	case nil:
		return &SQLStore{defaultChainID: DefaultChainID}
	default:
		panic(fmt.Sprintf("unsupported application postgres store db %T", db))
	}
}

func NewSQLStoreWithQuerier(querier appsqlc.Querier) *SQLStore {
	return &SQLStore{queries: querier, defaultChainID: DefaultChainID}
}

func (s *SQLStore) WithDefaultChainID(chainID int64) *SQLStore {
	if s == nil {
		return nil
	}
	if chainID > 0 {
		s.defaultChainID = chainID
	}
	return s
}

func NewSQLStoreSource() func(context.Context) (*SQLStore, error) {
	return func(ctx context.Context) (*SQLStore, error) {
		pool, err := postgres.ConnectAndMigrate(ctx, postgres.Options{
			Module:       "application",
			DSNEnv:       "ATHENA_APPLICATION_POSTGRES_DSN",
			Database:     "application",
			Migrations:   migrations,
			MigrationDir: "migrations",
		})
		if err != nil {
			return nil, fmt.Errorf("connect application postgres: %w", err)
		}
		log.Info("application postgres migrations are up to date")
		chainID := env.ParseInt64FromEnv("ATHENA_APPLICATION_CHAIN_ID", DefaultChainID, 1, math.MaxInt64)
		return NewSQLStore(pool).WithDefaultChainID(chainID), nil
	}
}

func (s *SQLStore) Close() error {
	if s.pool == nil {
		return nil
	}
	s.pool.Close()
	return nil
}

func (s *SQLStore) querier() (appsqlc.Querier, error) {
	if s.queries == nil {
		return nil, fmt.Errorf("application postgres database is not configured")
	}
	return s.queries, nil
}

func (s *SQLStore) chainIDForProject(chainID int64) int64 {
	if chainID > 0 {
		return chainID
	}
	if s != nil && s.defaultChainID > 0 {
		return s.defaultChainID
	}
	return DefaultChainID
}
