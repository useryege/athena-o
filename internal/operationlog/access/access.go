// Package access verifies operation-log viewers against persistent account state.
package access

import (
	"context"
	"errors"
)

var (
	ErrForbidden        = errors.New("operation log viewer forbidden")
	ErrUnavailable      = errors.New("operation log account state unavailable")
	ErrStoreUnavailable = errors.New("account state store unavailable")
)

type State struct {
	Administrator bool
	LoginEnabled  bool
	Revision      uint64
}
type Reader interface {
	ReadAccountAccess(context.Context, string) (State, error)
}
type Checker interface {
	Check(context.Context, string) error
}
type CheckerFunc func(context.Context, string) (State, error)

func (f CheckerFunc) Check(ctx context.Context, id string) error {
	state, err := f(ctx, id)
	if err != nil {
		if errors.Is(err, ErrStoreUnavailable) {
			return ErrUnavailable
		}
		return err
	}
	if !state.Administrator || !state.LoginEnabled {
		return ErrForbidden
	}
	return nil
}

type PersistentChecker struct{ Reader Reader }

func (c PersistentChecker) Check(ctx context.Context, id string) error {
	if c.Reader == nil {
		return ErrUnavailable
	}
	state, err := c.Reader.ReadAccountAccess(ctx, id)
	if err != nil {
		return ErrUnavailable
	}
	if !state.Administrator || !state.LoginEnabled {
		return ErrForbidden
	}
	return nil
}
