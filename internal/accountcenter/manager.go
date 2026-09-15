package accountcenter

import (
	"context"
	"fmt"
	"math"
	"strings"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Store persists profiles with optimistic concurrency revisions.
type Store interface {
	AccountExists(ctx context.Context, accountID string) (bool, error)
	GetProfile(ctx context.Context, accountID string) (Profile, bool, error)
	ListAvatarObjectKeys(ctx context.Context) ([]string, error)
	UpdateProfile(ctx context.Context, accountID string, next Profile, expectedRevision uint64) (Profile, error)
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

func (m *Manager) requireAccount(ctx context.Context, accountID string) error {
	if m == nil || m.store == nil {
		return status.Error(codes.Internal, "account center is not configured")
	}
	exists, err := m.store.AccountExists(ctx, accountID)
	if err != nil {
		return err
	}
	if !exists {
		return status.Errorf(codes.NotFound, "account %q does not exist", accountID)
	}
	return nil
}

func (m *Manager) GetProfile(ctx context.Context, accountID string) (Profile, error) {
	if err := m.requireAccount(ctx, accountID); err != nil {
		return Profile{}, err
	}
	profile, found, err := m.store.GetProfile(ctx, accountID)
	if err != nil {
		return Profile{}, err
	}
	if !found {
		return Profile{}, fmt.Errorf("account %q has no persisted profile", accountID)
	}
	if profile.Revision == 0 {
		return Profile{}, fmt.Errorf("account %q has persisted profile revision 0", accountID)
	}
	if err := validateProfile(profile); err != nil {
		return Profile{}, fmt.Errorf("account %q has invalid persisted profile: %w", accountID, err)
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

func (m *Manager) UpdateDisplayName(ctx context.Context, accountID, displayName string, expectedRevision uint64) (Profile, error) {
	displayName = strings.TrimSpace(displayName)
	if err := validateDisplayName(displayName); err != nil {
		return Profile{}, err
	}
	return m.updateProfile(ctx, accountID, expectedRevision, func(profile *Profile) {
		profile.DisplayName = displayName
	})
}

func (m *Manager) UpdateTier(ctx context.Context, accountID string, tier Tier, expectedRevision uint64) (Profile, error) {
	if err := validateTier(tier); err != nil {
		return Profile{}, err
	}
	return m.updateProfile(ctx, accountID, expectedRevision, func(profile *Profile) {
		profile.Tier = tier
	})
}

func (m *Manager) ReplaceAvatar(ctx context.Context, accountID string, avatar AvatarMetadata, expectedRevision uint64) (Profile, error) {
	if avatar.Empty() {
		return Profile{}, status.Error(codes.InvalidArgument, "avatar object metadata is required")
	}
	if err := avatar.validate(); err != nil {
		return Profile{}, err
	}
	return m.updateProfile(ctx, accountID, expectedRevision, func(profile *Profile) {
		profile.Avatar = avatar
	})
}

func (m *Manager) DeleteAvatar(ctx context.Context, accountID string, expectedRevision uint64) (Profile, error) {
	return m.updateProfile(ctx, accountID, expectedRevision, func(profile *Profile) {
		profile.Avatar = AvatarMetadata{}
	})
}

func (m *Manager) updateProfile(ctx context.Context, accountID string, expectedRevision uint64, mutate func(*Profile)) (Profile, error) {
	if expectedRevision >= math.MaxInt64 {
		return Profile{}, status.Error(codes.InvalidArgument, "account profile revision is out of range")
	}
	current, err := m.GetProfile(ctx, accountID)
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
	return m.store.UpdateProfile(ctx, accountID, current, expectedRevision)
}
