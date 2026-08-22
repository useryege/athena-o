package settings

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/useryege/athena/common"
	"github.com/useryege/athena/util"
	"github.com/useryege/athena/util/password"
)

type SettingsManagerOpts func(mgs *SettingsManager)

// NewSettingsManagerFromEnv loads settings once from environment variables.
func NewSettingsManagerFromEnv(ctx context.Context, opts ...SettingsManagerOpts) (*SettingsManager, error) {
	raw, err := loadRawSettingsFromEnv()
	if err != nil {
		return nil, err
	}

	settings, err := loadSettingsFromEnv(raw.Secrets)
	if err != nil {
		return nil, err
	}

	accounts, accountLoginDefaults, err := parseAccountsFromRaw(raw)
	if err != nil {
		return nil, err
	}

	mgr := &SettingsManager{
		ctx:                  ctx,
		raw:                  raw,
		settings:             settings,
		help:                 loadHelpFromEnv(),
		accounts:             accounts,
		accountLoginDefaults: accountLoginDefaults,
		mutex:                &sync.RWMutex{},
	}
	for i := range opts {
		opts[i](mgr)
	}

	logLoadedAccounts(raw, accounts)

	return mgr, nil
}

func logLoadedAccounts(raw RawSettings, accounts map[string]Account) {
	accountNames := make([]string, 0, len(accounts))
	for name := range accounts {
		accountNames = append(accountNames, name)
	}
	sort.Strings(accountNames)

	envAccountVars := 0
	for _, item := range os.Environ() {
		key, _, ok := strings.Cut(item, "=")
		if !ok {
			continue
		}
		if strings.HasPrefix(key, "ATHENA_ACCOUNT_") {
			envAccountVars++
		}
	}

	secretAccountKeys := 0
	for key := range raw.Secrets {
		if strings.HasPrefix(key, accountsKeyPrefix+".") {
			secretAccountKeys++
		}
	}

	log.Infof("Loaded local accounts from env: count=%d accounts=%v", len(accountNames), accountNames)
	log.Infof("Account source hints: ATHENA_ACCOUNT_* variables=%d, ATHENA_SECRET_accounts.* keys=%d", envAccountVars, secretAccountKeys)
}

// NewSettingsManager is kept as a compatibility shim for older call sites. The
// Kubernetes client argument is ignored because settings are now loaded from env.
func NewSettingsManager(ctx context.Context, _ any, _ string, opts ...SettingsManagerOpts) *SettingsManager {
	mgr, err := NewSettingsManagerFromEnv(ctx, opts...)
	if err != nil {
		panic(err)
	}

	return mgr
}

func (mgr *SettingsManager) GetPasswordPattern() (string, error) {
	mgr.mutex.RLock()
	defer mgr.mutex.RUnlock()

	pattern := mgr.settings.PasswordPattern
	if pattern == "" {
		return common.PasswordPatten, nil
	}

	return pattern, nil
}

func (mgr *SettingsManager) GetHelp() (*Help, error) {
	mgr.mutex.RLock()
	defer mgr.mutex.RUnlock()

	return &mgr.help, nil
}

// GetSettings returns settings loaded at process startup.
func (mgr *SettingsManager) GetSettings() (*AthenaSettings, error) {
	mgr.mutex.RLock()
	defer mgr.mutex.RUnlock()

	return &mgr.settings, nil
}

// InitializeSettings initializes transient admin password and JWT signature if missing.
func (mgr *SettingsManager) InitializeSettings() (*AthenaSettings, error) {
	mgr.mutex.Lock()
	defer mgr.mutex.Unlock()

	log.Debug("InitializeSettings started")

	adminAccount := mgr.accounts[common.AthenaAdminUsername]
	if adminAccount.PasswordHash == "" {
		initialPasswordBytes, err := util.MakeSignature(initialPasswordLength)
		if err != nil {
			return nil, err
		}

		initialPassword := base64.RawURLEncoding.EncodeToString(initialPasswordBytes)
		hashedPassword, err := password.HashPassword(initialPassword)
		if err != nil {
			return nil, err
		}

		now := time.Now().UTC()
		adminAccount.PasswordHash = hashedPassword
		adminAccount.PasswordMtime = &now
		mgr.accounts[common.AthenaAdminUsername] = adminAccount

		log.Warnf("Generated transient admin password because ATHENA_ADMIN_PASSWORD_HASH is not set. It will not persist across restarts: %s", initialPassword)
	} else if adminAccount.PasswordMtime == nil || adminAccount.PasswordMtime.IsZero() {
		now := time.Now().UTC()
		adminAccount.PasswordMtime = &now
		mgr.accounts[common.AthenaAdminUsername] = adminAccount
	}

	if len(mgr.settings.ServerSignature) == 0 {
		signature, err := util.MakeSignature(32)
		if err != nil {
			return nil, fmt.Errorf("error setting JWT signature: %w", err)
		}

		mgr.settings.ServerSignature = signature
		log.Warnf("Generated transient JWT secret because ATHENA_JWT_SECRET is not set, existing sessions will be invalid after restart: %s", string(signature))
	}

	log.Debug("InitializeSettings completed successfully")

	return &mgr.settings, nil
}
