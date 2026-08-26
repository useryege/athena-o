package settings

import (
	"context"

	settingspkg "github.com/useryege/athena/pkg/apiclient/settings"
	sessionmgr "github.com/useryege/athena/util/session"
	"github.com/useryege/athena/util/settings"
)

// Projector builds the settings representation returned by application bootstrap.
type Projector struct {
	mgr *settings.SettingsManager
	// appsInAnyNamespaceEnabled bool
	// hydratorEnabled        bool
	// syncWithReplaceAllowed bool
}

// NewProjector creates the transport-independent settings projector.
func NewProjector(mgr *settings.SettingsManager) *Projector {
	return &Projector{mgr: mgr}
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
		UserLoginsDisabled: false,
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
	return &settings, nil
}
