//go:build integration

package store

import (
	"context"
	"github.com/ethereum/go-ethereum/common"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/useryege/athena/internal/accountaccess"
	"github.com/useryege/athena/internal/accountcredentials"
	accountstore "github.com/useryege/athena/internal/accountstate/store"
	"github.com/useryege/athena/internal/accountstate/store/migrations"
	"github.com/useryege/athena/internal/accountstate/txgate"
	"github.com/useryege/athena/internal/testutil/pgtest"
	tm "github.com/useryege/athena/internal/tradersync/types"
	"testing"
)

func TestSubscriptionSchemaExists(t *testing.T) {
	db := pgtest.New(t, migrations.FS, migrations.Dir)
	var n int
	if e := db.Pool.QueryRow(context.Background(), "SELECT count(*) FROM trader_sync_subscriptions").Scan(&n); e != nil {
		t.Fatal(e)
	}
}

func TestEmptyIntervalKeepsActualEndpoint(t *testing.T) {
	db := pgtest.New(t, migrations.FS, migrations.Dir)
	ctx := context.Background()
	a := accountstore.NewSQLStore(db.Pool)
	account, e := a.EnsureDevelopmentAccount(ctx, accountcredentials.DevelopmentRoleMember)
	if e != nil {
		t.Fatal(e)
	}
	var sub, attempt string
	if e = db.Pool.QueryRow(ctx, `INSERT INTO trader_sync_subscriptions(owner_id,wallet,desired_state,observation_state,effective_at,target_display) VALUES($1,decode(repeat('11',20),'hex'),'enabled','healthy',clock_timestamp()+interval '1 hour','{"displayName":{"availability":"unavailable","reasonCode":"fixture_not_queried","source":"fixture"},"avatar":{"availability":"unavailable","reasonCode":"fixture_not_queried","source":"fixture"},"profileURL":{"availability":"unavailable","reasonCode":"fixture_not_queried","source":"fixture"}}'::jsonb) RETURNING id`, account.ID).Scan(&sub); e != nil {
		t.Fatal(e)
	}
	if e = db.Pool.QueryRow(ctx, `INSERT INTO trader_sync_baseline_attempts(owner_id,subscription_id,activation_generation,expected_revision,state) VALUES($1,$2,1,1,'succeeded') RETURNING id`, account.ID, sub).Scan(&attempt); e != nil {
		t.Fatal(e)
	}
	if _, e = db.Pool.Exec(ctx, `INSERT INTO trader_sync_monitor_intervals(owner_id,subscription_id,baseline_attempt_id,activation_generation,collector_epoch,filter_revision,expected_revision,registered_high,candidate_effective_at,effective_at) VALUES($1,$2,$3,1,1,1,1,1,clock_timestamp()+interval '1 hour',clock_timestamp()+interval '1 hour')`, account.ID, sub, attempt); e != nil {
		t.Fatal(e)
	}
	s := NewSQLStore(db.Pool)
	if e = txgate.WithAccountTx(ctx, db.Pool, account.ID, func(tx pgx.Tx) error { return s.RevokeTx(ctx, tx, account.ID, "permission_revoked") }); e != nil {
		t.Fatal(e)
	}
	var empty bool
	if e = db.Pool.QueryRow(ctx, `SELECT ended_at<effective_at AND ended_at<=clock_timestamp() FROM trader_sync_monitor_intervals WHERE subscription_id=$1`, sub).Scan(&empty); e != nil || !empty {
		t.Fatal(empty, e)
	}
}

func TestAccessFlagsDoNotRevokeProduct(t *testing.T) {
	db := pgtest.New(t, migrations.FS, migrations.Dir)
	ctx := context.Background()
	a := accountstore.NewSQLStore(db.Pool)
	account, e := a.EnsureDevelopmentAccount(ctx, accountcredentials.DevelopmentRoleMember)
	if e != nil {
		t.Fatal(e)
	}
	s := NewSQLStore(db.Pool)
	a.SetAccessChangeHook(s.ApplyAccessChangeTx)
	var id string
	if e = db.Pool.QueryRow(ctx, `INSERT INTO trader_sync_subscriptions(owner_id,wallet,desired_state,observation_state,target_display) VALUES($1,decode(repeat('12',20),'hex'),'enabled','pending_baseline','{"displayName":{"availability":"unavailable","reasonCode":"fixture_not_queried","source":"fixture"},"avatar":{"availability":"unavailable","reasonCode":"fixture_not_queried","source":"fixture"},"profileURL":{"availability":"unavailable","reasonCode":"fixture_not_queried","source":"fixture"}}'::jsonb) RETURNING id`, account.ID).Scan(&id); e != nil {
		t.Fatal(e)
	}
	access, e := a.GetAccountAccess(ctx, account.ID)
	if e != nil {
		t.Fatal(e)
	}
	access.LoginEnabled = false
	access.APIKeyEnabled = false
	access, e = a.UpdateAccountAccess(ctx, account.ID, access, access.Revision)
	if e != nil {
		t.Fatal(e)
	}
	var state string
	if e = db.Pool.QueryRow(ctx, "SELECT desired_state FROM trader_sync_subscriptions WHERE id=$1", id).Scan(&state); e != nil || state != "enabled" {
		t.Fatal(state, e)
	}
	access.Modules[accountaccess.ModuleTraderSync] = accountaccess.AccessLevelNone
	access, e = a.UpdateAccountAccess(ctx, account.ID, access, access.Revision)
	if e != nil {
		t.Fatal(e)
	}
	access.Modules[accountaccess.ModuleTraderSync] = accountaccess.AccessLevelReadWrite
	if _, e = a.UpdateAccountAccess(ctx, account.ID, access, access.Revision); e != nil {
		t.Fatal(e)
	}
	if e = db.Pool.QueryRow(ctx, "SELECT desired_state FROM trader_sync_subscriptions WHERE id=$1", id).Scan(&state); e != nil || state != "permission_disabled" {
		t.Fatal(state, e)
	}
}

func TestResolutionContextOwnerNotesAndQuota(t *testing.T) {
	db := pgtest.New(t, migrations.FS, migrations.Dir)
	ctx := context.Background()
	a := accountstore.NewSQLStore(db.Pool)
	account, e := a.EnsureDevelopmentAccount(ctx, accountcredentials.DevelopmentRoleMember)
	if e != nil {
		t.Fatal(e)
	}
	s := NewSQLStore(db.Pool)
	wallet := common.HexToAddress("0x3333333333333333333333333333333333333333")
	check := func(owner string, fn func(tm.ResolutionContext)) {
		t.Helper()
		if e := txgate.WithAccountTx(ctx, db.Pool, owner, func(tx pgx.Tx) error {
			value, e := s.ResolveContextTx(ctx, tx, owner, wallet)
			if e == nil {
				fn(value)
			}
			return e
		}); e != nil {
			t.Fatal(e)
		}
	}
	check(account.ID, func(c tm.ResolutionContext) {
		if c.SavedNote != nil || c.Existing != nil || c.Quota.Used != 0 || c.Quota.Limit != 10 {
			t.Fatal(c)
		}
	})
	if _, e = db.Pool.Exec(ctx, `INSERT INTO trader_sync_target_notes(owner_id,wallet,note,revision) VALUES($1,$2,'',1)`, account.ID, wallet.Bytes()); e != nil {
		t.Fatal(e)
	}
	var id string
	if e = db.Pool.QueryRow(ctx, `INSERT INTO trader_sync_subscriptions(owner_id,wallet,desired_state,observation_state,target_display) VALUES($1,$2,'enabled','pending_baseline','{"displayName":{"availability":"unavailable","reasonCode":"fixture_not_queried","source":"fixture"},"avatar":{"availability":"unavailable","reasonCode":"fixture_not_queried","source":"fixture"},"profileURL":{"availability":"unavailable","reasonCode":"fixture_not_queried","source":"fixture"}}'::jsonb) RETURNING id`, account.ID, wallet.Bytes()).Scan(&id); e != nil {
		t.Fatal(e)
	}
	for _, pair := range [][3]string{{"enabled", "pending_baseline", "pending_baseline"}, {"enabled", "healthy", "healthy"}, {"enabled", "interrupted", "interrupted"}, {"paused", "healthy", "paused"}, {"permission_disabled", "healthy", "permission_disabled"}} {
		if _, e = db.Pool.Exec(ctx, "UPDATE trader_sync_subscriptions SET desired_state=$2,observation_state=$3 WHERE id=$1", id, pair[0], pair[1]); e != nil {
			t.Fatal(e)
		}
		check(account.ID, func(c tm.ResolutionContext) {
			if c.SavedNote == nil || c.SavedNote.Note != "" || c.Existing == nil || c.Existing.Status != pair[2] || c.Quota.Used != 1 {
				t.Fatal(c)
			}
		})
	}
	check(uuid.NewString(), func(c tm.ResolutionContext) {
		if c.SavedNote != nil || c.Existing != nil || c.Quota.Used != 0 {
			t.Fatal(c)
		}
	})
	if _, e = db.Pool.Exec(ctx, "UPDATE trader_sync_subscriptions SET desired_state='cancelled' WHERE id=$1", id); e != nil {
		t.Fatal(e)
	}
	check(account.ID, func(c tm.ResolutionContext) {
		if c.SavedNote == nil || c.Existing != nil || c.Quota.Used != 0 {
			t.Fatal(c)
		}
	})
	if _, e = db.Pool.Exec(ctx, "UPDATE account_module_access SET access_level='read' WHERE account_id=$1 AND module='trader_sync'", account.ID); e == nil {
		t.Fatal("SQL allowed read grant")
	}
}
