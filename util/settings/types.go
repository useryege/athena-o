package settings

import (
	"context"
	"crypto/tls"
	"sync"
	"time"

	"github.com/useryege/athena/internal/server/settings/oidc"
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
	ExecEnabled bool `json:"execEnabled"`
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

// oidcConfig is the same as the public OIDCConfig, except the public one excludes the AllowedAudiences and the
// SkipAudienceCheckWhenTokenHasNoAudience fields.
// AllowedAudiences should be accessed via AthenaSettings.OAuth2AllowedAudiences.
// SkipAudienceCheckWhenTokenHasNoAudience should be accessed via AthenaSettings.SkipAudienceCheckWhenTokenHasNoAudience.
type oidcConfig struct {
	OIDCConfig
	AllowedAudiences                        []string `json:"allowedAudiences,omitempty"`
	SkipAudienceCheckWhenTokenHasNoAudience *bool    `json:"skipAudienceCheckWhenTokenHasNoAudience,omitempty"`
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
