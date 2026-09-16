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
	if len(AllModules()) != 8 {
		t.Fatal(len(AllModules()))
	}
	a.Modules[Module("trader_sync")] = AccessLevelReadWrite
	if err := a.Validate(); err != nil {
		t.Fatal(err)
	}
}

func TestSolanaIsReadOnlyInCompleteModuleMatrix(t *testing.T) {
	if len(AllModules()) != 8 {
		t.Fatalf("module count = %d", len(AllModules()))
	}
	if got, ok := MaxAccessLevel(Module("solana")); !ok || got != AccessLevelRead {
		t.Fatalf("Solana maximum = %q, %t", got, ok)
	}
	none := NoModuleAccess()
	if none[Module("solana")] != AccessLevelNone {
		t.Fatalf("new account Solana access = %q", none[Module("solana")])
	}
	maximum := MaximumModuleAccess()
	if maximum[Module("solana")] != AccessLevelRead {
		t.Fatalf("maximum Solana access = %q", maximum[Module("solana")])
	}
	a := Access{Modules: maximum}
	if err := a.Validate(); err != nil {
		t.Fatal(err)
	}
	a.Modules[Module("solana")] = AccessLevelReadWrite
	if err := a.Validate(); err == nil {
		t.Fatal("Solana READ_WRITE grant accepted")
	}
}
func TestRetiredModulesAreRejected(t *testing.T) {
	for _, old := range []Module{"sports_live", "sports_history", "world_cup_corners"} {
		if _, ok := MaxAccessLevel(old); ok {
			t.Errorf("retired module accepted: %s", old)
		}
	}
	if got := len(AllModules()); got != 8 {
		t.Errorf("module count=%d", got)
	}
	if level, ok := MaxAccessLevel(ModuleWormMarkets); !ok || level != AccessLevelRead {
		t.Fatal("Worm Markets permission changed")
	}
	if level, ok := MaxAccessLevel(ModuleWormTrading); !ok || level != AccessLevelReadWrite {
		t.Fatal("Worm Trading permission changed")
	}
}
