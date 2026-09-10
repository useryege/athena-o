//go:build integration

package tradersync

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/useryege/athena/internal/accountaccess"
	"github.com/useryege/athena/internal/accountcredentials"
	as "github.com/useryege/athena/internal/accountstate/store"
	"github.com/useryege/athena/internal/accountstate/store/migrations"
	"github.com/useryege/athena/internal/accountstate/txgate"
	"github.com/useryege/athena/internal/notification/delivery"
	notificationstore "github.com/useryege/athena/internal/notification/store"
	"github.com/useryege/athena/internal/testutil/pgtest"
	ts "github.com/useryege/athena/internal/tradersync/store"
	tm "github.com/useryege/athena/internal/tradersync/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"math/big"
	"testing"
	"time"
)

type fakeBaseline struct{ fail bool }

func (f *fakeBaseline) RegisterTx(ctx context.Context, tx pgx.Tx, s tm.Subscription) error {
	if f.fail {
		return errors.New("baseline unavailable")
	}
	_, e := tx.Exec(ctx, `INSERT INTO trader_sync_baseline_attempts(owner_id,subscription_id,activation_generation,expected_revision) VALUES($1,$2,$3,$4)`, s.OwnerID, s.ID, int64(s.Generation), int64(s.Revision))
	return e
}

type fakeRevalidator struct {
	fail    bool
	barrier chan struct{}
	arrived chan struct{}
}

func (f *fakeRevalidator) Revalidate(_ context.Context, i tm.Identity) error {
	if i.ResolutionInput == "" {
		return errors.New("source missing")
	}
	if f.fail {
		return errors.New("external must not run")
	}
	if f.barrier != nil {
		f.arrived <- struct{}{}
		<-f.barrier
	}
	return nil
}
func TestSubscriptionTransactions(t *testing.T) {
	db := pgtest.New(t, migrations.FS, migrations.Dir)
	ctx := context.Background()
	a := as.NewSQLStore(db.Pool)
	acct, e := a.EnsureDevelopmentAccount(ctx, accountcredentials.DevelopmentRoleMember)
	if e != nil {
		t.Fatal(e)
	}
	owner := acct.ID
	s := ts.NewSQLStore(db.Pool)
	r := &fakeRevalidator{}
	b := &fakeBaseline{}
	svc, e := NewSubscriptionService(db.Pool, s, r, b)
	if e != nil {
		t.Fatal(e)
	}
	token := func(w int) tm.CreateInput {
		t.Helper()
		raw := []byte(uuid.NewString())
		d := sha256.Sum256(raw)
		wallet := common.BigToAddress(big.NewInt(int64(w)))
		id := tm.Identity{Wallet: wallet, ResolutionInput: wallet.Hex(), Digest: sha256.Sum256(wallet.Bytes())}
		e := txgate.WithAccountTx(ctx, db.Pool, owner, func(tx pgx.Tx) error {
			return s.SaveConfirmationTx(ctx, tx, owner, id, d[:], time.Now().Add(time.Minute))
		})
		if e != nil {
			t.Fatal(e)
		}
		return tm.CreateInput{Token: base64.RawURLEncoding.EncodeToString(raw), RequestID: uuid.NewString()}
	}
	in := token(1)
	note := "🙂"
	in.Note = &note
	sub, e := svc.Create(ctx, owner, in)
	if e != nil {
		t.Fatal(e)
	}
	if sub.Note != note || sub.Generation != 1 || sub.NoteRevision != 1 {
		t.Fatal(sub)
	}
	r.fail = true
	_, e = db.Pool.Exec(ctx, "UPDATE trader_sync_target_confirmations SET expires_at=created_at")
	if e != nil {
		t.Fatal(e)
	}
	retry, e := svc.Create(ctx, owner, in)
	if e != nil || retry.ID != sub.ID {
		t.Fatal(retry, e)
	}
	changed := in
	changed.Note = nil
	if _, e = svc.Create(ctx, owner, changed); status.Code(e) != codes.AlreadyExists {
		t.Fatal(e)
	}
	r.fail = false
	saved, e := svc.UpdateNote(ctx, owner, tm.NoteInput{Wallet: sub.Wallet, RequestID: "note", Note: "", ExpectedRevision: 1})
	if e != nil || saved.Revision != 2 {
		t.Fatal(saved, e)
	}
	saved, e = svc.UpdateNote(ctx, owner, tm.NoteInput{Wallet: sub.Wallet, RequestID: "note", Note: "", ExpectedRevision: 1})
	if e != nil || saved.Revision != 2 {
		t.Fatal(saved, e)
	}
	var queuedID int64
	if e = db.Pool.QueryRow(ctx, `INSERT INTO account_notification_deliveries(account_id,idempotency_key,payload_digest,source,severity,body,channel,status,telegram_chat_id,binding_revision) VALUES($1,'before-pause',decode(repeat('cd',32),'hex'),'trader_sync','info','frozen note','telegram','pending',123,1) RETURNING id`, owner).Scan(&queuedID); e != nil {
		t.Fatal(e)
	}
	var attemptID string
	if e = db.Pool.QueryRow(ctx, "UPDATE trader_sync_baseline_attempts SET state='succeeded' WHERE subscription_id=$1 RETURNING id", sub.ID).Scan(&attemptID); e != nil {
		t.Fatal(e)
	}
	if _, e = db.Pool.Exec(ctx, `UPDATE trader_sync_subscriptions SET observation_state='healthy',effective_at=clock_timestamp()+interval '1 hour' WHERE id=$1`, sub.ID); e != nil {
		t.Fatal(e)
	}
	if _, e = db.Pool.Exec(ctx, `INSERT INTO trader_sync_monitor_intervals(owner_id,subscription_id,baseline_attempt_id,activation_generation,collector_epoch,filter_revision,expected_revision,registered_high,candidate_effective_at,effective_at) VALUES($1,$2,$3,1,1,1,1,1,clock_timestamp()+interval '1 hour',clock_timestamp()+interval '1 hour')`, owner, sub.ID, attemptID); e != nil {
		t.Fatal(e)
	}
	paused, e := svc.Change(ctx, owner, "pause", tm.ChangeInput{SubscriptionID: sub.ID, RequestID: "pause", ExpectedRevision: 1})
	if e != nil || paused.Generation != 1 || paused.Revision != 2 {
		t.Fatal(paused, e)
	}
	var emptyCoverage bool
	if e = db.Pool.QueryRow(ctx, "SELECT ended_at<effective_at FROM trader_sync_monitor_intervals WHERE subscription_id=$1", sub.ID).Scan(&emptyCoverage); e != nil || !emptyCoverage {
		t.Fatal(emptyCoverage, e)
	}
	resumed, e := svc.Change(ctx, owner, "resume", tm.ChangeInput{SubscriptionID: sub.ID, RequestID: "resume", ExpectedRevision: 2})
	if e != nil || resumed.Generation != 2 {
		t.Fatal(resumed, e)
	}
	cancelled, e := svc.Change(ctx, owner, "cancel", tm.ChangeInput{SubscriptionID: sub.ID, RequestID: "cancel", ExpectedRevision: 3})
	if e != nil || cancelled.DesiredState != "cancelled" {
		t.Fatal(cancelled, e)
	}
	var queuedState, queuedBody string
	if e = db.Pool.QueryRow(ctx, "SELECT status,body FROM account_notification_deliveries WHERE id=$1", queuedID).Scan(&queuedState, &queuedBody); e != nil || queuedState != "pending" || queuedBody != "frozen note" {
		t.Fatal(queuedState, queuedBody, e)
	}
	if _, e = svc.Change(ctx, owner, "resume", tm.ChangeInput{SubscriptionID: sub.ID, RequestID: "bad", ExpectedRevision: 4}); status.Code(e) != codes.FailedPrecondition {
		t.Fatal(e)
	}
	rebuilt, e := svc.Create(ctx, owner, token(1))
	if e != nil || rebuilt.ID == sub.ID || rebuilt.Note != "" || rebuilt.NoteRevision != 2 {
		t.Fatal(rebuilt, e)
	}
	b.fail = true
	bad := token(2)
	if _, e = svc.Create(ctx, owner, bad); e == nil {
		t.Fatal("registrar failure committed")
	}
	b.fail = false
	if _, e = svc.Create(ctx, owner, bad); e != nil {
		t.Fatal(e)
	}
	for i := 3; i <= 9; i++ {
		if _, e = svc.Create(ctx, owner, token(i)); e != nil {
			t.Fatal(e)
		}
	}
	// Both external checks finish before either final account transaction: only the tenth slot wins.
	x, y := token(10), token(11)
	r.barrier = make(chan struct{})
	r.arrived = make(chan struct{}, 2)
	out := make(chan error, 2)
	go func() { _, e := svc.Create(ctx, owner, x); out <- e }()
	go func() { _, e := svc.Create(ctx, owner, y); out <- e }()
	<-r.arrived
	<-r.arrived
	close(r.barrier)
	e1, e2 := <-out, <-out
	if !((e1 == nil && status.Code(e2) == codes.ResourceExhausted) || (e2 == nil && status.Code(e1) == codes.ResourceExhausted)) {
		t.Fatal(e1, e2)
	}
	a.SetAccessChangeHook(s.ApplyAccessChangeTx)
	access, e := a.GetAccountAccess(ctx, owner)
	if e != nil {
		t.Fatal(e)
	}
	access.Modules[accountaccess.ModuleTraderSync] = accountaccess.AccessLevelNone
	if _, e = a.UpdateAccountAccess(ctx, owner, access, access.Revision); e != nil {
		t.Fatal(e)
	}
	if _, e = svc.Create(ctx, owner, in); status.Code(e) != codes.PermissionDenied {
		t.Fatal("revoked request replay accepted", e)
	}
}

func TestAccessHookAtomicRevocation(t *testing.T) {
	db := pgtest.New(t, migrations.FS, migrations.Dir)
	ctx := context.Background()
	a := as.NewSQLStore(db.Pool)
	acct, e := a.EnsureDevelopmentAccount(ctx, accountcredentials.DevelopmentRoleMember)
	if e != nil {
		t.Fatal(e)
	}
	s := ts.NewSQLStore(db.Pool)
	var subID string
	if e = db.Pool.QueryRow(ctx, `INSERT INTO trader_sync_subscriptions(owner_id,wallet,desired_state,observation_state) VALUES($1,decode(repeat('44',20),'hex'),'enabled','pending_baseline') RETURNING id`, acct.ID).Scan(&subID); e != nil {
		t.Fatal(e)
	}
	before, e := a.GetAccountAccess(ctx, acct.ID)
	if e != nil {
		t.Fatal(e)
	}
	next := before.Clone()
	next.Modules[accountaccess.ModuleTraderSync] = accountaccess.AccessLevelNone
	fail := true
	calls := 0
	a.SetAccessChangeHook(func(ctx context.Context, tx pgx.Tx, id string, previous, next accountaccess.Access) error {
		calls++
		if previous.Modules[accountaccess.ModuleTraderSync] != accountaccess.AccessLevelReadWrite {
			t.Fatal(previous)
		}
		if e := s.RevokeTx(ctx, tx, id, "permission_revoked"); e != nil {
			return e
		}
		if fail {
			return errors.New("hook failed")
		}
		return nil
	})
	if _, e = a.UpdateAccountAccess(ctx, acct.ID, next, before.Revision); e == nil {
		t.Fatal("hook error committed")
	}
	got, e := a.GetAccountAccess(ctx, acct.ID)
	if e != nil || got.Revision != before.Revision || got.Modules[accountaccess.ModuleTraderSync] != accountaccess.AccessLevelReadWrite {
		t.Fatal(got, e)
	}
	var state string
	if e = db.Pool.QueryRow(ctx, "SELECT desired_state FROM trader_sync_subscriptions WHERE id=$1", subID).Scan(&state); e != nil || state != "enabled" {
		t.Fatal(state, e)
	}
	fail = false
	if _, e = a.UpdateAccountAccess(ctx, acct.ID, next, before.Revision); e != nil {
		t.Fatal(e)
	}
	if calls != 2 {
		t.Fatal(calls)
	}
}

func TestConcurrentDuplicateAndIdempotentCreate(t *testing.T) {
	for _, sameRequest := range []bool{false, true} {
		t.Run(fmt.Sprint(sameRequest), func(t *testing.T) {
			db := pgtest.New(t, migrations.FS, migrations.Dir)
			ctx := context.Background()
			a := as.NewSQLStore(db.Pool)
			acct, e := a.EnsureDevelopmentAccount(ctx, accountcredentials.DevelopmentRoleMember)
			if e != nil {
				t.Fatal(e)
			}
			store := ts.NewSQLStore(db.Pool)
			r := &fakeRevalidator{barrier: make(chan struct{}), arrived: make(chan struct{}, 2)}
			b := &fakeBaseline{}
			c1, e := db.Pool.Acquire(ctx)
			if e != nil {
				t.Fatal(e)
			}
			defer c1.Release()
			c2, e := db.Pool.Acquire(ctx)
			if e != nil {
				t.Fatal(e)
			}
			defer c2.Release()
			var p1, p2 int
			if e = c1.QueryRow(ctx, "select pg_backend_pid()").Scan(&p1); e != nil {
				t.Fatal(e)
			}
			if e = c2.QueryRow(ctx, "select pg_backend_pid()").Scan(&p2); e != nil || p1 == p2 {
				t.Fatal(p1, p2, e)
			}
			svc1, _ := NewSubscriptionService(c1, store, r, b)
			svc2, _ := NewSubscriptionService(c2, store, r, b)
			makeInput := func() tm.CreateInput {
				raw := []byte(uuid.NewString())
				d := sha256.Sum256(raw)
				wallet := common.HexToAddress("0x1111111111111111111111111111111111111111")
				id := tm.Identity{Wallet: wallet, ResolutionInput: wallet.Hex(), Digest: sha256.Sum256(wallet.Bytes())}
				if e := txgate.WithAccountTx(ctx, db.Pool, acct.ID, func(tx pgx.Tx) error {
					return store.SaveConfirmationTx(ctx, tx, acct.ID, id, d[:], time.Now().Add(time.Minute))
				}); e != nil {
					t.Fatal(e)
				}
				return tm.CreateInput{Token: base64.RawURLEncoding.EncodeToString(raw), RequestID: uuid.NewString()}
			}
			x, y := makeInput(), makeInput()
			if sameRequest {
				y = x
			}
			type outcome struct {
				s tm.Subscription
				e error
			}
			out := make(chan outcome, 2)
			go func() { s, e := svc1.Create(ctx, acct.ID, x); out <- outcome{s, e} }()
			go func() { s, e := svc2.Create(ctx, acct.ID, y); out <- outcome{s, e} }()
			<-r.arrived
			<-r.arrived
			close(r.barrier)
			one, two := <-out, <-out
			if sameRequest {
				if one.e != nil || two.e != nil || one.s.ID != two.s.ID {
					t.Fatal(one, two)
				}
			} else if !((one.e == nil && status.Code(two.e) == codes.AlreadyExists) || (two.e == nil && status.Code(one.e) == codes.AlreadyExists)) {
				t.Fatal(one, two)
			}
			var count int
			if e = db.Pool.QueryRow(ctx, "SELECT count(*) FROM trader_sync_subscriptions WHERE owner_id=$1", acct.ID).Scan(&count); e != nil || count != 1 {
				t.Fatal(count, e)
			}
		})
	}
}

func TestRevocationAndPermitShareOwnerGate(t *testing.T) {
	for _, permitFirst := range []bool{true, false} {
		for _, outcome := range []string{"sent", "unknown", "failed"} {
			t.Run(fmt.Sprint(permitFirst, outcome), func(t *testing.T) {
				db := pgtest.New(t, migrations.FS, migrations.Dir)
				ctx := context.Background()
				a := as.NewSQLStore(db.Pool)
				acct, e := a.EnsureDevelopmentAccount(ctx, accountcredentials.DevelopmentRoleMember)
				if e != nil {
					t.Fatal(e)
				}
				s := ts.NewSQLStore(db.Pool)
				ns := notificationstore.NewSQLStore(db.Pool)
				entered, release := make(chan struct{}), make(chan struct{})
				a.SetAccessChangeHook(func(ctx context.Context, tx pgx.Tx, id string, prev, next accountaccess.Access) error {
					if prev.Modules[accountaccess.ModuleTraderSync] == accountaccess.AccessLevelReadWrite && next.Modules[accountaccess.ModuleTraderSync] == accountaccess.AccessLevelNone {
						if !permitFirst {
							close(entered)
							<-release
						}
						return s.RevokeTx(ctx, tx, id, "permission_revoked")
					}
					return nil
				})
				if _, e = db.Pool.Exec(ctx, `INSERT INTO telegram_bindings(account_id,telegram_user_id,telegram_chat_id,telegram_display_name,revision) VALUES($1,123,123,'test',1)`, acct.ID); e != nil {
					t.Fatal(e)
				}
				var id int64
				if e = db.Pool.QueryRow(ctx, `INSERT INTO account_notification_deliveries(account_id,idempotency_key,payload_digest,source,severity,body,channel,status,telegram_chat_id,binding_revision) VALUES($1,'old',decode(repeat('ab',32),'hex'),'trader_sync','info','hello','telegram','pending',123,1) RETURNING id`, acct.ID).Scan(&id); e != nil {
					t.Fatal(e)
				}
				candidate := delivery.Candidate{Ref: delivery.WorkRef{Kind: "account", ID: id}, ChatID: 123}
				access, e := a.GetAccountAccess(ctx, acct.ID)
				if e != nil {
					t.Fatal(e)
				}
				access.Modules[accountaccess.ModuleTraderSync] = accountaccess.AccessLevelNone
				type permitResult struct {
					p delivery.Permit
					e error
				}
				permits := make(chan permitResult, 1)
				revoked := make(chan error, 1)
				if permitFirst {
					go func() {
						p, e := ns.Authorize(ctx, candidate, uuid.New(), func() error { close(entered); <-release; return nil })
						permits <- permitResult{p, e}
					}()
					<-entered
					go func() { _, e := a.UpdateAccountAccess(ctx, acct.ID, access, access.Revision); revoked <- e }()
					select {
					case early := <-revoked:
						t.Error("revocation bypassed held permit gate")
						revoked <- early
					case <-time.After(50 * time.Millisecond):
					}
					close(release)
				} else {
					go func() { _, e := a.UpdateAccountAccess(ctx, acct.ID, access, access.Revision); revoked <- e }()
					<-entered
					go func() { p, e := ns.Authorize(ctx, candidate, uuid.New(), nil); permits <- permitResult{p, e} }()
					select {
					case early := <-permits:
						t.Error("permit bypassed held revocation gate")
						permits <- early
					case <-time.After(50 * time.Millisecond):
					}
					close(release)
				}
				p := <-permits
				if e = <-revoked; e != nil {
					t.Fatal(e)
				}
				if permitFirst {
					if p.e != nil {
						t.Fatal(p.e)
					}
					if e = ns.RecordOutcome(ctx, p.p, delivery.Outcome{Kind: outcome, Code: "provider_failure", MessageID: "77"}, time.Now()); e != nil {
						t.Fatal(e)
					}
				} else if !errors.Is(p.e, notificationstore.ErrDeliveryNotEligible) {
					t.Fatal(p.e)
				}
				access, e = a.GetAccountAccess(ctx, acct.ID)
				if e != nil {
					t.Fatal(e)
				}
				access.Modules[accountaccess.ModuleTraderSync] = accountaccess.AccessLevelReadWrite
				if _, e = a.UpdateAccountAccess(ctx, acct.ID, access, access.Revision); e != nil {
					t.Fatal(e)
				}
				if _, e = ns.Authorize(ctx, candidate, uuid.New(), nil); !errors.Is(e, notificationstore.ErrDeliveryNotEligible) {
					t.Fatal("old work revived", e)
				}
				var state string
				var tombstone bool
				if e = db.Pool.QueryRow(ctx, "SELECT status,eligibility_revoked_at IS NOT NULL FROM account_notification_deliveries WHERE id=$1", id).Scan(&state, &tombstone); e != nil {
					t.Fatal(e)
				}
				want := "cancelled"
				if permitFirst && outcome != "failed" {
					want = outcome
				}
				if state != want || !tombstone {
					t.Fatal(state, want, tombstone)
				}
			})
		}
	}
}

func TestControllerUpdatesDifferentOwnersIndependently(t *testing.T) {
	db := pgtest.New(t, migrations.FS, migrations.Dir)
	ctx := context.Background()
	a := as.NewSQLStore(db.Pool)
	one, e := a.EnsureDevelopmentAccount(ctx, accountcredentials.DevelopmentRoleMember)
	if e != nil {
		t.Fatal(e)
	}
	two := uuid.NewString()
	if _, e = db.Pool.Exec(ctx, `INSERT INTO athena_account(account_id,username,identity_provider,identity_subject,verified_email) VALUES($1,'other-user','google','subject-two','two@example.test')`, two); e != nil {
		t.Fatal(e)
	}
	if _, e = db.Pool.Exec(ctx, `INSERT INTO account_access(account_id,login_enabled,api_key_enabled,profit_sharing_enabled,revision) VALUES($1,true,false,false,1)`, two); e != nil {
		t.Fatal(e)
	}
	if _, e = db.Pool.Exec(ctx, `INSERT INTO account_module_access(account_id,module,access_level) SELECT $1,module,'none' FROM account_module_access WHERE account_id=$2`, two, one.ID); e != nil {
		t.Fatal(e)
	}
	entered, release := make(chan struct{}), make(chan struct{})
	a.SetAccessChangeHook(func(ctx context.Context, tx pgx.Tx, id string, prev, next accountaccess.Access) error {
		if id == one.ID {
			close(entered)
			<-release
		}
		return nil
	})
	controller, e := accountaccess.NewController(ctx, a)
	if e != nil {
		t.Fatal(e)
	}
	first, _ := controller.Get(one.ID)
	second, _ := controller.Get(two)
	first.APIKeyEnabled = false
	second.APIKeyEnabled = true
	done := make(chan error, 1)
	go func() { _, e := controller.Update(ctx, one.ID, first, first.Revision); done <- e }()
	<-entered
	otherDone := make(chan error, 1)
	go func() { _, e := controller.Update(ctx, two, second, second.Revision); otherDone <- e }()
	select {
	case e := <-otherDone:
		if e != nil {
			t.Error(e)
		}
	case <-time.After(2 * time.Second):
		t.Error("unrelated owner blocked by global controller mutex")
	}
	close(release)
	if e = <-done; e != nil {
		t.Fatal(e)
	}
}
