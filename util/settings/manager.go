package settings

import "context"

// NewSettingsManagerFromEnv loads immutable runtime and Help configuration.
func NewSettingsManagerFromEnv(_ context.Context) (*SettingsManager, error) {
	settings, err := loadSettingsFromEnv()
	if err != nil {
		return nil, err
	}
	return &SettingsManager{settings: settings, help: loadHelpFromEnv()}, nil
}

// GetHelp returns a deep copy of the startup Help configuration.
func (mgr *SettingsManager) GetHelp() (*Help, error) {
	help := mgr.help
	help.BinaryURLs = copyStringMap(help.BinaryURLs)
	return &help, nil
}

// GetSettings returns a deep copy of the runtime settings loaded at startup.
func (mgr *SettingsManager) GetSettings() (*AthenaSettings, error) {
	settings := mgr.settings
	settings.AdditionalURLs = append([]string(nil), settings.AdditionalURLs...)
	settings.BinaryUrls = copyStringMap(settings.BinaryUrls)
	return &settings, nil
}

func copyStringMap(source map[string]string) map[string]string {
	copy := make(map[string]string, len(source))
	for key, value := range source {
		copy[key] = value
	}
	return copy
}
