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
	if s == nil || s.pool == nil || s.queries == nil {
		return nil, fmt.Errorf("account-access postgres database is not configured")
	}

	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{
		IsoLevel:   pgx.RepeatableRead,
		AccessMode: pgx.ReadOnly,
	})
	if err != nil {
		return nil, fmt.Errorf("begin account-access snapshot transaction: %w", err)
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback(context.Background())
		}
	}()
	txQueries := accountaccesssqlc.New(tx)

	heads, err := txQueries.ListAccountAccessOverrideHeads(ctx)
	if err != nil {
		return nil, fmt.Errorf("list account access override heads: %w", err)
	}
	moduleRows, err := txQueries.ListAccountModuleAccessOverrides(ctx)
	if err != nil {
		return nil, fmt.Errorf("list account module access overrides: %w", err)
	}

	overrides := make(map[string]accountaccess.Access, len(heads))
	for _, head := range heads {
		if head.Revision <= 0 {
			return nil, fmt.Errorf("account %q has non-positive access revision %d", head.AccountName, head.Revision)
		}
		if _, duplicate := overrides[head.AccountName]; duplicate {
			return nil, fmt.Errorf("account %q has duplicate access override heads", head.AccountName)
		}
		overrides[head.AccountName] = accountaccess.Access{
			LoginEnabled: head.LoginEnabled,
			Modules:      make(map[accountaccess.Module]accountaccess.AccessLevel, len(accountaccess.AllModules())),
			Revision:     uint64(head.Revision),
		}
	}

	for _, row := range moduleRows {
		access, exists := overrides[row.AccountName]
		if !exists {
			return nil, fmt.Errorf("account module access for %q has no override head", row.AccountName)
		}
		module := accountaccess.Module(row.Module)
		if _, known := accountaccess.MaxAccessLevel(module); !known {
			return nil, fmt.Errorf("account %q has unknown access module %q", row.AccountName, row.Module)
		}
		if _, duplicate := access.Modules[module]; duplicate {
			return nil, fmt.Errorf("account %q has duplicate access rows for module %q", row.AccountName, row.Module)
		}
		access.Modules[module] = accountaccess.AccessLevel(row.AccessLevel)
		overrides[row.AccountName] = access
	}

	for name, access := range overrides {
		if err := access.Validate(); err != nil {
			return nil, fmt.Errorf("account %q has invalid persisted module access: %w", name, err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit account-access snapshot transaction: %w", err)
	}
	committed = true
	return overrides, nil
}

func (s *SQLStore) UpdateAccountAccessOverride(ctx context.Context, name string, next accountaccess.Access, expectedRevision uint64) (accountaccess.Access, error) {
	if s == nil || s.pool == nil || s.queries == nil {
		return accountaccess.Access{}, fmt.Errorf("account-access postgres database is not configured")
	}
	if expectedRevision > math.MaxInt64 {
		return accountaccess.Access{}, fmt.Errorf("account %q expected revision is out of range", name)
	}
	if next.Revision != expectedRevision {
		return accountaccess.Access{}, fmt.Errorf("account %q access revision does not match expected revision", name)
	}
	next = next.Clone()
	if err := next.Validate(); err != nil {
		return accountaccess.Access{}, fmt.Errorf("validate account %q access override: %w", name, err)
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return accountaccess.Access{}, fmt.Errorf("begin account %q access update: %w", name, err)
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback(context.Background())
		}
	}()
	txQueries := accountaccesssqlc.New(tx)

	var persistedName string
	var persistedLoginEnabled bool
	var persistedRevision int64
	if expectedRevision == 0 {
		created, err := txQueries.CreateAccountAccessOverrideHead(ctx, accountaccesssqlc.CreateAccountAccessOverrideHeadParams{
			AccountName:  name,
			LoginEnabled: next.LoginEnabled,
		})
		if errors.Is(err, pgx.ErrNoRows) {
			return accountaccess.Access{}, accountaccess.ErrRevisionConflict
		}
		if err != nil {
			return accountaccess.Access{}, fmt.Errorf("create account %q access override head: %w", name, err)
		}
		persistedName = created.AccountName
		persistedLoginEnabled = created.LoginEnabled
		persistedRevision = created.Revision
	} else {
		updated, err := txQueries.UpdateAccountAccessOverrideHead(ctx, accountaccesssqlc.UpdateAccountAccessOverrideHeadParams{
			AccountName:      name,
			LoginEnabled:     next.LoginEnabled,
			ExpectedRevision: int64(expectedRevision),
		})
		if errors.Is(err, pgx.ErrNoRows) {
			return accountaccess.Access{}, accountaccess.ErrRevisionConflict
		}
		if err != nil {
			return accountaccess.Access{}, fmt.Errorf("update account %q access override head: %w", name, err)
		}
		persistedName = updated.AccountName
		persistedLoginEnabled = updated.LoginEnabled
		persistedRevision = updated.Revision
	}

	if persistedName != name || persistedLoginEnabled != next.LoginEnabled {
		return accountaccess.Access{}, fmt.Errorf("account %q access update returned inconsistent head", name)
	}
	if persistedRevision <= 0 || uint64(persistedRevision) != expectedRevision+1 {
		return accountaccess.Access{}, fmt.Errorf(
			"account %q access update returned revision %d after expected revision %d",
			name,
			persistedRevision,
			expectedRevision,
		)
	}

	if err := txQueries.UpsertAccountModuleAccessOverrides(ctx, accountaccesssqlc.UpsertAccountModuleAccessOverridesParams{
		AccountName:                    name,
		MarketRadarAccessLevel:         string(next.Modules[accountaccess.ModuleMarketRadar]),
		SportsLiveAccessLevel:          string(next.Modules[accountaccess.ModuleSportsLive]),
		SportsHistoryAccessLevel:       string(next.Modules[accountaccess.ModuleSportsHistory]),
		ManagedOoAccessLevel:           string(next.Modules[accountaccess.ModuleManagedOO]),
		WormMarketsAccessLevel:         string(next.Modules[accountaccess.ModuleWormMarkets]),
		FifaMarketDashboardAccessLevel: string(next.Modules[accountaccess.ModuleFIFAMarketDashboard]),
		WorldCupCornersAccessLevel:     string(next.Modules[accountaccess.ModuleWorldCupCorners]),
		TokenAccessLevel:               string(next.Modules[accountaccess.ModuleToken]),
		WalletAccessLevel:              string(next.Modules[accountaccess.ModuleWallet]),
		NotificationsAccessLevel:       string(next.Modules[accountaccess.ModuleNotifications]),
	}); err != nil {
		return accountaccess.Access{}, fmt.Errorf("replace account %q module access overrides: %w", name, err)
	}

	if err := tx.Commit(ctx); err != nil {
		return accountaccess.Access{}, fmt.Errorf("commit account %q access update: %w", name, err)
	}
	committed = true
	persisted := accountaccess.Access{
		LoginEnabled: persistedLoginEnabled,
		Modules:      next.Modules,
		Revision:     uint64(persistedRevision),
	}
	return persisted.Clone(), nil
}
