package settings

import (
	"github.com/useryege/athena/common"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

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

// ApplyAccountEnabledOverrides applies persisted availability overrides to
// accounts that are currently configured by the environment. Overrides for
// removed accounts and for the built-in administrator are intentionally
// ignored.
func (mgr *SettingsManager) ApplyAccountEnabledOverrides(overrides map[string]bool) {
	mgr.mutex.Lock()
	defer mgr.mutex.Unlock()

	for name, enabled := range overrides {
		if name == common.AthenaAdminUsername {
			continue
		}
		account, ok := mgr.accounts[name]
		if !ok {
			continue
		}
		account.Enabled = enabled
		mgr.accounts[name] = account
	}
}

// SetAccountEnabled updates the effective in-memory availability of a
// configured non-administrator account. Persistence is deliberately handled
// by the caller before this method is invoked.
func (mgr *SettingsManager) SetAccountEnabled(name string, enabled bool) (*Account, error) {
	mgr.mutex.Lock()
	defer mgr.mutex.Unlock()

	account, ok := mgr.accounts[name]
	if !ok {
		return nil, status.Errorf(codes.NotFound, "account '%s' does not exist", name)
	}
	if name == common.AthenaAdminUsername {
		return nil, status.Errorf(codes.InvalidArgument, "account '%s' is always enabled", name)
	}

	account.Enabled = enabled
	mgr.accounts[name] = account
	accountCopy := copyAccount(account)
	return &accountCopy, nil
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
