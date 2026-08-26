package accountcredentials

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var (
	ErrAdministratorIdentityConflict = errors.New("administrator Google identity is already registered")
	ErrUsernameUnavailable           = errors.New("username is unavailable")
	ErrLoginDisabled                 = errors.New("account login is disabled")
	ErrAPIKeyAccessDisabled          = errors.New("account API Key access is disabled")
)

// Store is the durable identity and API Key boundary. Implementations commit
// mutations before exposing them to the process-local registry.
type Store interface {
	ListCredentialAccounts(ctx context.Context) (map[string]Account, error)
	GetCredentialAccountByGoogleSubject(ctx context.Context, subject string) (Account, bool, error)
	UsernameExists(ctx context.Context, username string) (bool, error)
	RegisterGoogleAccount(ctx context.Context, subject, verifiedEmail, username string, administrator bool) (Account, bool, error)
	RecordGoogleLogin(ctx context.Context, accountID, subject, verifiedEmail string) (Account, error)
	CreateAPIKeyMetadata(ctx context.Context, accountID string, token Token) error
	DeleteAPIKeyMetadata(ctx context.Context, accountID, id string) error
}

type accountRecord struct {
	mutex   sync.RWMutex
	account Account
}

// CredentialManager is keyed exclusively by canonical account UUID. Username
// is immutable presentation metadata and never participates in authentication.
type CredentialManager struct {
	mutex          sync.RWMutex
	accounts       map[string]*accountRecord
	googleAccounts map[string]string
	store          Store
	jwtCodec       *JWTCodec
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
		accounts:       make(map[string]*accountRecord, len(accounts)),
		googleAccounts: make(map[string]string, len(accounts)),
		store:          store,
		jwtCodec:       jwtCodec,
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

// GetByGoogleSubject resolves only already-registered identities. Unknown
// verified identities must complete the separate username registration flow.
func (m *CredentialManager) GetByGoogleSubject(ctx context.Context, subject string) (Account, bool, error) {
	subject = strings.TrimSpace(subject)
	if subject == "" {
		return Account{}, false, nil
	}
	m.mutex.RLock()
	accountID := m.googleAccounts[subject]
	m.mutex.RUnlock()
	if accountID != "" {
		account, err := m.Get(accountID)
		return account, err == nil, err
	}
	account, found, err := m.store.GetCredentialAccountByGoogleSubject(ctx, subject)
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

// RegisterGoogleAccount atomically creates the complete account aggregate or
// converges on an account concurrently created for the same Google subject.
func (m *CredentialManager) RegisterGoogleAccount(ctx context.Context, subject, verifiedEmail, username string, administrator bool) (Account, bool, error) {
	subject = strings.TrimSpace(subject)
	verifiedEmail = strings.TrimSpace(verifiedEmail)
	if subject == "" || verifiedEmail == "" {
		return Account{}, false, status.Error(codes.PermissionDenied, "verified Google identity is required")
	}
	if existing, found, err := m.GetByGoogleSubject(ctx, subject); err != nil || found {
		return existing, false, err
	}
	if err := ValidateUsername(username, administrator); err != nil {
		return Account{}, false, err
	}
	account, created, err := m.store.RegisterGoogleAccount(ctx, subject, verifiedEmail, username, administrator)
	if err != nil {
		return Account{}, false, err
	}
	if account.GoogleSubject != subject || account.IdentityProvider != IdentityProviderGoogle {
		return Account{}, false, fmt.Errorf("durable Google registration returned an inconsistent identity")
	}
	if err := m.publish(account); err != nil {
		return Account{}, false, err
	}
	snapshot, err := m.Get(account.ID)
	return snapshot, created, err
}

func (m *CredentialManager) RecordGoogleLogin(ctx context.Context, accountID, subject, verifiedEmail string) (Account, error) {
	account, err := m.store.RecordGoogleLogin(ctx, accountID, subject, strings.TrimSpace(verifiedEmail))
	if err != nil {
		return Account{}, err
	}
	if account.ID != accountID || account.GoogleSubject != subject {
		return Account{}, fmt.Errorf("record Google login returned an inconsistent account")
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

func (m *CredentialManager) IssueGoogleLoginSession(accountID, verifiedSubject, id string, expiresIn int64) (string, error) {
	record, err := m.record(accountID)
	if err != nil {
		return "", err
	}
	record.mutex.RLock()
	defer record.mutex.RUnlock()
	if verifiedSubject == "" || record.account.IdentityProvider != IdentityProviderGoogle || record.account.GoogleSubject != verifiedSubject {
		return "", status.Error(codes.PermissionDenied, "Google identity binding changed")
	}
	now := time.Now().UTC()
	tokenString, _, err := m.jwtCodec.Issue(record.account.ID, CapabilityLogin, id, expiresIn, now, googleIdentityBinding(record.account.GoogleSubject))
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
		if record.account.IdentityProvider != IdentityProviderGoogle || record.account.GoogleSubject == "" || tokenIdentityBinding != googleIdentityBinding(record.account.GoogleSubject) {
			return fmt.Errorf("account Google identity binding has changed")
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
	case IdentityProviderGoogle:
		if !account.HasGoogleBinding() || strings.TrimSpace(account.VerifiedEmail) == "" {
			return fmt.Errorf("credential account %q has an incomplete Google identity", account.ID)
		}
		if err := ValidateUsername(account.Username, account.Administrator); err != nil {
			return fmt.Errorf("credential account %q has an invalid username: %w", account.ID, err)
		}
	case IdentityProviderDevelopment:
		if !account.Administrator || account.Username != "local-admin" || account.GoogleSubject != "" || account.VerifiedEmail != "" {
			return fmt.Errorf("credential account %q has an invalid development identity", account.ID)
		}
	default:
		return fmt.Errorf("credential account %q has unsupported identity provider %q", account.ID, account.IdentityProvider)
	}
	if account.GoogleSubject != "" {
		if existing := m.googleAccounts[account.GoogleSubject]; existing != "" && existing != account.ID {
			return fmt.Errorf("Google subject is assigned to multiple accounts")
		}
	}
	record, exists := m.accounts[account.ID]
	if !exists {
		m.accounts[account.ID] = &accountRecord{account: cloneAccount(account)}
		if account.GoogleSubject != "" {
			m.googleAccounts[account.GoogleSubject] = account.ID
		}
		return nil
	}
	record.mutex.Lock()
	defer record.mutex.Unlock()
	previous := record.account
	if previous.Username != account.Username || previous.IdentityProvider != account.IdentityProvider || previous.GoogleSubject != account.GoogleSubject || previous.Administrator != account.Administrator {
		return fmt.Errorf("credential account %q immutable identity changed unexpectedly", account.ID)
	}
	account.Tokens = append([]Token(nil), previous.Tokens...)
	record.account = cloneAccount(account)
	if account.GoogleSubject != "" {
		m.googleAccounts[account.GoogleSubject] = account.ID
	}
	return nil
}

func googleIdentityBinding(subject string) string {
	digest := sha256.Sum256([]byte("google\x00" + subject))
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
