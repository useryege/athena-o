//go:build integration

package store_test

import (
	"context"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/useryege/athena/internal/accountcredentials"
	ac "github.com/useryege/athena/internal/accountstate/store"
	"github.com/useryege/athena/internal/accountstate/store/migrations"
	"github.com/useryege/athena/internal/accountstate/txgate"
	"github.com/useryege/athena/internal/notification/delivery"
	ns "github.com/useryege/athena/internal/notification/store"
	"github.com/useryege/athena/internal/testutil/pgtest"
	"testing"
	"time"
)

func frozenActivityFixture(t *testing.T, pool *pgxpool.Pool, owner string, index int, form string) int64 {
	t.Helper()
	ctx := context.Background()
	var id int64
	if _, e := pool.Exec(ctx, `INSERT INTO trader_sync_market_metadata(cache_key,metadata_json)VALUES('fixture','{}')ON CONFLICT DO NOTHING`); e != nil {
		t.Fatal(e)
	}
	e := txgate.WithAccountTx(ctx, pool, owner, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, `
 WITH sub AS(INSERT INTO trader_sync_subscriptions(owner_id,wallet,desired_state,observation_state,target_display)VALUES($1,decode(lpad(to_hex($2::int),40,'0'),'hex'),'enabled','healthy','{"DisplayName":{"Availability":"unavailable","ReasonCode":"fixture"}}')RETURNING *),
 attempt AS(INSERT INTO trader_sync_baseline_attempts(owner_id,subscription_id,activation_generation,expected_revision,state)SELECT owner_id,id,1,1,'succeeded' FROM sub RETURNING *),
 epoch AS(INSERT INTO trader_sync_collector_epochs(fencing_token)VALUES(1)RETURNING id),
 interval AS(INSERT INTO trader_sync_monitor_intervals(owner_id,subscription_id,baseline_attempt_id,activation_generation,collector_epoch,filter_revision,expected_revision,registered_high,candidate_effective_at,effective_at)SELECT owner_id,subscription_id,id,1,(SELECT id FROM epoch),1,1,0,clock_timestamp()-interval '1 minute',clock_timestamp()-interval '1 minute' FROM attempt RETURNING *),
 source AS(INSERT INTO trader_sync_source_records(chain_id,exchange_address,wallet,block_hash,transaction_hash,log_index,block_number,raw_json,collector_epoch,read_sequence,received_at,removed)SELECT 137,decode(repeat('11',20),'hex'),wallet,decode(repeat('aa',32),'hex'),decode(repeat('bb',32),'hex'),$2,1,'{}',(SELECT id FROM epoch),1,clock_timestamp(),false FROM sub RETURNING id),
 candidate AS(INSERT INTO trader_sync_source_candidates(source_record_id,owner_id,subscription_id,activation_generation,baseline_attempt_id,received_at)SELECT source.id,attempt.owner_id,attempt.subscription_id,1,attempt.id,clock_timestamp() FROM source,attempt RETURNING *)
 INSERT INTO trader_sync_activities(owner_id,subscription_id,source_record_id,interval_id,activation_generation,trade_json,metadata_key,target_display_snapshot,note_snapshot,notification_mode,notification_reason,settled_at,received_at,recorded_at)
 SELECT c.owner_id,c.subscription_id,c.source_record_id,i.id,1,'{}','fixture','{"DisplayName":{"Availability":"unavailable","ReasonCode":"fixture"}}','',$3,'',clock_timestamp(),clock_timestamp(),clock_timestamp() FROM candidate c JOIN interval i ON c.subscription_id=i.subscription_id RETURNING id`, owner, index, form).Scan(&id)
	})
	if e != nil {
		t.Fatal(e)
	}
	state := "waiting"
	if form == "ordinary" {
		state = "frozen"
	}
	if _, e = pool.Exec(ctx, `INSERT INTO trader_sync_alert_memberships(activity_id,owner_id,binding_revision,chat_id,form,state,created_at)VALUES($1,$2,1,123,$3,$4,clock_timestamp())`, id, owner, form, state); e != nil {
		t.Fatal(e)
	}
	return id
}
func TestTransactionalEnqueueRollbackFrozenBindingAndDigestIdempotency(t *testing.T) {
	db := pgtest.New(t, migrations.FS, migrations.Dir)
	ctx := context.Background()
	owner, e := ac.NewSQLStore(db.Pool).EnsureDevelopmentAccount(ctx, accountcredentials.DevelopmentRoleMember)
	if e != nil {
		t.Fatal(e)
	}
	if _, e = db.Pool.Exec(ctx, `INSERT INTO telegram_bindings(account_id,telegram_user_id,telegram_chat_id,telegram_display_name,revision)VALUES($1,456,456,'new binding',2)`, owner.ID); e != nil {
		t.Fatal(e)
	}
	id := frozenActivityFixture(t, db.Pool, owner.ID, 1, "ordinary")
	s := ns.NewSQLStore(db.Pool)
	payload, e := delivery.EncodePayload(delivery.Payload{Format: "plain", Text: " frozen <>&🙂\nhttps://athena.test/a?x=1&y=2 "})
	if e != nil {
		t.Fatal(e)
	}
	in := delivery.AccountEnqueue{OwnerID: owner.ID, Source: "trader_sync", ActivityID: id, BindingRevision: 1, ChatID: 123, Payload: payload, RecordedAt: time.Now()}
	rollback := errors.New("abort after enqueue")
	e = txgate.WithAccountTx(ctx, db.Pool, owner.ID, func(tx pgx.Tx) error {
		if _, e := s.EnqueueAccountTx(ctx, tx, in); e != nil {
			return e
		}
		return rollback
	})
	if !errors.Is(e, rollback) {
		t.Fatal(e)
	}
	var count int
	if e = db.Pool.QueryRow(ctx, `SELECT count(*) FROM account_notification_deliveries`).Scan(&count); e != nil || count != 0 {
		t.Fatal(count, e)
	}
	var deliveryID int64
	e = txgate.WithAccountTx(ctx, db.Pool, owner.ID, func(tx pgx.Tx) error { var e error; deliveryID, e = s.EnqueueAccountTx(ctx, tx, in); return e })
	if e != nil {
		t.Fatal(e)
	}
	e = txgate.WithAccountTx(ctx, db.Pool, owner.ID, func(tx pgx.Tx) error {
		got, e := s.EnqueueAccountTx(ctx, tx, in)
		if e == nil && got != deliveryID {
			return fmt.Errorf("duplicate delivery")
		}
		return e
	})
	if e != nil {
		t.Fatal(e)
	}
	changed := in
	changed.Payload, _ = delivery.EncodePayload(delivery.Payload{Format: "html", Text: "changed"})
	e = txgate.WithAccountTx(ctx, db.Pool, owner.ID, func(tx pgx.Tx) error { _, e := s.EnqueueAccountTx(ctx, tx, changed); return e })
	if !errors.Is(e, ns.ErrAccountNotificationIdempotencyConflict) {
		t.Fatal(e)
	}
	if _, e = s.Authorize(ctx, delivery.Candidate{Ref: delivery.WorkRef{Kind: "account", ID: deliveryID}, ChatID: 123}, uuid.New(), nil); !errors.Is(e, ns.ErrDeliveryNotEligible) {
		t.Fatal("old frozen binding gained permit", e)
	}
	var chat, revision int64
	var raw []byte
	if e = db.Pool.QueryRow(ctx, `SELECT telegram_chat_id,binding_revision,payload FROM account_notification_deliveries WHERE id=$1`, deliveryID).Scan(&chat, &revision, &raw); e != nil || chat != 123 || revision != 1 || string(raw) != string(payload) {
		t.Fatal("enqueue reread current binding or changed payload", chat, revision, string(raw), e)
	}
}
func TestBindingTerminationIncludesWaitingSummaryWithoutDeliveryAndRollsBack(t *testing.T) {
	for _, kind := range []string{"delete", "unreachable"} {
		t.Run(kind, func(t *testing.T) {
			db := pgtest.New(t, migrations.FS, migrations.Dir)
			ctx := context.Background()
			a, e := ac.NewSQLStore(db.Pool).EnsureDevelopmentAccount(ctx, accountcredentials.DevelopmentRoleMember)
			if e != nil {
				t.Fatal(e)
			}
			if _, e = db.Pool.Exec(ctx, `INSERT INTO telegram_bindings(account_id,telegram_user_id,telegram_chat_id,telegram_display_name,revision)VALUES($1,123,123,'test',1)`, a.ID); e != nil {
				t.Fatal(e)
			}
			id := frozenActivityFixture(t, db.Pool, a.ID, 1, "summary")
			frozen := frozenActivityFixture(t, db.Pool, a.ID, 2, "summary")
			// The FK now requires a real batch fixture; this test only exercises binding
			// tombstones. Task11's freeze tests cover the actual sealed batch flow.
			if _, e = db.Pool.Exec(ctx, `INSERT INTO trader_sync_summary_batches(id,owner_id,binding_revision,chat_id,oldest_at)VALUES(99,$1,1,123,clock_timestamp())`, a.ID); e != nil {
				t.Fatal(e)
			}
			if _, e = db.Pool.Exec(ctx, `UPDATE trader_sync_alert_memberships SET state='frozen',batch_id=99 WHERE activity_id=$1`, frozen); e != nil {
				t.Fatal(e)
			}
			s := ns.NewSQLStore(db.Pool)
			if _, e = db.Pool.Exec(ctx, `CREATE FUNCTION reject_membership()RETURNS trigger LANGUAGE plpgsql AS $$BEGIN RAISE EXCEPTION 'injected';END$$;CREATE TRIGGER reject_membership BEFORE UPDATE ON trader_sync_alert_memberships FOR EACH ROW EXECUTE FUNCTION reject_membership()`); e != nil {
				t.Fatal(e)
			}
			change := func() error {
				if kind == "delete" {
					_, e := s.DeleteTelegramBinding(ctx, a.ID)
					return e
				}
				return s.MarkTelegramBindingUnreachable(ctx, a.ID, 123, 1, "blocked")
			}
			if e = change(); e == nil {
				t.Fatal("expected rollback")
			}
			var state string
			if e = db.Pool.QueryRow(ctx, `SELECT status FROM telegram_bindings WHERE account_id=$1`, a.ID).Scan(&state); e != nil || state != "connected" {
				t.Fatal("binding escaped failed transaction", state, e)
			}
			if _, e = db.Pool.Exec(ctx, `DROP TRIGGER reject_membership ON trader_sync_alert_memberships;DROP FUNCTION reject_membership()`); e != nil {
				t.Fatal(e)
			}
			if e = change(); e != nil {
				t.Fatal(e)
			}
			var reason string
			var revoked bool
			if e = db.Pool.QueryRow(ctx, `SELECT state,reason,eligibility_revoked_at IS NOT NULL FROM trader_sync_alert_memberships WHERE activity_id=$1`, id).Scan(&state, &reason, &revoked); e != nil || state != "cancelled" || reason != "binding_changed" || !revoked {
				t.Fatal(state, reason, revoked, e)
			}
			var batch int64
			if e = db.Pool.QueryRow(ctx, `SELECT state,batch_id,eligibility_revoked_at IS NOT NULL FROM trader_sync_alert_memberships WHERE activity_id=$1`, frozen).Scan(&state, &batch, &revoked); e != nil || state != "frozen" || batch != 99 || !revoked {
				t.Fatal(state, batch, revoked, e)
			}
		})
	}
}
