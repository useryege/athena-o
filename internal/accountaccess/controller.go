package accountaccess

import (
	"context"
	"fmt"
	"math"
	"sync"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Store persists complete access aggregates with optimistic concurrency.
// Every returned aggregate must contain all product modules.
type Store interface {
	ListAccountAccess(ctx context.Context) (map[string]Access, error)
	GetAccountAccess(ctx context.Context, accountID string) (Access, error)
	UpdateAccountAccess(ctx context.Context, accountID string, next Access, expectedRevision uint64) (Access, error)
}

// Controller is the sole process-local source of effective account access.
type Controller struct {
	mu      sync.RWMutex
	store   Store
	access  map[string]Access
	updates sync.Map
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
	for accountID, access := range effective {
		if access.Revision == 0 {
			return nil, fmt.Errorf("account %q has invalid persisted access revision 0", accountID)
		}
		if err := access.Validate(); err != nil {
			return nil, fmt.Errorf("account %q has invalid persisted access: %w", accountID, err)
		}
		effective[accountID] = access.Clone()
	}
	return &Controller{store: store, access: effective}, nil
}

// Register publishes a newly committed durable account into the current
// process snapshot. A newer revision already present is never overwritten.
func (c *Controller) Register(ctx context.Context, accountID string) (Access, error) {
	if c == nil || c.store == nil {
		return Access{}, fmt.Errorf("account access controller is not configured")
	}
	persisted, err := c.store.GetAccountAccess(ctx, accountID)
	if err != nil {
		return Access{}, err
	}
	if persisted.Revision == 0 {
		return Access{}, fmt.Errorf("account %q has invalid persisted access revision 0", accountID)
	}
	if err := persisted.Validate(); err != nil {
		return Access{}, fmt.Errorf("account %q has invalid persisted access: %w", accountID, err)
	}
	persisted = persisted.Clone()
	c.mu.Lock()
	current, exists := c.access[accountID]
	if !exists || current.Revision < persisted.Revision {
		c.access[accountID] = persisted
		current = persisted
	}
	c.mu.Unlock()
	return current.Clone(), nil
}

// Get returns a detached copy of the current effective access for an account.
func (c *Controller) Get(accountID string) (Access, error) {
	if c == nil {
		return Access{}, fmt.Errorf("account access controller is not configured")
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	access, ok := c.access[accountID]
	if !ok {
		return Access{}, status.Errorf(codes.NotFound, "account %q does not exist", accountID)
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
	for accountID, access := range c.access {
		result[accountID] = access.Clone()
	}
	return result
}

// Update replaces an ordinary account's complete access aggregate. The store
// commit succeeds before the new snapshot becomes visible to authentication.
func (c *Controller) Update(ctx context.Context, accountID string, next Access, expectedRevision uint64) (Access, error) {
	if c == nil || c.store == nil {
		return Access{}, fmt.Errorf("account access controller is not configured")
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

	lock, _ := c.updates.LoadOrStore(accountID, &sync.Mutex{})
	updateLock := lock.(*sync.Mutex)
	updateLock.Lock()
	defer updateLock.Unlock()

	c.mu.RLock()
	current, exists := c.access[accountID]
	c.mu.RUnlock()
	if !exists {
		return Access{}, status.Errorf(codes.NotFound, "account %q does not exist", accountID)
	}
	if current.Administrator {
		return Access{}, status.Error(codes.InvalidArgument, "administrator account access is fixed")
	}
	if current.Revision != expectedRevision {
		return Access{}, ErrRevisionConflict
	}

	persisted, err := c.store.UpdateAccountAccess(ctx, accountID, next.Clone(), expectedRevision)
	if err != nil {
		return Access{}, err
	}
	if persisted.Revision <= expectedRevision {
		return Access{}, fmt.Errorf("store returned non-incremented account access revision for %q", accountID)
	}
	if err := persisted.Validate(); err != nil {
		return Access{}, fmt.Errorf("store returned invalid account access for %q: %w", accountID, err)
	}
	persisted = persisted.Clone()

	c.mu.Lock()
	c.access[accountID] = persisted
	c.mu.Unlock()
	return persisted.Clone(), nil
}

// Authorize checks the current snapshot for one explicit role, entitlement, or
// module rule. Administrator status does not imply member business access.
func (c *Controller) Authorize(accountID string, requirement Requirement) error {
	if err := requirement.validate(); err != nil {
		return err
	}
	access, err := c.Get(accountID)
	if err != nil {
		return err
	}
	if requirement.Administrator {
		if access.Administrator {
			return nil
		}
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
		return status.Errorf(codes.Internal, "account %q is missing access for module %q", accountID, requirement.Module)
	}
	if accessLevelSatisfies(effective, requirement.AccessLevel) {
		return nil
	}
	return moduleAccessDeniedError(requirement.Module, requirement.AccessLevel, effective)
}
