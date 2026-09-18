package access

import (
	"context"
	"github.com/useryege/athena/internal/accountaccess"
	"testing"
)

type accountReaderFunc func(context.Context, string) (accountaccess.Access, error)

func (f accountReaderFunc) GetAccountAccess(c context.Context, id string) (accountaccess.Access, error) {
	return f(c, id)
}
func TestAccountStateCheckerReadsDurableAdministrator(t *testing.T) {
	c := AccountStateChecker{Reader: accountReaderFunc(func(context.Context, string) (accountaccess.Access, error) {
		return accountaccess.Access{Administrator: true, LoginEnabled: true}, nil
	})}
	if err := c.Check(context.Background(), "admin"); err != nil {
		t.Fatal(err)
	}
}
