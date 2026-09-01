package accountcredentials

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var (
	ErrAdministratorIdentityConflict = errors.New("administrator identity is already registered")
	ErrUsernameUnavailable           = errors.New("username is unavailable")
	ErrLoginDisabled                 = errors.New("account login is disabled")
	ErrAPIKeyAccessDisabled          = errors.New("account API Key access is disabled")
)

// Store is the durable identity and API Key boundary. Implementations commit
// mutations before exposing them to the process-local registry.
type Store interface {
	ListCredentialAccounts(ctx context.Context) (map[string]Account, error)
	GetCredentialAccountByIdentity(ctx context.Context, provider IdentityProvider, subject string) (Account, bool, error)
	UsernameExists(ctx context.Context, username string) (bool, error)
	RegisterExternalAccount(ctx context.Context, provider IdentityProvider, subject, verifiedEmail, username string, administrator bool) (Account, bool, error)
	RecordLogin(ctx context.Context, accountID string, provider IdentityProvider, subject, verifiedEmail string) (Account, error)
	CreateAPIKeyMetadata(ctx context.Context, accountID string, token Token) error
	DeleteAPIKeyMetadata(ctx context.Context, accountID, id string) error
}

type accountRecord struct {
	mutex   sync.RWMutex
	account Account
}

type identityKey struct {
	provider IdentityProvider
	subject  string
}

// CredentialManager is keyed exclusively by canonical account UUID. Username
// is immutable presentation metadata and never participates in authentication.
type CredentialManager struct {
	mutex            sync.RWMutex
	accounts         map[string]*accountRecord
	identityAccounts map[identityKey]string
	store            Store
	jwtCodec         *JWTCodec
}

func NewCredentialManager(ctx context.Context, store Store, jwtCodec *JWTCodec) (*CredentialManager, error) {
	if store == nil || jwtCodec == nil {
		return nil, fmt.Errorf("credential store and JWT codec are required")
	}
	accounts, err := store.ListCredentialAccounts(ctx)
	if err != nil {
		return nil, fmt.Errorf("load credential accounts: %w", err)
	}
	manager := &CredentialManager{
		accounts:         make(map[string]*accountRecord, len(accounts)),
		identityAccounts: make(map[identityKey]string, len(accounts)),
		store:            store,
		jwtCodec:         jwtCodec,
	}
	for id, account := range accounts {
		if account.ID == "" {
			account.ID = id
		}
		if account.ID != id {
			return nil, fmt.Errorf("credential account key %q does not match ID %q", id, account.ID)
		}
		if err := manager.publishLocked(account); err != nil {
			return nil, err
		}
	}
	return manager, nil
}

func (m *CredentialManager) Get(accountID string) (Account, error) {
	canonicalID, err := CanonicalAccountID(accountID)
	if err != nil {
		return Account{}, status.Error(codes.NotFound, "account does not exist")
	}
	m.mutex.RLock()
	record, ok := m.accounts[canonicalID]
	m.mutex.RUnlock()
	if !ok {
		return Account{}, status.Error(codes.NotFound, "account does not exist")
	}
	record.mutex.RLock()
	defer record.mutex.RUnlock()
	return cloneAccount(record.account), nil
}

func (m *CredentialManager) List() map[string]Account {
	m.mutex.RLock()
	records := make(map[string]*accountRecord, len(m.accounts))
	for id, record := range m.accounts {
		records[id] = record
	}
	m.mutex.RUnlock()
	accounts := make(map[string]Account, len(records))
	for id, record := range records {
		record.mutex.RLock()
		accounts[id] = cloneAccount(record.account)
		record.mutex.RUnlock()
	}
	return accounts
}

// GetByIdentity resolves only an already-registered external identity. Unknown
// identities must complete their provider's separate username registration flow.
func (m *CredentialManager) GetByIdentity(ctx context.Context, provider IdentityProvider, subject string) (Account, bool, error) {
	subject, err := NormalizeIdentitySubject(provider, subject)
	if err != nil {
		return Account{}, false, nil
	}
	key := identityKey{provider: provider, subject: subject}
	m.mutex.RLock()
	accountID := m.identityAccounts[key]
	m.mutex.RUnlock()
	if accountID != "" {
		account, err := m.Get(accountID)
		return account, err == nil, err
	}
	account, found, err := m.store.GetCredentialAccountByIdentity(ctx, provider, subject)
	if err != nil || !found {
		return Account{}, found, err
	}
	if err := m.publish(account); err != nil {
		return Account{}, false, err
	}
	account, err = m.Get(account.ID)
	return account, err == nil, err
}

// UsernameAvailable is advisory. Registration still relies on PostgreSQL's
// case-insensitive unique index as the final concurrency-safe decision.
func (m *CredentialManager) UsernameAvailable(ctx context.Context, username string, administrator bool) (bool, error) {
	if err := ValidateUsername(username, administrator); err != nil {
		return false, err
	}
	exists, err := m.store.UsernameExists(ctx, username)
	return !exists, err
}

// RegisterExternalAccount atomically creates a complete account aggregate or
// converges on an account concurrently created for the same provider identity.
func (m *CredentialManager) RegisterExternalAccount(ctx context.Context, provider IdentityProvider, subject, verifiedEmail, username string, administrator bool) (Account, bool, error) {
	subject, verifiedEmail, err := NormalizeExternalIdentity(provider, subject, verifiedEmail, administrator)
	if err != nil {
		return Account{}, false, status.Error(codes.PermissionDenied, "verified external identity is required")
	}
	if administrator && provider != IdentityProviderGoogle {
		return Account{}, false, status.Error(codes.PermissionDenied, "administrator registration requires Google")
	}
	if existing, found, err := m.GetByIdentity(ctx, provider, subject); err != nil || found {
		return existing, false, err
	}
	if err := ValidateUsername(username, administrator); err != nil {
		return Account{}, false, err
	}
	account, created, err := m.store.RegisterExternalAccount(ctx, provider, subject, verifiedEmail, username, administrator)
	if err != nil {
		return Account{}, false, err
	}
	if account.IdentitySubject != subject || account.IdentityProvider != provider {
		return Account{}, false, fmt.Errorf("durable external registration returned an inconsistent identity")
	}
	if err := m.publish(account); err != nil {
		return Account{}, false, err
	}
	snapshot, err := m.Get(account.ID)
	return snapshot, created, err
}

func (m *CredentialManager) RecordLogin(ctx context.Context, accountID string, provider IdentityProvider, subject, verifiedEmail string) (Account, error) {
	subject, verifiedEmail, err := NormalizeExternalIdentity(provider, subject, verifiedEmail, false)
	if err != nil {
		return Account{}, status.Error(codes.PermissionDenied, "verified external identity is required")
	}
	account, err := m.store.RecordLogin(ctx, accountID, provider, subject, verifiedEmail)
	if err != nil {
		return Account{}, err
	}
	if account.ID != accountID || account.IdentityProvider != provider || account.IdentitySubject != subject {
		return Account{}, fmt.Errorf("record external login returned an inconsistent account")
	}
	if err := m.publish(account); err != nil {
		return Account{}, err
	}
	return m.Get(accountID)
}

func (m *CredentialManager) IssueAPIKey(ctx context.Context, accountID, id string, expiresIn int64) (string, error) {
	record, err := m.record(accountID)
	if err != nil {
		return "", err
	}
	if !IsValidAPIKeyDisplayID(id) {
		return "", fmt.Errorf("invalid API Key display ID %q", id)
	}
	record.mutex.Lock()
	defer record.mutex.Unlock()
	if tokenIDIndex(record.account.Tokens, id) >= 0 {
		return "", fmt.Errorf("account already has token with id %q", id)
	}
	jti, err := uuid.NewRandom()
	if err != nil {
		return "", fmt.Errorf("generate API Key JTI: %w", err)
	}
	now := time.Now().UTC()
	tokenString, metadata, err := m.jwtCodec.Issue(record.account.ID, CapabilityAPIKey, jti.String(), expiresIn, now, "")
	if err != nil {
		return "", err
	}
	metadata.ID = id
	if err := m.store.CreateAPIKeyMetadata(ctx, record.account.ID, metadata); err != nil {
		return "", err
	}
	record.account.Tokens = append(record.account.Tokens, metadata)
	return tokenString, nil
}

// IssueLoginSession binds an Athena session to the currently persisted
// external provider identity. Changing either half of that identity invalidates
// the session on its next request.
func (m *CredentialManager) IssueLoginSession(accountID string, verifiedProvider IdentityProvider, verifiedSubject, id string, expiresIn int64) (string, error) {
	verifiedSubject, err := NormalizeIdentitySubject(verifiedProvider, verifiedSubject)
	if err != nil {
		return "", status.Error(codes.PermissionDenied, "external identity binding changed")
	}
	record, err := m.record(accountID)
	if err != nil {
		return "", err
	}
	record.mutex.RLock()
	defer record.mutex.RUnlock()
	if !record.account.HasExternalIdentity() || record.account.IdentityProvider != verifiedProvider || record.account.IdentitySubject != verifiedSubject {
		return "", status.Error(codes.PermissionDenied, "external identity binding changed")
	}
	now := time.Now().UTC()
	tokenString, _, err := m.jwtCodec.Issue(record.account.ID, CapabilityLogin, id, expiresIn, now, identityBinding(record.account.IdentityProvider, record.account.IdentitySubject))
	return tokenString, err
}

func (m *CredentialManager) DeleteAPIKey(ctx context.Context, accountID, id string) error {
	record, err := m.record(accountID)
	if err != nil {
		return err
	}
	record.mutex.Lock()
	defer record.mutex.Unlock()
	index := tokenIDIndex(record.account.Tokens, id)
	if index < 0 {
		return status.Errorf(codes.NotFound, "token with id %q does not exist", id)
	}
	if err := m.store.DeleteAPIKeyMetadata(ctx, record.account.ID, id); err != nil {
		return err
	}
	record.account.Tokens = append(record.account.Tokens[:index], record.account.Tokens[index+1:]...)
	return nil
}

func (m *CredentialManager) ValidateCredential(accountID string, capability Capability, jti, tokenIdentityBinding string) error {
	record, err := m.record(accountID)
	if err != nil {
		return err
	}
	record.mutex.RLock()
	defer record.mutex.RUnlock()
	switch capability {
	case CapabilityLogin:
		if !record.account.HasExternalIdentity() || tokenIdentityBinding != identityBinding(record.account.IdentityProvider, record.account.IdentitySubject) {
			return fmt.Errorf("account external identity binding has changed")
		}
	case CapabilityAPIKey:
		if tokenJTIIndex(record.account.Tokens, jti) < 0 {
			return fmt.Errorf("account does not have an API Key with the supplied JTI")
		}
	default:
		return fmt.Errorf("unsupported account capability %q", capability)
	}
	return nil
}

func (m *CredentialManager) record(accountID string) (*accountRecord, error) {
	canonicalID, err := CanonicalAccountID(accountID)
	if err != nil {
		return nil, status.Error(codes.NotFound, "account does not exist")
	}
	m.mutex.RLock()
	record, ok := m.accounts[canonicalID]
	m.mutex.RUnlock()
	if !ok {
		return nil, status.Error(codes.NotFound, "account does not exist")
	}
	return record, nil
}

func (m *CredentialManager) publish(account Account) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	return m.publishLocked(account)
}

func (m *CredentialManager) publishLocked(account Account) error {
	canonicalID, err := CanonicalAccountID(account.ID)
	if err != nil || canonicalID != account.ID {
		return fmt.Errorf("credential account ID %q is not a canonical UUID", account.ID)
	}
	switch account.IdentityProvider {
	case IdentityProviderGoogle, IdentityProviderSolanaWallet:
		subject, email, err := NormalizeExternalIdentity(account.IdentityProvider, account.IdentitySubject, account.VerifiedEmail, account.Administrator)
		if err != nil || subject != account.IdentitySubject || email != account.VerifiedEmail {
			return fmt.Errorf("credential account %q has an invalid external identity", account.ID)
		}
		if err := ValidateUsername(account.Username, account.Administrator); err != nil {
			return fmt.Errorf("credential account %q has an invalid username: %w", account.ID, err)
		}
	case IdentityProviderDevelopment:
		if _, err := account.DevelopmentRole(); err != nil {
			return fmt.Errorf("credential account %q has an invalid development identity", account.ID)
		}
	default:
		return fmt.Errorf("credential account %q has unsupported identity provider %q", account.ID, account.IdentityProvider)
	}
	if account.HasExternalIdentity() {
		key := identityKey{provider: account.IdentityProvider, subject: account.IdentitySubject}
		if existing := m.identityAccounts[key]; existing != "" && existing != account.ID {
			return fmt.Errorf("external identity is assigned to multiple accounts")
		}
	}
	record, exists := m.accounts[account.ID]
	if !exists {
		m.accounts[account.ID] = &accountRecord{account: cloneAccount(account)}
		if account.HasExternalIdentity() {
			m.identityAccounts[identityKey{provider: account.IdentityProvider, subject: account.IdentitySubject}] = account.ID
		}
		return nil
	}
	record.mutex.Lock()
	defer record.mutex.Unlock()
	previous := record.account
	if previous.Username != account.Username || previous.IdentityProvider != account.IdentityProvider || previous.IdentitySubject != account.IdentitySubject || previous.Administrator != account.Administrator {
		return fmt.Errorf("credential account %q immutable identity changed unexpectedly", account.ID)
	}
	account.Tokens = append([]Token(nil), previous.Tokens...)
	record.account = cloneAccount(account)
	if account.HasExternalIdentity() {
		m.identityAccounts[identityKey{provider: account.IdentityProvider, subject: account.IdentitySubject}] = account.ID
	}
	return nil
}

func identityBinding(provider IdentityProvider, subject string) string {
	digest := sha256.Sum256([]byte(string(provider) + "\x00" + subject))
	return hex.EncodeToString(digest[:])
}

func tokenIDIndex(tokens []Token, id string) int {
	for index, token := range tokens {
		if token.ID == id {
			return index
		}
	}
	return -1
}

func tokenJTIIndex(tokens []Token, jti string) int {
	for index, token := range tokens {
		if token.JTI == jti {
			return index
		}
	}
	return -1
}
