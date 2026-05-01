package settings

import (
	"context"

	"sigs.k8s.io/yaml"

	sessionmgr "github.com/useryege/athena/util/session"

	settingspkg "github.com/useryege/athena/pkg/apiclient/settings"
	"github.com/useryege/athena/pkg/apis/application/v1alpha1"
	"github.com/useryege/athena/util/settings"
)

// Server provides a Settings service
type Server struct {
	mgr           *settings.SettingsManager
	authenticator Authenticator
	disableAuth   bool
	// appsInAnyNamespaceEnabled bool
	// hydratorEnabled        bool
	// syncWithReplaceAllowed bool
}

type Authenticator interface {
	Authenticate(ctx context.Context) (context.Context, error)
}

// NewServer returns a new instance of the Settings service
func NewServer(mgr *settings.SettingsManager, authenticator Authenticator, disableAuth bool) *Server {
	return &Server{mgr: mgr, authenticator: authenticator, disableAuth: disableAuth}
}

// Get returns Athena settings
func (s *Server) Get(ctx context.Context, _ *settingspkg.SettingsQuery) (*settingspkg.Settings, error) {
	resourceOverrides, err := s.mgr.GetResourceOverrides()
	if err != nil {
		return nil, err
	}
	overrides := make(map[string]*v1alpha1.ResourceOverride)
	for k := range resourceOverrides {
		val := resourceOverrides[k]
		overrides[k] = &val
	}
	appInstanceLabelKey, err := s.mgr.GetAppInstanceLabelKey()
	if err != nil {
		return nil, err
	}
	athenaSettings, err := s.mgr.GetSettings()
	if err != nil {
		return nil, err
	}
	gaSettings, err := s.mgr.GetGoogleAnalytics()
	if err != nil {
		return nil, err
	}
	help, err := s.mgr.GetHelp()
	if err != nil {
		return nil, err
	}
	userLoginsDisabled := true
	accounts, err := s.mgr.GetAccounts()
	if err != nil {
		return nil, err
	}
	for _, account := range accounts {
		if account.Enabled && account.HasCapability(settings.AccountCapabilityLogin) {
			userLoginsDisabled = false
			break
		}
	}

	// kustomizeSettings, err := s.mgr.GetKustomizeSettings()
	// if err != nil {
	// 	return nil, err
	// }
	// var kustomizeVersions []string
	// for i := range kustomizeSettings.Versions {
	// 	kustomizeVersions = append(kustomizeVersions, kustomizeSettings.Versions[i].Name)
	// }

	// trackingMethod, err := s.mgr.GetTrackingMethod()
	// if err != nil {
	// 	return nil, err
	// }

	installationID, err := s.mgr.GetInstallationID()
	if err != nil {
		return nil, err
	}

	settings := settingspkg.Settings{
		URL:                athenaSettings.URL,
		AdditionalURLs:     athenaSettings.AdditionalURLs,
		AppLabelKey:        appInstanceLabelKey,
		StatusBadgeEnabled: athenaSettings.StatusBadgeEnabled,
		StatusBadgeRootUrl: athenaSettings.StatusBadgeRootUrl,
		// KustomizeOptions: &v1alpha1.KustomizeOptions{
		// 	BuildOptions: athenaSettings.KustomizeBuildOptions,
		// },
		GoogleAnalytics: &settingspkg.GoogleAnalyticsConfig{
			TrackingID:     gaSettings.TrackingID,
			AnonymizeUsers: gaSettings.AnonymizeUsers,
		},
		Help: &settingspkg.Help{
			ChatUrl:    help.ChatURL,
			ChatText:   help.ChatText,
			BinaryUrls: help.BinaryURLs,
		},
		UserLoginsDisabled: userLoginsDisabled,
		// KustomizeVersions:  kustomizeVersions,
		UiCssURL: athenaSettings.UiCssURL,
		// TrackingMethod: trackingMethod,
		InstallationID: installationID,
		ExecEnabled:    athenaSettings.ExecEnabled,
		// AppsInAnyNamespaceEnabled: s.appsInAnyNamespaceEnabled,
		ImpersonationEnabled: athenaSettings.ImpersonationEnabled,
		// HydratorEnabled:        s.hydratorEnabled,
		// SyncWithReplaceAllowed: s.syncWithReplaceAllowed,
	}

	if sessionmgr.LoggedIn(ctx) || s.disableAuth {
		settings.UiBannerContent = athenaSettings.UiBannerContent
		settings.UiBannerURL = athenaSettings.UiBannerURL
		settings.UiBannerPermanent = athenaSettings.UiBannerPermanent
		settings.UiBannerPosition = athenaSettings.UiBannerPosition
		settings.ControllerNamespace = s.mgr.GetNamespace()
		settings.ResourceOverrides = overrides
	}
	if sessionmgr.LoggedIn(ctx) {
		settings.PasswordPattern = athenaSettings.PasswordPattern
	}
	if athenaSettings.DexConfig != "" {
		var cfg settingspkg.DexConfig
		err = yaml.Unmarshal([]byte(athenaSettings.DexConfig), &cfg)
		if err == nil {
			settings.DexConfig = &cfg
		}
	}
	if oidcConfig := athenaSettings.OIDCConfig(); oidcConfig != nil {
		settings.OIDCConfig = &settingspkg.OIDCConfig{
			Name:                     oidcConfig.Name,
			Issuer:                   oidcConfig.Issuer,
			ClientID:                 oidcConfig.ClientID,
			CLIClientID:              oidcConfig.CLIClientID,
			Scopes:                   oidcConfig.RequestedScopes,
			EnablePKCEAuthentication: oidcConfig.EnablePKCEAuthentication,
		}
		if len(athenaSettings.OIDCConfig().RequestedIDTokenClaims) > 0 {
			settings.OIDCConfig.IDTokenClaims = athenaSettings.OIDCConfig().RequestedIDTokenClaims
		}
	}
	return &settings, nil
}

// AuthFuncOverride disables authentication for settings service
func (s *Server) AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error) {
	ctx, err := s.authenticator.Authenticate(ctx)
	if fullMethodName == "/cluster.SettingsService/Get" {
		// SettingsService/Get API is used by login page.
		// This authenticates the user, but ignores any error, so that we have claims populated
		err = nil
	}
	return ctx, err
}
