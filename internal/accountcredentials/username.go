package accountcredentials

import (
	"errors"
	"regexp"
	"strings"
)

const (
	MinUsernameLength = 3
	MaxUsernameLength = 42
)

var (
	// ErrUsernameInvalid intentionally combines format, reserved-name, and
	// safety-list failures so availability endpoints do not reveal which rule a
	// rejected candidate matched.
	ErrUsernameInvalid = errors.New("username is invalid")

	usernamePattern       = regexp.MustCompile(`^[A-Za-z0-9.-]+$`)
	usernameAlphanumeric  = regexp.MustCompile(`[A-Za-z0-9]`)
	walletAddressUsername = regexp.MustCompile(`(?i)^0x[0-9a-f]{40}$`)
)

// usernameBlocklist is matched after ASCII case folding and removal of dots
// and hyphens. This makes punctuation variants such as "a.d-m-i-n" resolve to
// the same checked-in safety key without changing the username that is stored.
var usernameBlocklist = map[string]struct{}{
	// Athena and system identities.
	"admin":          {},
	"administrator":  {},
	"anonymous":      {},
	"api":            {},
	"athena":         {},
	"athenaadmin":    {},
	"athenaofficial": {},
	"athenasecurity": {},
	"athenasupport":  {},
	"deleted":        {},
	"help":           {},
	"helpdesk":       {},
	"localadmin":     {},
	"mod":            {},
	"moderator":      {},
	"notifications":  {},
	"null":           {},
	"official":       {},
	"operator":       {},
	"owner":          {},
	"profitsharing":  {},
	"root":           {},
	"security":       {},
	"staff":          {},
	"status":         {},
	"superuser":      {},
	"support":        {},
	"system":         {},
	"systemadmin":    {},
	"undefined":      {},
	"wallet":         {},
	"webmaster":      {},

	// High-risk third-party impersonation names.
	"binance":            {},
	"chatgpt":            {},
	"coinbase":           {},
	"discord":            {},
	"google":             {},
	"googlesupport":      {},
	"meta":               {},
	"metamask":           {},
	"openai":             {},
	"openaisupport":      {},
	"polymarket":         {},
	"polymarketadmin":    {},
	"polymarketofficial": {},
	"polymarketsupport":  {},
	"telegram":           {},

	// Obvious abuse and slurs.
	"asshole":  {},
	"bitch":    {},
	"bullshit": {},
	"cock":     {},
	"cunt":     {},
	"dick":     {},
	"faggot":   {},
	"fuck":     {},
	"fucker":   {},
	"fucking":  {},
	"nazi":     {},
	"nigga":    {},
	"nigger":   {},
	"pussy":    {},
	"rape":     {},
	"rapist":   {},
	"retard":   {},
	"shit":     {},
	"slut":     {},
	"whore":    {},
}

// ValidateUsername validates a permanent public Athena username without
// trimming, normalizing, or changing its stored casing. The reserved lowercase
// username "admin" is available only to the administrator registration path;
// case or punctuation variants remain invalid for every caller.
func ValidateUsername(username string, administrator bool) error {
	if len(username) < MinUsernameLength || len(username) > MaxUsernameLength ||
		!usernamePattern.MatchString(username) ||
		!usernameAlphanumeric.MatchString(username) ||
		walletAddressUsername.MatchString(username) {
		return ErrUsernameInvalid
	}

	blocklistKey := strings.NewReplacer(".", "", "-", "").Replace(strings.ToLower(username))
	if _, blocked := usernameBlocklist[blocklistKey]; !blocked {
		return nil
	}
	if administrator && username == "admin" {
		return nil
	}
	return ErrUsernameInvalid
}
