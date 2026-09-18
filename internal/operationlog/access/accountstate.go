package access

import (
	"context"
	"github.com/useryege/athena/internal/accountaccess"
)

// AccountReader is the narrow durable account-state contract used by the
// operation-log boundary. It intentionally excludes runtime/navigation state.
type AccountReader interface {
	GetAccountAccess(context.Context, string) (accountaccess.Access, error)
}
type AccountStateChecker struct{ Reader AccountReader }

func (c AccountStateChecker) Check(ctx context.Context, accountID string) error {
	if c.Reader == nil {
		return ErrUnavailable
	}
	a, err := c.Reader.GetAccountAccess(ctx, accountID)
	if err != nil {
		return ErrUnavailable
	}
	if !a.Administrator || !a.LoginEnabled {
		return ErrForbidden
	}
	return nil
}
