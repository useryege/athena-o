package migration

import "testing"

func TestSelectAllModules(t *testing.T) {
	modules, err := Select(AllModules)
	if err != nil {
		t.Fatalf("Select(all) error = %v", err)
	}
	if len(modules) != 6 {
		t.Fatalf("Select(all) returned %d modules, want 6", len(modules))
	}
}

func TestSelectSingleModule(t *testing.T) {
	modules, err := Select("wallet")
	if err != nil {
		t.Fatalf("Select(wallet) error = %v", err)
	}
	if len(modules) != 1 || modules[0].Name != "wallet" || modules[0].Database != "wallet" {
		t.Fatalf("Select(wallet) = %+v, want wallet module", modules)
	}
}

func TestSelectUnknownModule(t *testing.T) {
	if _, err := Select("missing"); err == nil {
		t.Fatalf("Select(missing) error = nil, want error")
	}
}
