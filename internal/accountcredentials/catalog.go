package accountcredentials

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/useryege/athena/common"
	"github.com/useryege/athena/util"
	"github.com/useryege/athena/util/password"
)

const initialPasswordLength = 16

// Catalog is the immutable startup result for process-local credentials and
// environment login baselines. Its secrets are copied into runtime consumers
// and are never exposed through accessors.
type Catalog struct {
	accounts      map[string]accountSeed
	loginDefaults map[string]bool
	signingKey    []byte
}

// LoadCatalog reads the environment account registry and JWT signing key once.
func LoadCatalog() (*Catalog, error) {
	secrets := loadSecretsFromEnv()
	accounts, loginDefaults, err := parseAccounts(secrets)
	if err != nil {
		return nil, err
	}
	if err := initializeAdmin(accounts); err != nil {
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
		log.Warnf("Generated transient JWT secret because ATHENA_JWT_SECRET is not set, existing sessions will be invalid after restart: %s", string(signingKey))
	}

	logLoadedAccounts(accounts, secrets)
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

func initializeAdmin(accounts map[string]accountSeed) error {
	admin := accounts[common.AthenaAdminUsername]
	if admin.passwordHash == "" {
		initialPasswordBytes, err := util.MakeSignature(initialPasswordLength)
		if err != nil {
			return err
		}
		initialPassword := base64.RawURLEncoding.EncodeToString(initialPasswordBytes)
		admin.passwordHash, err = password.HashPassword(initialPassword)
		if err != nil {
			return err
		}
		now := time.Now().UTC()
		admin.passwordMtime = &now
		accounts[common.AthenaAdminUsername] = admin
		log.Warnf("Generated transient admin password because ATHENA_ADMIN_PASSWORD_HASH is not set. It will not persist across restarts: %s", initialPassword)
	} else if admin.passwordMtime == nil || admin.passwordMtime.IsZero() {
		now := time.Now().UTC()
		admin.passwordMtime = &now
		accounts[common.AthenaAdminUsername] = admin
	}
	return nil
}

func parseAccounts(secrets map[string]string) (map[string]accountSeed, map[string]bool, error) {
	admin, err := parseAdminAccount()
	if err != nil {
		return nil, nil, err
	}
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
				return nil, nil, err
			}
		case "PASSWORD_HASH":
			seed.passwordHash = value
		case "PASSWORD_MTIME":
			modifiedAt, parseErr := time.Parse(time.RFC3339, value)
			if parseErr != nil {
				return nil, nil, parseErr
			}
			seed.passwordMtime = &modifiedAt
		case "TOKENS":
			seed.tokens = []Token{}
			if value != "" && json.Unmarshal([]byte(value), &seed.tokens) != nil {
				log.Errorf("Account '%s' has invalid token in %s", name, key)
			}
		}
		accounts[name] = seed
	}

	for key, value := range secrets {
		if !strings.HasPrefix(key, "accounts.") {
			continue
		}
		parts := strings.Split(key, ".")
		if len(parts) != 3 {
			log.Warnf("Unexpected account secret key %s", key)
			continue
		}
		name, suffix := parts[1], parts[2]
		if name == common.AthenaAdminUsername {
			continue
		}
		seed := accounts[name]
		switch suffix {
		case "password":
			seed.passwordHash = value
		case "passwordMtime":
			modifiedAt, parseErr := time.Parse(time.RFC3339, value)
			if parseErr != nil {
				return nil, nil, parseErr
			}
			seed.passwordMtime = &modifiedAt
		case "tokens":
			seed.tokens = []Token{}
			if value != "" && json.Unmarshal([]byte(value), &seed.tokens) != nil {
				log.Errorf("Account '%s' has invalid token in settings", name)
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
	return accounts, loginDefaults, nil
}

func parseAdminAccount() (accountSeed, error) {
	seed := accountSeed{capabilities: []Capability{CapabilityLogin}}
	passwordHash, err := envOrFile("ATHENA_ADMIN_PASSWORD_HASH")
	if err != nil {
		return accountSeed{}, err
	}
	seed.passwordHash = passwordHash
	if value := os.Getenv("ATHENA_ADMIN_PASSWORD_MTIME"); value != "" {
		if modifiedAt, parseErr := time.Parse(time.RFC3339, value); parseErr == nil {
			seed.passwordMtime = &modifiedAt
		}
	}
	tokens, err := envOrFile("ATHENA_ADMIN_TOKENS")
	if err != nil {
		return accountSeed{}, err
	}
	seed.tokens = []Token{}
	if tokens != "" {
		if err := json.Unmarshal([]byte(tokens), &seed.tokens); err != nil {
			return accountSeed{}, err
		}
	}
	return seed, nil
}

func parseCapabilities(value, key string) []Capability {
	capabilities := []Capability{}
	for _, value := range strings.Split(value, ",") {
		capability := Capability(strings.TrimSpace(value))
		switch capability {
		case CapabilityLogin, CapabilityAPIKey:
			capabilities = append(capabilities, capability)
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
	for _, suffix := range []string{"_CAPABILITIES", "_ENABLED", "_PASSWORD_HASH", "_PASSWORD_MTIME", "_TOKENS"} {
		if strings.HasSuffix(raw, suffix) {
			return strings.TrimSuffix(raw, suffix), strings.TrimPrefix(suffix, "_"), true
		}
	}
	return "", "", false
}

func loadSecretsFromEnv() map[string]string {
	secrets := map[string]string{}
	for _, item := range os.Environ() {
		key, value, ok := strings.Cut(item, "=")
		if ok && strings.HasPrefix(key, "ATHENA_SECRET_") {
			secrets[strings.TrimPrefix(key, "ATHENA_SECRET_")] = value
		}
	}
	return secrets
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

func logLoadedAccounts(accounts map[string]accountSeed, secrets map[string]string) {
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
	secretKeys := 0
	for key := range secrets {
		if strings.HasPrefix(key, "accounts.") {
			secretKeys++
		}
	}
	log.Infof("Loaded local accounts from env: count=%d accounts=%v", len(names), names)
	log.Infof("Account source hints: ATHENA_ACCOUNT_* variables=%d, ATHENA_SECRET_accounts.* keys=%d", envVars, secretKeys)
}
