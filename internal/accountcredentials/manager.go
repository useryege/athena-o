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

// ErrAdministratorIdentityConflict indicates that the configured bootstrap
// email matched an identity after the permanent administrator binding had
// already been claimed by a different Google subject.
var (
	ErrAdministratorIdentityConflict = errors.New("administrator Google identity is already bound")
	ErrLoginDisabled                 = errors.New("account login is disabled")
	ErrAPIKeyAccessDisabled          = errors.New("account API Key access is disabled")
)

// Store is the durable identity and API Key boundary. Implementations must
// commit every mutation before returning it to the runtime registry.
type Store interface {
	ListCredentialAccounts(ctx context.Context) (map[string]Account, error)
	ResolveOrProvisionGoogleAccount(ctx context.Context, subject, verifiedEmail, administratorEmail string) (Account, bool, error)
	RecordGoogleLogin(ctx context.Context, name, subject, verifiedEmail string) (Account, error)
	CreateAPIKeyMetadata(ctx context.Context, name string, token Token) error
	DeleteAPIKeyMetadata(ctx context.Context, name, id string) error
}

type accountRecord struct {
	mutex   sync.RWMutex
	account Account
}

// CredentialManager keeps a process-local read-through registry while
// PostgreSQL remains the source of truth and mutation commit point.
type CredentialManager struct {
	mutex          sync.RWMutex
	accounts       map[string]*accountRecord
	googleAccounts map[string]string
	store          Store
	jwtCodec       *JWTCodec
}

// NewCredentialManager loads every durable account and API Key into the
// single-server runtime registry.
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
	for name, account := range accounts {
		if account.Name == "" {
			account.Name = name
		}
		if account.Name != name {
			return nil, fmt.Errorf("credential account key %q does not match name %q", name, account.Name)
		}
		if err := manager.publishLocked(account); err != nil {
			return nil, err
		}
	}
	return manager, nil
}

// Get returns a bearer-secret-free snapshot of a durable account.
func (m *CredentialManager) Get(name string) (Account, error) {
	m.mutex.RLock()
	record, ok := m.accounts[name]
	m.mutex.RUnlock()
	if !ok {
		return Account{}, status.Errorf(codes.NotFound, "account '%s' does not exist", name)
	}
	record.mutex.RLock()
	defer record.mutex.RUnlock()
	return cloneAccount(record.account), nil
}

// List returns snapshots of all durable accounts.
func (m *CredentialManager) List() map[string]Account {
	m.mutex.RLock()
	records := make(map[string]*accountRecord, len(m.accounts))
	for name, record := range m.accounts {
		records[name] = record
	}
	m.mutex.RUnlock()

	accounts := make(map[string]Account, len(records))
	for name, record := range records {
		record.mutex.RLock()
		accounts[name] = cloneAccount(record.account)
		record.mutex.RUnlock()
	}
	return accounts
}

// ResolveOrProvisionGoogleAccount resolves an existing subject or atomically
// provisions a zero-access account. The bootstrap email can claim only the
// fixed, previously unbound administrator row.
func (m *CredentialManager) ResolveOrProvisionGoogleAccount(ctx context.Context, subject, verifiedEmail, administratorEmail string) (Account, bool, error) {
	subject = strings.TrimSpace(subject)
	verifiedEmail = strings.TrimSpace(verifiedEmail)
	administratorEmail = strings.TrimSpace(administratorEmail)
	if subject == "" || verifiedEmail == "" {
		return Account{}, false, status.Error(codes.PermissionDenied, "verified Google subject and email are required")
	}

	m.mutex.RLock()
	name := m.googleAccounts[subject]
	m.mutex.RUnlock()
	if name != "" {
		account, err := m.Get(name)
		return account, false, err
	}

	account, created, err := m.store.ResolveOrProvisionGoogleAccount(ctx, subject, verifiedEmail, administratorEmail)
	if err != nil {
		return Account{}, false, err
	}
	if account.Name == "" || account.GoogleSubject != subject {
		return Account{}, false, fmt.Errorf("durable Google identity resolution returned an inconsistent account")
	}
	if err := m.publish(account); err != nil {
		return Account{}, false, err
	}
	snapshot, err := m.Get(account.Name)
	return snapshot, created, err
}

// RecordGoogleLogin persists mutable audit identity attributes only after a
// login has passed access checks and a local session has been signed.
func (m *CredentialManager) RecordGoogleLogin(ctx context.Context, name, subject, verifiedEmail string) (Account, error) {
	account, err := m.store.RecordGoogleLogin(ctx, name, subject, strings.TrimSpace(verifiedEmail))
	if err != nil {
		return Account{}, err
	}
	if account.Name != name || account.GoogleSubject != subject {
		return Account{}, fmt.Errorf("record Google login returned an inconsistent account")
	}
	if err := m.publish(account); err != nil {
		return Account{}, err
	}
	return m.Get(name)
}

// IssueAPIKey signs a bearer first, durably records its metadata, and exposes
// the bearer only after the database commit succeeds.
func (m *CredentialManager) IssueAPIKey(ctx context.Context, name, id string, expiresIn int64) (string, error) {
	record, err := m.record(name)
	if err != nil {
		return "", err
	}
	if !IsValidAPIKeyDisplayID(id) {
		return "", fmt.Errorf("invalid API Key display ID %q", id)
	}

	record.mutex.Lock()
	defer record.mutex.Unlock()
	if tokenIDIndex(record.account.Tokens, id) >= 0 {
		return "", fmt.Errorf("account already has token with id '%s'", id)
	}
	jti, err := uuid.NewRandom()
	if err != nil {
		return "", fmt.Errorf("generate API Key JTI: %w", err)
	}
	now := time.Now().UTC()
	tokenString, metadata, err := m.jwtCodec.Issue(name, CapabilityAPIKey, jti.String(), expiresIn, now, "")
	if err != nil {
		return "", err
	}
	metadata.ID = id
	if err := m.store.CreateAPIKeyMetadata(ctx, name, metadata); err != nil {
		return "", err
	}
	record.account.Tokens = append(record.account.Tokens, metadata)
	return tokenString, nil
}

// IssueGoogleLoginSession signs a login JWT while the verified Google subject
// still matches the account's permanent binding.
func (m *CredentialManager) IssueGoogleLoginSession(name, verifiedSubject, id string, expiresIn int64) (string, error) {
	record, err := m.record(name)
	if err != nil {
		return "", err
	}
	record.mutex.RLock()
	defer record.mutex.RUnlock()
	if verifiedSubject == "" || record.account.GoogleSubject != verifiedSubject {
		return "", status.Error(codes.PermissionDenied, "Google identity binding changed")
	}
	now := time.Now().UTC()
	tokenString, _, err := m.jwtCodec.Issue(name, CapabilityLogin, id, expiresIn, now, googleIdentityBinding(record.account.GoogleSubject))
	return tokenString, err
}

// DeleteAPIKey durably removes one API Key before updating the runtime registry.
func (m *CredentialManager) DeleteAPIKey(ctx context.Context, name, id string) error {
	record, err := m.record(name)
	if err != nil {
		return err
	}
	record.mutex.Lock()
	defer record.mutex.Unlock()
	index := tokenIDIndex(record.account.Tokens, id)
	if index < 0 {
		return status.Errorf(codes.NotFound, "token with id '%s' does not exist", id)
	}
	if err := m.store.DeleteAPIKeyMetadata(ctx, name, id); err != nil {
		return err
	}
	record.account.Tokens = append(record.account.Tokens[:index], record.account.Tokens[index+1:]...)
	return nil
}

// ValidateCredential checks the immutable login binding or current durable API
// Key membership. Entitlements and Redis revocation are composed by SessionManager.
func (m *CredentialManager) ValidateCredential(name string, capability Capability, jti, tokenIdentityBinding string) error {
	record, err := m.record(name)
	if err != nil {
		return err
	}
	record.mutex.RLock()
	defer record.mutex.RUnlock()
	switch capability {
	case CapabilityLogin:
		if record.account.GoogleSubject == "" || tokenIdentityBinding != googleIdentityBinding(record.account.GoogleSubject) {
			return fmt.Errorf("account Google identity binding has changed")
		}
	case CapabilityAPIKey:
		if tokenJTIIndex(record.account.Tokens, jti) < 0 {
			return fmt.Errorf("account %s does not have an API Key with the supplied JTI", name)
		}
	default:
		return fmt.Errorf("unsupported account capability %q", capability)
	}
	return nil
}

func (m *CredentialManager) record(name string) (*accountRecord, error) {
	m.mutex.RLock()
	record, ok := m.accounts[name]
	m.mutex.RUnlock()
	if !ok {
		return nil, status.Errorf(codes.NotFound, "account '%s' does not exist", name)
	}
	return record, nil
}

func (m *CredentialManager) publish(account Account) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	return m.publishLocked(account)
}

func (m *CredentialManager) publishLocked(account Account) error {
	if account.Name == "" || strings.Contains(account.Name, ":") {
		return fmt.Errorf("credential account name %q is invalid", account.Name)
	}
	if account.Administrator != (account.Name == "admin") {
		return fmt.Errorf("credential account %q has an invalid administrator identity", account.Name)
	}
	if !account.Administrator && (!account.HasGoogleBinding() || strings.TrimSpace(account.VerifiedEmail) == "") {
		return fmt.Errorf("ordinary credential account %q has no verified Google identity", account.Name)
	}
	if account.GoogleSubject != "" {
		if existing := m.googleAccounts[account.GoogleSubject]; existing != "" && existing != account.Name {
			return fmt.Errorf("Google subject is assigned to both %q and %q", existing, account.Name)
		}
	}
	record, exists := m.accounts[account.Name]
	if !exists {
		m.accounts[account.Name] = &accountRecord{account: cloneAccount(account)}
		if account.GoogleSubject != "" {
			m.googleAccounts[account.GoogleSubject] = account.Name
		}
		return nil
	}

	record.mutex.Lock()
	defer record.mutex.Unlock()
	previousSubject := record.account.GoogleSubject
	if previousSubject != "" && account.GoogleSubject != previousSubject {
		return fmt.Errorf("account %q Google identity binding changed unexpectedly", account.Name)
	}
	// Identity updates do not replace the independently managed API Key set.
	account.Tokens = append([]Token(nil), record.account.Tokens...)
	record.account = cloneAccount(account)
	if account.GoogleSubject != "" {
		m.googleAccounts[account.GoogleSubject] = account.Name
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
