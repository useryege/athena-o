package settings

import (
	"encoding/json"
	"os"
	"strconv"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/useryege/athena/common"
)

func parseAdminAccount() (*Account, error) {
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
	adminAccount, err := parseAdminAccount()
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
