package store

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	log "github.com/sirupsen/logrus"
	"github.com/useryege/athena/common"
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
		if _, duplicate := durableAccounts[row.AccountName]; duplicate {
			return nil, fmt.Errorf("durable account %q is duplicated", row.AccountName)
		}
		durableAccounts[row.AccountName] = struct{}{}
	}
	result := make(map[string]accountaccess.Access, len(heads))
	for _, head := range heads {
		if _, exists := durableAccounts[head.AccountName]; !exists {
			return nil, fmt.Errorf("account access head references unknown account %q", head.AccountName)
		}
		if _, duplicate := result[head.AccountName]; duplicate {
			return nil, fmt.Errorf("account %q has duplicate access heads", head.AccountName)
		}
		access, err := accessFromHead(head.AccountName, head.LoginEnabled, head.ApiKeyEnabled, head.ProfitSharingEnabled, head.Revision)
		if err != nil {
			return nil, err
		}
		result[head.AccountName] = access
	}
	for name := range durableAccounts {
		if _, exists := result[name]; !exists {
			return nil, fmt.Errorf("durable account %q has no access head", name)
		}
	}
	if err := attachModuleAccess(result, moduleRows); err != nil {
		return nil, err
	}
	for name, access := range result {
		if err := access.Validate(); err != nil {
			return nil, fmt.Errorf("account %q has invalid persisted access: %w", name, err)
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
func (s *SQLStore) GetAccountAccess(ctx context.Context, name string) (accountaccess.Access, error) {
	if err := s.requireDatabase(); err != nil {
		return accountaccess.Access{}, err
	}
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if err != nil {
		return accountaccess.Access{}, fmt.Errorf("begin account %q access snapshot: %w", name, err)
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback(context.Background())
		}
	}()
	txQueries := accountstatesqlc.New(tx)

	head, err := txQueries.GetAccountAccessHead(ctx, name)
	if err != nil {
		return accountaccess.Access{}, fmt.Errorf("get account %q access head: %w", name, err)
	}
	access, err := accessFromHead(head.AccountName, head.LoginEnabled, head.ApiKeyEnabled, head.ProfitSharingEnabled, head.Revision)
	if err != nil {
		return accountaccess.Access{}, err
	}
	if head.AccountName != name {
		return accountaccess.Access{}, fmt.Errorf("account %q access query returned head for %q", name, head.AccountName)
	}
	moduleRows, err := txQueries.ListAccountModuleAccessByAccount(ctx, name)
	if err != nil {
		return accountaccess.Access{}, fmt.Errorf("list account %q module access: %w", name, err)
	}
	result := map[string]accountaccess.Access{name: access}
	if err := attachModuleAccess(result, moduleRows); err != nil {
		return accountaccess.Access{}, err
	}
	access = result[name]
	if err := access.Validate(); err != nil {
		return accountaccess.Access{}, fmt.Errorf("account %q has invalid persisted access: %w", name, err)
	}
	if err := tx.Commit(ctx); err != nil {
		return accountaccess.Access{}, fmt.Errorf("commit account %q access snapshot: %w", name, err)
	}
	committed = true
	return access.Clone(), nil
}

// UpdateAccountAccess replaces one ordinary account's full access aggregate.
// The head CAS and all ten module updates commit atomically.
func (s *SQLStore) UpdateAccountAccess(ctx context.Context, name string, next accountaccess.Access, expectedRevision uint64) (accountaccess.Access, error) {
	if err := s.requireDatabase(); err != nil {
		return accountaccess.Access{}, err
	}
	if expectedRevision == 0 || expectedRevision >= math.MaxInt64 {
		return accountaccess.Access{}, fmt.Errorf("account %q expected access revision is out of range", name)
	}
	if next.Revision != expectedRevision {
		return accountaccess.Access{}, fmt.Errorf("account %q access revision does not match expected revision", name)
	}
	next = next.Clone()
	if err := next.Validate(); err != nil {
		return accountaccess.Access{}, fmt.Errorf("validate account %q access: %w", name, err)
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
	head, err := txQueries.UpdateAccountAccessHead(ctx, accountstatesqlc.UpdateAccountAccessHeadParams{
		LoginEnabled: next.LoginEnabled, ApiKeyEnabled: next.APIKeyEnabled,
		ProfitSharingEnabled: next.ProfitSharingEnabled, AccountName: name,
		ExpectedRevision: int64(expectedRevision),
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return accountaccess.Access{}, accountaccess.ErrRevisionConflict
	}
	if err != nil {
		return accountaccess.Access{}, fmt.Errorf("update account %q access head: %w", name, err)
	}
	if head.AccountName != name || head.LoginEnabled != next.LoginEnabled || head.ApiKeyEnabled != next.APIKeyEnabled || head.ProfitSharingEnabled != next.ProfitSharingEnabled {
		return accountaccess.Access{}, fmt.Errorf("account %q access update returned an inconsistent head", name)
	}
	if head.Revision <= 0 || uint64(head.Revision) != expectedRevision+1 {
		return accountaccess.Access{}, fmt.Errorf("account %q access update returned revision %d after expected revision %d", name, head.Revision, expectedRevision)
	}
	rowsAffected, err := txQueries.ReplaceAccountModuleAccess(ctx, accountstatesqlc.ReplaceAccountModuleAccessParams{
		MarketRadarAccessLevel: string(next.Modules[accountaccess.ModuleMarketRadar]), SportsLiveAccessLevel: string(next.Modules[accountaccess.ModuleSportsLive]),
		SportsHistoryAccessLevel: string(next.Modules[accountaccess.ModuleSportsHistory]), ManagedOoAccessLevel: string(next.Modules[accountaccess.ModuleManagedOO]),
		WormMarketsAccessLevel: string(next.Modules[accountaccess.ModuleWormMarkets]), FifaMarketDashboardAccessLevel: string(next.Modules[accountaccess.ModuleFIFAMarketDashboard]),
		WorldCupCornersAccessLevel: string(next.Modules[accountaccess.ModuleWorldCupCorners]), TokenAccessLevel: string(next.Modules[accountaccess.ModuleToken]),
		WalletAccessLevel: string(next.Modules[accountaccess.ModuleWallet]), NotificationsAccessLevel: string(next.Modules[accountaccess.ModuleNotifications]),
		AccountName: name,
	})
	if err != nil {
		return accountaccess.Access{}, fmt.Errorf("replace account %q module access: %w", name, err)
	}
	expectedModuleRows := int64(len(accountaccess.AllModules()))
	if rowsAffected != expectedModuleRows {
		return accountaccess.Access{}, fmt.Errorf("replace account %q module access affected %d rows, expected %d", name, rowsAffected, expectedModuleRows)
	}
	if err := tx.Commit(ctx); err != nil {
		return accountaccess.Access{}, fmt.Errorf("commit account %q access update: %w", name, err)
	}
	committed = true
	return accountaccess.Access{
		LoginEnabled: head.LoginEnabled, APIKeyEnabled: head.ApiKeyEnabled,
		ProfitSharingEnabled: head.ProfitSharingEnabled, Modules: next.Modules,
		Revision: uint64(head.Revision),
	}.Clone(), nil
}

func accessFromHead(name string, loginEnabled, apiKeyEnabled, profitSharingEnabled bool, revision int64) (accountaccess.Access, error) {
	if strings.TrimSpace(name) == "" || strings.Contains(name, ":") {
		return accountaccess.Access{}, fmt.Errorf("persisted account access name %q is invalid", name)
	}
	if revision <= 0 {
		return accountaccess.Access{}, fmt.Errorf("account %q has non-positive access revision %d", name, revision)
	}
	return accountaccess.Access{
		LoginEnabled: loginEnabled, APIKeyEnabled: apiKeyEnabled,
		ProfitSharingEnabled: profitSharingEnabled,
		Modules:              make(map[accountaccess.Module]accountaccess.AccessLevel, len(accountaccess.AllModules())),
		Revision:             uint64(revision),
	}, nil
}

func attachModuleAccess(accessByName map[string]accountaccess.Access, rows []accountstatesqlc.AccountModuleAccess) error {
	for _, row := range rows {
		access, exists := accessByName[row.AccountName]
		if !exists {
			return fmt.Errorf("account module access for %q has no access head", row.AccountName)
		}
		module := accountaccess.Module(row.Module)
		if _, known := accountaccess.MaxAccessLevel(module); !known {
			return fmt.Errorf("account %q has unknown access module %q", row.AccountName, row.Module)
		}
		if _, duplicate := access.Modules[module]; duplicate {
			return fmt.Errorf("account %q has duplicate access rows for module %q", row.AccountName, row.Module)
		}
		access.Modules[module] = accountaccess.AccessLevel(row.AccessLevel)
		accessByName[row.AccountName] = access
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
	seenSubject := make(map[string]string, len(rows))
	for _, row := range rows {
		account, err := credentialAccountFromAthenaRow(row)
		if err != nil {
			return nil, err
		}
		if _, duplicate := accounts[account.Name]; duplicate {
			return nil, fmt.Errorf("credential account %q is duplicated", account.Name)
		}
		if account.GoogleSubject != "" {
			if previous := seenSubject[account.GoogleSubject]; previous != "" {
				return nil, fmt.Errorf("Google subject is assigned to both %q and %q", previous, account.Name)
			}
			seenSubject[account.GoogleSubject] = account.Name
		}
		accounts[account.Name] = account
	}
	keyRows, err := txQueries.ListAccountAPIKeyRecords(ctx)
	if err != nil {
		return nil, fmt.Errorf("list account API Key metadata: %w", err)
	}
	seenJTI := make(map[string]string, len(keyRows))
	seenDisplayID := make(map[string]struct{}, len(keyRows))
	for _, row := range keyRows {
		account, exists := accounts[row.AccountName]
		if !exists {
			return nil, fmt.Errorf("API Key metadata references unknown account %q", row.AccountName)
		}
		token, err := credentialTokenFromRow(row.DisplayID, row.Jti, row.IssuedAt, row.ExpiresAt)
		if err != nil {
			return nil, fmt.Errorf("project account %q API Key %q: %w", row.AccountName, row.DisplayID, err)
		}
		if previous := seenJTI[token.JTI]; previous != "" {
			return nil, fmt.Errorf("API Key JTI is duplicated across accounts %q and %q", previous, row.AccountName)
		}
		seenJTI[token.JTI] = row.AccountName
		displayKey := row.AccountName + "\x00" + token.ID
		if _, duplicate := seenDisplayID[displayKey]; duplicate {
			return nil, fmt.Errorf("account %q has duplicate API Key display ID %q", row.AccountName, token.ID)
		}
		seenDisplayID[displayKey] = struct{}{}
		account.Tokens = append(account.Tokens, token)
		accounts[row.AccountName] = account
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit credential account snapshot transaction: %w", err)
	}
	committed = true
	return accounts, nil
}

// ResolveOrProvisionGoogleAccount resolves an existing subject before
// considering administrator bootstrap or ordinary-account creation.
func (s *SQLStore) ResolveOrProvisionGoogleAccount(ctx context.Context, subject, verifiedEmail, administratorEmail string) (accountcredentials.Account, bool, error) {
	if err := s.requireDatabase(); err != nil {
		return accountcredentials.Account{}, false, err
	}
	subject = strings.TrimSpace(subject)
	verifiedEmail = strings.TrimSpace(verifiedEmail)
	administratorEmail = strings.TrimSpace(administratorEmail)
	if subject == "" || verifiedEmail == "" {
		return accountcredentials.Account{}, false, fmt.Errorf("verified Google subject and email are required")
	}
	account, found, err := credentialAccountByGoogleSubject(ctx, s.queries, subject)
	if err != nil {
		return accountcredentials.Account{}, false, err
	}
	if found {
		return account, false, nil
	}
	if administratorEmail != "" && strings.EqualFold(verifiedEmail, administratorEmail) {
		return s.claimAdministratorIdentity(ctx, subject, verifiedEmail)
	}
	row, err := s.queries.CreateOrdinaryAccount(ctx, accountstatesqlc.CreateOrdinaryAccountParams{GoogleSubject: subject, VerifiedEmail: verifiedEmail})
	if errors.Is(err, pgx.ErrNoRows) || isUniqueViolation(err) {
		return s.resolveConcurrentIdentity(ctx, subject, false)
	}
	if err != nil {
		return accountcredentials.Account{}, false, fmt.Errorf("create ordinary account for Google subject: %w", err)
	}
	account, err = credentialAccountFromCreateRow(row)
	if err != nil {
		return accountcredentials.Account{}, false, err
	}
	if account.GoogleSubject != subject || account.Administrator {
		return accountcredentials.Account{}, false, fmt.Errorf("ordinary account creation returned an inconsistent identity")
	}
	return account, true, nil
}

func (s *SQLStore) claimAdministratorIdentity(ctx context.Context, subject, verifiedEmail string) (accountcredentials.Account, bool, error) {
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.ReadCommitted})
	if err != nil {
		return accountcredentials.Account{}, false, fmt.Errorf("begin administrator identity claim: %w", err)
	}
	finished := false
	defer func() {
		if !finished {
			_ = tx.Rollback(context.Background())
		}
	}()
	txQueries := accountstatesqlc.New(tx)
	adminRow, err := txQueries.GetAdministratorForUpdate(ctx)
	if err != nil {
		return accountcredentials.Account{}, false, fmt.Errorf("lock administrator identity: %w", err)
	}
	existing, found, err := credentialAccountByGoogleSubject(ctx, txQueries, subject)
	if err != nil {
		return accountcredentials.Account{}, false, err
	}
	if found {
		if err := tx.Commit(ctx); err != nil {
			return accountcredentials.Account{}, false, fmt.Errorf("commit existing Google identity lookup: %w", err)
		}
		finished = true
		return existing, false, nil
	}
	administrator, err := credentialAccountFromAthenaRow(adminRow)
	if err != nil {
		return accountcredentials.Account{}, false, err
	}
	if !administrator.Administrator || administrator.Name != common.AthenaAdminUsername {
		return accountcredentials.Account{}, false, fmt.Errorf("persisted administrator identity is invalid")
	}
	if administrator.GoogleSubject != "" {
		return accountcredentials.Account{}, false, accountcredentials.ErrAdministratorIdentityConflict
	}
	claimedRow, err := txQueries.ClaimAdministratorIdentity(ctx, accountstatesqlc.ClaimAdministratorIdentityParams{GoogleSubject: subject, VerifiedEmail: verifiedEmail})
	if errors.Is(err, pgx.ErrNoRows) || isUniqueViolation(err) {
		if rollbackErr := tx.Rollback(context.Background()); rollbackErr != nil && !errors.Is(rollbackErr, pgx.ErrTxClosed) {
			return accountcredentials.Account{}, false, fmt.Errorf("rollback contested administrator identity claim: %w", rollbackErr)
		}
		finished = true
		return s.resolveConcurrentIdentity(ctx, subject, true)
	}
	if err != nil {
		return accountcredentials.Account{}, false, fmt.Errorf("claim administrator Google identity: %w", err)
	}
	claimed, err := credentialAccountFromAthenaRow(claimedRow)
	if err != nil {
		return accountcredentials.Account{}, false, err
	}
	if !claimed.Administrator || claimed.Name != common.AthenaAdminUsername || claimed.GoogleSubject != subject {
		return accountcredentials.Account{}, false, fmt.Errorf("administrator identity claim returned an inconsistent account")
	}
	if err := tx.Commit(ctx); err != nil {
		return accountcredentials.Account{}, false, fmt.Errorf("commit administrator identity claim: %w", err)
	}
	finished = true
	return claimed, false, nil
}

func (s *SQLStore) resolveConcurrentIdentity(ctx context.Context, subject string, administratorClaim bool) (accountcredentials.Account, bool, error) {
	account, found, err := credentialAccountByGoogleSubject(ctx, s.queries, subject)
	if err != nil {
		return accountcredentials.Account{}, false, err
	}
	if found {
		return account, false, nil
	}
	if administratorClaim {
		row, err := s.queries.GetAccountRecord(ctx, common.AthenaAdminUsername)
		if err != nil {
			return accountcredentials.Account{}, false, fmt.Errorf("re-read administrator identity after claim conflict: %w", err)
		}
		administrator, err := credentialAccountFromAthenaRow(row)
		if err != nil {
			return accountcredentials.Account{}, false, err
		}
		if administrator.GoogleSubject != "" && administrator.GoogleSubject != subject {
			return accountcredentials.Account{}, false, accountcredentials.ErrAdministratorIdentityConflict
		}
	}
	return accountcredentials.Account{}, false, fmt.Errorf("Google identity conflict completed without a durable subject mapping")
}

func credentialAccountByGoogleSubject(ctx context.Context, queries accountstatesqlc.Querier, subject string) (accountcredentials.Account, bool, error) {
	row, err := queries.GetAccountByGoogleSubject(ctx, subject)
	if errors.Is(err, pgx.ErrNoRows) {
		return accountcredentials.Account{}, false, nil
	}
	if err != nil {
		return accountcredentials.Account{}, false, fmt.Errorf("resolve Google subject: %w", err)
	}
	account, err := credentialAccountFromAthenaRow(row)
	if err != nil {
		return accountcredentials.Account{}, false, err
	}
	return account, true, nil
}

// RecordGoogleLogin updates mutable identity audit fields only for the current
// permanent subject and an account whose login remains enabled.
func (s *SQLStore) RecordGoogleLogin(ctx context.Context, name, subject, verifiedEmail string) (accountcredentials.Account, error) {
	if err := s.requireDatabase(); err != nil {
		return accountcredentials.Account{}, err
	}
	name = strings.TrimSpace(name)
	subject = strings.TrimSpace(subject)
	verifiedEmail = strings.TrimSpace(verifiedEmail)
	if name == "" || subject == "" || verifiedEmail == "" {
		return accountcredentials.Account{}, fmt.Errorf("account name, Google subject, and verified email are required")
	}
	row, err := s.queries.RecordAccountLogin(ctx, accountstatesqlc.RecordAccountLoginParams{VerifiedEmail: verifiedEmail, AccountName: name, GoogleSubject: subject})
	if errors.Is(err, pgx.ErrNoRows) {
		return accountcredentials.Account{}, accountcredentials.ErrLoginDisabled
	}
	if err != nil {
		return accountcredentials.Account{}, fmt.Errorf("record account %q Google login: %w", name, err)
	}
	account, err := credentialAccountFromAthenaRow(row)
	if err != nil {
		return accountcredentials.Account{}, err
	}
	if account.Name != name || account.GoogleSubject != subject || account.LastLoginAt.IsZero() {
		return accountcredentials.Account{}, fmt.Errorf("record account %q Google login returned an inconsistent identity", name)
	}
	return account, nil
}

// CreateAPIKeyMetadata commits API Key metadata only while both login and API
// Key access remain enabled.
func (s *SQLStore) CreateAPIKeyMetadata(ctx context.Context, name string, token accountcredentials.Token) error {
	if err := s.requireDatabase(); err != nil {
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
		ExpiresAt: expiresAt, AccountName: name,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return accountcredentials.ErrAPIKeyAccessDisabled
	}
	if err != nil {
		return fmt.Errorf("create account %q API Key metadata: %w", name, err)
	}
	persisted, err := credentialTokenFromRow(row.DisplayID, row.Jti, row.IssuedAt, row.ExpiresAt)
	if err != nil {
		return fmt.Errorf("project created account %q API Key metadata: %w", name, err)
	}
	if row.AccountName != name || persisted != token {
		return fmt.Errorf("create account %q API Key metadata returned inconsistent values", name)
	}
	return nil
}

func (s *SQLStore) DeleteAPIKeyMetadata(ctx context.Context, name, id string) error {
	if err := s.requireDatabase(); err != nil {
		return err
	}
	jti, err := s.queries.DeleteAccountAPIKey(ctx, accountstatesqlc.DeleteAccountAPIKeyParams{AccountName: name, DisplayID: id})
	if err != nil {
		return fmt.Errorf("delete account %q API Key %q: %w", name, id, err)
	}
	if strings.TrimSpace(jti) == "" {
		return fmt.Errorf("delete account %q API Key %q returned an empty JTI", name, id)
	}
	return nil
}

func credentialAccountFromAthenaRow(row accountstatesqlc.AthenaAccount) (accountcredentials.Account, error) {
	return credentialAccountFromFields(row.AccountName, row.GoogleSubject, row.VerifiedEmail, row.Administrator, row.CreatedAt, row.LastLoginAt)
}

func credentialAccountFromCreateRow(row accountstatesqlc.CreateOrdinaryAccountRow) (accountcredentials.Account, error) {
	return credentialAccountFromFields(row.AccountName, row.GoogleSubject, row.VerifiedEmail, row.Administrator, row.CreatedAt, row.LastLoginAt)
}

func credentialAccountFromFields(name string, googleSubject pgtype.Text, verifiedEmail string, administrator bool, createdAt, lastLoginAt pgtype.Timestamptz) (accountcredentials.Account, error) {
	if strings.TrimSpace(name) == "" || name != strings.TrimSpace(name) || strings.Contains(name, ":") {
		return accountcredentials.Account{}, fmt.Errorf("persisted credential account name %q is invalid", name)
	}
	if administrator != (name == common.AthenaAdminUsername) {
		return accountcredentials.Account{}, fmt.Errorf("persisted account %q has an invalid administrator flag", name)
	}
	created, err := finiteTimestamp(createdAt, true)
	if err != nil {
		return accountcredentials.Account{}, fmt.Errorf("account %q created time: %w", name, err)
	}
	lastLogin, err := finiteTimestamp(lastLoginAt, false)
	if err != nil {
		return accountcredentials.Account{}, fmt.Errorf("account %q last login time: %w", name, err)
	}
	if !lastLogin.IsZero() && lastLogin.Before(created) {
		return accountcredentials.Account{}, fmt.Errorf("account %q last login predates account creation", name)
	}
	subject := ""
	if googleSubject.Valid {
		subject = strings.TrimSpace(googleSubject.String)
		if subject == "" || subject != googleSubject.String {
			return accountcredentials.Account{}, fmt.Errorf("account %q has an invalid Google subject", name)
		}
	}
	email := strings.TrimSpace(verifiedEmail)
	if email != verifiedEmail {
		return accountcredentials.Account{}, fmt.Errorf("account %q has an untrimmed verified email", name)
	}
	if subject == "" {
		if !administrator || email != "" {
			return accountcredentials.Account{}, fmt.Errorf("account %q has an incomplete Google identity binding", name)
		}
	} else if email == "" {
		return accountcredentials.Account{}, fmt.Errorf("account %q has a Google subject without a verified email", name)
	}
	return accountcredentials.Account{
		Name: name, GoogleSubject: subject, VerifiedEmail: email,
		Administrator: administrator, CreatedAt: created, LastLoginAt: lastLogin,
	}, nil
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

func isUniqueViolation(err error) bool {
	var postgresError *pgconn.PgError
	return errors.As(err, &postgresError) && postgresError.Code == "23505"
}

// ListAccountDirectory returns a stable page of account names and the matching
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
	names := make([]string, 0, len(rows))
	seen := make(map[string]struct{}, len(rows))
	for _, row := range rows {
		if strings.TrimSpace(row.AccountName) == "" {
			return nil, 0, fmt.Errorf("account directory returned an empty account name")
		}
		if _, duplicate := seen[row.AccountName]; duplicate {
			return nil, 0, fmt.Errorf("account directory returned duplicate account %q", row.AccountName)
		}
		seen[row.AccountName] = struct{}{}
		names = append(names, row.AccountName)
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, 0, fmt.Errorf("commit account directory snapshot: %w", err)
	}
	committed = true
	return names, total, nil
}

func (s *SQLStore) AccountExists(ctx context.Context, name string) (bool, error) {
	if err := s.requireDatabase(); err != nil {
		return false, err
	}
	exists, err := s.queries.AccountExists(ctx, name)
	if err != nil {
		return false, fmt.Errorf("check account %q existence: %w", name, err)
	}
	return exists, nil
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
		AccountName: name, DisplayName: next.DisplayName, AccountTier: string(next.Tier),
		AvatarObjectKey: next.Avatar.ObjectKey, AvatarContentType: next.Avatar.ContentType,
		AvatarEtag: next.Avatar.ETag, AvatarSizeBytes: next.Avatar.SizeBytes,
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
			DisplayName: params.DisplayName, AccountTier: params.AccountTier,
			AvatarObjectKey: params.AvatarObjectKey, AvatarContentType: params.AvatarContentType,
			AvatarEtag: params.AvatarEtag, AvatarSizeBytes: params.AvatarSizeBytes,
			AccountName: name, ExpectedRevision: int64(expectedRevision),
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
		DisplayName: displayName, Tier: accountcenter.Tier(tier),
		Avatar:   accountcenter.AvatarMetadata{ObjectKey: objectKey, ContentType: contentType, ETag: etag, SizeBytes: sizeBytes},
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
		row, err := s.queries.CreateAccountPreferences(ctx, accountstatesqlc.CreateAccountPreferencesParams{AccountName: name, Theme: string(next.Theme)})
		if errors.Is(err, pgx.ErrNoRows) {
			return accountcenter.Preferences{}, accountcenter.ErrPreferencesRevisionConflict
		}
		if err != nil {
			return accountcenter.Preferences{}, fmt.Errorf("create account %q preferences: %w", name, err)
		}
		theme, revision = row.Theme, row.Revision
	} else {
		row, err := s.queries.UpdateAccountPreferences(ctx, accountstatesqlc.UpdateAccountPreferencesParams{
			Theme: string(next.Theme), AccountName: name, ExpectedRevision: int64(expectedRevision),
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

var (
	_ accountaccess.Store      = (*SQLStore)(nil)
	_ accountcredentials.Store = (*SQLStore)(nil)
	_ accountcenter.Store      = (*SQLStore)(nil)
)
