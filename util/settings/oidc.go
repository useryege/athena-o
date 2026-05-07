package settings

import (
	"time"

	log "github.com/sirupsen/logrus"
	"sigs.k8s.io/yaml"

	"github.com/useryege/athena/common"
	"github.com/useryege/athena/util/env"
)

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
