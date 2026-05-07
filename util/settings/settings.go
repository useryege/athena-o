package settings

import (
	"context"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path"
	"strings"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/useryege/athena/common"
	"github.com/useryege/athena/internal/server/settings/oidc"
	timeutil "github.com/useryege/athena/pkg/time"
	"github.com/useryege/athena/util"
	"github.com/useryege/athena/util/crypto"
	"github.com/useryege/athena/util/env"
	"github.com/useryege/athena/util/password"
	tlsutil "github.com/useryege/athena/util/tls"
	"sigs.k8s.io/yaml"
)

// AthenaSettings holds in-memory runtime configuration options.
type AthenaSettings struct {
	// URL is the externally facing URL users will visit to reach Athena.
	// The value here is used when configuring SSO. Omitting this value will disable SSO.
	URL string `json:"url,omitempty"`
	// URLs is a list of externally facing URLs users will visit to reach Athena.
	// The value here is used when configuring SSO reachable from multiple domains.
	AdditionalURLs []string `json:"additionalUrls,omitempty"`
	// Indicates if status badge is enabled or not.
	StatusBadgeEnabled bool `json:"statusBadgeEnable"`
	// Indicates if status badge custom root URL should be used.
	StatusBadgeRootUrl string `json:"statusBadgeRootUrl,omitempty"` //nolint:revive //FIXME(var-naming)
	// DexConfig contains portions of a dex config yaml
	DexConfig string `json:"dexConfig,omitempty"`
	// OIDCConfigRAW holds OIDC configuration as a raw string
	OIDCConfigRAW string `json:"oidcConfig,omitempty"`
	// ServerSignature holds the key used to generate JWT tokens.
	ServerSignature []byte `json:"serverSignature,omitempty"`
	// Certificate holds the certificate/private key for the Athena API server.
	// If nil, will run insecure without TLS.
	Certificate *tls.Certificate `json:"-"`
	// Secrets holds all secrets in athena-secret as a map[string]string
	Secrets map[string]string `json:"secrets,omitempty"`
	// Indicates if anonymous user is enabled or not
	AnonymousUserEnabled bool `json:"anonymousUserEnabled,omitempty"`
	// Specifies token expiration duration
	UserSessionDuration time.Duration `json:"userSessionDuration,omitempty"`
	// UiCssURL local or remote path to user-defined CSS to customize Athena UI
	UiCssURL string `json:"uiCssURL,omitempty"` //nolint:revive //FIXME(var-naming)
	// Content of UI Banner
	UiBannerContent string `json:"uiBannerContent,omitempty"` //nolint:revive //FIXME(var-naming)
	// URL for UI Banner
	UiBannerURL string `json:"uiBannerURL,omitempty"` //nolint:revive //FIXME(var-naming)
	// Make Banner permanent and not closeable
	UiBannerPermanent bool `json:"uiBannerPermanent,omitempty"` //nolint:revive //FIXME(var-naming)
	// Position of UI Banner
	UiBannerPosition string `json:"uiBannerPosition,omitempty"` //nolint:revive //FIXME(var-naming)
	// PasswordPattern for password regular expression
	PasswordPattern string `json:"passwordPattern,omitempty"`
	// BinaryUrls contains the URLs for downloading athena binaries
	BinaryUrls map[string]string `json:"binaryUrls,omitempty"`
	// ExecEnabled indicates whether the UI exec feature is enabled
	// ExecEnabled bool `json:"execEnabled"`
	// OIDCTLSInsecureSkipVerify determines whether certificate verification is skipped when verifying tokens with the
	// configured OIDC provider (either external or the bundled Dex instance). Setting this to `true` will cause JWT
	// token verification to pass despite the OIDC provider having an invalid certificate. Only set to `true` if you
	// understand the risks.
	OIDCTLSInsecureSkipVerify bool `json:"oidcTLSInsecureSkipVerify"`
}

// Help settings
type Help struct {
	// the URL for getting chat help, this will typically be your Slack channel for support
	ChatURL string `json:"chatUrl,omitempty"`
	// the text for getting chat help, defaults to "Chat now!"
	ChatText string `json:"chatText,omitempty"`
	// the URLs for downloading athena binaries
	BinaryURLs map[string]string `json:"binaryUrl,omitempty"`
}

// oidcConfig is the same as the public OIDCConfig, except the public one excludes the AllowedAudiences and the
// SkipAudienceCheckWhenTokenHasNoAudience fields.
// AllowedAudiences should be accessed via AthenaSettings.OAuth2AllowedAudiences.
// SkipAudienceCheckWhenTokenHasNoAudience should be accessed via AthenaSettings.SkipAudienceCheckWhenTokenHasNoAudience.
type oidcConfig struct {
	OIDCConfig
	AllowedAudiences                        []string `json:"allowedAudiences,omitempty"`
	SkipAudienceCheckWhenTokenHasNoAudience *bool    `json:"skipAudienceCheckWhenTokenHasNoAudience,omitempty"`
}

func (o *oidcConfig) toExported() *OIDCConfig {
	if o == nil {
		return nil
	}
	return &OIDCConfig{
		Name:                     o.Name,
		Issuer:                   o.Issuer,
		ClientID:                 o.ClientID,
		ClientSecret:             o.ClientSecret,
		Azure:                    o.Azure,
		CLIClientID:              o.CLIClientID,
		UserInfoPath:             o.UserInfoPath,
		EnableUserInfoGroups:     o.EnableUserInfoGroups,
		UserInfoCacheExpiration:  o.UserInfoCacheExpiration,
		RefreshTokenThreshold:    o.RefreshTokenThreshold,
		RequestedScopes:          o.RequestedScopes,
		RequestedIDTokenClaims:   o.RequestedIDTokenClaims,
		LogoutURL:                o.LogoutURL,
		RootCA:                   o.RootCA,
		EnablePKCEAuthentication: o.EnablePKCEAuthentication,
		DomainHint:               o.DomainHint,
	}
}

type OIDCConfig struct {
	Name                     string                 `json:"name,omitempty"`
	Issuer                   string                 `json:"issuer,omitempty"`
	ClientID                 string                 `json:"clientID,omitempty"`
	ClientSecret             string                 `json:"clientSecret,omitempty"`
	CLIClientID              string                 `json:"cliClientID,omitempty"`
	EnableUserInfoGroups     bool                   `json:"enableUserInfoGroups,omitempty"`
	UserInfoPath             string                 `json:"userInfoPath,omitempty"`
	UserInfoCacheExpiration  string                 `json:"userInfoCacheExpiration,omitempty"`
	RequestedScopes          []string               `json:"requestedScopes,omitempty"`
	RequestedIDTokenClaims   map[string]*oidc.Claim `json:"requestedIDTokenClaims,omitempty"`
	LogoutURL                string                 `json:"logoutURL,omitempty"`
	RootCA                   string                 `json:"rootCA,omitempty"`
	EnablePKCEAuthentication bool                   `json:"enablePKCEAuthentication,omitempty"`
	DomainHint               string                 `json:"domainHint,omitempty"`
	Azure                    *AzureOIDCConfig       `json:"azure,omitempty"`
	RefreshTokenThreshold    string                 `json:"refreshTokenThreshold,omitempty"`
}

type AzureOIDCConfig struct {
	UseWorkloadIdentity bool `json:"useWorkloadIdentity,omitempty"`
}

const (
	// initialPasswordLength defines the length of the generated initial password
	initialPasswordLength = 16
)

// RawSettings holds secret values loaded during process startup.
type RawSettings struct {
	Secrets map[string]string
}

// SettingsManager holds immutable runtime settings loaded during process startup.
type SettingsManager struct {
	ctx           context.Context
	raw           RawSettings
	settings      AthenaSettings
	help          Help
	accounts      map[string]Account
	mutex         *sync.RWMutex
	tlsCertParser func([]byte, []byte) (tls.Certificate, error)
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

func getDownloadBinaryUrlsFromEnv() map[string]string {
	binaryUrls := map[string]string{}
	for _, archType := range []string{"darwin-amd64", "darwin-arm64", "windows-amd64", "linux-amd64", "linux-arm64", "linux-ppc64le", "linux-s390x"} {
		envName := "ATHENA_HELP_DOWNLOAD_" + strings.ToUpper(strings.NewReplacer("-", "_").Replace(archType))
		if val := os.Getenv(envName); val != "" {
			binaryUrls[archType] = val
		}
	}
	return binaryUrls
}

func loadHelpFromEnv() Help {
	chatURL := os.Getenv("ATHENA_HELP_CHAT_URL")
	chatText := ""
	if chatURL != "" {
		chatText = env.StringFromEnv("ATHENA_HELP_CHAT_TEXT", "Chat now!")
	}
	return Help{
		ChatURL:    chatURL,
		ChatText:   chatText,
		BinaryURLs: getDownloadBinaryUrlsFromEnv(),
	}
}

func loadSettingsFromEnv(secrets map[string]string) (AthenaSettings, error) {
	serverSignature, err := envOrFile("ATHENA_JWT_SECRET")
	if err != nil {
		return AthenaSettings{}, err
	}
	settings := AthenaSettings{
		DexConfig:            os.Getenv("ATHENA_DEX_CONFIG"),
		StatusBadgeEnabled:   env.ParseBoolFromEnv("ATHENA_STATUS_BADGE_ENABLED", false),
		StatusBadgeRootUrl:   os.Getenv("ATHENA_STATUS_BADGE_ROOT_URL"),
		AnonymousUserEnabled: env.ParseBoolFromEnv("ATHENA_ANONYMOUS_USER_ENABLED", false),
		UiCssURL:             os.Getenv("ATHENA_UI_CSS_URL"),
		UiBannerContent:      os.Getenv("ATHENA_UI_BANNER_CONTENT"),
		UiBannerPermanent:    env.ParseBoolFromEnv("ATHENA_UI_BANNER_PERMANENT", false),
		UiBannerPosition:     os.Getenv("ATHENA_UI_BANNER_POSITION"),
		BinaryUrls:           getDownloadBinaryUrlsFromEnv(),
		UiBannerURL:          os.Getenv("ATHENA_UI_BANNER_URL"),
		UserSessionDuration:  time.Hour * 24,
		PasswordPattern:      env.StringFromEnv("ATHENA_PASSWORD_PATTERN", common.PasswordPatten),
		// ExecEnabled:               env.ParseBoolFromEnv("ATHENA_EXEC_ENABLED", false),
		OIDCTLSInsecureSkipVerify: env.ParseBoolFromEnv("ATHENA_OIDC_TLS_INSECURE_SKIP_VERIFY", false),
		ServerSignature:           []byte(serverSignature),
		Secrets:                   secrets,
	}

	oidcConfig, err := loadOIDCConfigFromEnv()
	if err != nil {
		return settings, err
	}
	settings.OIDCConfigRAW = oidcConfig

	settings.URL = os.Getenv("ATHENA_URL")
	if err := ValidateExternalURL(settings.URL); err != nil {
		log.Warnf("Failed to validate URL in settings: %v", err)
	}
	if err := ValidateExternalURL(settings.UiBannerURL); err != nil {
		log.Warnf("Failed to validate UI banner URL in settings: %v", err)
	}

	additionalURLs := os.Getenv("ATHENA_ADDITIONAL_URLS")
	if additionalURLs != "" {
		if err := yaml.Unmarshal([]byte(additionalURLs), &settings.AdditionalURLs); err != nil {
			settings.AdditionalURLs = splitCommaSeparated(additionalURLs)
		}
	}
	for _, url := range settings.AdditionalURLs {
		if err := ValidateExternalURL(url); err != nil {
			log.Warnf("Failed to validate external URL in settings: %v", err)
		}
	}

	if userSessionDurationStr := os.Getenv("ATHENA_SESSION_DURATION"); userSessionDurationStr != "" {
		if val, err := timeutil.ParseDuration(userSessionDurationStr); err != nil {
			log.Warnf("Failed to parse ATHENA_SESSION_DURATION: %v", err)
		} else {
			settings.UserSessionDuration = *val
		}
	}
	return settings, nil
}

// ValidateExternalURL ensures the external URL that is set on the configmap is valid
func ValidateExternalURL(u string) error {
	if u == "" {
		return nil
	}
	URL, err := url.Parse(u)
	if err != nil {
		return fmt.Errorf("failed to parse URL: %w", err)
	}
	if URL.Scheme != "http" && URL.Scheme != "https" {
		return errors.New("URL must include http or https protocol")
	}
	return nil
}

func splitCommaSeparated(value string) []string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	parts := strings.Split(value, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			result = append(result, part)
		}
	}
	return result
}

func envOrFile(envName string) (string, error) {
	if value := os.Getenv(envName); value != "" {
		return value, nil
	}
	fileName := envName + "_FILE"
	if filePath := os.Getenv(fileName); filePath != "" {
		data, err := os.ReadFile(filePath)
		if err != nil {
			return "", fmt.Errorf("failed reading %s: %w", fileName, err)
		}
		return strings.TrimSpace(string(data)), nil
	}
	return "", nil
}

func loadOIDCConfigFromEnv() (string, error) {
	if config, err := envOrFile("ATHENA_OIDC_CONFIG"); err != nil {
		return "", err
	} else if config != "" {
		return config, nil
	}
	issuer := env.StringFromEnv("ATHENA_OIDC_ISSUER", "")
	clientID := env.StringFromEnv("ATHENA_OIDC_CLIENT_ID", "")
	clientSecret, err := envOrFile("ATHENA_OIDC_CLIENT_SECRET")
	if err != nil {
		return "", err
	}
	if issuer == "" && clientID == "" && clientSecret == "" {
		return "", nil
	}
	config := oidcConfig{
		OIDCConfig: OIDCConfig{
			Name:                     env.StringFromEnv("ATHENA_OIDC_NAME", "OIDC"),
			Issuer:                   issuer,
			ClientID:                 clientID,
			ClientSecret:             clientSecret,
			CLIClientID:              env.StringFromEnv("ATHENA_OIDC_CLI_CLIENT_ID", ""),
			RequestedScopes:          env.StringsFromEnv("ATHENA_OIDC_SCOPES", []string{"openid", "profile", "email"}, ","),
			LogoutURL:                env.StringFromEnv("ATHENA_OIDC_LOGOUT_URL", ""),
			EnablePKCEAuthentication: env.ParseBoolFromEnv("ATHENA_OIDC_ENABLE_PKCE", false),
		},
	}
	data, err := yaml.Marshal(config)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func loadRawSettingsFromEnv() (RawSettings, error) {
	raw := RawSettings{Secrets: map[string]string{}}
	for _, item := range os.Environ() {
		key, value, ok := strings.Cut(item, "=")
		if !ok || !strings.HasPrefix(key, "ATHENA_SECRET_") {
			continue
		}
		raw.Secrets[strings.TrimPrefix(key, "ATHENA_SECRET_")] = value
	}
	return raw, nil
}

func loadTLSCertificateFromEnv(parser func([]byte, []byte) (tls.Certificate, error)) (*tls.Certificate, error) {
	certFile := os.Getenv("ATHENA_TLS_CERT_FILE")
	keyFile := os.Getenv("ATHENA_TLS_KEY_FILE")
	if certFile == "" && keyFile == "" {
		return nil, nil
	}
	if certFile == "" || keyFile == "" {
		return nil, errors.New("ATHENA_TLS_CERT_FILE and ATHENA_TLS_KEY_FILE must be set together")
	}
	certBytes, err := os.ReadFile(certFile)
	if err != nil {
		return nil, fmt.Errorf("failed reading ATHENA_TLS_CERT_FILE: %w", err)
	}
	keyBytes, err := os.ReadFile(keyFile)
	if err != nil {
		return nil, fmt.Errorf("failed reading ATHENA_TLS_KEY_FILE: %w", err)
	}
	cert, err := parser(certBytes, keyBytes)
	if err != nil {
		return nil, err
	}
	return &cert, nil
}

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

// IsSSOConfigured returns whether or not single-sign-on is configured
func (a *AthenaSettings) IsSSOConfigured() bool {
	if a.IsDexConfigured() {
		return true
	}
	if a.OIDCConfig() != nil {
		return true
	}
	return false
}

func (a *AthenaSettings) IsDexConfigured() bool {
	if a.URL == "" {
		return false
	}
	dexCfg, err := UnmarshalDexConfig(a.DexConfig)
	if err != nil {
		log.Warnf("invalid dex yaml config: %s", err.Error())
		return false
	}
	return len(dexCfg) > 0
}

// GetServerEncryptionKey generates a new server encryption key using the server signature as a passphrase
func (a *AthenaSettings) GetServerEncryptionKey() ([]byte, error) {
	return crypto.KeyFromPassphrase(string(a.ServerSignature))
}

func UnmarshalDexConfig(config string) (map[string]any, error) {
	var dexCfg map[string]any
	err := yaml.Unmarshal([]byte(config), &dexCfg)
	return dexCfg, err
}

func (a *AthenaSettings) oidcConfig() *oidcConfig {
	if a.OIDCConfigRAW == "" {
		return nil
	}
	configMap := map[string]any{}
	err := yaml.Unmarshal([]byte(a.OIDCConfigRAW), &configMap)
	if err != nil {
		log.Warnf("invalid oidc config: %v", err)
		return nil
	}

	configMap = ReplaceMapSecrets(configMap, a.Secrets)
	data, err := yaml.Marshal(configMap)
	if err != nil {
		log.Warnf("invalid oidc config: %v", err)
		return nil
	}

	config, err := unmarshalOIDCConfig(string(data))
	if err != nil {
		log.Warnf("invalid oidc config: %v", err)
		return nil
	}

	return &config
}

func (a *AthenaSettings) OIDCConfig() *OIDCConfig {
	config := a.oidcConfig()
	if config == nil {
		return nil
	}
	return config.toExported()
}

func unmarshalOIDCConfig(configStr string) (oidcConfig, error) {
	var config oidcConfig
	err := yaml.Unmarshal([]byte(configStr), &config)
	return config, err
}

func ValidateOIDCConfig(configStr string) error {
	_, err := unmarshalOIDCConfig(configStr)
	return err
}

// TLSConfig returns a tls.Config with the configured certificates
func (a *AthenaSettings) TLSConfig() *tls.Config {
	if a.Certificate == nil {
		return nil
	}
	certPool := x509.NewCertPool()
	pemCertBytes, _ := tlsutil.EncodeX509KeyPair(*a.Certificate)
	ok := certPool.AppendCertsFromPEM(pemCertBytes)
	if !ok {
		panic("bad certs")
	}
	return &tls.Config{
		RootCAs: certPool,
	}
}

func (a *AthenaSettings) IssuerURL() string {
	if oidcConfig := a.OIDCConfig(); oidcConfig != nil {
		return oidcConfig.Issuer
	}
	if a.DexConfig != "" {
		return a.URL + common.DexAPIEndpoint
	}
	return ""
}

// UserInfoGroupsEnabled returns whether group claims should be fetch from UserInfo endpoint
func (a *AthenaSettings) UserInfoGroupsEnabled() bool {
	if oidcConfig := a.OIDCConfig(); oidcConfig != nil {
		return oidcConfig.EnableUserInfoGroups
	}
	return false
}

// UserInfoPath returns the sub-path on which the IDP exposes the UserInfo endpoint
func (a *AthenaSettings) UserInfoPath() string {
	if oidcConfig := a.OIDCConfig(); oidcConfig != nil {
		return oidcConfig.UserInfoPath
	}
	return ""
}

// UserInfoCacheExpiration returns the expiry time of the UserInfo cache
func (a *AthenaSettings) UserInfoCacheExpiration() time.Duration {
	if oidcConfig := a.OIDCConfig(); oidcConfig != nil && oidcConfig.UserInfoCacheExpiration != "" {
		userInfoCacheExpiration, err := time.ParseDuration(oidcConfig.UserInfoCacheExpiration)
		if err != nil {
			log.Warnf("Failed to parse 'oidc.config.userInfoCacheExpiration' key: %v", err)
		}
		return userInfoCacheExpiration
	}
	return 0
}

// RefreshTokenThreshold returns the duration before token expiration that a token should be refreshed by the server
func (a *AthenaSettings) RefreshTokenThreshold() time.Duration {
	return a.RefreshTokenThresholdWithConfig(a.OIDCConfig())
}

// RefreshTokenThresholdWithConfig takes oidcConfig as param and returns the duration before token expiration that a token should be refreshed by the server
func (a *AthenaSettings) RefreshTokenThresholdWithConfig(oidcConfig *OIDCConfig) time.Duration {
	if oidcConfig != nil && oidcConfig.RefreshTokenThreshold != "" {
		refreshTokenThreshold, err := time.ParseDuration(oidcConfig.RefreshTokenThreshold)
		if err != nil {
			log.Warnf("Failed to parse 'oidc.config.refreshTokenThreshold' key: %v", err)
		}
		return refreshTokenThreshold
	}
	return 0
}

func (a *AthenaSettings) OAuth2ClientID() string {
	if oidcConfig := a.OIDCConfig(); oidcConfig != nil {
		return oidcConfig.ClientID
	}
	if a.DexConfig != "" {
		return common.AthenaClientAppID
	}
	return ""
}

// OAuth2AllowedAudiences returns a list of audiences that are allowed for the OAuth2 client. If the user has not
// explicitly configured the list of audiences (or has configured an empty list), then the OAuth2 client ID is returned
// as the only allowed audience. When using the bundled Dex, that client ID is always "athena".
func (a *AthenaSettings) OAuth2AllowedAudiences() []string {
	if config := a.oidcConfig(); config != nil {
		if len(config.AllowedAudiences) == 0 {
			allowedAudiences := []string{config.ClientID}
			if config.CLIClientID != "" {
				allowedAudiences = append(allowedAudiences, config.CLIClientID)
			}
			return allowedAudiences
		}
		return config.AllowedAudiences
	}
	if a.DexConfig != "" {
		return []string{common.AthenaClientAppID, common.AthenaCLIClientAppID}
	}
	return nil
}

func (a *AthenaSettings) SkipAudienceCheckWhenTokenHasNoAudience() bool {
	if config := a.oidcConfig(); config != nil {
		if config.SkipAudienceCheckWhenTokenHasNoAudience != nil {
			return *config.SkipAudienceCheckWhenTokenHasNoAudience
		}
		return false
	}
	// When using the bundled Dex, the audience check is required. Dex will always send JWTs with an audience.
	return false
}

func (a *AthenaSettings) OAuth2ClientSecret() string {
	if oidcConfig := a.OIDCConfig(); oidcConfig != nil {
		return oidcConfig.ClientSecret
	}
	if a.DexConfig != "" {
		return a.DexOAuth2ClientSecret()
	}
	return ""
}

func (a *AthenaSettings) OAuth2UsePKCE() bool {
	if oidcConfig := a.OIDCConfig(); oidcConfig != nil {
		return oidcConfig.EnablePKCEAuthentication
	}
	return false
}

func (a *AthenaSettings) UseAzureWorkloadIdentity() bool {
	if oidcConfig := a.OIDCConfig(); oidcConfig != nil && oidcConfig.Azure != nil {
		return oidcConfig.Azure.UseWorkloadIdentity
	}
	return false
}

// OIDCTLSConfig returns the TLS config for the OIDC provider. If an external provider is configured, returns a TLS
// config using the root CAs (if any) specified in the OIDC config. If an external OIDC provider is not configured,
// returns the API server TLS config, because the API server proxies requests to Dex.
func (a *AthenaSettings) OIDCTLSConfig() *tls.Config {
	var tlsConfig *tls.Config

	oidcConfig := a.OIDCConfig()
	if oidcConfig != nil {
		tlsConfig = &tls.Config{}
		if oidcConfig.RootCA != "" {
			certPool := x509.NewCertPool()
			ok := certPool.AppendCertsFromPEM([]byte(oidcConfig.RootCA))
			if !ok {
				log.Warn("failed to append certificates from PEM: proceeding without custom rootCA")
			} else {
				tlsConfig.RootCAs = certPool
			}
		}
	} else {
		tlsConfig = a.TLSConfig()
	}
	if tlsConfig != nil && a.OIDCTLSInsecureSkipVerify {
		tlsConfig.InsecureSkipVerify = true
	}
	return tlsConfig
}

func appendURLPath(inputURL string, inputPath string) (string, error) {
	u, err := url.Parse(inputURL)
	if err != nil {
		return "", err
	}
	u.Path = path.Join(u.Path, inputPath)
	return u.String(), nil
}

func (a *AthenaSettings) RedirectURL() (string, error) {
	return appendURLPath(a.URL, common.CallbackEndpoint)
}

func (a *AthenaSettings) AthenaURLForRequest(r *http.Request) (string, error) {
	for _, candidateURL := range append([]string{a.URL}, a.AdditionalURLs...) {
		u, err := url.Parse(candidateURL)
		if err != nil {
			return "", err
		}
		if u.Host == r.Host && strings.HasPrefix(r.URL.RequestURI(), u.RequestURI()) {
			return candidateURL, nil
		}
	}
	return a.URL, nil
}

func (a *AthenaSettings) RedirectURLForRequest(r *http.Request) (string, error) {
	if r == nil {
		return "", errors.New("request is nil")
	}
	base, err := a.AthenaURLForRequest(r)
	if err != nil {
		return "", err
	}
	return appendURLPath(base, common.CallbackEndpoint)
}

func (a *AthenaSettings) RedirectAdditionalURLs() ([]string, error) {
	RedirectAdditionalURLs := []string{}
	for _, url := range a.AdditionalURLs {
		redirectURL, err := appendURLPath(url, common.CallbackEndpoint)
		if err != nil {
			return []string{}, err
		}
		RedirectAdditionalURLs = append(RedirectAdditionalURLs, redirectURL)
	}
	return RedirectAdditionalURLs, nil
}

func (a *AthenaSettings) DexRedirectURL() (string, error) {
	return appendURLPath(a.URL, common.DexCallbackEndpoint)
}

// DexOAuth2ClientSecret calculates an arbitrary, but predictable OAuth2 client secret string derived
// from the server secret. This is called by the dex startup wrapper (athena-dex rundex), as well
// as the API server, such that they both independently come to the same conclusion of what the
// OAuth2 shared client secret should be.
func (a *AthenaSettings) DexOAuth2ClientSecret() string {
	h := sha256.New()
	_, err := h.Write(a.ServerSignature)
	if err != nil {
		panic(err)
	}
	sha := h.Sum(nil)
	return base64.URLEncoding.EncodeToString(sha)[:40]
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
		log.Warn("Generated transient JWT secret because ATHENA_JWT_SECRET is not set. Existing sessions will be invalid after restart.")
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

// ReplaceMapSecrets takes a json object and recursively looks for any secret key references in the
// object and replaces the value with the secret value
func ReplaceMapSecrets(obj map[string]any, secretValues map[string]string) map[string]any {
	newObj := make(map[string]any)
	for k, v := range obj {
		switch val := v.(type) {
		case map[string]any:
			newObj[k] = ReplaceMapSecrets(val, secretValues)
		case []any:
			newObj[k] = replaceListSecrets(val, secretValues)
		case string:
			newObj[k] = ReplaceStringSecret(val, secretValues)
		default:
			newObj[k] = val
		}
	}
	return newObj
}

func replaceListSecrets(obj []any, secretValues map[string]string) []any {
	newObj := make([]any, len(obj))
	for i, v := range obj {
		switch val := v.(type) {
		case map[string]any:
			newObj[i] = ReplaceMapSecrets(val, secretValues)
		case []any:
			newObj[i] = replaceListSecrets(val, secretValues)
		case string:
			newObj[i] = ReplaceStringSecret(val, secretValues)
		default:
			newObj[i] = val
		}
	}
	return newObj
}

// ReplaceStringSecret checks if given string is a secret key reference ( starts with $ ) and returns corresponding value from provided map
func ReplaceStringSecret(val string, secretValues map[string]string) string {
	if val == "" || !strings.HasPrefix(val, "$") {
		return val
	}
	secretKey := val[1:]
	secretVal, ok := secretValues[secretKey]
	if !ok {
		log.Warnf("config referenced '%s', but key does not exist in secret", val)
		return val
	}
	return strings.TrimSpace(secretVal)
}
