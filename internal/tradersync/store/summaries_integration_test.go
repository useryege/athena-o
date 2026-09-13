//go:build integration

package store

import (
	"context"
	"encoding/json"
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
	tm "github.com/useryege/athena/internal/tradersync/types"
	"strings"
	"testing"
	"time"
)

func TestSummaryFreezeIsAtomicCompleteAndUsesRealPartEligibility(t *testing.T) {
	db := pgtest.New(t, migrations.FS, migrations.Dir)
	ctx := context.Background()
	owner, e := ac.NewSQLStore(db.Pool).EnsureDevelopmentAccount(ctx, accountcredentials.DevelopmentRoleMember)
	if e != nil {
		t.Fatal(e)
	}
	if _, e = db.Pool.Exec(ctx, `INSERT INTO telegram_bindings(account_id,telegram_user_id,telegram_chat_id,telegram_display_name,revision)VALUES($1,123,123,'test',1)`, owner.ID); e != nil {
		t.Fatal(e)
	}
	s := runtimeTestStore(t, db.Pool)
	if e = s.ConfigureActivities("https://athena.test/base"); e != nil {
		t.Fatal(e)
	}
	first := activityFixture(t, db.Pool, owner.ID, 1)
	for i := 1; i <= 12; i++ {
		in := first
		if i > 1 {
			in = activitySource(t, db.Pool, owner.ID, first.Candidate.SubscriptionID, first.Candidate.AttemptID, first.Trade.Wallet, first.Confirmation.SettledAt, i)
		}
		if _, _, e = s.Project(ctx, in); e != nil {
			t.Fatal(e)
		}
	}
	rollback := errors.New("rollback frozen permit")
	var batch tm.SummaryBatch
	freeze := func(tx pgx.Tx) error { var e error; batch, e = s.FreezeSummaryTx(ctx, tx, owner.ID, 1, 123); return e }
	e = txgate.WithAccountTx(ctx, db.Pool, owner.ID, func(tx pgx.Tx) error {
		if e := freeze(tx); e != nil {
			return e
		}
		if batch.ID <= 0 || len(batch.Parts) == 0 {
			t.Error("waiting members were not frozen into a complete batch")
		}
		return rollback
	})
	if !errors.Is(e, rollback) {
		t.Fatal(e)
	}
	if batch.ID <= 0 || len(batch.Parts) == 0 {
		t.Fatal("freeze returned no batch")
	}
	var count int
	if e = db.Pool.QueryRow(ctx, `SELECT count(*) FROM trader_sync_summary_batches`).Scan(&count); e != nil || count != 0 {
		t.Fatal("rollback left batch", count, e)
	}
	var permit delivery.Permit
	e = txgate.WithAccountTx(ctx, db.Pool, owner.ID, freeze)
	if e != nil {
		t.Fatal(e)
	}
	if len(batch.Parts[0].ActivityIDs) != 2 {
		t.Fatal("freeze did not include all waiting activities", batch.Parts)
	}
	n := ns.NewSQLStore(db.Pool)
	permit, e = n.Authorize(ctx, delivery.Candidate{Ref: delivery.WorkRef{Kind: "account", ID: batch.Parts[0].DeliveryID}, ChatID: 123}, uuid.New(), nil)
	if e != nil {
		t.Fatal("real summary part was not eligible", e)
	}
	payload, e := delivery.DecodePayload(permit.Payload)
	if e != nil || payload.Format != "plain" || payload.Text != batch.Parts[0].Text {
		t.Fatal("permit does not preserve frozen payload", e)
	}
	if e = db.Pool.QueryRow(ctx, `SELECT count(*) FROM account_notification_deliveries WHERE id=$1 AND activity_id IS NULL AND source='trader_sync'`, permit.Work.ID).Scan(&count); e != nil || count != 1 {
		t.Fatal("part impersonated ordinary activity", count, e)
	}
	if _, e = db.Pool.Exec(ctx, `UPDATE trader_sync_summary_parts SET text='changed' WHERE delivery_id=$1`, permit.Work.ID); e == nil {
		t.Fatal("frozen part text mutable")
	}
	if _, e = db.Pool.Exec(ctx, `UPDATE trader_sync_alert_memberships SET batch_id=NULL WHERE batch_id=$1`, batch.ID); e == nil {
		t.Fatal("frozen membership detached")
	}
	if _, e = n.DeleteTelegramBinding(ctx, owner.ID); e != nil {
		t.Fatal(e)
	}
	if e = n.RecordOutcome(ctx, permit, delivery.Outcome{Kind: "retryable", Code: "429"}, batch.FrozenAt); e != nil {
		t.Fatal(e)
	}
	var state string
	if e = db.Pool.QueryRow(ctx, `SELECT status FROM account_notification_deliveries WHERE id=$1`, permit.Work.ID).Scan(&state); e != nil || state != "cancelled" {
		t.Fatal("revoked part retried", state, e)
	}
}

func summaryStoreFixture(t *testing.T) (*pgxpool.Pool, *SQLStore, string) {
	db := pgtest.New(t, migrations.FS, migrations.Dir)
	ctx := context.Background()
	owner, e := ac.NewSQLStore(db.Pool).EnsureDevelopmentAccount(ctx, accountcredentials.DevelopmentRoleMember)
	if e != nil {
		t.Fatal(e)
	}
	if _, e = db.Pool.Exec(ctx, `INSERT INTO telegram_bindings(account_id,telegram_user_id,telegram_chat_id,telegram_display_name,revision)VALUES($1,123,123,'test',1)`, owner.ID); e != nil {
		t.Fatal(e)
	}
	s := runtimeTestStore(t, db.Pool)
	if e = s.ConfigureActivities("https://athena.test/base"); e != nil {
		t.Fatal(e)
	}
	first := activityFixture(t, db.Pool, owner.ID, 1)
	for i := 1; i <= 12; i++ {
		in := first
		if i > 1 {
			in = activitySource(t, db.Pool, owner.ID, first.Candidate.SubscriptionID, first.Candidate.AttemptID, first.Trade.Wallet, first.Confirmation.SettledAt, i)
		}
		if _, _, e = s.Project(ctx, in); e != nil {
			t.Fatal(e)
		}
	}
	return db.Pool, s, owner.ID
}

func TestSummaryStoppedRecoveryIncludesSentWithoutStartAndNeverResends(t *testing.T) {
	for _, kind := range []string{"sending", "sent", "unknown", "actual_start"} {
		t.Run(kind, func(t *testing.T) {
			pool, s, owner := summaryStoreFixture(t)
			ctx := context.Background()
			n := ns.NewSQLStore(pool)
			inc := uuid.New()
			if _, e := pool.Exec(ctx, `INSERT INTO notification_sender_instances(incarnation,hostname,process_id,process_identity)VALUES($1,'fixture',1,'stopped-test')`, inc); e != nil {
				t.Fatal(e)
			}
			var batch tm.SummaryBatch
			var permit delivery.Permit
			e := txgate.WithAccountTx(ctx, pool, owner, func(tx pgx.Tx) error {
				var e error
				batch, e = s.FreezeSummaryTx(ctx, tx, owner, 1, 123)
				if e != nil {
					return e
				}
				permit, e = n.AuthorizeTx(ctx, tx, delivery.Candidate{Ref: delivery.WorkRef{Kind: "account", ID: batch.Parts[0].DeliveryID}, OwnerID: owner, ChatID: 123}, inc, nil)
				if e != nil {
					return e
				}
				return n.AttachSummaryPermitTx(ctx, tx, batch.ID, permit)
			})
			if e != nil {
				t.Fatal(e)
			}
			if kind == "sent" || kind == "unknown" {
				out := delivery.Outcome{Kind: kind}
				if kind == "sent" {
					out.MessageID = "42"
				}
				if e = n.RecordOutcome(ctx, permit, out, time.Now()); e != nil {
					t.Fatal(e)
				}
			}
			if kind == "actual_start" {
				if e = n.RecordStarted(ctx, permit, time.Now().Add(-time.Minute)); e != nil {
					t.Fatal(e)
				}
			}
			if e = n.RecoverSender(ctx, inc); !errors.Is(e, ns.ErrSenderActive) {
				t.Fatal("recovery skipped stop proof", e)
			}
			var stopped time.Time
			if e = pool.QueryRow(ctx, `UPDATE notification_sender_instances SET stopped_at=clock_timestamp(),stop_confirmation='operator' WHERE incarnation=$1 RETURNING stopped_at`, inc).Scan(&stopped); e != nil {
				t.Fatal(e)
			}
			if e = n.RecoverSender(ctx, inc); e != nil {
				t.Fatal(e)
			}
			var unresolved, started bool
			var basis *time.Time
			var reason string
			if e = pool.QueryRow(ctx, `SELECT h.current_batch_id IS NOT NULL,b.first_started_at IS NOT NULL,b.recovery_basis_at,b.recovery_reason FROM trader_sync_summary_heads h JOIN trader_sync_summary_batches b ON b.owner_id=h.owner_id WHERE b.id=$1`, batch.ID).Scan(&unresolved, &started, &basis, &reason); e != nil {
				t.Fatal(e)
			}
			if unresolved || basis == nil || started != (kind == "actual_start") || reason == "" {
				t.Fatal("stopped head did not resolve honestly", unresolved, started, basis, reason)
			}
			if kind != "actual_start" && !basis.Equal(stopped) {
				t.Fatal("missing start did not use conservative stop basis", basis, stopped)
			}
			var state string
			var attempts int
			if e = pool.QueryRow(ctx, `SELECT status,attempts FROM account_notification_deliveries WHERE id=$1`, permit.Work.ID).Scan(&state, &attempts); e != nil {
				t.Fatal(e)
			}
			want := "unknown"
			if kind == "sent" {
				want = "sent"
			}
			if state != want || attempts != 1 {
				t.Fatal("recovery repeated or lost result", state, attempts)
			}
			if e = n.RecoverSender(ctx, inc); e != nil {
				t.Fatal("recovery not idempotent", e)
			}
		})
	}
}

func TestSummaryPartsHaveIndependentOutcomesAndCompleteMembership(t *testing.T) {
	pool, s, owner := summaryStoreFixture(t)
	ctx := context.Background()
	n := ns.NewSQLStore(pool)
	// Metadata enrichment before freeze is visible; after freeze the payload never
	// changes. Each single Combo activity intentionally spans several parts.
	var metadata tm.TradeMetadata
	metadata.Relationship = "AND"
	metadata.Market.Outcome = "YES"
	metadata.LegsEvidence.Availability = "available"
	for i := 0; i < 36; i++ {
		metadata.Legs = append(metadata.Legs, tm.ComboLeg{PositionID: fmt.Sprint(i + 1), Market: tm.MarketRef{Evidence: tm.Evidence{Availability: "available"}, Title: strings.Repeat("🙂", 70), ConditionID: fmt.Sprint(i + 2), URL: fmt.Sprintf("https://market.test/leg/%d", i)}})
	}
	data, e := json.Marshal(metadata)
	if e != nil {
		t.Fatal(e)
	}
	if _, e = pool.Exec(ctx, `UPDATE trader_sync_market_metadata SET metadata_json=$1`, data); e != nil {
		t.Fatal(e)
	}
	var batch tm.SummaryBatch
	e = txgate.WithAccountTx(ctx, pool, owner, func(tx pgx.Tx) error { var e error; batch, e = s.FreezeSummaryTx(ctx, tx, owner, 1, 123); return e })
	if e != nil {
		t.Fatal(e)
	}
	if len(batch.Parts) < 3 {
		t.Fatal("fixture lacks split members")
	}
	progress, e := s.SummaryProgress(ctx, owner, batch.ID)
	if e != nil || progress.Total != int64(len(batch.Parts)) || progress.Success {
		t.Fatal("initial part progress missing", progress, e)
	}
	for i, part := range batch.Parts {
		p, e := n.Authorize(ctx, delivery.Candidate{Ref: delivery.WorkRef{Kind: "account", ID: part.DeliveryID}, OwnerID: owner, ChatID: 123}, uuid.New(), nil)
		if e != nil {
			t.Fatal(e)
		}
		outcome := delivery.Outcome{Kind: "sent", MessageID: fmt.Sprint(i)}
		if i == 1 {
			outcome = delivery.Outcome{Kind: "unknown", Code: "response_lost"}
		}
		if e = n.RecordOutcome(ctx, p, outcome, time.Now()); e != nil {
			t.Fatal(e)
		}
		if _, e = n.Authorize(ctx, delivery.Candidate{Ref: p.Work, ChatID: 123}, uuid.New(), nil); !errors.Is(e, ns.ErrDeliveryNotEligible) {
			t.Fatal("terminal part resent", e)
		}
	}
	progress, e = s.SummaryProgress(ctx, owner, batch.ID)
	if e != nil || progress.Success || progress.Counts["unknown"] != 1 || progress.Counts["sent"] != int64(len(batch.Parts)-1) {
		t.Fatal("partial success masqueraded as full batch success", progress, e)
	}
	var distinct, links int
	if e = pool.QueryRow(ctx, `SELECT count(DISTINCT activity_id),count(*) FROM trader_sync_summary_part_items WHERE batch_id=$1`, batch.ID).Scan(&distinct, &links); e != nil || distinct != 2 || links <= 2 {
		t.Fatal("cross-part activity mappings incomplete", distinct, links, e)
	}
}

func TestSummaryAllSentIsOnlyCompleteSuccess(t *testing.T) {
	pool, s, owner := summaryStoreFixture(t)
	ctx := context.Background()
	n := ns.NewSQLStore(pool)
	var batch tm.SummaryBatch
	if e := txgate.WithAccountTx(ctx, pool, owner, func(tx pgx.Tx) error { var e error; batch, e = s.FreezeSummaryTx(ctx, tx, owner, 1, 123); return e }); e != nil {
		t.Fatal(e)
	}
	for _, part := range batch.Parts {
		p, e := n.Authorize(ctx, delivery.Candidate{Ref: delivery.WorkRef{Kind: "account", ID: part.DeliveryID}, ChatID: 123}, uuid.New(), nil)
		if e != nil {
			t.Fatal(e)
		}
		if e = n.RecordOutcome(ctx, p, delivery.Outcome{Kind: "sent", MessageID: "42"}, time.Now()); e != nil {
			t.Fatal(e)
		}
	}
	progress, e := s.SummaryProgress(ctx, owner, batch.ID)
	if e != nil || !progress.Success || progress.Counts["sent"] != progress.Total || progress.Total == 0 {
		t.Fatal("complete successful batch not recognized", progress, e)
	}
	another, e := s.SummaryProgress(ctx, uuid.NewString(), batch.ID)
	if e != nil || another.Total != 0 || another.Success {
		t.Fatal("cross-owner summary leaked", another, e)
	}
}

func TestSummaryHeadCannotAttachUnrelatedOrdinaryPermit(t *testing.T) {
	pool, s, owner := summaryStoreFixture(t)
	ctx := context.Background()
	n := ns.NewSQLStore(pool)
	var ordinary int64
	if e := pool.QueryRow(ctx, `SELECT id FROM account_notification_deliveries WHERE activity_id IS NOT NULL ORDER BY id LIMIT 1`).Scan(&ordinary); e != nil {
		t.Fatal(e)
	}
	e := txgate.WithAccountTx(ctx, pool, owner, func(tx pgx.Tx) error {
		batch, e := s.FreezeSummaryTx(ctx, tx, owner, 1, 123)
		if e != nil {
			return e
		}
		p, e := n.AuthorizeTx(ctx, tx, delivery.Candidate{Ref: delivery.WorkRef{Kind: "account", ID: ordinary}, OwnerID: owner, ChatID: 123}, uuid.New(), nil)
		if e != nil {
			return e
		}
		e = n.AttachSummaryPermitTx(ctx, tx, batch.ID, p)
		if !errors.Is(e, ns.ErrStalePermit) {
			t.Errorf("head accepted permit unrelated to its frozen parts: %v", e)
		}
		return ns.ErrStalePermit // roll back both test permissions and the frozen batch
	})
	if !errors.Is(e, ns.ErrStalePermit) {
		t.Fatal(e)
	}
}

func TestSummaryRevokedWaitingNeverBuildsEmptyBatch(t *testing.T) {
	pool, s, owner := summaryStoreFixture(t)
	ctx := context.Background()
	n := ns.NewSQLStore(pool)
	if _, e := n.DeleteTelegramBinding(ctx, owner); e != nil {
		t.Fatal(e)
	}
	e := txgate.WithAccountTx(ctx, pool, owner, func(tx pgx.Tx) error { _, e := s.FreezeSummaryTx(ctx, tx, owner, 1, 123); return e })
	if !errors.Is(e, ns.ErrSummaryNotReady) {
		t.Fatal("revoked waiting member froze", e)
	}
	var count int
	if e = pool.QueryRow(ctx, `SELECT count(*) FROM trader_sync_summary_batches`).Scan(&count); e != nil || count != 0 {
		t.Fatal("empty/revoked batch persisted", count, e)
	}
}
