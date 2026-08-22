package settings

import (
	"context"

	"github.com/useryege/athena/internal/accountaccess"
	settingspkg "github.com/useryege/athena/pkg/apiclient/settings"
	sessionmgr "github.com/useryege/athena/util/session"
	"github.com/useryege/athena/util/settings"
)

// Projector builds the settings representation returned by application bootstrap.
type Projector struct {
	mgr    *settings.SettingsManager
	access *accountaccess.Controller
	// appsInAnyNamespaceEnabled bool
	// hydratorEnabled        bool
	// syncWithReplaceAllowed bool
}

// NewProjector creates the transport-independent settings projector.
func NewProjector(mgr *settings.SettingsManager, access *accountaccess.Controller) *Projector {
	return &Projector{mgr: mgr, access: access}
}

// Project returns settings visible to the identity, if any, in ctx.
func (p *Projector) Project(ctx context.Context) (*settingspkg.Settings, error) {
	athenaSettings, err := p.mgr.GetSettings()
	if err != nil {
		return nil, err
	}
	// gaSettings, err := p.mgr.GetGoogleAnalytics()
	// if err != nil {
	// 	return nil, err
	// }
	help, err := p.mgr.GetHelp()
	if err != nil {
		return nil, err
	}
	userLoginsDisabled := true
	accounts, err := p.mgr.GetAccounts()
	if err != nil {
		return nil, err
	}
	for name, account := range accounts {
		access, err := p.access.Get(name)
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

	if sessionmgr.LoggedIn(ctx) {
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
