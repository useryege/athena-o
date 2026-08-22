package settings

import (
	"context"

	"github.com/useryege/athena/internal/accountaccess"
	sessionmgr "github.com/useryege/athena/util/session"

	settingspkg "github.com/useryege/athena/pkg/apiclient/settings"
	"github.com/useryege/athena/util/settings"
)

// Server provides a Settings service
type Server struct {
	mgr           *settings.SettingsManager
	access        *accountaccess.Controller
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
func NewServer(mgr *settings.SettingsManager, access *accountaccess.Controller, authenticator Authenticator, disableAuth bool) *Server {
	return &Server{mgr: mgr, access: access, authenticator: authenticator, disableAuth: disableAuth}
}

// Get returns Athena settings
func (s *Server) Get(ctx context.Context, _ *settingspkg.SettingsQuery) (*settingspkg.Settings, error) {
	athenaSettings, err := s.mgr.GetSettings()
	if err != nil {
		return nil, err
	}
	// gaSettings, err := s.mgr.GetGoogleAnalytics()
	// if err != nil {
	// 	return nil, err
	// }
	help, err := s.mgr.GetHelp()
	if err != nil {
		return nil, err
	}
	userLoginsDisabled := true
	accounts, err := s.mgr.GetAccounts()
	if err != nil {
		return nil, err
	}
	for name, account := range accounts {
		access, err := s.access.Get(name)
		if err != nil {
			return nil, err
		}
		if access.LoginEnabled && account.HasCapability(settings.AccountCapabilityLogin) {
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

	settings := settingspkg.Settings{
		URL:                athenaSettings.URL,
		AdditionalURLs:     athenaSettings.AdditionalURLs,
		StatusBadgeEnabled: athenaSettings.StatusBadgeEnabled,
		StatusBadgeRootUrl: athenaSettings.StatusBadgeRootUrl,
		GoogleAnalytics:    &settingspkg.GoogleAnalyticsConfig{
			// TrackingID:     gaSettings.TrackingID,
			// AnonymizeUsers: gaSettings.AnonymizeUsers,
		},
		Help: &settingspkg.Help{
			ChatUrl:    help.ChatURL,
			ChatText:   help.ChatText,
			BinaryUrls: help.BinaryURLs,
		},
		UserLoginsDisabled: userLoginsDisabled,
		// KustomizeVersions:  kustomizeVersions,
		UiCssURL:    athenaSettings.UiCssURL,
		ExecEnabled: athenaSettings.ExecEnabled,
		// HydratorEnabled:        s.hydratorEnabled,
		// SyncWithReplaceAllowed: s.syncWithReplaceAllowed,
	}

	if sessionmgr.LoggedIn(ctx) || s.disableAuth {
		settings.UiBannerContent = athenaSettings.UiBannerContent
		settings.UiBannerURL = athenaSettings.UiBannerURL
		settings.UiBannerPermanent = athenaSettings.UiBannerPermanent
		settings.UiBannerPosition = athenaSettings.UiBannerPosition
	}
	if sessionmgr.LoggedIn(ctx) {
		settings.PasswordPattern = athenaSettings.PasswordPattern
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
