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
// Every returned aggregate must contain all product modules.
type Store interface {
	ListAccountAccess(ctx context.Context) (map[string]Access, error)
	GetAccountAccess(ctx context.Context, name string) (Access, error)
	UpdateAccountAccess(ctx context.Context, name string, next Access, expectedRevision uint64) (Access, error)
}

// Controller is the sole process-local source of effective account access.
type Controller struct {
	mu      sync.RWMutex
	store   Store
	access  map[string]Access
	updates sync.Mutex
}

// NewController loads the durable, complete access state for every account.
func NewController(ctx context.Context, store Store) (*Controller, error) {
	if store == nil {
		return nil, fmt.Errorf("account-access store is required")
	}
	effective, err := store.ListAccountAccess(ctx)
	if err != nil {
		return nil, fmt.Errorf("load account access: %w", err)
	}
	for name, access := range effective {
		if access.Revision == 0 {
			return nil, fmt.Errorf("account %q has invalid persisted access revision 0", name)
		}
		if err := access.Validate(); err != nil {
			return nil, fmt.Errorf("account %q has invalid persisted access: %w", name, err)
		}
		effective[name] = access.Clone()
	}
	administrator, ok := effective[common.AthenaAdminUsername]
	if !ok {
		return nil, fmt.Errorf("built-in administrator account is not configured")
	}
	if !administrator.LoginEnabled || !administrator.ProfitSharingEnabled || administrator.APIKeyEnabled {
		return nil, fmt.Errorf("built-in administrator access flags are invalid")
	}
	for module, maximum := range MaximumModuleAccess() {
		if administrator.Modules[module] != maximum {
			return nil, fmt.Errorf("built-in administrator module %q is not at maximum access", module)
		}
	}

	return &Controller{store: store, access: effective}, nil
}

// Register publishes a newly committed durable account into the current
// process snapshot. A newer revision already present is never overwritten.
func (c *Controller) Register(ctx context.Context, name string) (Access, error) {
	if c == nil || c.store == nil {
		return Access{}, fmt.Errorf("account access controller is not configured")
	}
	persisted, err := c.store.GetAccountAccess(ctx, name)
	if err != nil {
		return Access{}, err
	}
	if persisted.Revision == 0 {
		return Access{}, fmt.Errorf("account %q has invalid persisted access revision 0", name)
	}
	if err := persisted.Validate(); err != nil {
		return Access{}, fmt.Errorf("account %q has invalid persisted access: %w", name, err)
	}
	persisted = persisted.Clone()
	c.mu.Lock()
	current, exists := c.access[name]
	if !exists || current.Revision < persisted.Revision {
		c.access[name] = persisted
		current = persisted
	}
	c.mu.Unlock()
	return current.Clone(), nil
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
	if expectedRevision >= math.MaxInt64 {
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

	persisted, err := c.store.UpdateAccountAccess(ctx, name, next.Clone(), expectedRevision)
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
	if requirement.ProfitSharing {
		if access.ProfitSharingEnabled {
			return nil
		}
		return ErrProfitSharingAccessDenied
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
