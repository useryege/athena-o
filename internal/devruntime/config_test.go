package devruntime

import (
	"io"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestDotenvIsDataAndExportedEmptyWins(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "env")
	marker := filepath.Join(root, "executed")
	content := "A=from-file\nEMPTY=from-file\nCOMMAND='$(touch " + marker + ")'\nSECRET=do-not-inject\n"
	if e := os.WriteFile(path, []byte(content), 0600); e != nil {
		t.Fatal(e)
	}
	env, e := LoadEnvironment(path, []string{"A=exported", "EMPTY="})
	if e != nil {
		t.Fatal(e)
	}
	if env["A"] != "exported" || env["EMPTY"] != "" || env["COMMAND"] != "$(touch "+marker+")" {
		t.Fatal(env)
	}
	if _, e = os.Stat(marker); !os.IsNotExist(e) {
		t.Fatal("executed dotenv")
	}
	if got := EnvironmentFor(env, []string{"EMPTY", "A", "MISSING"}); !reflect.DeepEqual(got, []string{"A=exported", "EMPTY="}) {
		t.Fatal(got)
	}
}
func TestSecretFileIsRestrictedAndCannotEscape(t *testing.T) {
	k := testKey(t)
	m := NewManager(k)
	if e := m.SaveSecret("db.env", []byte("PASSWORD=secret\n")); e != nil {
		t.Fatal(e)
	}
	p := filepath.Join(k.Dir(), "db.env")
	st, e := os.Stat(p)
	if e != nil || st.Mode().Perm() != 0600 {
		t.Fatalf("secret permission %v %v", st, e)
	}
	if e = m.SaveSecret("../escape", []byte("secret")); e == nil {
		t.Fatal("accepted secret traversal")
	}
}

// Unchanged bind-mounted secrets must keep the file Docker originally mounted.
func TestSaveSecretUnchangedPreservesFileAndRestrictsPermissions(t *testing.T) {
	for _, mode := range []os.FileMode{0600, 0644} {
		t.Run(mode.String(), func(t *testing.T) {
			m := NewManager(testKey(t))
			data := []byte("requirepass fixture\n")
			if err := m.SaveSecret("redis.conf", data); err != nil {
				t.Fatal(err)
			}
			path := filepath.Join(m.Key.Dir(), "redis.conf")
			if err := os.Chmod(path, mode); err != nil {
				t.Fatal(err)
			}
			before, err := os.Stat(path)
			if err != nil {
				t.Fatal(err)
			}
			if err = m.SaveSecret("redis.conf", data); err != nil {
				t.Fatal(err)
			}
			after, err := os.Stat(path)
			if err != nil {
				t.Fatal(err)
			}
			if !os.SameFile(before, after) {
				t.Error("unchanged secret replaced the bind-mounted file")
			}
			if after.Mode().Perm() != 0600 {
				t.Errorf("secret mode = %o", after.Mode().Perm())
			}
		})
	}
}

func TestSaveSecretChangedAtomicallyReplacesFile(t *testing.T) {
	m := NewManager(testKey(t))
	if err := m.SaveSecret("db.env", []byte("old")); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(m.Key.Dir(), "db.env")
	old, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer old.Close()
	if err = m.SaveSecret("db.env", []byte("new secret")); err != nil {
		t.Fatal(err)
	}
	retained, err := io.ReadAll(old)
	if err != nil || string(retained) != "old" {
		t.Fatalf("open reader saw changed bytes: %q %v", retained, err)
	}
	current, err := os.ReadFile(path)
	if err != nil || string(current) != "new secret" {
		t.Fatalf("new reader: %q %v", current, err)
	}
	st, err := os.Stat(path)
	if err != nil || st.Mode().Perm() != 0600 {
		t.Fatalf("replacement mode: %v %v", st, err)
	}
}

func TestSaveSecretReplacesSymlinkEvenWhenContentMatches(t *testing.T) {
	m := NewManager(testKey(t))
	if err := m.SaveSecret("db.env", []byte("secret")); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(m.Key.Dir(), "db.env")
	target := filepath.Join(t.TempDir(), "external")
	if err := os.WriteFile(target, []byte("secret"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, path); err != nil {
		t.Fatal(err)
	}
	if err := m.SaveSecret("db.env", []byte("secret")); err != nil {
		t.Fatal(err)
	}
	st, err := os.Lstat(path)
	if err != nil || !st.Mode().IsRegular() || st.Mode().Perm() != 0600 {
		t.Fatalf("secret retained link: %v %v", st, err)
	}
	targetInfo, err := os.Stat(target)
	if err != nil || targetInfo.Mode().Perm() != 0644 {
		t.Fatalf("external target changed: %v %v", targetInfo, err)
	}
}
