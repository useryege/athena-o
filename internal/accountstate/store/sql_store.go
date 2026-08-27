package store

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	log "github.com/sirupsen/logrus"
	"github.com/useryege/athena/internal/accountaccess"
	"github.com/useryege/athena/internal/accountcenter"
	"github.com/useryege/athena/internal/accountcredentials"
	accountstatesqlc "github.com/useryege/athena/internal/accountstate/store/sqlc"
	"github.com/useryege/athena/util/db/postgres"
)

//go:embed migrations/*.sql
var migrations embed.FS

func Migrations() embed.FS {
	return migrations
}

// SQLStore is the shared durable adapter for account identity, access, API
// Keys, profiles, and preferences.
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

func accountIDParam(value string) (pgtype.UUID, string, error) {
	canonical, err := accountcredentials.CanonicalAccountID(value)
	if err != nil {
		return pgtype.UUID{}, "", fmt.Errorf("account ID %q is not a UUID: %w", value, err)
	}
	parsed, err := uuid.Parse(canonical)
	if err != nil || parsed == uuid.Nil {
		return pgtype.UUID{}, "", fmt.Errorf("account ID %q is not a non-zero UUID", value)
	}
	return pgtype.UUID{Bytes: [16]byte(parsed), Valid: true}, canonical, nil
}

func accountIDFromPG(value pgtype.UUID) (string, error) {
	if !value.Valid {
		return "", fmt.Errorf("account ID is missing")
	}
	parsed := uuid.UUID(value.Bytes)
	if parsed == uuid.Nil {
		return "", fmt.Errorf("account ID is the zero UUID")
	}
	return parsed.String(), nil
}

// ListAccountAccess returns one complete, consistent access aggregate for
// every durable account.
func (s *SQLStore) ListAccountAccess(ctx context.Context) (map[string]accountaccess.Access, error) {
	if err := s.requireDatabase(); err != nil {
		return nil, err
	}
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
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

	accountRows, err := txQueries.ListAccountRecords(ctx)
	if err != nil {
		return nil, fmt.Errorf("list durable accounts for access snapshot: %w", err)
	}
	heads, err := txQueries.ListAccountAccessHeads(ctx)
	if err != nil {
		return nil, fmt.Errorf("list account access heads: %w", err)
	}
	moduleRows, err := txQueries.ListAccountModuleAccess(ctx)
	if err != nil {
		return nil, fmt.Errorf("list account module access: %w", err)
	}

	durableAccounts := make(map[string]struct{}, len(accountRows))
	for _, row := range accountRows {
		accountID, err := accountIDFromPG(row.AccountID)
		if err != nil {
			return nil, fmt.Errorf("project durable account ID: %w", err)
		}
		if _, duplicate := durableAccounts[accountID]; duplicate {
			return nil, fmt.Errorf("durable account %q is duplicated", accountID)
		}
		durableAccounts[accountID] = struct{}{}
	}
	result := make(map[string]accountaccess.Access, len(heads))
	for _, head := range heads {
		accountID, err := accountIDFromPG(head.AccountID)
		if err != nil {
			return nil, fmt.Errorf("project account access head ID: %w", err)
		}
		if _, exists := durableAccounts[accountID]; !exists {
			return nil, fmt.Errorf("account access head references unknown account %q", accountID)
		}
		if _, duplicate := result[accountID]; duplicate {
			return nil, fmt.Errorf("account %q has duplicate access heads", accountID)
		}
		access, err := accessFromHead(accountID, head.Administrator, head.LoginEnabled, head.ApiKeyEnabled, head.ProfitSharingEnabled, head.Revision)
		if err != nil {
			return nil, err
		}
		result[accountID] = access
	}
	for accountID := range durableAccounts {
		if _, exists := result[accountID]; !exists {
			return nil, fmt.Errorf("durable account %q has no access head", accountID)
		}
	}
	if err := attachModuleAccess(result, moduleRows); err != nil {
		return nil, err
	}
	for accountID, access := range result {
		if err := access.Validate(); err != nil {
			return nil, fmt.Errorf("account %q has invalid persisted access: %w", accountID, err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit account-access snapshot transaction: %w", err)
	}
	committed = true
	return result, nil
}

// GetAccountAccess returns one complete access aggregate from a consistent
// head-and-module snapshot.
func (s *SQLStore) GetAccountAccess(ctx context.Context, accountID string) (accountaccess.Access, error) {
	if err := s.requireDatabase(); err != nil {
		return accountaccess.Access{}, err
	}
	accountIDValue, canonicalID, err := accountIDParam(accountID)
	if err != nil {
		return accountaccess.Access{}, err
	}
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if err != nil {
		return accountaccess.Access{}, fmt.Errorf("begin account %q access snapshot: %w", canonicalID, err)
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback(context.Background())
		}
	}()
	txQueries := accountstatesqlc.New(tx)

	head, err := txQueries.GetAccountAccessHead(ctx, accountIDValue)
	if err != nil {
		return accountaccess.Access{}, fmt.Errorf("get account %q access head: %w", canonicalID, err)
	}
	returnedID, err := accountIDFromPG(head.AccountID)
	if err != nil {
		return accountaccess.Access{}, fmt.Errorf("project account %q access head ID: %w", canonicalID, err)
	}
	if returnedID != canonicalID {
		return accountaccess.Access{}, fmt.Errorf("account %q access query returned head for %q", canonicalID, returnedID)
	}
	access, err := accessFromHead(returnedID, head.Administrator, head.LoginEnabled, head.ApiKeyEnabled, head.ProfitSharingEnabled, head.Revision)
	if err != nil {
		return accountaccess.Access{}, err
	}
	moduleRows, err := txQueries.ListAccountModuleAccessByAccount(ctx, accountIDValue)
	if err != nil {
		return accountaccess.Access{}, fmt.Errorf("list account %q module access: %w", canonicalID, err)
	}
	result := map[string]accountaccess.Access{canonicalID: access}
	if err := attachModuleAccess(result, moduleRows); err != nil {
		return accountaccess.Access{}, err
	}
	access = result[canonicalID]
	if err := access.Validate(); err != nil {
		return accountaccess.Access{}, fmt.Errorf("account %q has invalid persisted access: %w", canonicalID, err)
	}
	if err := tx.Commit(ctx); err != nil {
		return accountaccess.Access{}, fmt.Errorf("commit account %q access snapshot: %w", canonicalID, err)
	}
	committed = true
	return access.Clone(), nil
}

// UpdateAccountAccess replaces one ordinary account's full access aggregate.
// The head CAS and all nine module updates commit atomically.
func (s *SQLStore) UpdateAccountAccess(ctx context.Context, accountID string, next accountaccess.Access, expectedRevision uint64) (accountaccess.Access, error) {
	if err := s.requireDatabase(); err != nil {
		return accountaccess.Access{}, err
	}
	accountIDValue, canonicalID, err := accountIDParam(accountID)
	if err != nil {
		return accountaccess.Access{}, err
	}
	if expectedRevision == 0 || expectedRevision >= math.MaxInt64 {
		return accountaccess.Access{}, fmt.Errorf("account %q expected access revision is out of range", canonicalID)
	}
	if next.Revision != expectedRevision {
		return accountaccess.Access{}, fmt.Errorf("account %q access revision does not match expected revision", canonicalID)
	}
	if next.Administrator {
		return accountaccess.Access{}, fmt.Errorf("account %q administrator role cannot be supplied by an access update", canonicalID)
	}
	next = next.Clone()
	if err := next.Validate(); err != nil {
		return accountaccess.Access{}, fmt.Errorf("validate account %q access: %w", canonicalID, err)
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return accountaccess.Access{}, fmt.Errorf("begin account %q access update: %w", canonicalID, err)
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback(context.Background())
		}
	}()
	txQueries := accountstatesqlc.New(tx)
	head, err := txQueries.UpdateAccountAccessHead(ctx, accountstatesqlc.UpdateAccountAccessHeadParams{
		LoginEnabled: next.LoginEnabled, ApiKeyEnabled: next.APIKeyEnabled,
		ProfitSharingEnabled: next.ProfitSharingEnabled, AccountID: accountIDValue,
		ExpectedRevision: int64(expectedRevision),
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return accountaccess.Access{}, accountaccess.ErrRevisionConflict
	}
	if err != nil {
		return accountaccess.Access{}, fmt.Errorf("update account %q access head: %w", canonicalID, err)
	}
	returnedID, err := accountIDFromPG(head.AccountID)
	if err != nil {
		return accountaccess.Access{}, fmt.Errorf("project updated account %q access head ID: %w", canonicalID, err)
	}
	if returnedID != canonicalID || head.LoginEnabled != next.LoginEnabled || head.ApiKeyEnabled != next.APIKeyEnabled || head.ProfitSharingEnabled != next.ProfitSharingEnabled {
		return accountaccess.Access{}, fmt.Errorf("account %q access update returned an inconsistent head", canonicalID)
	}
	if head.Revision <= 0 || uint64(head.Revision) != expectedRevision+1 {
		return accountaccess.Access{}, fmt.Errorf("account %q access update returned revision %d after expected revision %d", canonicalID, head.Revision, expectedRevision)
	}
	rowsAffected, err := txQueries.ReplaceAccountModuleAccess(ctx, accountstatesqlc.ReplaceAccountModuleAccessParams{
		MarketRadarAccessLevel: string(next.Modules[accountaccess.ModuleMarketRadar]), SportsLiveAccessLevel: string(next.Modules[accountaccess.ModuleSportsLive]),
		SportsHistoryAccessLevel: string(next.Modules[accountaccess.ModuleSportsHistory]), ManagedOoAccessLevel: string(next.Modules[accountaccess.ModuleManagedOO]),
		WormMarketsAccessLevel:     string(next.Modules[accountaccess.ModuleWormMarkets]),
		WorldCupCornersAccessLevel: string(next.Modules[accountaccess.ModuleWorldCupCorners]), TokenAccessLevel: string(next.Modules[accountaccess.ModuleToken]),
		WalletAccessLevel: string(next.Modules[accountaccess.ModuleWallet]), NotificationsAccessLevel: string(next.Modules[accountaccess.ModuleNotifications]),
		AccountID: accountIDValue,
	})
	if err != nil {
		return accountaccess.Access{}, fmt.Errorf("replace account %q module access: %w", canonicalID, err)
	}
	expectedModuleRows := int64(len(accountaccess.AllModules()))
	if rowsAffected != expectedModuleRows {
		return accountaccess.Access{}, fmt.Errorf("replace account %q module access affected %d rows, expected %d", canonicalID, rowsAffected, expectedModuleRows)
	}
	if err := tx.Commit(ctx); err != nil {
		return accountaccess.Access{}, fmt.Errorf("commit account %q access update: %w", canonicalID, err)
	}
	committed = true
	return accountaccess.Access{
		Administrator: false,
		LoginEnabled:  head.LoginEnabled, APIKeyEnabled: head.ApiKeyEnabled,
		ProfitSharingEnabled: head.ProfitSharingEnabled, Modules: next.Modules,
		Revision: uint64(head.Revision),
	}.Clone(), nil
}

func accessFromHead(accountID string, administrator, loginEnabled, apiKeyEnabled, profitSharingEnabled bool, revision int64) (accountaccess.Access, error) {
	if canonicalID, err := accountcredentials.CanonicalAccountID(accountID); err != nil || canonicalID != accountID {
		return accountaccess.Access{}, fmt.Errorf("persisted account access ID %q is invalid", accountID)
	}
	if revision <= 0 {
		return accountaccess.Access{}, fmt.Errorf("account %q has non-positive access revision %d", accountID, revision)
	}
	return accountaccess.Access{
		Administrator: administrator,
		LoginEnabled:  loginEnabled, APIKeyEnabled: apiKeyEnabled,
		ProfitSharingEnabled: profitSharingEnabled,
		Modules:              make(map[accountaccess.Module]accountaccess.AccessLevel, len(accountaccess.AllModules())),
		Revision:             uint64(revision),
	}, nil
}

func attachModuleAccess(accessByID map[string]accountaccess.Access, rows []accountstatesqlc.AccountModuleAccess) error {
	for _, row := range rows {
		accountID, err := accountIDFromPG(row.AccountID)
		if err != nil {
			return fmt.Errorf("project account module access ID: %w", err)
		}
		access, exists := accessByID[accountID]
		if !exists {
			return fmt.Errorf("account module access for %q has no access head", accountID)
		}
		module := accountaccess.Module(row.Module)
		if _, known := accountaccess.MaxAccessLevel(module); !known {
			return fmt.Errorf("account %q has unknown access module %q", accountID, row.Module)
		}
		if _, duplicate := access.Modules[module]; duplicate {
			return fmt.Errorf("account %q has duplicate access rows for module %q", accountID, row.Module)
		}
		access.Modules[module] = accountaccess.AccessLevel(row.AccessLevel)
		accessByID[accountID] = access
	}
	return nil
}

// ListCredentialAccounts loads every durable identity together with all API
// Key metadata from one consistent database snapshot.
func (s *SQLStore) ListCredentialAccounts(ctx context.Context) (map[string]accountcredentials.Account, error) {
	if err := s.requireDatabase(); err != nil {
		return nil, err
	}
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if err != nil {
		return nil, fmt.Errorf("begin credential account snapshot transaction: %w", err)
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback(context.Background())
		}
	}()
	txQueries := accountstatesqlc.New(tx)
	rows, err := txQueries.ListAccountRecords(ctx)
	if err != nil {
		return nil, fmt.Errorf("list credential accounts: %w", err)
	}
	accounts := make(map[string]accountcredentials.Account, len(rows))
	seenIdentity := make(map[string]string, len(rows))
	for _, row := range rows {
		account, err := credentialAccountFromAthenaRow(row)
		if err != nil {
			return nil, err
		}
		if _, duplicate := accounts[account.ID]; duplicate {
			return nil, fmt.Errorf("credential account %q is duplicated", account.ID)
		}
		if account.HasExternalIdentity() {
			key := credentialIdentityKey(account.IdentityProvider, account.IdentitySubject)
			if previous := seenIdentity[key]; previous != "" {
				return nil, fmt.Errorf("external identity is assigned to both %q and %q", previous, account.ID)
			}
			seenIdentity[key] = account.ID
		}
		accounts[account.ID] = account
	}
	keyRows, err := txQueries.ListAccountAPIKeyRecords(ctx)
	if err != nil {
		return nil, fmt.Errorf("list account API Key metadata: %w", err)
	}
	seenJTI := make(map[string]string, len(keyRows))
	seenDisplayID := make(map[string]struct{}, len(keyRows))
	for _, row := range keyRows {
		accountID, err := accountIDFromPG(row.AccountID)
		if err != nil {
			return nil, fmt.Errorf("project API Key account ID: %w", err)
		}
		account, exists := accounts[accountID]
		if !exists {
			return nil, fmt.Errorf("API Key metadata references unknown account %q", accountID)
		}
		token, err := credentialTokenFromRow(row.DisplayID, row.Jti, row.IssuedAt, row.ExpiresAt)
		if err != nil {
			return nil, fmt.Errorf("project account %q API Key %q: %w", accountID, row.DisplayID, err)
		}
		if previous := seenJTI[token.JTI]; previous != "" {
			return nil, fmt.Errorf("API Key JTI is duplicated across accounts %q and %q", previous, accountID)
		}
		seenJTI[token.JTI] = accountID
		displayKey := accountID + "\x00" + token.ID
		if _, duplicate := seenDisplayID[displayKey]; duplicate {
			return nil, fmt.Errorf("account %q has duplicate API Key display ID %q", accountID, token.ID)
		}
		seenDisplayID[displayKey] = struct{}{}
		account.Tokens = append(account.Tokens, token)
		accounts[accountID] = account
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit credential account snapshot transaction: %w", err)
	}
	committed = true
	return accounts, nil
}

// GetCredentialAccountByIdentity resolves only an already-registered external
// provider identity. Unknown identities remain outside the durable account
// model until their provider's username registration flow commits.
func (s *SQLStore) GetCredentialAccountByIdentity(ctx context.Context, provider accountcredentials.IdentityProvider, subject string) (accountcredentials.Account, bool, error) {
	if err := s.requireDatabase(); err != nil {
		return accountcredentials.Account{}, false, err
	}
	subject, err := accountcredentials.NormalizeIdentitySubject(provider, subject)
	if err != nil {
		return accountcredentials.Account{}, false, nil
	}
	return credentialAccountByIdentity(ctx, s.queries, provider, subject)
}

func (s *SQLStore) UsernameExists(ctx context.Context, username string) (bool, error) {
	if err := s.requireDatabase(); err != nil {
		return false, err
	}
	exists, err := s.queries.UsernameExists(ctx, username)
	if err != nil {
		return false, fmt.Errorf("check username availability: %w", err)
	}
	return exists, nil
}

// RegisterExternalAccount creates the complete identity, access, module,
// profile, and preference aggregate in one SQL statement. A concurrent
// registration for the same provider identity converges on the first commit.
func (s *SQLStore) RegisterExternalAccount(ctx context.Context, provider accountcredentials.IdentityProvider, subject, verifiedEmail, username string, administrator bool) (accountcredentials.Account, bool, error) {
	if err := s.requireDatabase(); err != nil {
		return accountcredentials.Account{}, false, err
	}
	subject, verifiedEmail, err := accountcredentials.NormalizeExternalIdentity(provider, subject, verifiedEmail, administrator)
	if err != nil {
		return accountcredentials.Account{}, false, fmt.Errorf("validate external identity: %w", err)
	}
	if administrator && provider != accountcredentials.IdentityProviderGoogle {
		return accountcredentials.Account{}, false, fmt.Errorf("administrator registration requires Google")
	}
	if err := accountcredentials.ValidateUsername(username, administrator); err != nil {
		return accountcredentials.Account{}, false, err
	}
	if existing, found, err := credentialAccountByIdentity(ctx, s.queries, provider, subject); err != nil || found {
		return existing, false, err
	}

	var account accountcredentials.Account
	if administrator {
		row, queryErr := s.queries.CreateAdministratorAccount(ctx, accountstatesqlc.CreateAdministratorAccountParams{
			Username: username, IdentityProvider: string(provider), IdentitySubject: subject, VerifiedEmail: verifiedEmail,
		})
		if queryErr != nil {
			return s.resolveRegistrationConflict(ctx, provider, subject, true, queryErr)
		}
		account, err = credentialAccountFromFields(
			row.AccountID, row.Username, row.IdentityProvider, row.IdentitySubject,
			row.VerifiedEmail, row.Administrator, row.CreatedAt, row.LastLoginAt,
		)
	} else {
		row, queryErr := s.queries.CreateOrdinaryAccount(ctx, accountstatesqlc.CreateOrdinaryAccountParams{
			Username: username, IdentityProvider: string(provider), IdentitySubject: subject, VerifiedEmail: verifiedEmail,
		})
		if queryErr != nil {
			return s.resolveRegistrationConflict(ctx, provider, subject, false, queryErr)
		}
		account, err = credentialAccountFromFields(
			row.AccountID, row.Username, row.IdentityProvider, row.IdentitySubject,
			row.VerifiedEmail, row.Administrator, row.CreatedAt, row.LastLoginAt,
		)
	}
	if err != nil {
		return accountcredentials.Account{}, false, fmt.Errorf("project registered external account: %w", err)
	}
	if account.Username != username || account.IdentitySubject != subject || account.Administrator != administrator || account.IdentityProvider != provider {
		return accountcredentials.Account{}, false, fmt.Errorf("external registration returned an inconsistent identity")
	}
	return account, true, nil
}

func (s *SQLStore) resolveRegistrationConflict(ctx context.Context, provider accountcredentials.IdentityProvider, subject string, administrator bool, registrationErr error) (accountcredentials.Account, bool, error) {
	// Re-read first for every conflict. A single registration can violate both
	// subject and username/admin uniqueness, but an existing subject must always
	// converge instead of being reported as an unrelated username conflict.
	account, found, lookupErr := credentialAccountByIdentity(ctx, s.queries, provider, subject)
	if lookupErr != nil {
		return accountcredentials.Account{}, false, lookupErr
	}
	if found {
		return account, false, nil
	}
	if errors.Is(registrationErr, pgx.ErrNoRows) {
		return accountcredentials.Account{}, false, fmt.Errorf("external registration conflict completed without a durable identity mapping")
	}
	constraint, unique := uniqueViolationConstraint(registrationErr)
	if !unique {
		return accountcredentials.Account{}, false, fmt.Errorf("register external account: %w", registrationErr)
	}
	if administrator {
		accounts, err := s.queries.ListAccountRecords(ctx)
		if err != nil {
			return accountcredentials.Account{}, false, fmt.Errorf("check administrator registration conflict: %w", err)
		}
		for _, existing := range accounts {
			if existing.Administrator {
				return accountcredentials.Account{}, false, accountcredentials.ErrAdministratorIdentityConflict
			}
		}
	}
	switch constraint {
	case "athena_account_username_lower_uidx":
		return accountcredentials.Account{}, false, accountcredentials.ErrUsernameUnavailable
	case "athena_account_single_administrator_uidx":
		return accountcredentials.Account{}, false, accountcredentials.ErrAdministratorIdentityConflict
	default:
		return accountcredentials.Account{}, false, fmt.Errorf("register external account violated unique constraint %q: %w", constraint, registrationErr)
	}
}

// EnsureDevelopmentAdministrator returns the existing isolated disabled-auth
// identity or creates its complete administrator aggregate. A normal Google
// administrator intentionally conflicts, requiring a full state reset before
// changing authentication modes.
func (s *SQLStore) EnsureDevelopmentAdministrator(ctx context.Context) (accountcredentials.Account, error) {
	if err := s.requireDatabase(); err != nil {
		return accountcredentials.Account{}, err
	}
	if account, found, err := developmentAdministrator(ctx, s.queries); err != nil || found {
		return account, err
	}
	row, err := s.queries.CreateDevelopmentAdministrator(ctx)
	if err != nil {
		if account, found, lookupErr := developmentAdministrator(ctx, s.queries); lookupErr != nil {
			return accountcredentials.Account{}, lookupErr
		} else if found {
			return account, nil
		}
		if constraint, unique := uniqueViolationConstraint(err); unique && constraint == "athena_account_single_administrator_uidx" {
			return accountcredentials.Account{}, accountcredentials.ErrAdministratorIdentityConflict
		}
		return accountcredentials.Account{}, fmt.Errorf("create development administrator: %w", err)
	}
	account, err := credentialAccountFromFields(
		row.AccountID, row.Username, row.IdentityProvider, row.IdentitySubject,
		row.VerifiedEmail, row.Administrator, row.CreatedAt, row.LastLoginAt,
	)
	if err != nil {
		return accountcredentials.Account{}, fmt.Errorf("project development administrator: %w", err)
	}
	if account.IdentityProvider != accountcredentials.IdentityProviderDevelopment || !account.Administrator || account.Username != "local-admin" {
		return accountcredentials.Account{}, fmt.Errorf("development administrator creation returned an inconsistent identity")
	}
	return account, nil
}

func developmentAdministrator(ctx context.Context, queries accountstatesqlc.Querier) (accountcredentials.Account, bool, error) {
	row, err := queries.GetDevelopmentAdministrator(ctx)
	if errors.Is(err, pgx.ErrNoRows) {
		return accountcredentials.Account{}, false, nil
	}
	if err != nil {
		return accountcredentials.Account{}, false, fmt.Errorf("get development administrator: %w", err)
	}
	account, err := credentialAccountFromAthenaRow(row)
	if err != nil {
		return accountcredentials.Account{}, false, err
	}
	return account, true, nil
}

func credentialAccountByIdentity(ctx context.Context, queries accountstatesqlc.Querier, provider accountcredentials.IdentityProvider, subject string) (accountcredentials.Account, bool, error) {
	row, err := queries.GetAccountByIdentity(ctx, accountstatesqlc.GetAccountByIdentityParams{
		IdentityProvider: string(provider),
		IdentitySubject:  subject,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return accountcredentials.Account{}, false, nil
	}
	if err != nil {
		return accountcredentials.Account{}, false, fmt.Errorf("resolve external identity: %w", err)
	}
	account, err := credentialAccountFromAthenaRow(row)
	if err != nil {
		return accountcredentials.Account{}, false, err
	}
	return account, true, nil
}

// RecordLogin updates mutable identity audit fields only for the current
// permanent provider identity and an account whose login remains enabled.
func (s *SQLStore) RecordLogin(ctx context.Context, accountID string, provider accountcredentials.IdentityProvider, subject, verifiedEmail string) (accountcredentials.Account, error) {
	if err := s.requireDatabase(); err != nil {
		return accountcredentials.Account{}, err
	}
	accountIDValue, canonicalID, err := accountIDParam(accountID)
	if err != nil {
		return accountcredentials.Account{}, err
	}
	subject, verifiedEmail, err = accountcredentials.NormalizeExternalIdentity(provider, subject, verifiedEmail, false)
	if err != nil {
		return accountcredentials.Account{}, fmt.Errorf("validate account %q external login identity: %w", canonicalID, err)
	}
	row, err := s.queries.RecordAccountLogin(ctx, accountstatesqlc.RecordAccountLoginParams{
		VerifiedEmail: verifiedEmail, AccountID: accountIDValue,
		IdentityProvider: string(provider), IdentitySubject: subject,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return accountcredentials.Account{}, accountcredentials.ErrLoginDisabled
	}
	if err != nil {
		return accountcredentials.Account{}, fmt.Errorf("record account %q external login: %w", canonicalID, err)
	}
	account, err := credentialAccountFromAthenaRow(row)
	if err != nil {
		return accountcredentials.Account{}, err
	}
	if account.ID != canonicalID || account.IdentityProvider != provider || account.IdentitySubject != subject || account.VerifiedEmail != verifiedEmail || account.LastLoginAt.IsZero() {
		return accountcredentials.Account{}, fmt.Errorf("record account %q external login returned an inconsistent identity", canonicalID)
	}
	return account, nil
}

// CreateAPIKeyMetadata commits API Key metadata only while both login and API
// Key access remain enabled.
func (s *SQLStore) CreateAPIKeyMetadata(ctx context.Context, accountID string, token accountcredentials.Token) error {
	if err := s.requireDatabase(); err != nil {
		return err
	}
	accountIDValue, canonicalID, err := accountIDParam(accountID)
	if err != nil {
		return err
	}
	if !accountcredentials.IsValidAPIKeyDisplayID(token.ID) {
		return fmt.Errorf("invalid API Key display ID %q", token.ID)
	}
	if strings.TrimSpace(token.JTI) == "" || token.JTI != strings.TrimSpace(token.JTI) {
		return fmt.Errorf("API Key JTI is invalid")
	}
	if token.IssuedAt <= 0 {
		return fmt.Errorf("API Key issue time must be positive")
	}
	if token.ExpiresAt != 0 && token.ExpiresAt <= token.IssuedAt {
		return fmt.Errorf("API Key expiry must be later than its issue time")
	}
	expiresAt := pgtype.Timestamptz{}
	if token.ExpiresAt != 0 {
		expiresAt = pgtype.Timestamptz{Time: time.Unix(token.ExpiresAt, 0).UTC(), Valid: true}
	}
	row, err := s.queries.CreateAccountAPIKey(ctx, accountstatesqlc.CreateAccountAPIKeyParams{
		DisplayID: token.ID, Jti: token.JTI,
		IssuedAt:  pgtype.Timestamptz{Time: time.Unix(token.IssuedAt, 0).UTC(), Valid: true},
		ExpiresAt: expiresAt, AccountID: accountIDValue,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return accountcredentials.ErrAPIKeyAccessDisabled
	}
	if err != nil {
		return fmt.Errorf("create account %q API Key metadata: %w", canonicalID, err)
	}
	persisted, err := credentialTokenFromRow(row.DisplayID, row.Jti, row.IssuedAt, row.ExpiresAt)
	if err != nil {
		return fmt.Errorf("project created account %q API Key metadata: %w", canonicalID, err)
	}
	returnedID, err := accountIDFromPG(row.AccountID)
	if err != nil {
		return fmt.Errorf("project created account %q API Key account ID: %w", canonicalID, err)
	}
	if returnedID != canonicalID || persisted != token {
		return fmt.Errorf("create account %q API Key metadata returned inconsistent values", canonicalID)
	}
	return nil
}

func (s *SQLStore) DeleteAPIKeyMetadata(ctx context.Context, accountID, id string) error {
	if err := s.requireDatabase(); err != nil {
		return err
	}
	accountIDValue, canonicalID, err := accountIDParam(accountID)
	if err != nil {
		return err
	}
	jti, err := s.queries.DeleteAccountAPIKey(ctx, accountstatesqlc.DeleteAccountAPIKeyParams{AccountID: accountIDValue, DisplayID: id})
	if err != nil {
		return fmt.Errorf("delete account %q API Key %q: %w", canonicalID, id, err)
	}
	if strings.TrimSpace(jti) == "" {
		return fmt.Errorf("delete account %q API Key %q returned an empty JTI", canonicalID, id)
	}
	return nil
}

func credentialAccountFromAthenaRow(row accountstatesqlc.AthenaAccount) (accountcredentials.Account, error) {
	return credentialAccountFromFields(
		row.AccountID, row.Username, row.IdentityProvider, row.IdentitySubject,
		row.VerifiedEmail, row.Administrator, row.CreatedAt, row.LastLoginAt,
	)
}

func credentialAccountFromFields(accountID pgtype.UUID, username, identityProvider string, identitySubject pgtype.Text, verifiedEmail string, administrator bool, createdAt, lastLoginAt pgtype.Timestamptz) (accountcredentials.Account, error) {
	id, err := accountIDFromPG(accountID)
	if err != nil {
		return accountcredentials.Account{}, fmt.Errorf("persisted credential account ID: %w", err)
	}
	created, err := finiteTimestamp(createdAt, true)
	if err != nil {
		return accountcredentials.Account{}, fmt.Errorf("account %q created time: %w", id, err)
	}
	lastLogin, err := finiteTimestamp(lastLoginAt, false)
	if err != nil {
		return accountcredentials.Account{}, fmt.Errorf("account %q last login time: %w", id, err)
	}
	if !lastLogin.IsZero() && lastLogin.Before(created) {
		return accountcredentials.Account{}, fmt.Errorf("account %q last login predates account creation", id)
	}
	subject := ""
	if identitySubject.Valid {
		subject = identitySubject.String
		if strings.TrimSpace(subject) == "" || subject != strings.TrimSpace(subject) {
			return accountcredentials.Account{}, fmt.Errorf("account %q has an invalid identity subject", id)
		}
	}
	provider := accountcredentials.IdentityProvider(identityProvider)
	email := verifiedEmail
	switch provider {
	case accountcredentials.IdentityProviderGoogle, accountcredentials.IdentityProviderSolanaWallet:
		normalizedSubject, normalizedEmail, err := accountcredentials.NormalizeExternalIdentity(provider, subject, email, administrator)
		if err != nil || normalizedSubject != subject || normalizedEmail != email {
			return accountcredentials.Account{}, fmt.Errorf("account %q has an invalid external identity binding", id)
		}
		if err := accountcredentials.ValidateUsername(username, administrator); err != nil {
			return accountcredentials.Account{}, fmt.Errorf("account %q has an invalid username: %w", id, err)
		}
	case accountcredentials.IdentityProviderDevelopment:
		if !administrator || username != "local-admin" || subject != "" || email != "" {
			return accountcredentials.Account{}, fmt.Errorf("account %q has an invalid development identity", id)
		}
	default:
		return accountcredentials.Account{}, fmt.Errorf("account %q has unsupported identity provider %q", id, identityProvider)
	}
	return accountcredentials.Account{
		ID: id, Username: username, IdentityProvider: provider,
		IdentitySubject: subject, VerifiedEmail: email,
		Administrator: administrator, CreatedAt: created, LastLoginAt: lastLogin,
	}, nil
}

func credentialIdentityKey(provider accountcredentials.IdentityProvider, subject string) string {
	return string(provider) + "\x00" + subject
}

func credentialTokenFromRow(id, jti string, issuedAt, expiresAt pgtype.Timestamptz) (accountcredentials.Token, error) {
	if !accountcredentials.IsValidAPIKeyDisplayID(id) {
		return accountcredentials.Token{}, fmt.Errorf("invalid display ID %q", id)
	}
	if strings.TrimSpace(jti) == "" || jti != strings.TrimSpace(jti) {
		return accountcredentials.Token{}, fmt.Errorf("invalid JTI")
	}
	issued, err := finiteTimestamp(issuedAt, true)
	if err != nil {
		return accountcredentials.Token{}, fmt.Errorf("issue time: %w", err)
	}
	if issued.Unix() <= 0 {
		return accountcredentials.Token{}, fmt.Errorf("issue time must be positive")
	}
	expires, err := finiteTimestamp(expiresAt, false)
	if err != nil {
		return accountcredentials.Token{}, fmt.Errorf("expiry: %w", err)
	}
	if !expires.IsZero() && !expires.After(issued) {
		return accountcredentials.Token{}, fmt.Errorf("expiry must be later than issue time")
	}
	token := accountcredentials.Token{ID: id, JTI: jti, IssuedAt: issued.Unix()}
	if !expires.IsZero() {
		token.ExpiresAt = expires.Unix()
	}
	return token, nil
}

func finiteTimestamp(value pgtype.Timestamptz, required bool) (time.Time, error) {
	if !value.Valid {
		if required {
			return time.Time{}, fmt.Errorf("timestamp is missing")
		}
		return time.Time{}, nil
	}
	if value.InfinityModifier != pgtype.Finite {
		return time.Time{}, fmt.Errorf("timestamp must be finite")
	}
	return value.Time.UTC(), nil
}

func uniqueViolationConstraint(err error) (string, bool) {
	var postgresError *pgconn.PgError
	if !errors.As(err, &postgresError) || postgresError.Code != "23505" {
		return "", false
	}
	return postgresError.ConstraintName, true
}

// ListAccountDirectory returns a stable page of account IDs and the matching
// total from one repeatable-read snapshot.
func (s *SQLStore) ListAccountDirectory(ctx context.Context, query, status string, page, pageSize int32, profitSharingEligibleOnly bool) ([]string, int64, error) {
	if err := s.requireDatabase(); err != nil {
		return nil, 0, err
	}
	query = strings.TrimSpace(query)
	status = strings.ToLower(strings.TrimSpace(status))
	if status == "" {
		status = "all"
	}
	switch status {
	case "all", "pending", "active", "blocked":
	default:
		return nil, 0, fmt.Errorf("unsupported account directory status %q", status)
	}
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 50
	}
	if pageSize > 100 {
		pageSize = 100
	}
	offset := int64(page-1) * int64(pageSize)
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if err != nil {
		return nil, 0, fmt.Errorf("begin account directory snapshot: %w", err)
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback(context.Background())
		}
	}()
	txQueries := accountstatesqlc.New(tx)
	total, err := txQueries.CountAccountDirectory(ctx, accountstatesqlc.CountAccountDirectoryParams{
		SearchQuery: query, StatusFilter: status, ProfitSharingEligibleOnly: profitSharingEligibleOnly,
	})
	if err != nil {
		return nil, 0, fmt.Errorf("count account directory: %w", err)
	}
	if total < 0 {
		return nil, 0, fmt.Errorf("account directory returned a negative total")
	}
	if offset > math.MaxInt32 {
		if err := tx.Commit(ctx); err != nil {
			return nil, 0, fmt.Errorf("commit empty account directory page: %w", err)
		}
		committed = true
		return []string{}, total, nil
	}
	rows, err := txQueries.ListAccountDirectoryPage(ctx, accountstatesqlc.ListAccountDirectoryPageParams{
		SearchQuery: query, StatusFilter: status, ProfitSharingEligibleOnly: profitSharingEligibleOnly,
		OffsetCount: int32(offset), LimitCount: pageSize,
	})
	if err != nil {
		return nil, 0, fmt.Errorf("list account directory page: %w", err)
	}
	accountIDs := make([]string, 0, len(rows))
	seen := make(map[string]struct{}, len(rows))
	for _, row := range rows {
		accountID, err := accountIDFromPG(row.AccountID)
		if err != nil {
			return nil, 0, fmt.Errorf("project account directory ID: %w", err)
		}
		if _, duplicate := seen[accountID]; duplicate {
			return nil, 0, fmt.Errorf("account directory returned duplicate account %q", accountID)
		}
		seen[accountID] = struct{}{}
		accountIDs = append(accountIDs, accountID)
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, 0, fmt.Errorf("commit account directory snapshot: %w", err)
	}
	committed = true
	return accountIDs, total, nil
}

func (s *SQLStore) AccountExists(ctx context.Context, accountID string) (bool, error) {
	if err := s.requireDatabase(); err != nil {
		return false, err
	}
	accountIDValue, canonicalID, err := accountIDParam(accountID)
	if err != nil {
		return false, err
	}
	exists, err := s.queries.AccountExists(ctx, accountIDValue)
	if err != nil {
		return false, fmt.Errorf("check account %q existence: %w", canonicalID, err)
	}
	return exists, nil
}

func (s *SQLStore) GetProfile(ctx context.Context, accountID string) (accountcenter.Profile, bool, error) {
	if err := s.requireDatabase(); err != nil {
		return accountcenter.Profile{}, false, err
	}
	accountIDValue, canonicalID, err := accountIDParam(accountID)
	if err != nil {
		return accountcenter.Profile{}, false, err
	}
	row, err := s.queries.GetAccountProfile(ctx, accountIDValue)
	if errors.Is(err, pgx.ErrNoRows) {
		return accountcenter.Profile{}, false, nil
	}
	if err != nil {
		return accountcenter.Profile{}, false, fmt.Errorf("get account %q profile: %w", canonicalID, err)
	}
	profile, err := profileFromRow(row.DisplayName, row.AccountTier, row.AvatarObjectKey, row.AvatarContentType, row.AvatarEtag, row.AvatarSizeBytes, row.Revision)
	if err != nil {
		return accountcenter.Profile{}, false, fmt.Errorf("project account %q profile: %w", canonicalID, err)
	}
	return profile, true, nil
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

func (s *SQLStore) UpdateProfile(ctx context.Context, accountID string, next accountcenter.Profile, expectedRevision uint64) (accountcenter.Profile, error) {
	if err := s.requireDatabase(); err != nil {
		return accountcenter.Profile{}, err
	}
	accountIDValue, canonicalID, err := accountIDParam(accountID)
	if err != nil {
		return accountcenter.Profile{}, err
	}
	if expectedRevision == 0 || expectedRevision >= math.MaxInt64 || next.Revision != expectedRevision {
		return accountcenter.Profile{}, fmt.Errorf("account %q profile revision does not match expected revision", canonicalID)
	}
	row, queryErr := s.queries.UpdateAccountProfile(ctx, accountstatesqlc.UpdateAccountProfileParams{
		DisplayName: next.DisplayName, AccountTier: string(next.Tier),
		AvatarObjectKey: next.Avatar.ObjectKey, AvatarContentType: next.Avatar.ContentType,
		AvatarEtag: next.Avatar.ETag, AvatarSizeBytes: next.Avatar.SizeBytes,
		AccountID: accountIDValue, ExpectedRevision: int64(expectedRevision),
	})
	if errors.Is(queryErr, pgx.ErrNoRows) {
		return accountcenter.Profile{}, accountcenter.ErrProfileRevisionConflict
	}
	if queryErr != nil {
		return accountcenter.Profile{}, fmt.Errorf("update account %q profile: %w", canonicalID, queryErr)
	}
	profile, err := profileFromRow(row.DisplayName, row.AccountTier, row.AvatarObjectKey, row.AvatarContentType, row.AvatarEtag, row.AvatarSizeBytes, row.Revision)
	if err != nil {
		return accountcenter.Profile{}, fmt.Errorf("project account %q profile: %w", canonicalID, err)
	}
	if profile.Revision != expectedRevision+1 {
		return accountcenter.Profile{}, fmt.Errorf("account %q profile update returned revision %d after expected revision %d", canonicalID, profile.Revision, expectedRevision)
	}
	return profile, nil
}

func profileFromRow(displayName, tier, objectKey, contentType, etag string, sizeBytes, revision int64) (accountcenter.Profile, error) {
	if revision <= 0 {
		return accountcenter.Profile{}, fmt.Errorf("persisted profile revision is non-positive")
	}
	return accountcenter.Profile{
		DisplayName: displayName, Tier: accountcenter.Tier(tier),
		Avatar:   accountcenter.AvatarMetadata{ObjectKey: objectKey, ContentType: contentType, ETag: etag, SizeBytes: sizeBytes},
		Revision: uint64(revision),
	}, nil
}

func (s *SQLStore) GetPreferences(ctx context.Context, accountID string) (accountcenter.Preferences, bool, error) {
	if err := s.requireDatabase(); err != nil {
		return accountcenter.Preferences{}, false, err
	}
	accountIDValue, canonicalID, err := accountIDParam(accountID)
	if err != nil {
		return accountcenter.Preferences{}, false, err
	}
	row, err := s.queries.GetAccountPreferences(ctx, accountIDValue)
	if errors.Is(err, pgx.ErrNoRows) {
		return accountcenter.Preferences{}, false, nil
	}
	if err != nil {
		return accountcenter.Preferences{}, false, fmt.Errorf("get account %q preferences: %w", canonicalID, err)
	}
	if row.Revision <= 0 {
		return accountcenter.Preferences{}, false, fmt.Errorf("account %q has non-positive preferences revision", canonicalID)
	}
	return accountcenter.Preferences{Theme: accountcenter.ThemeMode(row.Theme), Revision: uint64(row.Revision)}, true, nil
}

func (s *SQLStore) UpdatePreferences(ctx context.Context, accountID string, next accountcenter.Preferences, expectedRevision uint64) (accountcenter.Preferences, error) {
	if err := s.requireDatabase(); err != nil {
		return accountcenter.Preferences{}, err
	}
	accountIDValue, canonicalID, err := accountIDParam(accountID)
	if err != nil {
		return accountcenter.Preferences{}, err
	}
	if expectedRevision == 0 || expectedRevision >= math.MaxInt64 || next.Revision != expectedRevision {
		return accountcenter.Preferences{}, fmt.Errorf("account %q preferences revision does not match expected revision", canonicalID)
	}
	row, err := s.queries.UpdateAccountPreferences(ctx, accountstatesqlc.UpdateAccountPreferencesParams{
		Theme: string(next.Theme), AccountID: accountIDValue, ExpectedRevision: int64(expectedRevision),
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return accountcenter.Preferences{}, accountcenter.ErrPreferencesRevisionConflict
	}
	if err != nil {
		return accountcenter.Preferences{}, fmt.Errorf("update account %q preferences: %w", canonicalID, err)
	}
	theme, revision := row.Theme, row.Revision
	if revision <= 0 || uint64(revision) != expectedRevision+1 {
		return accountcenter.Preferences{}, fmt.Errorf("account %q preferences update returned revision %d after expected revision %d", canonicalID, revision, expectedRevision)
	}
	return accountcenter.Preferences{Theme: accountcenter.ThemeMode(theme), Revision: uint64(revision)}, nil
}

var (
	_ accountaccess.Store      = (*SQLStore)(nil)
	_ accountcredentials.Store = (*SQLStore)(nil)
	_ accountcenter.Store      = (*SQLStore)(nil)
)
