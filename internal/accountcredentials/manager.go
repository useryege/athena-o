package accountcredentials

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type accountRecord struct {
	mutex         sync.RWMutex
	googleSubject string
	capabilities  []Capability
	tokens        []Token
}

// CredentialManager owns the fixed Google identity registry and process-local
// API Key metadata. Each account has an independent mutation lock.
type CredentialManager struct {
	accounts       map[string]*accountRecord
	googleAccounts map[string]string
	jwtCodec       *JWTCodec
}

// NewCredentialManager copies the startup catalog into per-account records.
func NewCredentialManager(catalog *Catalog, jwtCodec *JWTCodec) *CredentialManager {
	manager := &CredentialManager{
		accounts:       make(map[string]*accountRecord, len(catalog.accounts)),
		googleAccounts: make(map[string]string, len(catalog.accounts)),
		jwtCodec:       jwtCodec,
	}
	for name, rawSeed := range catalog.accounts {
		seed := cloneSeed(rawSeed)
		manager.accounts[name] = &accountRecord{
			googleSubject: seed.googleSubject,
			capabilities:  seed.capabilities,
			tokens:        seed.tokens,
		}
		if seed.googleSubject != "" {
			manager.googleAccounts[seed.googleSubject] = name
		}
	}
	return manager
}

// Get returns a bearer-secret-free snapshot of a configured account.
func (m *CredentialManager) Get(name string) (Account, error) {
	record, ok := m.accounts[name]
	if !ok {
		return Account{}, status.Errorf(codes.NotFound, "account '%s' does not exist", name)
	}
	record.mutex.RLock()
	defer record.mutex.RUnlock()
	return record.snapshot(), nil
}

// List returns snapshots of all configured accounts.
func (m *CredentialManager) List() map[string]Account {
	accounts := make(map[string]Account, len(m.accounts))
	for name, record := range m.accounts {
		record.mutex.RLock()
		accounts[name] = record.snapshot()
		record.mutex.RUnlock()
	}
	return accounts
}

// ResolveGoogleSubject maps one verified Google subject to its Athena account.
func (m *CredentialManager) ResolveGoogleSubject(subject string) (string, error) {
	name := m.googleAccounts[subject]
	if name == "" {
		return "", status.Error(codes.PermissionDenied, "Google identity is not authorized")
	}
	return name, nil
}

// IssueAPIKey atomically validates, signs, and records an API Key.
func (m *CredentialManager) IssueAPIKey(name, id string, expiresIn int64) (string, error) {
	record, ok := m.accounts[name]
	if !ok {
		return "", status.Errorf(codes.NotFound, "account '%s' does not exist", name)
	}
	if !IsValidAPIKeyDisplayID(id) {
		return "", fmt.Errorf("invalid API Key display ID %q", id)
	}
	record.mutex.Lock()
	defer record.mutex.Unlock()
	if tokenIDIndex(record.tokens, id) >= 0 {
		return "", fmt.Errorf("account already has token with id '%s'", id)
	}
	if !hasCapability(record.capabilities, CapabilityAPIKey) {
		return "", fmt.Errorf("account '%s' does not have %s capability", name, CapabilityAPIKey)
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
	record.tokens = append(record.tokens, metadata)
	return tokenString, nil
}

// IssueGoogleLoginSession signs a login JWT only while the verified Google
// subject still matches the account's current startup binding.
func (m *CredentialManager) IssueGoogleLoginSession(name, verifiedSubject, id string, expiresIn int64) (string, error) {
	record, ok := m.accounts[name]
	if !ok {
		return "", status.Errorf(codes.NotFound, "account '%s' does not exist", name)
	}
	record.mutex.RLock()
	defer record.mutex.RUnlock()
	if verifiedSubject == "" || record.googleSubject != verifiedSubject {
		return "", status.Error(codes.PermissionDenied, "Google identity binding changed")
	}
	if !hasCapability(record.capabilities, CapabilityLogin) {
		return "", fmt.Errorf("account '%s' does not have %s capability", name, CapabilityLogin)
	}
	now := time.Now().UTC()
	tokenString, _, err := m.jwtCodec.Issue(name, CapabilityLogin, id, expiresIn, now, googleIdentityBinding(record.googleSubject))
	return tokenString, err
}

// DeleteAPIKey removes one process-local API Key by identifier.
func (m *CredentialManager) DeleteAPIKey(name, id string) error {
	record, ok := m.accounts[name]
	if !ok {
		return status.Errorf(codes.NotFound, "account '%s' does not exist", name)
	}
	record.mutex.Lock()
	defer record.mutex.Unlock()
	index := tokenIDIndex(record.tokens, id)
	if index < 0 {
		return status.Errorf(codes.NotFound, "token with id '%s' does not exist", id)
	}
	record.tokens = append(record.tokens[:index], record.tokens[index+1:]...)
	return nil
}

// ValidateCredential checks current capability, identity binding or API Key
// membership. Access state and Redis revocation are composed by SessionManager.
func (m *CredentialManager) ValidateCredential(name string, capability Capability, jti, tokenIdentityBinding string) error {
	record, ok := m.accounts[name]
	if !ok {
		return status.Errorf(codes.NotFound, "account '%s' does not exist", name)
	}
	record.mutex.RLock()
	defer record.mutex.RUnlock()
	if !hasCapability(record.capabilities, capability) {
		return fmt.Errorf("account %s does not have '%s' capability", name, capability)
	}
	switch capability {
	case CapabilityLogin:
		if record.googleSubject == "" || tokenIdentityBinding != googleIdentityBinding(record.googleSubject) {
			return fmt.Errorf("account Google identity binding has changed")
		}
	case CapabilityAPIKey:
		if tokenJTIIndex(record.tokens, jti) < 0 {
			return fmt.Errorf("account %s does not have an API Key with the supplied JTI", name)
		}
	default:
		return fmt.Errorf("unsupported account capability %q", capability)
	}
	return nil
}

func googleIdentityBinding(subject string) string {
	digest := sha256.Sum256([]byte("google\x00" + subject))
	return hex.EncodeToString(digest[:])
}

func (record *accountRecord) snapshot() Account {
	return cloneAccount(Account{
		GoogleSubject: record.googleSubject,
		Capabilities:  record.capabilities,
		Tokens:        record.tokens,
	})
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

func hasCapability(capabilities []Capability, capability Capability) bool {
	for _, candidate := range capabilities {
		if candidate == capability {
			return true
		}
	}
	return false
}
