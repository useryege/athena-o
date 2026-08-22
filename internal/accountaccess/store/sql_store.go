package store

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"math"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	log "github.com/sirupsen/logrus"
	"github.com/useryege/athena/internal/accountaccess"
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

func (s *SQLStore) ListAccountAccessOverrides(ctx context.Context) (map[string]accountaccess.Access, error) {
	if s == nil || s.queries == nil {
		return nil, fmt.Errorf("account-access postgres database is not configured")
	}

	rows, err := s.queries.ListAccountAccessOverrides(ctx)
	if err != nil {
		return nil, fmt.Errorf("list account access overrides: %w", err)
	}
	overrides := make(map[string]accountaccess.Access, len(rows))
	for _, row := range rows {
		if row.Revision < 0 {
			return nil, fmt.Errorf("account %q has negative access revision", row.AccountName)
		}
		overrides[row.AccountName] = accountaccess.Access{
			LoginEnabled: row.LoginEnabled,
			DataAccess:   accountaccess.DataAccess(row.DataAccess),
			Revision:     uint64(row.Revision),
		}
	}
	return overrides, nil
}

func (s *SQLStore) UpdateAccountAccessOverride(ctx context.Context, name string, next accountaccess.Access, expectedRevision uint64) (accountaccess.Access, error) {
	if s == nil || s.queries == nil {
		return accountaccess.Access{}, fmt.Errorf("account-access postgres database is not configured")
	}
	if expectedRevision > math.MaxInt64 {
		return accountaccess.Access{}, fmt.Errorf("account %q expected revision is out of range", name)
	}

	updated, err := s.queries.UpdateAccountAccessOverride(ctx, accountaccesssqlc.UpdateAccountAccessOverrideParams{
		AccountName:      name,
		LoginEnabled:     next.LoginEnabled,
		DataAccess:       string(next.DataAccess),
		ExpectedRevision: int64(expectedRevision),
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return accountaccess.Access{}, accountaccess.ErrRevisionConflict
	}
	if err != nil {
		return accountaccess.Access{}, fmt.Errorf("update account %q access override: %w", name, err)
	}
	if updated.Revision < 0 {
		return accountaccess.Access{}, fmt.Errorf("account %q has negative access revision", name)
	}
	return accountaccess.Access{
		LoginEnabled: updated.LoginEnabled,
		DataAccess:   accountaccess.DataAccess(updated.DataAccess),
		Revision:     uint64(updated.Revision),
	}, nil
}
