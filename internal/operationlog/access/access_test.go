package access

import (
	"context"
	"errors"
	"testing"
)

func TestCheckerRequiresPersistentAdministratorAndLogin(t *testing.T) {
	checker := CheckerFunc(func(context.Context, string) (State, error) {
		return State{Administrator: true, LoginEnabled: false, Revision: 3}, nil
	})
	err := checker.Check(context.Background(), "a")
	if !errors.Is(err, ErrForbidden) {
		t.Fatalf("got %v", err)
	}
	checker = CheckerFunc(func(context.Context, string) (State, error) { return State{}, ErrStoreUnavailable })
	if err := checker.Check(context.Background(), "a"); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("got %v", err)
	}
	checker = CheckerFunc(func(context.Context, string) (State, error) {
		return State{Administrator: true, LoginEnabled: true, Revision: 4}, nil
	})
	if err := checker.Check(context.Background(), "a"); err != nil {
		t.Fatal(err)
	}
}
