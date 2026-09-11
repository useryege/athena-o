package accountaccess

import "testing"

func TestTraderSyncGrantDiffersFromReadRequirement(t *testing.T) {
	a := Access{Modules: NoModuleAccess()}
	a.Modules[Module("trader_sync")] = AccessLevelRead
	if a.Validate() == nil {
		t.Fatal("READ grant accepted")
	}
	if err := RequireModule(Module("trader_sync"), AccessLevelRead).validate(); err != nil {
		t.Fatal(err)
	}
	if !accessLevelSatisfies(AccessLevelReadWrite, AccessLevelRead) {
		t.Fatal("RW cannot read")
	}
	if len(AllModules()) != 10 {
		t.Fatal(len(AllModules()))
	}
	a.Modules[Module("trader_sync")] = AccessLevelReadWrite
	if err := a.Validate(); err != nil {
		t.Fatal(err)
	}
}
