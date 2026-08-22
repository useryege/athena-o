package accountaccess

import (
	"context"
	"fmt"
	"math"
	"sync"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/useryege/athena/common"
)

// Store persists complete access aggregates with optimistic concurrency.
// Every returned persisted aggregate must contain all product modules.
type Store interface {
	ListAccountAccessOverrides(ctx context.Context) (map[string]Access, error)
	UpdateAccountAccessOverride(ctx context.Context, name string, next Access, expectedRevision uint64) (Access, error)
}

// Controller is the sole process-local source of effective account access.
type Controller struct {
	mu      sync.RWMutex
	store   Store
	access  map[string]Access
	updates sync.Mutex
}

// NewController combines environment login defaults with durable, full-row
// overrides. Ordinary accounts have no module access until explicitly granted.
func NewController(ctx context.Context, loginDefaults map[string]bool, store Store) (*Controller, error) {
	if store == nil {
		return nil, fmt.Errorf("account-access store is required")
	}
	if _, ok := loginDefaults[common.AthenaAdminUsername]; !ok {
		return nil, fmt.Errorf("built-in administrator account is not configured")
	}

	effective := make(map[string]Access, len(loginDefaults))
	for name, loginEnabled := range loginDefaults {
		effective[name] = Access{
			LoginEnabled: loginEnabled,
			Modules:      NoModuleAccess(),
		}
	}
	effective[common.AthenaAdminUsername] = Access{
		LoginEnabled: true,
		Modules:      MaximumModuleAccess(),
	}

	overrides, err := store.ListAccountAccessOverrides(ctx)
	if err != nil {
		return nil, fmt.Errorf("load account access overrides: %w", err)
	}
	for name, override := range overrides {
		if name == common.AthenaAdminUsername {
			continue
		}
		if _, ok := effective[name]; !ok {
			continue
		}
		if override.Revision == 0 {
			return nil, fmt.Errorf("account %q has invalid persisted access revision 0", name)
		}
		if err := override.Validate(); err != nil {
			return nil, fmt.Errorf("account %q has invalid persisted access: %w", name, err)
		}
		effective[name] = override.Clone()
	}

	return &Controller{store: store, access: effective}, nil
}

// Get returns a detached copy of the current effective access for an account.
func (c *Controller) Get(name string) (Access, error) {
	if c == nil {
		return Access{}, fmt.Errorf("account access controller is not configured")
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	access, ok := c.access[name]
	if !ok {
		return Access{}, status.Errorf(codes.NotFound, "account %q does not exist", name)
	}
	return access.Clone(), nil
}

// List returns a detached point-in-time copy of every configured account's access.
func (c *Controller) List() map[string]Access {
	if c == nil {
		return nil
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	result := make(map[string]Access, len(c.access))
	for name, access := range c.access {
		result[name] = access.Clone()
	}
	return result
}

// Update replaces an ordinary account's complete access aggregate. The store
// commit succeeds before the new snapshot becomes visible to authentication.
func (c *Controller) Update(ctx context.Context, name string, next Access, expectedRevision uint64) (Access, error) {
	if c == nil || c.store == nil {
		return Access{}, fmt.Errorf("account access controller is not configured")
	}
	if name == common.AthenaAdminUsername {
		return Access{}, status.Error(codes.InvalidArgument, "admin account access is fixed")
	}
	if next.Revision != expectedRevision {
		return Access{}, status.Error(codes.InvalidArgument, "account access revision must match expected revision")
	}
	if expectedRevision > math.MaxInt64 {
		return Access{}, status.Error(codes.InvalidArgument, "account access revision is out of range")
	}
	next = next.Clone()
	if err := next.Validate(); err != nil {
		return Access{}, err
	}

	c.updates.Lock()
	defer c.updates.Unlock()

	c.mu.RLock()
	current, exists := c.access[name]
	c.mu.RUnlock()
	if !exists {
		return Access{}, status.Errorf(codes.NotFound, "account %q does not exist", name)
	}
	if current.Revision != expectedRevision {
		return Access{}, ErrRevisionConflict
	}

	persisted, err := c.store.UpdateAccountAccessOverride(ctx, name, next.Clone(), expectedRevision)
	if err != nil {
		return Access{}, err
	}
	if persisted.Revision <= expectedRevision {
		return Access{}, fmt.Errorf("store returned non-incremented account access revision for %q", name)
	}
	if err := persisted.Validate(); err != nil {
		return Access{}, fmt.Errorf("store returned invalid account access for %q: %w", name, err)
	}
	persisted = persisted.Clone()

	c.mu.Lock()
	c.access[name] = persisted
	c.mu.Unlock()
	return persisted.Clone(), nil
}

// Authorize checks the current snapshot for an administrator or module rule.
func (c *Controller) Authorize(name string, requirement Requirement) error {
	if err := requirement.validate(); err != nil {
		return err
	}
	access, err := c.Get(name)
	if err != nil {
		return err
	}
	if name == common.AthenaAdminUsername {
		return nil
	}
	if requirement.Administrator {
		return ErrAdministratorAccessDenied
	}

	effective, exists := access.Modules[requirement.Module]
	if !exists {
		return status.Errorf(codes.Internal, "account %q is missing access for module %q", name, requirement.Module)
	}
	if accessLevelSatisfies(effective, requirement.AccessLevel) {
		return nil
	}
	return moduleAccessDeniedError(requirement.Module, requirement.AccessLevel, effective)
}
