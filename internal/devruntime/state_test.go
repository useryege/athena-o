package devruntime

import (
	"os"
	"path/filepath"
	"sync"
	"testing"
)

func testKey(t *testing.T) InstanceKey {
	t.Helper()
	k, e := NewInstanceKey(t.TempDir(), "test")
	if e != nil {
		t.Fatal(e)
	}
	return k
}
func TestInstanceKeyIsolationAndPaths(t *testing.T) {
	root := t.TempDir()
	a, e := NewInstanceKey(root, "a")
	if e != nil {
		t.Fatal(e)
	}
	b, _ := NewInstanceKey(root, "b")
	c, _ := NewInstanceKey(t.TempDir(), "a")
	if a.Namespace == b.Namespace || a.Namespace == c.Namespace {
		t.Fatal("instances share ownership")
	}
	link := filepath.Join(t.TempDir(), "checkout")
	if e = os.Symlink(root, link); e != nil {
		t.Fatal(e)
	}
	same, _ := NewInstanceKey(link, "a")
	if a != same {
		t.Fatal("symlink checkout is not canonical")
	}
	for _, name := range []string{"", "../escape", "a/b", "a.b", "-a"} {
		if _, e := NewInstanceKey(root, name); e == nil {
			t.Fatalf("accepted %q", name)
		}
	}
}
func TestAtomicStateAndUnknownVersion(t *testing.T) {
	k := testKey(t)
	s := State{Version: 1, Key: k, RunID: "run", Phase: "starting"}
	p := k.StatePath()
	if e := SaveState(p, s); e != nil {
		t.Fatal(e)
	}
	var wg sync.WaitGroup
	for range 4 {
		wg.Go(func() {
			for range 40 {
				if e := SaveState(p, s); e != nil {
					t.Error(e)
				}
				if _, e := LoadState(p); e != nil {
					t.Error(e)
				}
			}
		})
	}
	wg.Wait()
	st, _ := os.Stat(p)
	if st.Mode().Perm() != 0600 {
		t.Fatalf("permissions %v", st.Mode())
	}
	if e := SaveState(filepath.Join(t.TempDir(), "state.json"), s); e == nil {
		t.Fatal("accepted unrelated state path")
	}
	if e := os.WriteFile(p, []byte(`{"Version":99}`), 0600); e != nil {
		t.Fatal(e)
	}
	if _, e := LoadState(p); e == nil {
		t.Fatal("accepted unknown version")
	}
}
func TestStateRejectsSymlinkedInstanceDirectory(t *testing.T) {
	k := testKey(t)
	outside := t.TempDir()
	if e := os.MkdirAll(filepath.Join(k.Checkout, ".run"), 0700); e != nil {
		t.Fatal(e)
	}
	if e := os.Symlink(outside, filepath.Join(k.Checkout, ".run", "instances")); e != nil {
		t.Fatal(e)
	}
	if e := SaveState(k.StatePath(), State{Version: 1, Key: k}); e == nil {
		t.Fatal("followed instance symlink")
	}
	files, _ := os.ReadDir(outside)
	if len(files) != 0 {
		t.Fatal("wrote outside checkout")
	}
}
