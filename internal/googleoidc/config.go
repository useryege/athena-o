package googleoidc

import (
	"fmt"
	"net"
	"net/url"
	"os"
	"strings"

	"github.com/useryege/athena/internal/authregistration"
)

const (
	envClientID         = "ATHENA_GOOGLE_OIDC_CLIENT_ID"
	envClientSecret     = "ATHENA_GOOGLE_OIDC_CLIENT_SECRET"
	envRedirectURI      = "ATHENA_GOOGLE_OIDC_REDIRECT_URI"
	envAdminEmail       = "ATHENA_ADMIN_GOOGLE_EMAIL"
	callbackPath        = "/auth/google/callback"
	googleAuthorization = "https://accounts.google.com/o/oauth2/v2/auth"
	googleToken         = "https://oauth2.googleapis.com/token"
	googleJWKS          = "https://www.googleapis.com/oauth2/v3/certs"
)

// Config contains the fixed Google Web OAuth client settings.
type Config struct {
	ClientID     string
	ClientSecret string
	RedirectURI  string
	AdminEmail   string
	secureCookie bool
}

// PublicOrigin returns the canonical authority used by same-origin wallet
// authentication. It is derived only from the already validated redirect URI,
// never from request Host or forwarding headers.
func (config Config) PublicOrigin() string {
	redirect, err := url.Parse(config.RedirectURI)
	if err != nil {
		return ""
	}
	scheme := strings.ToLower(redirect.Scheme)
	hostname := strings.ToLower(redirect.Hostname())
	port := redirect.Port()
	if (scheme == "https" && port == "443") || (scheme == "http" && port == "80") {
		port = ""
	}
	authority := hostname
	if port != "" {
		authority = net.JoinHostPort(hostname, port)
	} else if strings.Contains(hostname, ":") {
		authority = "[" + hostname + "]"
	}
	return scheme + "://" + authority
}

// SecureCookie reports the cookie transport policy derived from RedirectURI.
func (config Config) SecureCookie() bool {
	return config.secureCookie
}

// LoadConfigFromEnv reads and validates Google OIDC settings without making a
// network request. Client secret supports the standard _FILE form.
func LoadConfigFromEnv(baseHRef string) (Config, error) {
	secret, err := envOrFile(envClientSecret)
	if err != nil {
		return Config{}, err
	}
	config := Config{
		ClientID:     strings.TrimSpace(os.Getenv(envClientID)),
		ClientSecret: secret,
		RedirectURI:  strings.TrimSpace(os.Getenv(envRedirectURI)),
		AdminEmail:   strings.TrimSpace(os.Getenv(envAdminEmail)),
	}
	if config.ClientID == "" {
		return Config{}, fmt.Errorf("%s is required", envClientID)
	}
	if config.ClientSecret == "" {
		return Config{}, fmt.Errorf("%s or %s_FILE is required", envClientSecret, envClientSecret)
	}
	if config.AdminEmail == "" {
		return Config{}, fmt.Errorf("%s is required", envAdminEmail)
	}
	redirect, err := url.Parse(config.RedirectURI)
	if err != nil || !redirect.IsAbs() || redirect.Host == "" || redirect.User != nil || redirect.Opaque != "" {
		return Config{}, fmt.Errorf("%s must be an absolute URI", envRedirectURI)
	}
	expectedCallbackPath := authregistration.DeploymentPath(baseHRef, callbackPath)
	expectedEscapedPath := (&url.URL{Path: expectedCallbackPath}).EscapedPath()
	if redirect.Path != expectedCallbackPath || redirect.EscapedPath() != expectedEscapedPath || redirect.ForceQuery || redirect.RawQuery != "" || redirect.Fragment != "" {
		return Config{}, fmt.Errorf("%s path must be exactly %s without query or fragment", envRedirectURI, expectedCallbackPath)
	}
	switch redirect.Scheme {
	case "https":
		config.secureCookie = true
	case "http":
		if redirect.Hostname() != "localhost" {
			return Config{}, fmt.Errorf("%s must use HTTPS outside localhost", envRedirectURI)
		}
	default:
		return Config{}, fmt.Errorf("%s must use HTTP or HTTPS", envRedirectURI)
	}
	return config, nil
}

func envOrFile(name string) (string, error) {
	if value := os.Getenv(name); value != "" {
		return strings.TrimSpace(value), nil
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
