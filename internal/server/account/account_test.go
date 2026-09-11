package account

import (
	core "github.com/useryege/athena/internal/accountaccess"
	api "github.com/useryege/athena/pkg/apiclient/account"
	"testing"
)

func TestTraderSyncAPIMatrix(t *testing.T) {
	a := core.Access{Modules: core.MaximumModuleAccess()}
	wire := ToAPIAccountAccess(a)
	if len(wire.ModuleAccess) != 10 {
		t.Fatal(len(wire.ModuleAccess))
	}
	var sync *api.AccountModuleAccess
	for _, m := range wire.ModuleAccess {
		if m.Module == api.AccountDataModule(12) {
			sync = m
		}
	}
	if sync == nil {
		t.Fatal("Trader Sync omitted")
	}
	sync.DataAccess = api.AccountDataAccess_ACCOUNT_DATA_ACCESS_READ
	if _, e := fromAPIModuleAccess(wire.ModuleAccess); e == nil {
		t.Fatal("READ grant accepted")
	}
	sync.DataAccess = api.AccountDataAccess_ACCOUNT_DATA_ACCESS_READ_WRITE
	if _, e := fromAPIModuleAccess(wire.ModuleAccess); e != nil {
		t.Fatal(e)
	}
}
