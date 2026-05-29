package store

import (
	"context"
	"database/sql"
	"embed"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	log "github.com/sirupsen/logrus"
	appsqlc "github.com/useryege/athena/internal/application/store/sqlc"
	"github.com/useryege/athena/internal/postgres"
)

//go:embed migrations/*.sql
var migrations embed.FS

type SQLStore struct {
	pool    *pgxpool.Pool
	db      postgres.Executor
	queries appsqlc.Querier
}

func NewSQLStore(db any) *SQLStore {
	var queries appsqlc.Querier
	switch value := db.(type) {
	case *pgxpool.Pool:
		if value != nil {
			queries = appsqlc.New(value)
		}
		return &SQLStore{pool: value, db: postgres.NewDB(value), queries: queries}
	case *sql.DB:
		return &SQLStore{db: postgres.NewSQLDB(value)}
	case nil:
		return &SQLStore{}
	default:
		panic(fmt.Sprintf("unsupported application postgres store db %T", db))
	}
}

func NewSQLStoreWithQuerier(querier appsqlc.Querier) *SQLStore {
	return &SQLStore{queries: querier}
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
