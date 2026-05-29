package store

import (
	"context"
	"database/sql"
	"embed"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	log "github.com/sirupsen/logrus"
	"github.com/useryege/athena/internal/postgres"
	"github.com/useryege/athena/internal/solidity/store/sqlc"
)

//go:embed migrations/*.sql
var migrations embed.FS

type SQLStore struct {
	pool    *pgxpool.Pool
	db      postgres.Executor
	queries *sqlc.Queries
}

func NewSQLStore(db any) *SQLStore {
	var queries *sqlc.Queries
	switch value := db.(type) {
	case *pgxpool.Pool:
		if value != nil {
			queries = sqlc.New(value)
		}
		return &SQLStore{pool: value, db: postgres.NewDB(value), queries: queries}
	case *sql.DB:
		return &SQLStore{db: postgres.NewSQLDB(value)}
	case nil:
		return &SQLStore{}
	default:
		panic(fmt.Sprintf("unsupported solidity postgres store db %T", db))
	}
}

func NewSQLStoreSource() func(context.Context) (*SQLStore, error) {
	return func(ctx context.Context) (*SQLStore, error) {
		pool, err := postgres.ConnectAndMigrate(ctx, postgres.Options{
			Module:       "solidity",
			DSNEnv:       "ATHENA_SOLIDITY_POSTGRES_DSN",
			Database:     "solidity",
			Migrations:   migrations,
			MigrationDir: "migrations",
		})
		if err != nil {
			return nil, fmt.Errorf("connect solidity postgres: %w", err)
		}
		log.Info("solidity postgres migrations are up to date")
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
