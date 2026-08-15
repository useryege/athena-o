package store

import (
	"context"
	"embed"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	log "github.com/sirupsen/logrus"
	fifamarketdashboardsqlc "github.com/useryege/athena/internal/fifamarketdashboard/store/sqlc"
	"github.com/useryege/athena/util/db/postgres"
)

//go:embed migrations/*.sql
var migrations embed.FS

func Migrations() embed.FS {
	return migrations
}

type SQLStore struct {
	pool    *pgxpool.Pool
	queries fifamarketdashboardsqlc.Querier
}

func NewSQLStore(db any) *SQLStore {
	switch value := db.(type) {
	case *pgxpool.Pool:
		if value == nil {
			return &SQLStore{}
		}
		return &SQLStore{pool: value, queries: fifamarketdashboardsqlc.New(value)}
	case nil:
		return &SQLStore{}
	default:
		panic(fmt.Sprintf("unsupported FIFA Market Dashboard postgres store db %T", db))
	}
}

func NewSQLStoreWithQuerier(querier fifamarketdashboardsqlc.Querier) *SQLStore {
	return &SQLStore{queries: querier}
}

func NewSQLStoreSource() func(context.Context) (*SQLStore, error) {
	return func(ctx context.Context) (*SQLStore, error) {
		pool, err := postgres.ConnectAndMigrate(ctx, postgres.Options{
			Module:       "fifa-market-dashboard",
			DSNEnv:       "ATHENA_FIFA_MARKET_DASHBOARD_POSTGRES_DSN",
			Database:     "fifa_market_dashboard",
			Migrations:   migrations,
			MigrationDir: "migrations",
		})
		if err != nil {
			return nil, fmt.Errorf("connect FIFA Market Dashboard postgres: %w", err)
		}
		log.Info("FIFA Market Dashboard postgres migrations are up to date")
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

func (s *SQLStore) querier() (fifamarketdashboardsqlc.Querier, error) {
	if s == nil || s.queries == nil {
		return nil, fmt.Errorf("FIFA Market Dashboard postgres database is not configured")
	}
	return s.queries, nil
}
