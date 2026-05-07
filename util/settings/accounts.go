package settings

import (
	"encoding/json"
	"os"
	"strconv"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/useryege/athena/common"
)

const (
	accountsKeyPrefix          = "accounts"
	accountPasswordSuffix      = "password"
	accountPasswordMtimeSuffix = "passwordMtime"
	accountTokensSuffix        = "tokens"
)

type AccountCapability string

const (
	// AccountCapabilityLogin represents capability to create UI session tokens.
	AccountCapabilityLogin AccountCapability = "login"
	// AccountCapabilityLogin represents capability to generate API auth tokens.
	AccountCapabilityApiKey AccountCapability = "apiKey" //nolint:revive //FIXME(var-naming)
)

// Token holds the information about the generated auth token.
type Token struct {
	ID        string `json:"id"`
	IssuedAt  int64  `json:"iat"`
	ExpiresAt int64  `json:"exp,omitempty"`
}

// Account holds local account information
type Account struct {
	PasswordHash  string
	PasswordMtime *time.Time
	Enabled       bool
	Capabilities  []AccountCapability
	Tokens        []Token
}

// FormatPasswordMtime return the formatted password modify time or empty string of password modify time is nil.
func (a *Account) FormatPasswordMtime() string {
	if a.PasswordMtime == nil {
		return ""
	}
	return a.PasswordMtime.Format(time.RFC3339)
}

// FormatCapabilities returns comma separate list of user capabilities.
func (a *Account) FormatCapabilities() string {
	var items []string
	for i := range a.Capabilities {
		items = append(items, string(a.Capabilities[i]))
	}
	return strings.Join(items, ",")
}

// TokenIndex return an index of a token with the given identifier or -1 if token not found.
func (a *Account) TokenIndex(id string) int {
	for i := range a.Tokens {
		if a.Tokens[i].ID == id {
			return i
		}
	}
	return -1
}

// HasCapability return true if the account has the specified capability.
func (a *Account) HasCapability(capability AccountCapability) bool {
	for _, c := range a.Capabilities {
		if c == capability {
			return true
		}
	}
	return false
}

// AddAccount save an account with the given name and properties.
func (mgr *SettingsManager) AddAccount(name string, account Account) error {
	return status.Error(codes.FailedPrecondition, "account updates are disabled because settings are loaded from environment variables and are read-only at runtime")
}

// GetAccount return an account info by the specified name.
func (mgr *SettingsManager) GetAccount(name string) (*Account, error) {
	mgr.mutex.RLock()
	defer mgr.mutex.RUnlock()
	account, ok := mgr.accounts[name]
	if !ok {
		return nil, status.Errorf(codes.NotFound, "account '%s' does not exist", name)
	}
	accountCopy := copyAccount(account)
	return &accountCopy, nil
}

// UpdateAccount runs the callback function against an account that matches to the specified name
// and persist changes applied by the callback.
func (mgr *SettingsManager) UpdateAccount(name string, callback func(account *Account) error) error {
	return status.Error(codes.FailedPrecondition, "account updates are disabled because settings are loaded from environment variables and are read-only at runtime")
}

// GetAccounts returns list of configured accounts
func (mgr *SettingsManager) GetAccounts() (map[string]Account, error) {
	mgr.mutex.RLock()
	defer mgr.mutex.RUnlock()
	accounts := make(map[string]Account, len(mgr.accounts))
	for name, account := range mgr.accounts {
		accounts[name] = copyAccount(account)
	}
	return accounts, nil
}

func copyAccount(account Account) Account {
	account.Capabilities = append([]AccountCapability(nil), account.Capabilities...)
	account.Tokens = append([]Token(nil), account.Tokens...)
	if account.PasswordMtime != nil {
		mt := *account.PasswordMtime
		account.PasswordMtime = &mt
	}
	return account
}

func parseAdminAccount(raw RawSettings) (*Account, error) {
	adminAccount := &Account{Enabled: true, Capabilities: []AccountCapability{AccountCapabilityLogin}}
	if adminPasswordHash, err := envOrFile("ATHENA_ADMIN_PASSWORD_HASH"); err != nil {
		return nil, err
	} else if adminPasswordHash != "" {
		adminAccount.PasswordHash = adminPasswordHash
	}
	if adminPasswordMtime := os.Getenv("ATHENA_ADMIN_PASSWORD_MTIME"); adminPasswordMtime != "" {
		if mTime, err := time.Parse(time.RFC3339, adminPasswordMtime); err == nil {
			adminAccount.PasswordMtime = &mTime
		}
	}

	adminAccount.Tokens = make([]Token, 0)
	if tokensStr, err := envOrFile("ATHENA_ADMIN_TOKENS"); err != nil {
		return nil, err
	} else if tokensStr != "" {
		if err := json.Unmarshal([]byte(tokensStr), &adminAccount.Tokens); err != nil {
			return nil, err
		}
	}

	if enabledStr := os.Getenv("ATHENA_ADMIN_ENABLED"); enabledStr != "" {
		if enabled, err := strconv.ParseBool(enabledStr); err == nil {
			adminAccount.Enabled = enabled
		} else {
			log.Warnf("invalid ATHENA_ADMIN_ENABLED: %v", err)
		}
	}

	return adminAccount, nil
}

func parseAccountCapabilities(value string, key string) []AccountCapability {
	capabilities := []AccountCapability{}
	for _, capability := range strings.Split(value, ",") {
		capability = strings.TrimSpace(capability)
		if capability == "" {
			continue
		}

		switch capability {
		case string(AccountCapabilityLogin):
			capabilities = append(capabilities, AccountCapabilityLogin)
		case string(AccountCapabilityApiKey):
			capabilities = append(capabilities, AccountCapabilityApiKey)
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

func parseAccountsFromRaw(raw RawSettings) (map[string]Account, error) {
	adminAccount, err := parseAdminAccount(raw)
	if err != nil {
		return nil, err
	}
	accounts := map[string]Account{
		common.AthenaAdminUsername: *adminAccount,
	}

	for _, item := range os.Environ() {
		key, value, ok := strings.Cut(item, "=")
		if !ok {
			continue
		}
		accountName, suffix, ok := accountFromEnvKey(key)
		if !ok || accountName == "" {
			continue
		}
		account, ok := accounts[accountName]
		if !ok {
			account = Account{Enabled: true}
		}
		switch suffix {
		case "CAPABILITIES":
			account.Capabilities = parseAccountCapabilities(value, key)
		case "ENABLED":
			account.Enabled, err = strconv.ParseBool(value)
			if err != nil {
				return nil, err
			}
		case "PASSWORD_HASH":
			account.PasswordHash = value
		case "PASSWORD_MTIME":
			mTime, err := time.Parse(time.RFC3339, value)
			if err != nil {
				return nil, err
			}
			account.PasswordMtime = &mTime
		case "TOKENS":
			account.Tokens = make([]Token, 0)
			if value != "" {
				if err := json.Unmarshal([]byte(value), &account.Tokens); err != nil {
					log.Errorf("Account '%s' has invalid token in %s", accountName, key)
				}
			}
		}
		accounts[accountName] = account
	}

	for key, value := range raw.Secrets {
		if !strings.HasPrefix(key, accountsKeyPrefix+".") {
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
		account, ok := accounts[name]
		if !ok {
			account = Account{Enabled: true}
		}
		switch suffix {
		case accountPasswordSuffix:
			account.PasswordHash = value
		case accountPasswordMtimeSuffix:
			mTime, err := time.Parse(time.RFC3339, value)
			if err != nil {
				return nil, err
			}
			account.PasswordMtime = &mTime
		case accountTokensSuffix:
			account.Tokens = make([]Token, 0)
			if value != "" {
				if err := json.Unmarshal([]byte(value), &account.Tokens); err != nil {
					log.Errorf("Account '%s' has invalid token in settings", name)
				}
			}
		}
		accounts[name] = account
	}

	return accounts, nil
}
