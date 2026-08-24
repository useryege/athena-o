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
	"github.com/useryege/athena/internal/accountcenter"
	accountstatesqlc "github.com/useryege/athena/internal/accountstate/store/sqlc"
	"github.com/useryege/athena/util/db/postgres"
)

//go:embed migrations/*.sql
var migrations embed.FS

func Migrations() embed.FS {
	return migrations
}

// SQLStore is the shared durable adapter for account access, profiles, and preferences.
type SQLStore struct {
	pool    *pgxpool.Pool
	queries accountstatesqlc.Querier
}

func NewSQLStore(pool *pgxpool.Pool) *SQLStore {
	if pool == nil {
		return &SQLStore{}
	}
	return &SQLStore{pool: pool, queries: accountstatesqlc.New(pool)}
}

func NewSQLStoreSource() func(context.Context) (*SQLStore, error) {
	return func(ctx context.Context) (*SQLStore, error) {
		pool, err := postgres.ConnectAndMigrate(ctx, postgres.Options{
			Module:       "account-state",
			DSNEnv:       "ATHENA_SERVER_POSTGRES_DSN",
			Database:     "athena",
			Migrations:   migrations,
			MigrationDir: "migrations",
		})
		if err != nil {
			return nil, fmt.Errorf("connect account-state postgres: %w", err)
		}
		log.Info("account-state postgres migrations are up to date")
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

func (s *SQLStore) requireDatabase() error {
	if s == nil || s.pool == nil || s.queries == nil {
		return fmt.Errorf("account-state postgres database is not configured")
	}
	return nil
}

func (s *SQLStore) ListAccountAccessOverrides(ctx context.Context) (map[string]accountaccess.Access, error) {
	if err := s.requireDatabase(); err != nil {
		return nil, err
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
	txQueries := accountstatesqlc.New(tx)

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
	if err := s.requireDatabase(); err != nil {
		return accountaccess.Access{}, err
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
	txQueries := accountstatesqlc.New(tx)

	var persistedName string
	var persistedLoginEnabled bool
	var persistedRevision int64
	if expectedRevision == 0 {
		created, err := txQueries.CreateAccountAccessOverrideHead(ctx, accountstatesqlc.CreateAccountAccessOverrideHeadParams{
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
		updated, err := txQueries.UpdateAccountAccessOverrideHead(ctx, accountstatesqlc.UpdateAccountAccessOverrideHeadParams{
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
		return accountaccess.Access{}, fmt.Errorf("account %q access update returned revision %d after expected revision %d", name, persistedRevision, expectedRevision)
	}

	if err := txQueries.UpsertAccountModuleAccessOverrides(ctx, accountstatesqlc.UpsertAccountModuleAccessOverridesParams{
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

func (s *SQLStore) GetProfile(ctx context.Context, name string) (accountcenter.Profile, bool, error) {
	if err := s.requireDatabase(); err != nil {
		return accountcenter.Profile{}, false, err
	}
	row, err := s.queries.GetAccountProfile(ctx, name)
	if errors.Is(err, pgx.ErrNoRows) {
		return accountcenter.Profile{}, false, nil
	}
	if err != nil {
		return accountcenter.Profile{}, false, fmt.Errorf("get account %q profile: %w", name, err)
	}
	profile, err := profileFromRow(row.DisplayName, row.AccountTier, row.AvatarObjectKey, row.AvatarContentType, row.AvatarEtag, row.AvatarSizeBytes, row.Revision)
	return profile, true, err
}

func (s *SQLStore) ListAvatarObjectKeys(ctx context.Context) ([]string, error) {
	if err := s.requireDatabase(); err != nil {
		return nil, err
	}
	keys, err := s.queries.ListAvatarObjectKeys(ctx)
	if err != nil {
		return nil, fmt.Errorf("list referenced avatar object keys: %w", err)
	}
	return keys, nil
}

func (s *SQLStore) UpdateProfile(ctx context.Context, name string, next accountcenter.Profile, expectedRevision uint64) (accountcenter.Profile, error) {
	if err := s.requireDatabase(); err != nil {
		return accountcenter.Profile{}, err
	}
	if expectedRevision >= math.MaxInt64 || next.Revision != expectedRevision {
		return accountcenter.Profile{}, fmt.Errorf("account %q profile revision does not match expected revision", name)
	}
	params := accountstatesqlc.CreateAccountProfileParams{
		AccountName:       name,
		DisplayName:       next.DisplayName,
		AccountTier:       string(next.Tier),
		AvatarObjectKey:   next.Avatar.ObjectKey,
		AvatarContentType: next.Avatar.ContentType,
		AvatarEtag:        next.Avatar.ETag,
		AvatarSizeBytes:   next.Avatar.SizeBytes,
	}
	var profile accountcenter.Profile
	var err error
	if expectedRevision == 0 {
		row, queryErr := s.queries.CreateAccountProfile(ctx, params)
		if errors.Is(queryErr, pgx.ErrNoRows) {
			return accountcenter.Profile{}, accountcenter.ErrProfileRevisionConflict
		}
		if queryErr != nil {
			return accountcenter.Profile{}, fmt.Errorf("create account %q profile: %w", name, queryErr)
		}
		profile, err = profileFromRow(row.DisplayName, row.AccountTier, row.AvatarObjectKey, row.AvatarContentType, row.AvatarEtag, row.AvatarSizeBytes, row.Revision)
	} else {
		row, queryErr := s.queries.UpdateAccountProfile(ctx, accountstatesqlc.UpdateAccountProfileParams{
			DisplayName:       params.DisplayName,
			AccountTier:       params.AccountTier,
			AvatarObjectKey:   params.AvatarObjectKey,
			AvatarContentType: params.AvatarContentType,
			AvatarEtag:        params.AvatarEtag,
			AvatarSizeBytes:   params.AvatarSizeBytes,
			AccountName:       name,
			ExpectedRevision:  int64(expectedRevision),
		})
		if errors.Is(queryErr, pgx.ErrNoRows) {
			return accountcenter.Profile{}, accountcenter.ErrProfileRevisionConflict
		}
		if queryErr != nil {
			return accountcenter.Profile{}, fmt.Errorf("update account %q profile: %w", name, queryErr)
		}
		profile, err = profileFromRow(row.DisplayName, row.AccountTier, row.AvatarObjectKey, row.AvatarContentType, row.AvatarEtag, row.AvatarSizeBytes, row.Revision)
	}
	if err != nil {
		return accountcenter.Profile{}, fmt.Errorf("project account %q profile: %w", name, err)
	}
	if profile.Revision != expectedRevision+1 {
		return accountcenter.Profile{}, fmt.Errorf("account %q profile update returned revision %d after expected revision %d", name, profile.Revision, expectedRevision)
	}
	return profile, nil
}

func profileFromRow(displayName, tier, objectKey, contentType, etag string, sizeBytes, revision int64) (accountcenter.Profile, error) {
	if revision <= 0 {
		return accountcenter.Profile{}, fmt.Errorf("persisted profile revision is non-positive")
	}
	return accountcenter.Profile{
		DisplayName: displayName,
		Tier:        accountcenter.Tier(tier),
		Avatar: accountcenter.AvatarMetadata{
			ObjectKey: objectKey, ContentType: contentType, ETag: etag, SizeBytes: sizeBytes,
		},
		Revision: uint64(revision),
	}, nil
}

func (s *SQLStore) GetPreferences(ctx context.Context, name string) (accountcenter.Preferences, bool, error) {
	if err := s.requireDatabase(); err != nil {
		return accountcenter.Preferences{}, false, err
	}
	row, err := s.queries.GetAccountPreferences(ctx, name)
	if errors.Is(err, pgx.ErrNoRows) {
		return accountcenter.Preferences{}, false, nil
	}
	if err != nil {
		return accountcenter.Preferences{}, false, fmt.Errorf("get account %q preferences: %w", name, err)
	}
	if row.Revision <= 0 {
		return accountcenter.Preferences{}, false, fmt.Errorf("account %q has non-positive preferences revision", name)
	}
	return accountcenter.Preferences{Theme: accountcenter.ThemeMode(row.Theme), Revision: uint64(row.Revision)}, true, nil
}

func (s *SQLStore) UpdatePreferences(ctx context.Context, name string, next accountcenter.Preferences, expectedRevision uint64) (accountcenter.Preferences, error) {
	if err := s.requireDatabase(); err != nil {
		return accountcenter.Preferences{}, err
	}
	if expectedRevision >= math.MaxInt64 || next.Revision != expectedRevision {
		return accountcenter.Preferences{}, fmt.Errorf("account %q preferences revision does not match expected revision", name)
	}
	var theme string
	var revision int64
	if expectedRevision == 0 {
		row, err := s.queries.CreateAccountPreferences(ctx, accountstatesqlc.CreateAccountPreferencesParams{
			AccountName: name,
			Theme:       string(next.Theme),
		})
		if errors.Is(err, pgx.ErrNoRows) {
			return accountcenter.Preferences{}, accountcenter.ErrPreferencesRevisionConflict
		}
		if err != nil {
			return accountcenter.Preferences{}, fmt.Errorf("create account %q preferences: %w", name, err)
		}
		theme, revision = row.Theme, row.Revision
	} else {
		row, err := s.queries.UpdateAccountPreferences(ctx, accountstatesqlc.UpdateAccountPreferencesParams{
			Theme:            string(next.Theme),
			AccountName:      name,
			ExpectedRevision: int64(expectedRevision),
		})
		if errors.Is(err, pgx.ErrNoRows) {
			return accountcenter.Preferences{}, accountcenter.ErrPreferencesRevisionConflict
		}
		if err != nil {
			return accountcenter.Preferences{}, fmt.Errorf("update account %q preferences: %w", name, err)
		}
		theme, revision = row.Theme, row.Revision
	}
	if revision <= 0 || uint64(revision) != expectedRevision+1 {
		return accountcenter.Preferences{}, fmt.Errorf("account %q preferences update returned revision %d after expected revision %d", name, revision, expectedRevision)
	}
	return accountcenter.Preferences{Theme: accountcenter.ThemeMode(theme), Revision: uint64(revision)}, nil
}
