package accountcredentials

import (
	"regexp"
	"strings"
)

var apiKeyDisplayIDPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,63}$`)

// Capability identifies a credential purpose allowed for an account.
type Capability string

const (
	CapabilityLogin  Capability = "login"
	CapabilityAPIKey Capability = "apiKey"
)

// Token is the process-local metadata for an issued API Key.
type Token struct {
	ID        string `json:"id"`
	JTI       string `json:"jti"`
	IssuedAt  int64  `json:"iat"`
	ExpiresAt int64  `json:"exp,omitempty"`
}

// Account is the internal, bearer-secret-free projection of an account credential.
// GoogleSubject is never projected through the public Account API.
type Account struct {
	GoogleSubject string
	Capabilities  []Capability
	Tokens        []Token
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
	return hasCapability(a.Capabilities, capability)
}

// HasGoogleBinding reports whether the account can resolve a Google identity.
func (a Account) HasGoogleBinding() bool {
	return strings.TrimSpace(a.GoogleSubject) != ""
}

type accountSeed struct {
	googleSubject string
	capabilities  []Capability
	tokens        []Token
}

// IsValidAPIKeyDisplayID reports whether id is a valid user-visible API Key
// identifier for both configured metadata and runtime issuance.
func IsValidAPIKeyDisplayID(id string) bool {
	return apiKeyDisplayIDPattern.MatchString(id)
}

func cloneAccount(account Account) Account {
	account.Capabilities = append([]Capability(nil), account.Capabilities...)
	account.Tokens = append([]Token(nil), account.Tokens...)
	return account
}

func cloneSeed(seed accountSeed) accountSeed {
	seed.capabilities = append([]Capability(nil), seed.capabilities...)
	seed.tokens = append([]Token(nil), seed.tokens...)
	return seed
}
