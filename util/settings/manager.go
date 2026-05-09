package settings

import (
	"context"
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/useryege/athena/common"
	"github.com/useryege/athena/util"
	"github.com/useryege/athena/util/password"
	tlsutil "github.com/useryege/athena/util/tls"
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

	accounts, err := parseAccountsFromRaw(raw)
	if err != nil {
		return nil, err
	}

	mgr := &SettingsManager{
		ctx:           ctx,
		raw:           raw,
		settings:      settings,
		help:          loadHelpFromEnv(),
		accounts:      accounts,
		mutex:         &sync.RWMutex{},
		tlsCertParser: tls.X509KeyPair,
	}
	for i := range opts {
		opts[i](mgr)
	}

	if cert, err := loadTLSCertificateFromEnv(mgr.tlsCertParser); err != nil {
		return nil, err
	} else if cert != nil {
		mgr.settings.Certificate = cert
	}

	return mgr, nil
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

// InitializeSettings is used to initialize empty admin password, signature, certificate etc if missing
func (mgr *SettingsManager) InitializeSettings(insecureModeEnabled bool) (*AthenaSettings, error) {
	mgr.mutex.Lock()
	defer mgr.mutex.Unlock()

	log.Debugf("InitializeSettings started (insecureModeEnabled=%t)", insecureModeEnabled)

	adminAccount := mgr.accounts[common.AthenaAdminUsername]
	if adminAccount.Enabled && adminAccount.PasswordHash == "" {
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
	} else if adminAccount.Enabled && (adminAccount.PasswordMtime == nil || adminAccount.PasswordMtime.IsZero()) {
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

	if mgr.settings.Certificate == nil && !insecureModeEnabled {
		log.Debug("TLS certificate missing and insecure mode disabled, generating TLS certificate")

		hosts := []string{
			"localhost",
			"athena-server",
		}
		certOpts := tlsutil.CertOptions{
			Hosts:        hosts,
			Organization: "Athena",
			IsCA:         false,
		}

		cert, err := tlsutil.GenerateX509KeyPair(certOpts)
		if err != nil {
			return nil, err
		}

		mgr.settings.Certificate = cert
		log.Warn("Generated transient TLS certificate because ATHENA_TLS_CERT_FILE/ATHENA_TLS_KEY_FILE are not set.")
	}

	log.Debug("InitializeSettings completed successfully")

	return &mgr.settings, nil
}
