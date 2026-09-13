package devruntime

import (
	"context"
	"testing"
)

func TestRejectUnknownDatabaseInitializationBeforeDocker(t *testing.T) {
	k, _ := NewInstanceKey(t.TempDir(), "unknown-init")
	m := NewManager(k)
	if e := SaveState(k.StatePath(), State{Version: StateVersion, Key: k, RunID: NewRunID(), Phase: "starting", DBMode: "managed", InitializationMode: "some-other-layout"}); e != nil {
		t.Fatal(e)
	}
	called := false
	m.Docker.Exec = func(context.Context, string, ...string) ([]byte, error) { called = true; return nil, context.Canceled }
	if _, e := m.preparePostgres(context.Background()); e == nil || called {
		t.Fatal("unknown initialization reached Docker")
	}
}
func TestExternalDatabaseIdentityExcludesCredentials(t *testing.T) {
	address, database, e := externalDatabaseIdentity("postgres://owner:secret@127.0.0.1:35432/borrowed?sslmode=disable")
	if e != nil || address != "127.0.0.1:35432" || database != "borrowed" {
		t.Fatal("external target not observable without credentials", e)
	}
}
