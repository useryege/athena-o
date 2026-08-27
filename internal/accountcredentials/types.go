package accountcredentials

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"
	"unicode"

	"github.com/google/uuid"
	"github.com/mr-tron/base58/base58"
)

var apiKeyDisplayIDPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,63}$`)

// Capability identifies a credential purpose allowed for an account.
type Capability string

const (
	CapabilityLogin       Capability = "login"
	CapabilityAPIKey      Capability = "apiKey"
	CapabilityDevelopment Capability = "development"
)

// AuthenticatedCredential is the server-side projection of the credential
// that authenticated the current request. It deliberately keeps credential
// capability and binding metadata out of public claims while making security-
// sensitive handlers independent from transport headers.
type AuthenticatedCredential struct {
	AccountID       string
	Capability      Capability
	JTI             string
	IdentityBinding string
	AccessRevision  uint64
}

// IsInteractiveLogin reports whether this credential may cross a boundary
// that explicitly excludes API Keys. Development credentials exist only while
// the API Server is bound to loopback with authentication disabled.
func (c AuthenticatedCredential) IsInteractiveLogin() bool {
	return c.Capability == CapabilityLogin || c.Capability == CapabilityDevelopment
}

// Token is bearer-secret-free durable metadata for an issued API Key.
type Token struct {
	ID        string `json:"id"`
	JTI       string `json:"jti"`
	IssuedAt  int64  `json:"iat"`
	ExpiresAt int64  `json:"exp,omitempty"`
}

// IdentityProvider identifies the external or isolated development identity
// bound permanently to an Athena account.
type IdentityProvider string

const (
	IdentityProviderGoogle       IdentityProvider = "google"
	IdentityProviderSolanaWallet IdentityProvider = "solana_wallet"
	IdentityProviderDevelopment  IdentityProvider = "development"
)

// Account is the internal, bearer-secret-free projection of one durable
// Athena account. IdentitySubject is authentication state and must not be
// exposed through public account or session APIs.
type Account struct {
	ID               string
	Username         string
	IdentityProvider IdentityProvider
	IdentitySubject  string
	VerifiedEmail    string
	Administrator    bool
	CreatedAt        time.Time
	LastLoginAt      time.Time
	Tokens           []Token
}

// CanonicalAccountID validates and canonicalizes a public or token account ID.
func CanonicalAccountID(value string) (string, error) {
	parsed, err := uuid.Parse(strings.TrimSpace(value))
	if err != nil {
		return "", err
	}
	if parsed == uuid.Nil {
		return "", errors.New("account ID must not be the zero UUID")
	}
	return parsed.String(), nil
}

// NormalizeIdentitySubject validates and canonicalizes an external login key.
func NormalizeIdentitySubject(provider IdentityProvider, subject string) (string, error) {
	subject = strings.TrimSpace(subject)
	if subject == "" || strings.IndexFunc(subject, unicode.IsControl) >= 0 {
		return "", fmt.Errorf("identity subject is required")
	}

	switch provider {
	case IdentityProviderGoogle:
	case IdentityProviderSolanaWallet:
		if len(subject) < 32 || len(subject) > 44 {
			return "", fmt.Errorf("Solana wallet identity must be a canonical public key")
		}
		decoded, err := base58.Decode(subject)
		if err != nil || len(decoded) != 32 || base58.Encode(decoded) != subject {
			return "", fmt.Errorf("Solana wallet identity must be a canonical 32-byte public key")
		}
	default:
		return "", fmt.Errorf("identity provider %q is not an external login provider", provider)
	}
	return subject, nil
}

// NormalizeExternalIdentity validates and canonicalizes a complete login
// identity before it is written to PostgreSQL.
func NormalizeExternalIdentity(provider IdentityProvider, subject, verifiedEmail string, administrator bool) (string, string, error) {
	subject, err := NormalizeIdentitySubject(provider, subject)
	if err != nil {
		return "", "", err
	}
	verifiedEmail = strings.TrimSpace(verifiedEmail)
	switch provider {
	case IdentityProviderGoogle:
		if verifiedEmail == "" || strings.IndexFunc(verifiedEmail, unicode.IsControl) >= 0 {
			return "", "", fmt.Errorf("verified Google email is required")
		}
	case IdentityProviderSolanaWallet:
		if administrator {
			return "", "", fmt.Errorf("Solana wallet identities cannot be administrators")
		}
		if verifiedEmail != "" {
			return "", "", fmt.Errorf("Solana wallet identities cannot have a verified email")
		}
	}
	return subject, verifiedEmail, nil
}

// HasExternalIdentity reports whether the account can resolve a supported
// external login identity.
func (a Account) HasExternalIdentity() bool {
	if a.IdentityProvider != IdentityProviderGoogle && a.IdentityProvider != IdentityProviderSolanaWallet {
		return false
	}
	_, _, err := NormalizeExternalIdentity(a.IdentityProvider, a.IdentitySubject, a.VerifiedEmail, a.Administrator)
	return err == nil
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
