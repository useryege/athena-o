//go:build integration

package devruntime

import (
	"context"
	"github.com/jackc/pgx/v5/pgxpool"
	"os"
	"testing"
	"time"
)

func databaseFixtureManager(t *testing.T) *Manager {
	t.Helper()
	k, e := NewInstanceKey(t.TempDir(), "task9-"+NewRunID()[:8])
	if e != nil {
		t.Fatal(e)
	}
	m := NewManager(k)
	s := State{Version: StateVersion, Key: k, RunID: NewRunID(), DBMode: "managed", Phase: "starting", Processes: map[string]ProcessIdentity{}}
	if e = SaveState(k.StatePath(), s); e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
		defer cancel()
		if e := m.Stop(ctx); e != nil {
			t.Error(e)
			return
		}
		if e := m.Reset(ctx); e != nil {
			t.Error(e)
		}
	})
	return m
}
func TestManagedPostgresPersistsAndIsolates(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 240*time.Second)
	defer cancel()
	a := databaseFixtureManager(t)
	b := databaseFixtureManager(t)
	da, e := a.preparePostgres(ctx)
	if e != nil {
		t.Fatal(e)
	}
	pa, e := pgxpool.New(ctx, da)
	if e != nil {
		t.Fatal(e)
	}
	defer pa.Close()
	if _, e = pa.Exec(ctx, "CREATE TABLE task9_marker(value text); INSERT INTO task9_marker VALUES('persisted')"); e != nil {
		t.Fatal(e)
	}
	var fsync string
	if e = pa.QueryRow(ctx, "SHOW fsync").Scan(&fsync); e != nil || fsync != "on" {
		t.Fatal(fsync, e)
	}
	pa.Close()
	if e = a.Stop(ctx); e != nil {
		t.Fatal(e)
	}
	a.Update(func(s *State) error { s.Phase = "starting"; s.RunID = NewRunID(); return nil })
	da, e = a.preparePostgres(ctx)
	if e != nil {
		t.Fatal(e)
	}
	pa, e = pgxpool.New(ctx, da)
	if e != nil {
		t.Fatal(e)
	}
	defer pa.Close()
	var marker string
	if e = pa.QueryRow(ctx, "SELECT value FROM task9_marker").Scan(&marker); e != nil || marker != "persisted" {
		t.Fatal(marker, e)
	}
	db, e := b.preparePostgres(ctx)
	if e != nil {
		t.Fatal(e)
	}
	pb, e := pgxpool.New(ctx, db)
	if e != nil {
		t.Fatal(e)
	}
	defer pb.Close()
	var absent bool
	if e = pb.QueryRow(ctx, "SELECT to_regclass('task9_marker') IS NULL").Scan(&absent); e != nil || !absent {
		t.Fatal("cross-instance database", e)
	}
	state, _ := os.ReadFile(a.Key.StatePath())
	if len(state) == 0 {
		t.Fatal("missing resource evidence")
	}
	t.Logf("A namespace=%s B namespace=%s", a.Key.Namespace, b.Key.Namespace)
}
