package accountcredentials

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"

	log "github.com/sirupsen/logrus"

	"github.com/useryege/athena/common"
	"github.com/useryege/athena/util"
)

// Catalog is the immutable startup result for process-local credentials and
// environment login baselines. Its secrets are copied into runtime consumers
// and are never exposed through accessors.
type Catalog struct {
	accounts      map[string]accountSeed
	loginDefaults map[string]bool
	signingKey    []byte
}

// LoadCatalog reads the fixed account registry and JWT signing key once.
func LoadCatalog() (*Catalog, error) {
	accounts, loginDefaults, err := parseAccounts()
	if err != nil {
		return nil, err
	}

	signingKeyValue, err := envOrFile("ATHENA_JWT_SECRET")
	if err != nil {
		return nil, err
	}
	signingKey := []byte(signingKeyValue)
	if len(signingKey) == 0 {
		signingKey, err = util.MakeSignature(32)
		if err != nil {
			return nil, fmt.Errorf("error setting JWT signature: %w", err)
		}
		log.Warn("Generated transient JWT secret because ATHENA_JWT_SECRET is not set; existing sessions and API Keys will be invalid after restart")
	}
	if len(signingKey) < minimumJWTSigningKeyBytes {
		return nil, fmt.Errorf("ATHENA_JWT_SECRET must contain at least %d bytes", minimumJWTSigningKeyBytes)
	}

	logLoadedAccounts(accounts)
	return &Catalog{
		accounts:      accounts,
		loginDefaults: loginDefaults,
		signingKey:    append([]byte(nil), signingKey...),
	}, nil
}

// LoginDefaults returns a copy of the environment login baseline.
func (c *Catalog) LoginDefaults() map[string]bool {
	defaults := make(map[string]bool, len(c.loginDefaults))
	for name, enabled := range c.loginDefaults {
		defaults[name] = enabled
	}
	return defaults
}

// ValidateGoogleBindings validates the fail-closed Google login identity map.
// It is intentionally invoked only when authentication is enabled.
func (c *Catalog) ValidateGoogleBindings() error {
	if c == nil {
		return fmt.Errorf("account credential catalog is nil")
	}
	seen := make(map[string]string, len(c.accounts))
	for name, seed := range c.accounts {
		subject := strings.TrimSpace(seed.googleSubject)
		if hasCapability(seed.capabilities, CapabilityLogin) && subject == "" {
			return fmt.Errorf("Google subject is required for login-capable account %q", name)
		}
		if subject == "" {
			continue
		}
		if existing, ok := seen[subject]; ok {
			return fmt.Errorf("Google subject is bound to multiple Athena accounts: %q and %q", existing, name)
		}
		seen[subject] = name
	}
	admin, ok := c.accounts[common.AthenaAdminUsername]
	if !ok || !hasCapability(admin.capabilities, CapabilityLogin) || strings.TrimSpace(admin.googleSubject) == "" {
		return fmt.Errorf("admin must have a unique Google subject and login capability")
	}
	return nil
}

func parseAccounts() (map[string]accountSeed, map[string]bool, error) {
	admin := parseAdminAccount()
	var err error
	accounts := map[string]accountSeed{common.AthenaAdminUsername: admin}
	loginDefaults := map[string]bool{common.AthenaAdminUsername: true}

	for _, item := range os.Environ() {
		key, value, ok := strings.Cut(item, "=")
		if !ok {
			continue
		}
		name, suffix, ok := accountFromEnvKey(key)
		if !ok || name == "" || name == common.AthenaAdminUsername {
			continue
		}
		seed := accounts[name]
		switch suffix {
		case "CAPABILITIES":
			seed.capabilities = parseCapabilities(value, key)
		case "ENABLED":
			loginDefaults[name], err = strconv.ParseBool(value)
			if err != nil {
				return nil, nil, fmt.Errorf("invalid %s: %w", key, err)
			}
		case "GOOGLE_SUB":
			seed.googleSubject = strings.TrimSpace(value)
		case "TOKENS":
			seed.tokens = []Token{}
			if value != "" {
				if err := json.Unmarshal([]byte(value), &seed.tokens); err != nil {
					return nil, nil, fmt.Errorf("invalid API Key metadata in %s: %w", key, err)
				}
			}
		}
		accounts[name] = seed
	}

	for name := range accounts {
		if _, exists := loginDefaults[name]; !exists {
			loginDefaults[name] = false
		}
	}
	loginDefaults[common.AthenaAdminUsername] = true
	if err := validateTokenMetadata(accounts); err != nil {
		return nil, nil, err
	}
	return accounts, loginDefaults, nil
}

func parseAdminAccount() accountSeed {
	return accountSeed{
		googleSubject: strings.TrimSpace(os.Getenv("ATHENA_ADMIN_GOOGLE_SUB")),
		capabilities:  []Capability{CapabilityLogin},
		tokens:        []Token{},
	}
}

func validateTokenMetadata(accounts map[string]accountSeed) error {
	names := make([]string, 0, len(accounts))
	for name := range accounts {
		names = append(names, name)
	}
	sort.Strings(names)
	globalJTIs := make(map[string]string)
	for _, name := range names {
		if len(accounts[name].tokens) > 0 && !hasCapability(accounts[name].capabilities, CapabilityAPIKey) {
			return fmt.Errorf("account %q has API Key metadata without the apiKey capability", name)
		}
		ids := make(map[string]bool)
		for index, token := range accounts[name].tokens {
			if !IsValidAPIKeyDisplayID(token.ID) {
				return fmt.Errorf("API Key metadata %s[%d] has an invalid display ID", name, index)
			}
			if token.JTI == "" || strings.TrimSpace(token.JTI) != token.JTI {
				return fmt.Errorf("API Key metadata %s[%d] has an invalid JTI", name, index)
			}
			if token.IssuedAt <= 0 || (token.ExpiresAt != 0 && token.ExpiresAt <= token.IssuedAt) {
				return fmt.Errorf("API Key metadata %s[%d] has invalid time values", name, index)
			}
			if ids[token.ID] {
				return fmt.Errorf("API Key display ID %q is duplicated for account %q", token.ID, name)
			}
			ids[token.ID] = true
			if owner, exists := globalJTIs[token.JTI]; exists {
				return fmt.Errorf("API Key JTI is duplicated across accounts %q and %q", owner, name)
			}
			globalJTIs[token.JTI] = name
		}
	}
	return nil
}

func parseCapabilities(value, key string) []Capability {
	capabilities := []Capability{}
	seen := map[Capability]bool{}
	for _, value := range strings.Split(value, ",") {
		capability := Capability(strings.TrimSpace(value))
		switch capability {
		case CapabilityLogin, CapabilityAPIKey:
			if !seen[capability] {
				capabilities = append(capabilities, capability)
				seen[capability] = true
			}
		case "":
		default:
			log.Warnf("not supported account capability '%s' in %s", capability, key)
		}
	}
	return capabilities
}

func accountFromEnvKey(key string) (string, string, bool) {
	const prefix = "ATHENA_ACCOUNT_"
	if !strings.HasPrefix(key, prefix) {
		return "", "", false
	}
	raw := strings.TrimPrefix(key, prefix)
	for _, suffix := range []string{"_CAPABILITIES", "_ENABLED", "_GOOGLE_SUB", "_TOKENS"} {
		if strings.HasSuffix(raw, suffix) {
			return strings.TrimSuffix(raw, suffix), strings.TrimPrefix(suffix, "_"), true
		}
	}
	return "", "", false
}

func envOrFile(name string) (string, error) {
	if value := os.Getenv(name); value != "" {
		return value, nil
	}
	if path := os.Getenv(name + "_FILE"); path != "" {
		data, err := os.ReadFile(path)
		if err != nil {
			return "", fmt.Errorf("failed reading %s_FILE: %w", name, err)
		}
		return strings.TrimSpace(string(data)), nil
	}
	return "", nil
}

func logLoadedAccounts(accounts map[string]accountSeed) {
	names := make([]string, 0, len(accounts))
	for name := range accounts {
		names = append(names, name)
	}
	sort.Strings(names)
	envVars := 0
	for _, item := range os.Environ() {
		key, _, ok := strings.Cut(item, "=")
		if ok && strings.HasPrefix(key, "ATHENA_ACCOUNT_") {
			envVars++
		}
	}
	log.Infof("Loaded Athena accounts from environment: count=%d accounts=%v", len(names), names)
	log.Infof("Account source hints: ATHENA_ACCOUNT_* variables=%d", envVars)
}
