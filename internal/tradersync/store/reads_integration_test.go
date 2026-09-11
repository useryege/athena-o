//go:build integration

package store

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/ethereum/go-ethereum/common"
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
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"strings"
	"testing"
	"time"
)

func TestReadActivitiesSnapshotRefreshAndPrivacy(t *testing.T) {
	db := pgtest.New(t, migrations.FS, migrations.Dir)
	ctx := context.Background()
	owner, err := ac.NewSQLStore(db.Pool).EnsureDevelopmentAccount(ctx, accountcredentials.DevelopmentRoleMember)
	if err != nil {
		t.Fatal(err)
	}
	s := NewSQLStore(db.Pool)
	if err = s.ConfigureActivities("https://athena.test"); err != nil {
		t.Fatal(err)
	}
	in := activityFixture(t, db.Pool, owner.ID, 1)
	first, _, err := s.Project(ctx, in)
	if err != nil {
		t.Fatal(err)
	}
	page, err := s.ReadActivities(ctx, owner.ID, tm.ActivityReadInput{Limit: 2})
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Activities) != 1 || page.Activities[0].ID != first || page.Snapshot != first || page.HasNewer {
		t.Fatalf("first snapshot: %+v", page)
	}
	// A higher committed ID with earlier event time must remain outside the old page.
	next := activitySource(t, db.Pool, owner.ID, in.Candidate.SubscriptionID, in.Candidate.AttemptID, in.Trade.Wallet, in.Candidate.ReceivedAt.Add(-time.Second), 2)
	second, _, err := s.Project(ctx, next)
	if err != nil {
		t.Fatal(err)
	}
	old, err := s.ReadActivities(ctx, owner.ID, tm.ActivityReadInput{Limit: 2, Snapshot: &first, Lower: &first, Upper: &first, Refresh: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(old.Activities) != 1 || old.Activities[0].ID != first || !old.HasNewer {
		t.Fatalf("refresh changed original membership: %+v", old)
	}
	empty, err := s.ReadActivities(ctx, owner.ID, tm.ActivityReadInput{Limit: 2, Snapshot: &first, Refresh: true, Empty: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(empty.Activities) != 0 || !empty.HasNewer {
		t.Fatalf("empty refresh grew: %+v", empty)
	}
	latest, err := s.ReadActivities(ctx, owner.ID, tm.ActivityReadInput{Limit: 2})
	if err != nil {
		t.Fatal(err)
	}
	if len(latest.Activities) != 2 || latest.Activities[0].ID != second {
		t.Fatalf("newest missing: %+v", latest)
	}
	if latest.Activities[0].NotificationMode != "in_app_only" || latest.Activities[0].Delivery != nil {
		t.Fatal("invented delivery")
	}
	if latest.Activities[0].SourceLocation.LogIndex != 2 || latest.Activities[0].SourceLocation.ChainID != 137 {
		t.Fatal("original source location missing")
	}
	// Cross-owner and absent identifiers share NotFound after the owner grant check.
	_, err = s.ReadActivities(ctx, owner.ID, tm.ActivityReadInput{Limit: 1, ActivityID: second + 100000})
	if status.Code(err) != codes.NotFound {
		t.Fatalf("absent detail: %v", err)
	}
	_, err = s.ReadActivities(ctx, owner.ID, tm.ActivityReadInput{Limit: 1, Filter: tm.ActivityFilter{SubscriptionID: uuid.NewString()}})
	if status.Code(err) != codes.NotFound {
		t.Fatalf("absent subscription: %v", err)
	}
	if _, err = db.Pool.Exec(ctx, `UPDATE account_module_access SET access_level='none' WHERE account_id=$1 AND module='trader_sync'`, owner.ID); err != nil {
		t.Fatal(err)
	}
	_, err = s.ReadActivities(ctx, owner.ID, tm.ActivityReadInput{Limit: 2})
	if status.Code(err) != codes.PermissionDenied {
		t.Fatalf("revoked grant read: %v", err)
	}
}

func TestReadSubscriptionsAndAdminAreSeparate(t *testing.T) {
	db := pgtest.New(t, migrations.FS, migrations.Dir)
	ctx := context.Background()
	accounts := ac.NewSQLStore(db.Pool)
	member, err := accounts.EnsureDevelopmentAccount(ctx, accountcredentials.DevelopmentRoleMember)
	if err != nil {
		t.Fatal(err)
	}
	admin, err := accounts.EnsureDevelopmentAccount(ctx, accountcredentials.DevelopmentRoleAdministrator)
	if err != nil {
		t.Fatal(err)
	}
	s := NewSQLStore(db.Pool)
	in := activityFixture(t, db.Pool, member.ID, 1)
	got, err := s.ReadSubscriptions(ctx, member.ID, tm.SubscriptionFilter{View: "current"}, tm.ReadPage{Limit: 2})
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Subscriptions) != 1 || got.Subscriptions[0].ID != in.Candidate.SubscriptionID || got.Quota.Used != 1 || got.Quota.Limit != 10 {
		t.Fatalf("member subscription: %+v", got)
	}
	if got.Subscriptions[0].PausedAt != nil || got.Subscriptions[0].CancelledAt != nil {
		t.Fatal("invented lifecycle time")
	}
	summaries, err := s.ReadSubscriptionSummaries(ctx, admin.ID, tm.AdminSubscriptionFilter{}, tm.ReadPage{Limit: 2})
	if err != nil {
		t.Fatal(err)
	}
	if len(summaries.Summaries) != 1 || summaries.Summaries[0].AccountID != member.ID {
		t.Fatalf("admin overview: %+v", summaries)
	}
	_, err = s.ReadSubscriptions(ctx, admin.ID, tm.SubscriptionFilter{}, tm.ReadPage{Limit: 2})
	if status.Code(err) != codes.PermissionDenied {
		t.Fatalf("admin member read: %v", err)
	}
	_, err = s.ReadSubscriptionSummaries(ctx, member.ID, tm.AdminSubscriptionFilter{}, tm.ReadPage{Limit: 2})
	if status.Code(err) != codes.PermissionDenied {
		t.Fatalf("member admin read: %v", err)
	}
}

func TestReadHistoryAndSummaryOwnership(t *testing.T) {
	db := pgtest.New(t, migrations.FS, migrations.Dir)
	ctx := context.Background()
	owner, err := ac.NewSQLStore(db.Pool).EnsureDevelopmentAccount(ctx, accountcredentials.DevelopmentRoleMember)
	if err != nil {
		t.Fatal(err)
	}
	s := NewSQLStore(db.Pool)
	in := activityFixture(t, db.Pool, owner.ID, 1)
	got, err := s.ReadHistory(ctx, owner.ID, in.Candidate.SubscriptionID, tm.ReadPage{Limit: 2})
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Entries) != 1 || got.Entries[0].Kind != "interval" || got.Entries[0].Interval == nil {
		t.Fatalf("interval history: %+v", got)
	}
	_, err = s.ReadSummaryBatch(ctx, owner.ID, 999)
	if status.Code(err) != codes.NotFound {
		t.Fatalf("unknown batch: %v", err)
	}
	_, err = s.ReadSummaryParts(ctx, owner.ID, 999, 0, tm.ReadPage{Limit: 2})
	if status.Code(err) != codes.NotFound {
		t.Fatalf("unknown batch parts: %v", err)
	}
}

func TestReadHistoryBeyondFiftyKeepsStableBoundary(t *testing.T) {
	db := pgtest.New(t, migrations.FS, migrations.Dir)
	ctx := context.Background()
	owner, e := ac.NewSQLStore(db.Pool).EnsureDevelopmentAccount(ctx, accountcredentials.DevelopmentRoleMember)
	if e != nil {
		t.Fatal(e)
	}
	in := activityFixture(t, db.Pool, owner.ID, 1)
	for i := 0; i < 50; i++ {
		attempt := uuid.NewString()
		if _, e = db.Pool.Exec(ctx, `INSERT INTO trader_sync_baseline_attempts(id,owner_id,subscription_id,activation_generation,collector_epoch,filter_revision,expected_revision,state,effective_at) SELECT $1,owner_id,subscription_id,activation_generation,collector_epoch,filter_revision,expected_revision,'succeeded',effective_at FROM trader_sync_baseline_attempts WHERE id=$2`, attempt, in.Candidate.AttemptID); e != nil {
			t.Fatal(e)
		}
		if _, e = db.Pool.Exec(ctx, `INSERT INTO trader_sync_monitor_intervals(owner_id,subscription_id,baseline_attempt_id,activation_generation,collector_epoch,filter_revision,expected_revision,registered_high,candidate_effective_at,effective_at,state,ended_at) SELECT owner_id,subscription_id,$1,activation_generation,collector_epoch,filter_revision,expected_revision,registered_high,candidate_effective_at,effective_at,'closed',effective_at FROM trader_sync_monitor_intervals WHERE baseline_attempt_id=$2`, attempt, in.Candidate.AttemptID); e != nil {
			t.Fatal(e)
		}
	}
	s := NewSQLStore(db.Pool)
	first, e := s.ReadHistory(ctx, owner.ID, in.Candidate.SubscriptionID, tm.ReadPage{Limit: 51})
	if e != nil || len(first.Entries) != 51 {
		t.Fatal(len(first.Entries), e)
	}
	boundary := first.Entries[49]
	last, e := s.ReadHistory(ctx, owner.ID, in.Candidate.SubscriptionID, tm.ReadPage{Limit: 51, AfterTime: &boundary.SortAt, AfterID: boundary.ID})
	if e != nil || len(last.Entries) != 1 || last.Entries[0].ID != first.Entries[50].ID {
		t.Fatal("history tail truncated or repeated", last, e)
	}
	if first.Entries[0].SortAt != first.Entries[50].SortAt {
		t.Fatal("fixture must exercise equal-time ID tiebreak")
	}
}

func TestReadSummaryPartsBeyondHundredAndSentWithoutStart(t *testing.T) {
	pool, s, owner := summaryStoreFixture(t)
	ctx := context.Background()
	// Synthetic large display evidence exercises paging independently of module
	// leg-count evidence. The production freezer creates every actual part/link.
	var metadata tm.TradeMetadata
	metadata.Relationship = "AND"
	metadata.Market.Outcome = "YES"
	metadata.LegsEvidence.Availability = "available"
	for i := 0; i < 3000; i++ {
		metadata.Legs = append(metadata.Legs, tm.ComboLeg{PositionID: fmt.Sprint(i + 1), Market: tm.MarketRef{Evidence: tm.Evidence{Availability: "available"}, Title: strings.Repeat("🙂", 70), ConditionID: fmt.Sprint(i + 2), URL: fmt.Sprintf("https://market.test/%d", i)}})
	}
	data, e := json.Marshal(metadata)
	if e != nil {
		t.Fatal(e)
	}
	if _, e = pool.Exec(ctx, `UPDATE trader_sync_market_metadata SET metadata_json=$1`, data); e != nil {
		t.Fatal(e)
	}
	var batch tm.SummaryBatch
	if e = txgate.WithAccountTx(ctx, pool, owner, func(tx pgx.Tx) error {
		var err error
		batch, err = s.FreezeSummaryTx(ctx, tx, owner, 1, 123)
		return err
	}); e != nil {
		t.Fatal(e)
	}
	if len(batch.Parts) < 101 {
		t.Fatal("fixture lacks 101 real frozen parts", len(batch.Parts))
	}
	first, e := s.ReadSummaryParts(ctx, owner, batch.ID, 0, tm.ReadPage{Limit: 101})
	if e != nil || len(first.Parts) != 101 {
		t.Fatal(len(first.Parts), e)
	}
	tail, e := s.ReadSummaryParts(ctx, owner, batch.ID, 0, tm.ReadPage{Limit: 101, AfterIndex: 100})
	if e != nil || len(tail.Parts) == 0 || tail.Parts[0].Index != 101 {
		t.Fatal("part 101 lost", tail, e)
	}
	activity := batch.Parts[0].ActivityIDs[0]
	filtered, e := s.ReadSummaryParts(ctx, owner, batch.ID, activity, tm.ReadPage{Limit: 101})
	if e != nil || len(filtered.Parts) != 101 || filtered.Parts[0].Index != 1 || filtered.Parts[100].Index != 101 {
		t.Fatal("single activity crossing page lost associated parts", len(filtered.Parts), e)
	}
	n := ns.NewSQLStore(pool)
	permit, e := n.Authorize(ctx, delivery.Candidate{Ref: delivery.WorkRef{Kind: "account", ID: batch.Parts[0].DeliveryID}, OwnerID: owner, ChatID: 123}, uuid.New(), nil)
	if e != nil {
		t.Fatal(e)
	}
	if e = n.RecordOutcome(ctx, permit, delivery.Outcome{Kind: "sent", MessageID: "42"}, time.Now()); e != nil {
		t.Fatal(e)
	}
	sent, e := s.ReadSummaryParts(ctx, owner, batch.ID, 0, tm.ReadPage{Limit: 1})
	if e != nil || len(sent.Parts) != 1 {
		t.Fatal(e)
	}
	d := sent.Parts[0].Delivery
	if d.Status != "sent" || d.StartedAt != nil || d.ResultAt == nil || d.AuthorizedAt == nil || d.AttemptCount != 1 {
		t.Fatal("sent result fabricated start or dropped actual result", d)
	}
	progress, e := s.ReadSummaryBatch(ctx, owner, batch.ID)
	if e != nil || progress.PartCounts.Sent != 1 || progress.FirstStartedAt != nil || progress.PartCounts.Total != int64(len(batch.Parts)) {
		t.Fatal("batch counts or missing-start evidence wrong", progress, e)
	}
}

func TestReadWaitsForOwnerGrantAndDiscardsDeniedPage(t *testing.T) {
	db := pgtest.New(t, migrations.FS, migrations.Dir)
	ctx := context.Background()
	owner, e := ac.NewSQLStore(db.Pool).EnsureDevelopmentAccount(ctx, accountcredentials.DevelopmentRoleMember)
	if e != nil {
		t.Fatal(e)
	}
	other, e := pgxpool.New(ctx, db.Pool.Config().ConnString())
	if e != nil {
		t.Fatal(e)
	}
	defer other.Close()
	gate, e := txgate.AcquireAccountSession(ctx, db.Pool, owner.ID)
	if e != nil {
		t.Fatal(e)
	}
	defer gate.Release(ctx)
	done := make(chan error, 1)
	go func() {
		page, err := NewSQLStore(other).ReadActivities(ctx, owner.ID, tm.ActivityReadInput{Limit: 1})
		if len(page.Activities) != 0 {
			done <- fmt.Errorf("denied read leaked rows")
			return
		}
		done <- err
	}()
	deadline := time.Now().Add(2 * time.Second)
	for {
		var waiting int
		if e = db.Pool.QueryRow(ctx, `SELECT count(*) FROM pg_stat_activity WHERE datname=current_database() AND wait_event='advisory'`).Scan(&waiting); e != nil {
			t.Fatal(e)
		}
		if waiting > 0 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("read did not wait for owner gate")
		}
		time.Sleep(10 * time.Millisecond)
	}
	if _, e = gate.Conn.Exec(ctx, `UPDATE account_module_access SET access_level='none' WHERE account_id=$1 AND module='trader_sync'`, owner.ID); e != nil {
		t.Fatal(e)
	}
	if e = gate.Release(ctx); e != nil {
		t.Fatal(e)
	}
	select {
	case e = <-done:
		if status.Code(e) != codes.PermissionDenied {
			t.Fatal("read authorized from stale grant", e)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("read did not resume")
	}
}

func TestRolledBackActivityDoesNotMoveSnapshot(t *testing.T) {
	db := pgtest.New(t, migrations.FS, migrations.Dir)
	ctx := context.Background()
	owner, e := ac.NewSQLStore(db.Pool).EnsureDevelopmentAccount(ctx, accountcredentials.DevelopmentRoleMember)
	if e != nil {
		t.Fatal(e)
	}
	s := NewSQLStore(db.Pool)
	if e = s.ConfigureActivities("https://athena.test"); e != nil {
		t.Fatal(e)
	}
	in := activityFixture(t, db.Pool, owner.ID, 1)
	first, _, e := s.Project(ctx, in)
	if e != nil {
		t.Fatal(e)
	}
	candidate := activitySource(t, db.Pool, owner.ID, in.Candidate.SubscriptionID, in.Candidate.AttemptID, in.Trade.Wallet, in.Candidate.ReceivedAt.Add(-time.Second), 2)
	tx, e := db.Pool.Begin(ctx)
	if e != nil {
		t.Fatal(e)
	}
	defer tx.Rollback(ctx)
	var rolledID int64
	if e = tx.QueryRow(ctx, `INSERT INTO trader_sync_activities SELECT (jsonb_populate_record(NULL::trader_sync_activities,to_jsonb(a)||jsonb_build_object('id',nextval('trader_sync_activity_id_seq'),'source_record_id',$2::bigint))).* FROM trader_sync_activities a WHERE id=$1 RETURNING id`, first, candidate.Candidate.SourceID).Scan(&rolledID); e != nil {
		t.Fatal(e)
	}
	if e = tx.Rollback(ctx); e != nil {
		t.Fatal(e)
	}
	if rolledID <= first {
		t.Fatal("rollback fixture did not consume a greater ID")
	}
	result, e := s.ReadActivities(ctx, owner.ID, tm.ActivityReadInput{Limit: 2, Snapshot: &first})
	if e != nil || result.HasNewer || len(result.Activities) != 1 || result.Snapshot != first {
		t.Fatal("uncommitted activity changed snapshot", result, e)
	}
}

func TestAdminCountsDistinctDeliveryAcrossSubscriptionsAndRetries(t *testing.T) {
	pool, s, owner := summaryStoreFixture(t)
	ctx := context.Background()
	otherTarget := activityWalletFixture(t, pool, owner, 13, common.HexToAddress("0x2222222222222222222222222222222222222222"))
	if _, _, e := s.Project(ctx, otherTarget); e != nil {
		t.Fatal(e)
	}
	admin, e := ac.NewSQLStore(pool).EnsureDevelopmentAccount(ctx, accountcredentials.DevelopmentRoleAdministrator)
	if e != nil {
		t.Fatal(e)
	}
	var batch tm.SummaryBatch
	if e = txgate.WithAccountTx(ctx, pool, owner, func(tx pgx.Tx) error {
		var err error
		batch, err = s.FreezeSummaryTx(ctx, tx, owner, 1, 123)
		return err
	}); e != nil {
		t.Fatal(e)
	}
	if len(batch.Parts) != 1 || len(batch.Parts[0].ActivityIDs) != 3 {
		t.Fatal("fixture must share one part across two subscriptions", batch.Parts)
	}
	n := ns.NewSQLStore(pool)
	candidate := delivery.Candidate{Ref: delivery.WorkRef{Kind: "account", ID: batch.Parts[0].DeliveryID}, OwnerID: owner, ChatID: 123}
	for i := 0; i < 2; i++ {
		permit, err := n.Authorize(ctx, candidate, uuid.New(), nil)
		if err != nil {
			t.Fatal(err)
		}
		out := delivery.Outcome{Kind: "retryable", Code: "provider_429"}
		if i == 1 {
			out = delivery.Outcome{Kind: "sent", MessageID: "42"}
		}
		if err = n.RecordOutcome(ctx, permit, out, time.Now()); err != nil {
			t.Fatal(err)
		}
		if i == 0 {
			var next time.Time
			if err = pool.QueryRow(ctx, `SELECT next_attempt_at FROM account_notification_deliveries WHERE id=$1`, candidate.Ref.ID).Scan(&next); err != nil {
				t.Fatal(err)
			}
			if wait := time.Until(next); wait > 0 {
				timer := time.NewTimer(wait + 10*time.Millisecond)
				<-timer.C
			}
		}
	}
	summaries, e := s.ReadSubscriptionSummaries(ctx, admin.ID, tm.AdminSubscriptionFilter{AccountID: owner}, tm.ReadPage{Limit: 10})
	if e != nil || len(summaries.Summaries) != 2 {
		t.Fatal(summaries, e)
	}
	for _, v := range summaries.Summaries {
		want := int64(11)
		if v.SubscriptionID == otherTarget.Candidate.SubscriptionID {
			want = 1
		}
		if v.AssociatedDeliveryCounts.Total != want || v.AssociatedDeliveryCounts.Sent != 1 {
			t.Fatal("shared part or retry double counted", v)
		}
	}
	runtime, e := s.ReadRuntimeStatus(ctx, admin.ID)
	if e != nil {
		t.Fatal(e)
	}
	for _, metric := range runtime.Metrics {
		if metric.Name == "delivery_sent" && metric.Value != "1" {
			t.Fatal("global count duplicated shared part", metric)
		}
	}
	parts, e := s.ReadSummaryParts(ctx, owner, batch.ID, 0, tm.ReadPage{Limit: 1})
	if e != nil || parts.Parts[0].Delivery.AttemptCount != 2 || parts.Parts[0].Delivery.StartedAt != nil {
		t.Fatal("latest attempt/count or missing start lost", parts, e)
	}
}
