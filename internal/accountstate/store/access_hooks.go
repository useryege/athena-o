package store

import (
	"context"
	"fmt"
	"github.com/jackc/pgx/v5"
	"github.com/useryege/athena/internal/accountaccess"
)

type AccessChangeHook func(context.Context, pgx.Tx, string, accountaccess.Access, accountaccess.Access) error

// SetAccessChangeHook is startup-only. Replacing a running hook is not supported.
func (s *SQLStore) SetAccessChangeHook(hook AccessChangeHook) {
	if s.accessChangeHook != nil {
		panic("access change hook already configured")
	}
	s.accessChangeHook = hook
}
func (s *SQLStore) RequireAccessChangeHook() error {
	if s == nil || s.accessChangeHook == nil {
		return fmt.Errorf("account access change hook is required")
	}
	return nil
}
