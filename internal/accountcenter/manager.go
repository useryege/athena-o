package accountcenter

import (
	"context"
	"fmt"
	"math"
	"strings"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Store persists profile and preference aggregates with independent CAS revisions.
type Store interface {
	AccountExists(ctx context.Context, name string) (bool, error)
	GetProfile(ctx context.Context, name string) (Profile, bool, error)
	ListAvatarObjectKeys(ctx context.Context) ([]string, error)
	UpdateProfile(ctx context.Context, name string, next Profile, expectedRevision uint64) (Profile, error)
	GetPreferences(ctx context.Context, name string) (Preferences, bool, error)
	UpdatePreferences(ctx context.Context, name string, next Preferences, expectedRevision uint64) (Preferences, error)
}

// Manager provides uncached account-center state for durable accounts.
type Manager struct {
	store Store
}

func NewManager(store Store) (*Manager, error) {
	if store == nil {
		return nil, fmt.Errorf("account-center store is required")
	}
	return &Manager{store: store}, nil
}

func (m *Manager) requireAccount(ctx context.Context, name string) error {
	if m == nil || m.store == nil {
		return status.Error(codes.Internal, "account center is not configured")
	}
	exists, err := m.store.AccountExists(ctx, name)
	if err != nil {
		return err
	}
	if !exists {
		return status.Errorf(codes.NotFound, "account %q does not exist", name)
	}
	return nil
}

func defaultProfile(name string) Profile {
	return Profile{DisplayName: name, Tier: TierStandard}
}

func defaultPreferences() Preferences {
	return Preferences{Theme: ThemeModeSystem}
}

func (m *Manager) GetProfile(ctx context.Context, name string) (Profile, error) {
	if err := m.requireAccount(ctx, name); err != nil {
		return Profile{}, err
	}
	profile, found, err := m.store.GetProfile(ctx, name)
	if err != nil {
		return Profile{}, err
	}
	if !found {
		return defaultProfile(name), nil
	}
	if profile.Revision == 0 {
		return Profile{}, fmt.Errorf("account %q has persisted profile revision 0", name)
	}
	if err := validateProfile(profile); err != nil {
		return Profile{}, fmt.Errorf("account %q has invalid persisted profile: %w", name, err)
	}
	return profile, nil
}

// ListAvatarObjectKeys returns every currently referenced private avatar key.
// It is intended for object-store orphan collection and does not expose profiles.
func (m *Manager) ListAvatarObjectKeys(ctx context.Context) ([]string, error) {
	if m == nil || m.store == nil {
		return nil, status.Error(codes.Internal, "account center is not configured")
	}
	return m.store.ListAvatarObjectKeys(ctx)
}

func (m *Manager) GetPreferences(ctx context.Context, name string) (Preferences, error) {
	if err := m.requireAccount(ctx, name); err != nil {
		return Preferences{}, err
	}
	preferences, found, err := m.store.GetPreferences(ctx, name)
	if err != nil {
		return Preferences{}, err
	}
	if !found {
		return defaultPreferences(), nil
	}
	if preferences.Revision == 0 {
		return Preferences{}, fmt.Errorf("account %q has persisted preferences revision 0", name)
	}
	if err := validateThemeMode(preferences.Theme); err != nil {
		return Preferences{}, fmt.Errorf("account %q has invalid persisted preferences: %w", name, err)
	}
	return preferences, nil
}

func (m *Manager) UpdateDisplayName(ctx context.Context, name, displayName string, expectedRevision uint64) (Profile, error) {
	displayName = strings.TrimSpace(displayName)
	if err := validateDisplayName(displayName); err != nil {
		return Profile{}, err
	}
	return m.updateProfile(ctx, name, expectedRevision, func(profile *Profile) {
		profile.DisplayName = displayName
	})
}

func (m *Manager) UpdateTier(ctx context.Context, name string, tier Tier, expectedRevision uint64) (Profile, error) {
	if err := validateTier(tier); err != nil {
		return Profile{}, err
	}
	return m.updateProfile(ctx, name, expectedRevision, func(profile *Profile) {
		profile.Tier = tier
	})
}

func (m *Manager) ReplaceAvatar(ctx context.Context, name string, avatar AvatarMetadata, expectedRevision uint64) (Profile, error) {
	if avatar.Empty() {
		return Profile{}, status.Error(codes.InvalidArgument, "avatar object metadata is required")
	}
	if err := avatar.validate(); err != nil {
		return Profile{}, err
	}
	return m.updateProfile(ctx, name, expectedRevision, func(profile *Profile) {
		profile.Avatar = avatar
	})
}

func (m *Manager) DeleteAvatar(ctx context.Context, name string, expectedRevision uint64) (Profile, error) {
	return m.updateProfile(ctx, name, expectedRevision, func(profile *Profile) {
		profile.Avatar = AvatarMetadata{}
	})
}

func (m *Manager) updateProfile(ctx context.Context, name string, expectedRevision uint64, mutate func(*Profile)) (Profile, error) {
	if expectedRevision >= math.MaxInt64 {
		return Profile{}, status.Error(codes.InvalidArgument, "account profile revision is out of range")
	}
	current, err := m.GetProfile(ctx, name)
	if err != nil {
		return Profile{}, err
	}
	if current.Revision != expectedRevision {
		return Profile{}, ErrProfileRevisionConflict
	}
	mutate(&current)
	if err := validateProfile(current); err != nil {
		return Profile{}, err
	}
	return m.store.UpdateProfile(ctx, name, current, expectedRevision)
}

func (m *Manager) UpdatePreferences(ctx context.Context, name string, theme ThemeMode, expectedRevision uint64) (Preferences, error) {
	if expectedRevision >= math.MaxInt64 {
		return Preferences{}, status.Error(codes.InvalidArgument, "account preferences revision is out of range")
	}
	if err := validateThemeMode(theme); err != nil {
		return Preferences{}, err
	}
	current, err := m.GetPreferences(ctx, name)
	if err != nil {
		return Preferences{}, err
	}
	if current.Revision != expectedRevision {
		return Preferences{}, ErrPreferencesRevisionConflict
	}
	current.Theme = theme
	return m.store.UpdatePreferences(ctx, name, current, expectedRevision)
}
