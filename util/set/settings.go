package set

import (
	"context"
)

type AthenaSettings struct {
}

type SettingsManager interface {
	GetSettings() (*AthenaSettings, error)
}

type settingsManager struct {
	ctx context.Context
}

func NewSettingsManager(ctx context.Context) SettingsManager {
	return &settingsManager{ctx: ctx}
}

func (s *settingsManager) GetSettings() (*AthenaSettings, error) {
	return &AthenaSettings{}, nil
}
