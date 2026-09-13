package devruntime

import (
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
