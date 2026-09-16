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
	count := 0
	for _, module := range Modules() {
		if module.Name == "notification" || module.Database == "notification" {
			t.Fatalf("independent notification migration: %+v", module)
		}
		if module.Database == "athena" {
			count++
		}
	}
	if count != 1 {
		t.Fatalf("athena migration owners=%d", count)
	}
}
