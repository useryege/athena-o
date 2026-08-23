package accountcredentials

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	passwordutil "github.com/useryege/athena/util/password"
)

// ErrInvalidCredentials intentionally does not identify which credential part failed.
var ErrInvalidCredentials = errors.New("invalid account credentials")

type accountRecord struct {
	mutex         sync.RWMutex
	passwordHash  string
	passwordMtime *time.Time
	capabilities  []Capability
	tokens        []Token
}

// CredentialManager owns process-local mutable passwords and API key metadata.
// The account registry is fixed after construction, while each account has an
// independent lock so unrelated accounts never block one another.
type CredentialManager struct {
	accounts map[string]*accountRecord
	jwtCodec *JWTCodec
}

// NewCredentialManager copies the startup catalog into mutable per-account records.
func NewCredentialManager(catalog *Catalog, jwtCodec *JWTCodec) *CredentialManager {
	manager := &CredentialManager{
		accounts: make(map[string]*accountRecord, len(catalog.accounts)),
		jwtCodec: jwtCodec,
	}
	for name, rawSeed := range catalog.accounts {
		seed := cloneSeed(rawSeed)
		manager.accounts[name] = &accountRecord{
			passwordHash:  seed.passwordHash,
			passwordMtime: seed.passwordMtime,
			capabilities:  seed.capabilities,
			tokens:        seed.tokens,
		}
	}
	return manager
}

// Get returns a secret-free snapshot of a configured account.
func (m *CredentialManager) Get(name string) (Account, error) {
	record, ok := m.accounts[name]
	if !ok {
		return Account{}, status.Errorf(codes.NotFound, "account '%s' does not exist", name)
	}
	record.mutex.RLock()
	defer record.mutex.RUnlock()
	return record.snapshot(), nil
}

// List returns secret-free snapshots of all configured accounts.
func (m *CredentialManager) List() map[string]Account {
	accounts := make(map[string]Account, len(m.accounts))
	for name, record := range m.accounts {
		record.mutex.RLock()
		accounts[name] = record.snapshot()
		record.mutex.RUnlock()
	}
	return accounts
}

// VerifyPassword verifies a password without exposing the stored hash.
func (m *CredentialManager) VerifyPassword(name, password string) (PasswordVerification, error) {
	record, ok := m.accounts[name]
	if !ok {
		_, _ = passwordutil.HashPassword("for_consistent_response_time")
		return PasswordVerification{}, ErrInvalidCredentials
	}
	record.mutex.RLock()
	defer record.mutex.RUnlock()
	valid, _ := passwordutil.VerifyPassword(password, record.passwordHash)
	if !valid {
		return PasswordVerification{}, ErrInvalidCredentials
	}
	return PasswordVerification{account: name, passwordHash: record.passwordHash}, nil
}

// ChangePassword atomically verifies the current password and publishes its replacement.
func (m *CredentialManager) ChangePassword(name, currentPassword, passwordHash string) error {
	record, ok := m.accounts[name]
	if !ok {
		_, _ = passwordutil.HashPassword("for_consistent_response_time")
		return ErrInvalidCredentials
	}
	record.mutex.Lock()
	defer record.mutex.Unlock()
	valid, _ := passwordutil.VerifyPassword(currentPassword, record.passwordHash)
	if !valid {
		return ErrInvalidCredentials
	}
	record.passwordHash = passwordHash
	record.passwordMtime = timePointer(time.Now().UTC())
	return nil
}

// ResetPassword atomically replaces a password after caller-side administrator authorization.
func (m *CredentialManager) ResetPassword(name, passwordHash string) error {
	record, ok := m.accounts[name]
	if !ok {
		return status.Errorf(codes.NotFound, "account '%s' does not exist", name)
	}
	record.mutex.Lock()
	defer record.mutex.Unlock()
	record.passwordHash = passwordHash
	record.passwordMtime = timePointer(time.Now().UTC())
	return nil
}

// IssueAPIKey atomically validates, signs, and records an API key.
func (m *CredentialManager) IssueAPIKey(name, id string, expiresIn int64) (string, error) {
	record, ok := m.accounts[name]
	if !ok {
		return "", status.Errorf(codes.NotFound, "account '%s' does not exist", name)
	}
	record.mutex.Lock()
	defer record.mutex.Unlock()
	if tokenIndex(record.tokens, id) >= 0 {
		return "", fmt.Errorf("account already has token with id '%s'", id)
	}
	if !hasCapability(record.capabilities, CapabilityAPIKey) {
		return "", fmt.Errorf("account '%s' does not have %s capability", name, CapabilityAPIKey)
	}
	now := time.Now().UTC()
	tokenString, metadata, err := m.jwtCodec.Issue(name, CapabilityAPIKey, id, expiresIn, now, passwordEpoch(record.passwordMtime))
	if err != nil {
		return "", err
	}
	record.tokens = append(record.tokens, metadata)
	return tokenString, nil
}

// IssueLoginSession signs a login JWT only while the password version proven
// by verification is still current. The read lock orders issuance against a
// concurrent password replacement without calling external components.
func (m *CredentialManager) IssueLoginSession(verification PasswordVerification, id string, expiresIn int64) (string, error) {
	record, ok := m.accounts[verification.account]
	if !ok {
		return "", ErrInvalidCredentials
	}
	record.mutex.RLock()
	defer record.mutex.RUnlock()
	if verification.passwordHash == "" || record.passwordHash != verification.passwordHash {
		return "", ErrInvalidCredentials
	}
	if !hasCapability(record.capabilities, CapabilityLogin) {
		return "", fmt.Errorf("account '%s' does not have %s capability", verification.account, CapabilityLogin)
	}
	now := time.Now().UTC()
	tokenString, _, err := m.jwtCodec.Issue(verification.account, CapabilityLogin, id, expiresIn, now, passwordEpoch(record.passwordMtime))
	return tokenString, err
}

// DeleteAPIKey removes one process-local API key by identifier.
func (m *CredentialManager) DeleteAPIKey(name, id string) error {
	record, ok := m.accounts[name]
	if !ok {
		return status.Errorf(codes.NotFound, "account '%s' does not exist", name)
	}
	record.mutex.Lock()
	defer record.mutex.Unlock()
	index := tokenIndex(record.tokens, id)
	if index < 0 {
		return status.Errorf(codes.NotFound, "token with id '%s' does not exist", id)
	}
	record.tokens = append(record.tokens[:index], record.tokens[index+1:]...)
	return nil
}

// ValidateCredential checks current capability, API key membership, and password epoch.
func (m *CredentialManager) ValidateCredential(name string, capability Capability, id, tokenPasswordEpoch string) error {
	record, ok := m.accounts[name]
	if !ok {
		return status.Errorf(codes.NotFound, "account '%s' does not exist", name)
	}
	record.mutex.RLock()
	defer record.mutex.RUnlock()
	if !hasCapability(record.capabilities, capability) {
		return fmt.Errorf("account %s does not have '%s' capability", name, capability)
	}
	if capability == CapabilityAPIKey && tokenIndex(record.tokens, id) < 0 {
		return fmt.Errorf("account %s does not have token with id %s", name, id)
	}
	if tokenPasswordEpoch != passwordEpoch(record.passwordMtime) {
		return errors.New("account password has changed since token issued")
	}
	return nil
}

func passwordEpoch(modifiedAt *time.Time) string {
	if modifiedAt == nil {
		return ""
	}
	return modifiedAt.UTC().Format(time.RFC3339Nano)
}

func (record *accountRecord) snapshot() Account {
	return cloneAccount(Account{
		PasswordMtime: record.passwordMtime,
		Capabilities:  record.capabilities,
		Tokens:        record.tokens,
	})
}

func tokenIndex(tokens []Token, id string) int {
	for index, token := range tokens {
		if token.ID == id {
			return index
		}
	}
	return -1
}

func hasCapability(capabilities []Capability, capability Capability) bool {
	for _, candidate := range capabilities {
		if candidate == capability {
			return true
		}
	}
	return false
}

func timePointer(value time.Time) *time.Time {
	copy := value
	return &copy
}
