package account

import (
	core "github.com/useryege/athena/internal/accountaccess"
	api "github.com/useryege/athena/pkg/apiclient/account"
	"testing"
)

func TestTraderSyncAPIMatrix(t *testing.T) {
	a := core.Access{Modules: core.MaximumModuleAccess()}
	wire := ToAPIAccountAccess(a)
	if len(wire.ModuleAccess) != 8 {
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

func TestSolanaWireMatrixRetainsToken(t *testing.T) {
	modules := core.MaximumModuleAccess()
	wire := ToAPIAccountAccess(core.Access{Modules: modules})
	if len(wire.ModuleAccess) != 8 {
		t.Fatalf("module count = %d", len(wire.ModuleAccess))
	}
	var foundSolana, foundToken bool
	for _, row := range wire.ModuleAccess {
		if row.Module == api.AccountDataModule(13) {
			foundSolana = true
			if row.DataAccess != api.AccountDataAccess_ACCOUNT_DATA_ACCESS_READ {
				t.Fatalf("Solana access = %v", row.DataAccess)
			}
		}
		if row.Module == api.AccountDataModule_ACCOUNT_DATA_MODULE_TOKEN {
			foundToken = true
		}
	}
	if !foundSolana || !foundToken {
		t.Fatalf("Solana=%t Token=%t", foundSolana, foundToken)
	}
	roundTrip, err := fromAPIModuleAccess(wire.ModuleAccess)
	if err != nil {
		t.Fatal(err)
	}
	if roundTrip[core.ModuleToken] != core.AccessLevelReadWrite || roundTrip[core.Module("solana")] != core.AccessLevelRead {
		t.Fatalf("round trip = %v", roundTrip)
	}
}
