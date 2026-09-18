package migration

import "testing"

func TestModulesHaveOneAthenaOwner(t *testing.T) {
	for _, name := range []string{"sports-live", "sports-history"} {
		if _, err := Select(name); err == nil {
			t.Fatalf("retired module still migrates: %s", name)
		}
	}
	for _, name := range []string{"worm-trading"} {
		if _, err := Select(name); err != nil {
			t.Fatalf("Worm migration removed: %v", err)
		}
	}
	if _, err := Select("worm-markets"); err == nil {
		t.Fatal("retired worm-markets migration module remains registered")
	}
	accountStateOwners := 0
	for _, module := range Modules() {
		if module.Name == "notification" || module.Database == "notification" {
			t.Fatalf("independent notification migration: %+v", module)
		}
		if module.Name == "account-state" {
			accountStateOwners++
		}
	}
	if accountStateOwners != 1 {
		t.Fatalf("account-state migration owners=%d", accountStateOwners)
	}
}

func TestOperationLogMigrationOwnsTheSharedAthenaDatabase(t *testing.T) {
	modules, err := Select(AllModules)
	if err != nil {
		t.Fatal(err)
	}
	var found Module
	for _, module := range modules {
		if module.Name == "operation-log" {
			found = module
			break
		}
	}
	if found.Name == "" {
		t.Fatal("operation-log migration is missing from all")
	}
	if found.DSNEnv != "ATHENA_ACCOUNT_STATE_POSTGRES_DSN" {
		t.Fatalf("operation-log DSN env = %q", found.DSNEnv)
	}
	if found.Database != "athena" {
		t.Fatalf("operation-log database = %q", found.Database)
	}
	if found.Migrations != nil {
		t.Fatal("operation-log must use its schema owner instead of generic migrations")
	}
}
