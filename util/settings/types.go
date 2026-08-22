package settings

import (
	"context"
	"sync"
	"time"
)

// AthenaSettings holds in-memory runtime configuration options.
type AthenaSettings struct {
	// URL is the externally facing URL users will visit to reach Athena.
	URL string `json:"url,omitempty"`
	// URLs is a list of externally facing URLs users will visit to reach Athena.
	AdditionalURLs []string `json:"additionalUrls,omitempty"`
	// Indicates if status badge is enabled or not.
	StatusBadgeEnabled bool `json:"statusBadgeEnable"`
	// Indicates if status badge custom root URL should be used.
	StatusBadgeRootUrl string `json:"statusBadgeRootUrl,omitempty"` //nolint:revive //FIXME(var-naming)
	// ServerSignature holds the key used to generate JWT tokens.
	ServerSignature []byte `json:"serverSignature,omitempty"`
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
	ctx      context.Context
	raw      RawSettings
	settings AthenaSettings
	help     Help
	accounts map[string]Account
	// accountLoginDefaults contains the immutable environment baseline consumed
	// by accountaccess.Controller. Effective access never lives in settings.
	accountLoginDefaults map[string]bool
	mutex                *sync.RWMutex
}

const (
	// initialPasswordLength defines the length of the generated initial password
	initialPasswordLength = 16
)
