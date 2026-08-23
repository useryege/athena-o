package accountcredentials

import (
	"strings"
	"time"
)

// Capability identifies a credential purpose allowed for an account.
type Capability string

const (
	CapabilityLogin  Capability = "login"
	CapabilityAPIKey Capability = "apiKey"
)

// Token is the process-local metadata for an issued API key.
type Token struct {
	ID        string `json:"id"`
	IssuedAt  int64  `json:"iat"`
	ExpiresAt int64  `json:"exp,omitempty"`
}

// Account is the public, secret-free projection of an account credential.
type Account struct {
	PasswordMtime *time.Time
	Capabilities  []Capability
	Tokens        []Token
}

// PasswordVerification is an opaque, short-lived proof that a password matched
// one account credential version. Its fields remain private so callers cannot
// inspect the password hash used to establish the proof.
type PasswordVerification struct {
	account      string
	passwordHash string
}

// FormatPasswordMtime returns the password modification time in RFC3339 form.
func (a Account) FormatPasswordMtime() string {
	if a.PasswordMtime == nil {
		return ""
	}
	return a.PasswordMtime.Format(time.RFC3339)
}

// FormatCapabilities returns a comma-separated capability list.
func (a Account) FormatCapabilities() string {
	items := make([]string, 0, len(a.Capabilities))
	for _, capability := range a.Capabilities {
		items = append(items, string(capability))
	}
	return strings.Join(items, ",")
}

// HasCapability reports whether the account supports capability.
func (a Account) HasCapability(capability Capability) bool {
	for _, configured := range a.Capabilities {
		if configured == capability {
			return true
		}
	}
	return false
}

type accountSeed struct {
	passwordHash  string
	passwordMtime *time.Time
	capabilities  []Capability
	tokens        []Token
}

func cloneAccount(account Account) Account {
	account.Capabilities = append([]Capability(nil), account.Capabilities...)
	account.Tokens = append([]Token(nil), account.Tokens...)
	if account.PasswordMtime != nil {
		modifiedAt := *account.PasswordMtime
		account.PasswordMtime = &modifiedAt
	}
	return account
}

func cloneSeed(seed accountSeed) accountSeed {
	seed.capabilities = append([]Capability(nil), seed.capabilities...)
	seed.tokens = append([]Token(nil), seed.tokens...)
	if seed.passwordMtime != nil {
		modifiedAt := *seed.passwordMtime
		seed.passwordMtime = &modifiedAt
	}
	return seed
}
