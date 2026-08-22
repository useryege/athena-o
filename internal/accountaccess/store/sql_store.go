package store

import (
	"context"
	"embed"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	log "github.com/sirupsen/logrus"
	accountaccesssqlc "github.com/useryege/athena/internal/accountaccess/store/sqlc"
	"github.com/useryege/athena/util/db/postgres"
)

//go:embed migrations/*.sql
var migrations embed.FS

func Migrations() embed.FS {
	return migrations
}

type SQLStore struct {
	pool    *pgxpool.Pool
	queries accountaccesssqlc.Querier
}

func NewSQLStore(pool *pgxpool.Pool) *SQLStore {
	if pool == nil {
		return &SQLStore{}
	}
	return &SQLStore{pool: pool, queries: accountaccesssqlc.New(pool)}
}

func NewSQLStoreSource() func(context.Context) (*SQLStore, error) {
	return func(ctx context.Context) (*SQLStore, error) {
		pool, err := postgres.ConnectAndMigrate(ctx, postgres.Options{
			Module:       "account-access",
			DSNEnv:       "ATHENA_SERVER_POSTGRES_DSN",
			Database:     "athena",
			Migrations:   migrations,
			MigrationDir: "migrations",
		})
		if err != nil {
			return nil, fmt.Errorf("connect account-access postgres: %w", err)
		}
		log.Info("account-access postgres migrations are up to date")
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

func (s *SQLStore) ListAccountEnabledOverrides(ctx context.Context) (map[string]bool, error) {
	if s == nil || s.queries == nil {
		return nil, fmt.Errorf("account-access postgres database is not configured")
	}

	rows, err := s.queries.ListAccountEnabledOverrides(ctx)
	if err != nil {
		return nil, fmt.Errorf("list account enabled overrides: %w", err)
	}
	overrides := make(map[string]bool, len(rows))
	for _, row := range rows {
		overrides[row.AccountName] = row.Enabled
	}
	return overrides, nil
}

func (s *SQLStore) SetAccountEnabled(ctx context.Context, name string, enabled bool) error {
	if s == nil || s.queries == nil {
		return fmt.Errorf("account-access postgres database is not configured")
	}

	if err := s.queries.UpsertAccountEnabledOverride(ctx, accountaccesssqlc.UpsertAccountEnabledOverrideParams{
		AccountName: name,
		Enabled:     enabled,
	}); err != nil {
		return fmt.Errorf("set account %q enabled override: %w", name, err)
	}
	return nil
}
