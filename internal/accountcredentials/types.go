package accountcredentials

import (
	"regexp"
	"strings"
	"time"
)

var apiKeyDisplayIDPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,63}$`)

// Capability identifies a credential purpose allowed for an account.
type Capability string

const (
	CapabilityLogin  Capability = "login"
	CapabilityAPIKey Capability = "apiKey"
)

// Token is bearer-secret-free durable metadata for an issued API Key.
type Token struct {
	ID        string `json:"id"`
	JTI       string `json:"jti"`
	IssuedAt  int64  `json:"iat"`
	ExpiresAt int64  `json:"exp,omitempty"`
}

const IdentityProviderGoogle = "google"

// Account is the internal, bearer-secret-free projection of one durable
// Athena account. GoogleSubject is never projected through the public API.
type Account struct {
	Name          string
	GoogleSubject string
	VerifiedEmail string
	Administrator bool
	CreatedAt     time.Time
	LastLoginAt   time.Time
	Tokens        []Token
}

// HasGoogleBinding reports whether the account can resolve a Google identity.
func (a Account) HasGoogleBinding() bool {
	return strings.TrimSpace(a.GoogleSubject) != ""
}

// IsValidAPIKeyDisplayID reports whether id is a valid user-visible API Key
// identifier for both configured metadata and runtime issuance.
func IsValidAPIKeyDisplayID(id string) bool {
	return apiKeyDisplayIDPattern.MatchString(id)
}

func cloneAccount(account Account) Account {
	account.Tokens = append([]Token(nil), account.Tokens...)
	return account
}
