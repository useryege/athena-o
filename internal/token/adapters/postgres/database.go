package postgres

import (
	"context"
	"embed"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	log "github.com/sirupsen/logrus"
	tokensqlc "github.com/useryege/athena/internal/token/adapters/postgres/sqlc"
	"github.com/useryege/athena/util/db/postgres"
)

//go:embed migrations/*.sql
var migrations embed.FS

func Migrations() embed.FS {
	return migrations
}

type Database struct {
	pool    *pgxpool.Pool
	queries tokensqlc.Querier
}

func NewDatabase(pool *pgxpool.Pool) *Database {
	if pool == nil {
		return &Database{}
	}
	return &Database{pool: pool, queries: tokensqlc.New(pool)}
}

func NewDatabaseSource() func(context.Context) (*Database, error) {
	return func(ctx context.Context) (*Database, error) {
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
		return NewDatabase(pool), nil
	}
}

func (s *Database) Close() error {
	if s.pool == nil {
		return nil
	}
	s.pool.Close()
	return nil
}

func (s *Database) querier() (tokensqlc.Querier, error) {
	if s == nil || s.queries == nil {
		return nil, fmt.Errorf("token postgres database is not configured")
	}
	return s.queries, nil
}
