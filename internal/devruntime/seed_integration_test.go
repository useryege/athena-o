//go:build integration

package devruntime

import (
	"context"
	"github.com/useryege/athena/internal/accountaccess"
	"github.com/useryege/athena/internal/accountstate/schema"
	accountstore "github.com/useryege/athena/internal/accountstate/store"
	"github.com/useryege/athena/internal/accountstate/store/migrations"
	"github.com/useryege/athena/internal/testutil/pgtest"
	traderstore "github.com/useryege/athena/internal/tradersync/store"
	"testing"
	"time"
)

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
