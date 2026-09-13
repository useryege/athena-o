//go:build integration

package devruntime

import (
	"context"
	"encoding/json"
	"github.com/useryege/athena/internal/accountaccess"
	"github.com/useryege/athena/internal/accountstate/schema"
	accountstore "github.com/useryege/athena/internal/accountstate/store"
	"github.com/useryege/athena/internal/accountstate/store/migrations"
	"github.com/useryege/athena/internal/testutil/pgtest"
	traderstore "github.com/useryege/athena/internal/tradersync/store"
	"os"
	"os/exec"
	"testing"
	"time"
)

func TestSeedCLIPrintsStableIdentitiesAndPreservesRevokedGrant(t *testing.T) {
	o := runnerOptions(t, "managed", []string{"trader-sync"}, "")
	_, done := startRunner(t, o)
	awaitRunning(t, o, done)
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()
	invoke := func(command string, args ...string) map[string]string {
		t.Helper()
		cmd := exec.CommandContext(ctx, command, args...)
		cmd.Dir = o.Key.Checkout
		cmd.Env = EnvironmentFor(environmentMap(os.Environ()), []string{"PATH", "HOME", "TMPDIR", "GOCACHE", "GOMODCACHE", "GOPATH", "GOROOT", "GOTOOLCHAIN"})
		out, e := cmd.Output()
		if e != nil {
			t.Fatal(e)
		}
		var ids map[string]string
		if e = json.Unmarshal(out, &ids); e != nil {
			t.Fatalf("seed CLI must emit identity JSON: %v; stdout=%q", e, out)
		}
		if len(ids) != 2 || ids["member_id"] == "" || ids["administrator_id"] == "" || ids["member_id"] == ids["administrator_id"] {
			t.Fatalf("invalid seed identities: %v", ids)
		}
		return ids
	}
	first := invoke("make", "--no-print-directory", "seed-service", "SERVICE=trader-sync", "INSTANCE="+o.Key.Name)
	m := NewManager(o.Key)
	s, e := m.Status()
	if e != nil {
		t.Fatal(e)
	}
	dsn, e := m.currentManagedDSN(ctx, s)
	if e != nil {
		t.Fatal(e)
	}
	pool, e := schema.ConnectVerified(ctx, dsn)
	if e != nil {
		t.Fatal(e)
	}
	defer pool.Close()
	store := accountstore.NewSQLStore(pool)
	store.SetAccessChangeHook(traderstore.NewAccessRevocationAdapter().ApplyAccessChangeTx)
	access, e := store.GetAccountAccess(ctx, first["member_id"])
	if e != nil {
		t.Fatal(e)
	}
	access.Modules[accountaccess.ModuleTraderSync] = accountaccess.AccessLevelNone
	if _, e = store.UpdateAccountAccess(ctx, first["member_id"], access, access.Revision); e != nil {
		t.Fatal(e)
	}
	second := invoke("go", "run", "./tools/trader-sync-dev", "--instance", o.Key.Name)
	for key, value := range first {
		if second[key] != value {
			t.Fatal("seed identities changed", first, second)
		}
	}
	after, e := store.GetAccountAccess(ctx, first["member_id"])
	if e != nil || after.Modules[accountaccess.ModuleTraderSync] != accountaccess.AccessLevelNone {
		t.Fatal("CLI seed reset grant", e)
	}
	if e = Stop(ctx, o.Key); e != nil {
		t.Fatal(e)
	}
	<-done
	t.Logf("both seed CLIs returned stable IDs in owned namespace=%s; revoked grant preserved", o.Key.Namespace)
}

func TestSeedDoesNotRestoreRevokedGrant(t *testing.T) {
	db := pgtest.New(t, migrations.FS, migrations.Dir)
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()
	if e := schema.Up(ctx, db.DSN); e != nil {
		t.Fatal(e)
	}
	pool, e := schema.ConnectVerified(ctx, db.DSN)
	if e != nil {
		t.Fatal(e)
	}
	defer pool.Close()
	fixture, e := seedDatabase(ctx, pool)
	if e != nil {
		t.Fatal(e)
	}
	store := accountstore.NewSQLStore(pool)
	access, e := store.GetAccountAccess(ctx, fixture.MemberID)
	if e != nil {
		t.Fatal(e)
	}
	if access.Modules[accountaccess.ModuleTraderSync] != accountaccess.AccessLevelReadWrite {
		t.Fatal("missing initial grant")
	}
	access.Modules[accountaccess.ModuleTraderSync] = accountaccess.AccessLevelNone
	store.SetAccessChangeHook(traderstore.NewAccessRevocationAdapter().ApplyAccessChangeTx)
	if _, e = store.UpdateAccountAccess(ctx, fixture.MemberID, access, access.Revision); e != nil {
		t.Fatal(e)
	}
	again, e := seedDatabase(ctx, pool)
	if e != nil || again != fixture {
		t.Fatal(again, e)
	}
	after, e := store.GetAccountAccess(ctx, fixture.MemberID)
	if e != nil || after.Modules[accountaccess.ModuleTraderSync] != accountaccess.AccessLevelNone {
		t.Fatal("seed reset revocation", e)
	}
}
