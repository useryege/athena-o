//go:build integration

package store

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	et "github.com/ethereum/go-ethereum/core/types"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/useryege/athena/internal/accountcredentials"
	ac "github.com/useryege/athena/internal/accountstate/store"
	"github.com/useryege/athena/internal/accountstate/store/migrations"
	"github.com/useryege/athena/internal/accountstate/txgate"
	"github.com/useryege/athena/internal/testutil/pgtest"
	"github.com/useryege/athena/internal/tradersync/activity"
	q "github.com/useryege/athena/internal/tradersync/store/sqlc"
	tm "github.com/useryege/athena/internal/tradersync/types"
	"testing"
	"time"
)

func activityFixture(t *testing.T, pool *pgxpool.Pool, owner string, index int) tm.Projection {
	return activityWalletFixture(t, pool, owner, index, common.HexToAddress("0x1111111111111111111111111111111111111111"))
}
func activityWalletFixture(t *testing.T, pool *pgxpool.Pool, owner string, index int, wallet common.Address) tm.Projection {
	t.Helper()
	ctx := context.Background()
	sub, attempt, interval := uuid.NewString(), uuid.NewString(), uuid.NewString()
	at := time.Now().UTC().Truncate(time.Second).Add(-time.Second)
	if _, err := pool.Exec(ctx, `INSERT INTO trader_sync_subscriptions(id,owner_id,wallet,desired_state,observation_state,target_display) VALUES($1,$2,$3,'enabled','healthy','{"displayName":{"availability":"unavailable","reasonCode":"fixture_not_queried","source":"fixture"},"avatar":{"availability":"unavailable","reasonCode":"fixture_not_queried","source":"fixture"},"profileURL":{"availability":"unavailable","reasonCode":"fixture_not_queried","source":"fixture"}}'::jsonb)`, sub, owner, wallet.Bytes()); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO trader_sync_baseline_attempts(id,owner_id,subscription_id,activation_generation,expected_revision,state,effective_at)VALUES($1,$2,$3,1,1,'succeeded',$4)`, attempt, owner, sub, at.Add(-time.Minute)); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO trader_sync_monitor_intervals(id,owner_id,subscription_id,baseline_attempt_id,activation_generation,collector_epoch,filter_revision,expected_revision,registered_high,candidate_effective_at,effective_at)VALUES($1,$2,$3,$4,1,1,1,1,0,$5,$5)`, interval, owner, sub, attempt, at.Add(-time.Minute)); err != nil {
		t.Fatal(err)
	}
	return activitySource(t, pool, owner, sub, attempt, wallet, at, index)
}
func activitySource(t *testing.T, pool *pgxpool.Pool, owner, sub, attempt string, wallet common.Address, at time.Time, index int) tm.Projection {
	t.Helper()
	ctx := context.Background()
	var epoch int64
	if err := pool.QueryRow(ctx, `INSERT INTO trader_sync_collector_epochs(fencing_token)VALUES(1) RETURNING id`).Scan(&epoch); err != nil {
		t.Fatal(err)
	}
	raw := et.Log{Topics: []common.Hash{}, Data: []byte{}, Address: common.HexToAddress("0xe111180000d2663c0091e4f400237545b87b996b"), BlockHash: common.HexToHash("0xaa"), TxHash: common.BigToHash(common.Big1), Index: uint(index), BlockNumber: 1}
	data, _ := json.Marshal(raw)
	var id int64
	if err := pool.QueryRow(ctx, `INSERT INTO trader_sync_source_records(chain_id,exchange_address,wallet,block_hash,transaction_hash,log_index,block_number,raw_json,collector_epoch,read_sequence,received_at,removed)VALUES(137,$1,$2,$3,$4,$5,1,$6,$7,1,$8,false) RETURNING id`, raw.Address.Bytes(), wallet.Bytes(), raw.BlockHash.Bytes(), raw.TxHash.Bytes(), index, data, epoch, at).Scan(&id); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO trader_sync_source_candidates(source_record_id,owner_id,subscription_id,activation_generation,baseline_attempt_id,received_at)VALUES($1,$2,$3,1,$4,$5)`, id, owner, sub, attempt, at); err != nil {
		t.Fatal(err)
	}
	return tm.Projection{Candidate: tm.Candidate{SourceID: id, OwnerID: owner, SubscriptionID: sub, Generation: 1, AttemptID: attempt, ReceivedAt: at}, Trade: tm.Trade{Wallet: wallet, Exchange: raw.Address, Side: "BUY", PositionID: "123", CollateralRaw: "2500000", SharesRaw: "10000000", FeeRaw: "100", CollateralSymbol: "USDC", CollateralDecimals: 6, SharesDecimals: 6, SourceVersion: "polygon137-core-v2-ccc0596074f4", PriceNumerator: "1", PriceDenominator: "4"}, Confirmation: tm.CanonicalEvidence{Status: "confirmed", BlockHash: raw.BlockHash, SettledAt: at, CheckedAt: at}}
}
func TestActivityProjectPersistsAndDoesNotReplanDuplicate(t *testing.T) {
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
	id, created, err := s.Project(ctx, in)
	if err != nil || !created || id <= 0 {
		t.Fatalf("projected activity missing: id=%d created=%v err=%v", id, created, err)
	}
	if _, err = db.Pool.Exec(ctx, `INSERT INTO telegram_bindings(account_id,telegram_user_id,telegram_chat_id,telegram_display_name,revision)VALUES($1,123,123,'test',1)`, owner.ID); err != nil {
		t.Fatal(err)
	}
	duplicate, created, err := s.Project(ctx, in)
	if err != nil || created || duplicate != id {
		t.Fatalf("duplicate replanned: %d %v %v", duplicate, created, err)
	}
	var mode, reason string
	var deliveries int
	if err = db.Pool.QueryRow(ctx, `SELECT notification_mode,notification_reason FROM trader_sync_activities WHERE id=$1`, id).Scan(&mode, &reason); err != nil {
		t.Fatal(err)
	}
	if err = db.Pool.QueryRow(ctx, `SELECT count(*) FROM account_notification_deliveries`).Scan(&deliveries); err != nil {
		t.Fatal(err)
	}
	if mode != "in_app_only" || reason != "unbound_at_formation" || deliveries != 0 {
		t.Fatal(mode, reason, deliveries)
	}
}

func TestConfirmationDisplayUsesExactCardAndRejectsDifferentWallet(t *testing.T) {
	db := pgtest.New(t, migrations.FS, migrations.Dir)
	ctx := context.Background()
	owner, err := ac.NewSQLStore(db.Pool).EnsureDevelopmentAccount(ctx, accountcredentials.DevelopmentRoleMember)
	if err != nil {
		t.Fatal(err)
	}
	s := NewSQLStore(db.Pool)
	id := tm.Identity{Wallet: common.HexToAddress("0x123"), ResolutionInput: "address", Digest: [32]byte{1}}
	token := make([]byte, 32)
	name := "Pseudonym from card"
	queryTime := time.Now().UTC().Truncate(time.Microsecond)
	tx, err := db.Pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback(ctx)
	if err = s.SaveConfirmationTx(ctx, tx, owner.ID, id, token, time.Now().Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
	card := tm.ConfirmationCard{Identity: id, DisplayName: tm.Scalar{Value: &name, Evidence: tm.Evidence{Availability: "available", Source: "gamma", QueriedAt: queryTime}}}
	if err = s.SaveConfirmationCardTx(ctx, tx, owner.ID, token, card); err != nil {
		t.Fatal(err)
	}
	got, err := s.ReadConfirmationDisplayTx(ctx, tx, owner.ID, token, id)
	if err != nil || got.DisplayName.Value == nil || *got.DisplayName.Value != name || !got.DisplayName.QueriedAt.Equal(queryTime) {
		t.Fatalf("confirmed display lost: %+v %v", got, err)
	}
	if got.ProfileURL.Availability != "unavailable" || got.ProfileURL.Value != nil {
		t.Fatal("invented profile URL", got)
	}
	card.Identity.Wallet = common.HexToAddress("0x999")
	if err = s.SaveConfirmationCardTx(ctx, tx, owner.ID, token, card); err != nil {
		t.Fatal(err)
	}
	if _, err = s.ReadConfirmationDisplayTx(ctx, tx, owner.ID, token, id); err == nil {
		t.Fatal("accepted another wallet's card")
	}
}

func TestActivityWindowIncludesUnboundAndKeepsFirstTen(t *testing.T) {
	db := pgtest.New(t, migrations.FS, migrations.Dir)
	ctx := context.Background()
	owner, err := ac.NewSQLStore(db.Pool).EnsureDevelopmentAccount(ctx, accountcredentials.DevelopmentRoleMember)
	if err != nil {
		t.Fatal(err)
	}
	s := NewSQLStore(db.Pool)
	s.ConfigureActivities("https://athena.test")
	first := activityFixture(t, db.Pool, owner.ID, 1)
	for i := 1; i <= 12; i++ {
		in := first
		if i > 1 {
			in = activitySource(t, db.Pool, owner.ID, first.Candidate.SubscriptionID, first.Candidate.AttemptID, first.Trade.Wallet, first.Confirmation.SettledAt, i)
		}
		if i == 2 {
			_, err = db.Pool.Exec(ctx, `INSERT INTO telegram_bindings(account_id,telegram_user_id,telegram_chat_id,telegram_display_name,revision)VALUES($1,123,123,'test',1)`, owner.ID)
			if err != nil {
				t.Fatal(err)
			}
		}
		if _, created, err := s.Project(ctx, in); err != nil || !created {
			t.Fatal(i, created, err)
		}
	}
	var ordinary, summary, unbound int
	if err = db.Pool.QueryRow(ctx, `SELECT count(*) FILTER(WHERE notification_mode='ordinary'),count(*) FILTER(WHERE notification_mode='summary'),count(*) FILTER(WHERE notification_mode='in_app_only') FROM trader_sync_activities`).Scan(&ordinary, &summary, &unbound); err != nil {
		t.Fatal(err)
	}
	if ordinary != 9 || summary != 2 || unbound != 1 {
		t.Fatal(ordinary, summary, unbound)
	}
	var waiting, deliveries int
	db.Pool.QueryRow(ctx, `SELECT count(*) FROM trader_sync_alert_memberships WHERE state='waiting' AND batch_id IS NULL`).Scan(&waiting)
	db.Pool.QueryRow(ctx, `SELECT count(*) FROM account_notification_deliveries`).Scan(&deliveries)
	if waiting != 2 || deliveries != 9 {
		t.Fatal(waiting, deliveries)
	}
	// Existing pause intent must not cancel any already formed eligibility.
	if _, err = db.Pool.Exec(ctx, `UPDATE trader_sync_subscriptions SET desired_state='paused' WHERE id=$1`, first.Candidate.SubscriptionID); err != nil {
		t.Fatal(err)
	}
	extra := activitySource(t, db.Pool, owner.ID, first.Candidate.SubscriptionID, first.Candidate.AttemptID, first.Trade.Wallet, first.Confirmation.SettledAt, 13)
	if id, created, err := s.Project(ctx, extra); err != nil || created || id != 0 {
		t.Fatal("paused candidate projected", id, created, err)
	}
	db.Pool.QueryRow(ctx, `SELECT count(*) FROM account_notification_deliveries WHERE status='pending'`).Scan(&deliveries)
	if deliveries != 9 {
		t.Fatal("pause cancelled old queue", deliveries)
	}
	if err = txgate.WithAccountTx(ctx, db.Pool, owner.ID, func(tx pgx.Tx) error { return s.RevokeTx(ctx, tx, owner.ID, "permission_revoked") }); err != nil {
		t.Fatal(err)
	}
	var cancelled int
	db.Pool.QueryRow(ctx, `SELECT count(*) FROM trader_sync_alert_memberships WHERE form='summary' AND state='cancelled' AND batch_id IS NULL AND reason='permission_revoked' AND eligibility_revoked_at IS NOT NULL`).Scan(&cancelled)
	if cancelled != 2 {
		t.Fatal("waiting summary not permanently ended", cancelled)
	}
}

func TestActivityFrozenFactsRollbackAndSequence(t *testing.T) {
	db := pgtest.New(t, migrations.FS, migrations.Dir)
	ctx := context.Background()
	owner, err := ac.NewSQLStore(db.Pool).EnsureDevelopmentAccount(ctx, accountcredentials.DevelopmentRoleMember)
	if err != nil {
		t.Fatal(err)
	}
	s := NewSQLStore(db.Pool)
	s.ConfigureActivities("https://athena.test")
	in := activityFixture(t, db.Pool, owner.ID, 1)
	_, err = db.Pool.Exec(ctx, `INSERT INTO telegram_bindings(account_id,telegram_user_id,telegram_chat_id,telegram_display_name,revision)VALUES($1,123,123,'test',1)`, owner.ID)
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.Pool.Exec(ctx, `CREATE FUNCTION test_fail_delivery() RETURNS trigger LANGUAGE plpgsql AS $$BEGIN RAISE EXCEPTION 'injected enqueue failure';END;$$;CREATE TRIGGER test_fail_delivery BEFORE INSERT ON account_notification_deliveries FOR EACH ROW EXECUTE FUNCTION test_fail_delivery()`)
	if err != nil {
		t.Fatal(err)
	}
	if id, created, err := s.Project(ctx, in); err == nil || created || id != 0 {
		t.Fatal("enqueue failure committed partial activity", id, created, err)
	}
	var n int
	db.Pool.QueryRow(ctx, `SELECT count(*) FROM trader_sync_activities`).Scan(&n)
	if n != 0 {
		t.Fatal("activity escaped rollback")
	}
	db.Pool.QueryRow(ctx, `SELECT count(*) FROM trader_sync_alert_memberships`).Scan(&n)
	if n != 0 {
		t.Fatal("membership escaped rollback")
	}
	if _, err = db.Pool.Exec(ctx, `DROP TRIGGER test_fail_delivery ON account_notification_deliveries`); err != nil {
		t.Fatal(err)
	}
	id, created, err := s.Project(ctx, in)
	if err != nil || !created || id < 2 {
		t.Fatal("rollback gap not preserved", id, created, err)
	}
	var cache, increment int64
	var cycle bool
	if err = db.Pool.QueryRow(ctx, `SELECT cache_size,increment_by,cycle FROM pg_sequences WHERE sequencename='trader_sync_activity_id_seq'`).Scan(&cache, &increment, &cycle); err != nil || cache != 1 || increment != 1 || cycle {
		t.Fatal(cache, increment, cycle, err)
	}
	if _, err = db.Pool.Exec(ctx, `UPDATE trader_sync_activities SET note_snapshot='rewritten' WHERE id=$1`, id); err == nil {
		t.Fatal("historical snapshot mutable")
	}
	// A late display update touches only the shared metadata cache, never delivery text.
	var before []byte
	db.Pool.QueryRow(ctx, `SELECT payload FROM account_notification_deliveries WHERE activity_id=$1`, id).Scan(&before)
	if err = s.SaveMetadata(ctx, activity.MetadataKey(in.Trade, in.Confirmation.BlockHash.Hex()), tm.TradeMetadata{Market: tm.MarketRef{PositionID: in.Trade.PositionID, Title: "late title", Evidence: tm.Evidence{Availability: "available"}}}); err != nil {
		t.Fatal(err)
	}
	var after []byte
	db.Pool.QueryRow(ctx, `SELECT payload FROM account_notification_deliveries WHERE activity_id=$1`, id).Scan(&after)
	if !bytes.Equal(before, after) {
		t.Fatal("late metadata rewrote payload")
	}
}

func TestActivityGrantLossDoesNotStopOtherOwners(t *testing.T) {
	db := pgtest.New(t, migrations.FS, migrations.Dir)
	ctx := context.Background()
	owner, err := ac.NewSQLStore(db.Pool).EnsureDevelopmentAccount(ctx, accountcredentials.DevelopmentRoleMember)
	if err != nil {
		t.Fatal(err)
	}
	s := NewSQLStore(db.Pool)
	s.ConfigureActivities("https://athena.test")
	in := activityFixture(t, db.Pool, owner.ID, 1)
	if _, err = db.Pool.Exec(ctx, `UPDATE account_module_access SET access_level='none' WHERE account_id=$1 AND module='trader_sync'`, owner.ID); err != nil {
		t.Fatal(err)
	}
	if id, created, err := s.Project(ctx, in); err != nil || created || id != 0 {
		t.Fatalf("grant loss must terminate only this candidate, not projector: %d %v %v", id, created, err)
	}
}

func TestInvalidSourceTerminatesCandidatesAndRemovedPublicationIsAnomaly(t *testing.T) {
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
	if err = s.SaveProjectionEvidence(ctx, in.Candidate.SourceID, tm.CanonicalEvidence{Status: "invalid", Reason: "receipt_mismatch", CheckedAt: time.Now()}, nil); err != nil {
		t.Fatal(err)
	}
	sources, err := s.ProjectionSources(ctx, 100)
	if err != nil {
		t.Fatal(err)
	}
	if len(sources) != 0 {
		t.Fatalf("invalid source keeps pending candidates: %+v", sources)
	}
	next := activitySource(t, db.Pool, owner.ID, in.Candidate.SubscriptionID, in.Candidate.AttemptID, in.Trade.Wallet, in.Confirmation.SettledAt, 2)
	id, created, err := s.Project(ctx, next)
	if err != nil || !created {
		t.Fatal(id, created, err)
	}
	if _, err = db.Pool.Exec(ctx, `UPDATE trader_sync_source_records SET removed=true WHERE id=$1`, next.Candidate.SourceID); err != nil {
		t.Fatal(err)
	}
	if err = s.SaveProjectionEvidence(ctx, next.Candidate.SourceID, next.Confirmation, &next.Trade); err != nil {
		t.Fatal(err)
	}
	var anomalies, activities int
	if err = db.Pool.QueryRow(ctx, `SELECT (SELECT count(*) FROM trader_sync_finality_anomalies),(SELECT count(*) FROM trader_sync_activities)`).Scan(&anomalies, &activities); err != nil {
		t.Fatal(err)
	}
	if anomalies != 1 || activities != 1 {
		t.Fatal(anomalies, activities)
	}
	var unknown bool
	if err = db.Pool.QueryRow(ctx, `SELECT conflicting_block_hash IS NULL FROM trader_sync_finality_anomalies`).Scan(&unknown); err != nil || !unknown {
		t.Fatal("removed invented a replacement block", unknown, err)
	}
	var state, reason string
	var removed bool
	if err = db.Pool.QueryRow(ctx, `SELECT confirmation_state,confirmation_reason,removed FROM trader_sync_source_records WHERE id=$1`, next.Candidate.SourceID).Scan(&state, &reason, &removed); err != nil || state != "invalid" || reason != "removed" || !removed {
		t.Fatal(state, reason, removed, err)
	}
}
func TestActivityDeliveryPayloadCannotChangeAfterFormation(t *testing.T) {
	db := pgtest.New(t, migrations.FS, migrations.Dir)
	ctx := context.Background()
	owner, err := ac.NewSQLStore(db.Pool).EnsureDevelopmentAccount(ctx, accountcredentials.DevelopmentRoleMember)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = db.Pool.Exec(ctx, `INSERT INTO telegram_bindings(account_id,telegram_user_id,telegram_chat_id,telegram_display_name,revision)VALUES($1,123,123,'test',1)`, owner.ID); err != nil {
		t.Fatal(err)
	}
	s := NewSQLStore(db.Pool)
	if err = s.ConfigureActivities("https://athena.test"); err != nil {
		t.Fatal(err)
	}
	in := activityFixture(t, db.Pool, owner.ID, 1)
	id, _, err := s.Project(ctx, in)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = db.Pool.Exec(ctx, `UPDATE account_notification_deliveries SET payload=convert_to('{"format":"plain","text":"tampered"}','UTF8'),payload_digest=sha256(convert_to('{"format":"plain","text":"tampered"}','UTF8')) WHERE activity_id=$1`, id); err == nil {
		t.Fatal("frozen payload and digest were mutable")
	}
}

func activityOtherOwner(t *testing.T, pool *pgxpool.Pool, base string) string {
	t.Helper()
	ctx := context.Background()
	id := uuid.NewString()
	if _, e := pool.Exec(ctx, `INSERT INTO athena_account(account_id,username,identity_provider,identity_subject,verified_email)VALUES($1::text::uuid,$1::text,'google',$1::text,$1::text||'@test.invalid')`, id); e != nil {
		t.Fatal(e)
	}
	if _, e := pool.Exec(ctx, `INSERT INTO account_access(account_id,login_enabled,api_key_enabled,profit_sharing_enabled,revision)VALUES($1,true,false,false,1)`, id); e != nil {
		t.Fatal(e)
	}
	if _, e := pool.Exec(ctx, `INSERT INTO account_module_access(account_id,module,access_level)SELECT $1,module,access_level FROM account_module_access WHERE account_id=$2`, id, base); e != nil {
		t.Fatal(e)
	}
	return id
}
func TestActivityConcurrentOwnersCannotPublishConflictingForks(t *testing.T) {
	db := pgtest.New(t, migrations.FS, migrations.Dir)
	ctx := context.Background()
	a, e := ac.NewSQLStore(db.Pool).EnsureDevelopmentAccount(ctx, accountcredentials.DevelopmentRoleMember)
	if e != nil {
		t.Fatal(e)
	}
	b := activityOtherOwner(t, db.Pool, a.ID)
	s := NewSQLStore(db.Pool)
	if e = s.ConfigureActivities("https://athena.test"); e != nil {
		t.Fatal(e)
	}
	one := activityFixture(t, db.Pool, a.ID, 1)
	two := activityFixture(t, db.Pool, b, 2)
	two.Confirmation.BlockHash = common.HexToHash("0xbb")
	if _, e = db.Pool.Exec(ctx, `UPDATE trader_sync_source_records SET block_hash=$2 WHERE id=$1`, two.Candidate.SourceID, two.Confirmation.BlockHash.Bytes()); e != nil {
		t.Fatal(e)
	}
	hold, e := db.Pool.Begin(ctx)
	if e != nil {
		t.Fatal(e)
	}
	defer hold.Rollback(ctx)
	if _, e = hold.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended('athena:trade:'||encode($1::bytea,'hex'),0))`, common.BigToHash(common.Big1).Bytes()); e != nil {
		t.Fatal(e)
	}
	type outcome struct {
		id      int64
		created bool
		err     error
	}
	done := make(chan outcome, 2)
	for _, in := range []tm.Projection{one, two} {
		go func(in tm.Projection) { id, c, e := s.Project(ctx, in); done <- outcome{id, c, e} }(in)
	}
	deadline := time.Now().Add(time.Second)
	for {
		var waiting int
		if e = db.Pool.QueryRow(ctx, `SELECT count(*) FROM pg_locks WHERE locktype='advisory' AND NOT granted AND database=(SELECT oid FROM pg_database WHERE datname=current_database())`).Scan(&waiting); e != nil {
			t.Fatal(e)
		}
		if waiting >= 2 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("both owner transactions did not wait on shared trade evidence")
		}
		time.Sleep(time.Millisecond)
	}
	var before int
	if e = db.Pool.QueryRow(ctx, `SELECT count(*) FROM trader_sync_activities`).Scan(&before); e != nil || before != 0 {
		t.Fatal(before, e)
	}
	if e = hold.Commit(ctx); e != nil {
		t.Fatal(e)
	}
	created := 0
	for range 2 {
		r := <-done
		if r.err != nil {
			t.Fatal(r.err)
		}
		if r.created {
			created++
		}
	}
	var count, anomalies int
	var realFork bool
	if e = db.Pool.QueryRow(ctx, `SELECT (SELECT count(*) FROM trader_sync_activities),(SELECT count(*) FROM trader_sync_finality_anomalies),(SELECT published_block_hash<>conflicting_block_hash FROM trader_sync_finality_anomalies)`).Scan(&count, &anomalies, &realFork); e != nil || count != 1 || created != 1 || anomalies != 1 || !realFork {
		t.Fatal(created, count, anomalies, realFork, e)
	}
}
func TestActivitySameSourceOwnersHaveIndependentNotesAndEligibility(t *testing.T) {
	db := pgtest.New(t, migrations.FS, migrations.Dir)
	ctx := context.Background()
	a, e := ac.NewSQLStore(db.Pool).EnsureDevelopmentAccount(ctx, accountcredentials.DevelopmentRoleMember)
	if e != nil {
		t.Fatal(e)
	}
	b := activityOtherOwner(t, db.Pool, a.ID)
	one := activityFixture(t, db.Pool, a.ID, 1)
	two := activityFixture(t, db.Pool, b, 2)
	if _, e = db.Pool.Exec(ctx, `UPDATE trader_sync_source_candidates SET source_record_id=$2 WHERE source_record_id=$1`, two.Candidate.SourceID, one.Candidate.SourceID); e != nil {
		t.Fatal(e)
	}
	two.Candidate.SourceID = one.Candidate.SourceID
	for _, v := range []struct{ owner, note string }{{a.ID, "one"}, {b, "two"}} {
		if _, e = db.Pool.Exec(ctx, `INSERT INTO trader_sync_target_notes(owner_id,wallet,note,revision)VALUES($1,$2,$3,1)`, v.owner, one.Trade.Wallet.Bytes(), v.note); e != nil {
			t.Fatal(e)
		}
	}
	if _, e = db.Pool.Exec(ctx, `INSERT INTO telegram_bindings(account_id,telegram_user_id,telegram_chat_id,telegram_display_name,revision)VALUES($1,123,123,'test',1)`, a.ID); e != nil {
		t.Fatal(e)
	}
	s := NewSQLStore(db.Pool)
	s.ConfigureActivities("https://athena.test")
	done := make(chan error, 2)
	for _, in := range []tm.Projection{one, two} {
		go func(in tm.Projection) {
			_, c, e := s.Project(ctx, in)
			if e == nil && !c {
				e = fmt.Errorf("not created")
			}
			done <- e
		}(in)
	}
	for range 2 {
		if e = <-done; e != nil {
			t.Fatal(e)
		}
	}
	rows, e := db.Pool.Query(ctx, `SELECT owner_id::text,note_snapshot,notification_mode FROM trader_sync_activities`)
	if e != nil {
		t.Fatal(e)
	}
	defer rows.Close()
	got := map[string]string{}
	for rows.Next() {
		var id, note, mode string
		if e = rows.Scan(&id, &note, &mode); e != nil {
			t.Fatal(e)
		}
		got[id] = note + ":" + mode
	}
	if got[a.ID] != "one:ordinary" || got[b] != "two:in_app_only" {
		t.Fatal(got)
	}
}

func TestPublishedPartialRemainsRetryableAfterTemporaryConfirmationFailure(t *testing.T) {
	db := pgtest.New(t, migrations.FS, migrations.Dir)
	ctx := context.Background()
	owner, e := ac.NewSQLStore(db.Pool).EnsureDevelopmentAccount(ctx, accountcredentials.DevelopmentRoleMember)
	if e != nil {
		t.Fatal(e)
	}
	s := NewSQLStore(db.Pool)
	s.ConfigureActivities("https://athena.test")
	in := activityFixture(t, db.Pool, owner.ID, 1)
	if _, _, e = s.Project(ctx, in); e != nil {
		t.Fatal(e)
	}
	if e = s.SaveProjectionEvidence(ctx, in.Candidate.SourceID, tm.CanonicalEvidence{Status: "unverified", Reason: "provider_unavailable", CheckedAt: time.Now()}, nil); e != nil {
		t.Fatal(e)
	}
	rows, e := s.ProjectionSources(ctx, 10)
	if e != nil {
		t.Fatal(e)
	}
	if len(rows) != 1 || len(rows[0].Candidates) != 0 {
		t.Fatalf("temporary failure stranded published partial: %+v", rows)
	}
}

func TestActivityWindowEndpointsAndBackwardTimeSnapshot(t *testing.T) {
	db := pgtest.New(t, migrations.FS, migrations.Dir)
	ctx := context.Background()
	owner, e := ac.NewSQLStore(db.Pool).EnsureDevelopmentAccount(ctx, accountcredentials.DevelopmentRoleMember)
	if e != nil {
		t.Fatal(e)
	}
	s := NewSQLStore(db.Pool)
	s.ConfigureActivities("https://athena.test")
	in := activityFixture(t, db.Pool, owner.ID, 1)
	base, _, e := s.Project(ctx, in)
	if e != nil {
		t.Fatal(e)
	}
	at := time.Now().UTC().Add(-time.Hour).Truncate(time.Microsecond)
	var ids []int64
	for i, stamp := range []time.Time{at.Add(-60 * time.Second), at, at.Add(-60*time.Second + time.Microsecond), at.Add(time.Microsecond)} {
		next := activitySource(t, db.Pool, owner.ID, in.Candidate.SubscriptionID, in.Candidate.AttemptID, in.Trade.Wallet, in.Confirmation.SettledAt, i+2)
		var id int64
		e = txgate.WithAccountTx(ctx, db.Pool, owner.ID, func(tx pgx.Tx) error {
			return tx.QueryRow(ctx, `INSERT INTO trader_sync_activities(owner_id,subscription_id,source_record_id,interval_id,activation_generation,trade_json,metadata_key,target_display_snapshot,note_snapshot,notification_mode,notification_reason,settled_at,received_at,recorded_at)SELECT owner_id,subscription_id,$2,interval_id,activation_generation,trade_json,metadata_key,target_display_snapshot,note_snapshot,notification_mode,notification_reason,settled_at,received_at,$3 FROM trader_sync_activities WHERE id=$1 RETURNING id`, base, next.Candidate.SourceID, stamp).Scan(&id)
		})
		if e != nil {
			t.Fatal(e)
		}
		ids = append(ids, id)
	}
	ownerUUID, e := confirmationOwner(owner.ID)
	if e != nil {
		t.Fatal(e)
	}
	queries := q.New(db.Pool)
	n, e := queries.CountActivityWindow(ctx, q.CountActivityWindowParams{OwnerID: ownerUUID, Column2: pgtype.Timestamptz{Time: at, Valid: true}})
	if e != nil || n != 2 {
		t.Fatal("window must exclude left and include right, ignore future", n, e)
	}
	snap, e := queries.ReadOwnerActivitySnapshot(ctx, ownerUUID)
	if e != nil || snap != ids[3] {
		t.Fatal(snap, ids, e)
	}
	rows, e := queries.ReadOwnerActivities(ctx, q.ReadOwnerActivitiesParams{OwnerID: ownerUUID, ID: ids[2], Limit: 10})
	if e != nil {
		t.Fatal(e)
	}
	if len(rows) != 4 || rows[0].ID != ids[2] || rows[1].ID != ids[1] || rows[3].ID != base {
		t.Fatal("snapshot follows committed IDs across time rollback", rows)
	}
}
func TestActivityOwnerGatePreventsIDPreallocationAndProjectionAfterPause(t *testing.T) {
	db := pgtest.New(t, migrations.FS, migrations.Dir)
	ctx := context.Background()
	a, e := ac.NewSQLStore(db.Pool).EnsureDevelopmentAccount(ctx, accountcredentials.DevelopmentRoleMember)
	if e != nil {
		t.Fatal(e)
	}
	b := activityOtherOwner(t, db.Pool, a.ID)
	s := NewSQLStore(db.Pool)
	s.ConfigureActivities("https://athena.test")
	one := activityFixture(t, db.Pool, a.ID, 1)
	two := activityFixture(t, db.Pool, b, 2)
	entered, release := make(chan struct{}), make(chan struct{})
	changed := make(chan error, 1)
	go func() {
		changed <- txgate.WithAccountTx(ctx, db.Pool, a.ID, func(tx pgx.Tx) error {
			if _, e := tx.Exec(ctx, `UPDATE trader_sync_subscriptions SET desired_state='paused' WHERE id=$1`, one.Candidate.SubscriptionID); e != nil {
				return e
			}
			close(entered)
			<-release
			return nil
		})
	}()
	<-entered
	done := make(chan error, 1)
	go func() {
		id, c, e := s.Project(ctx, one)
		if e == nil && (id != 0 || c) {
			e = fmt.Errorf("paused projection published")
		}
		done <- e
	}()
	select {
	case e = <-done:
		t.Fatalf("bypassed owner gate: %v", e)
	case <-time.After(20 * time.Millisecond):
	}
	id, c, e := s.Project(ctx, two)
	if e != nil || !c || id != 1 {
		t.Fatal("blocked owner allocated ID or blocked independent owner", id, c, e)
	}
	close(release)
	if e = <-changed; e != nil {
		t.Fatal(e)
	}
	if e = <-done; e != nil {
		t.Fatal(e)
	}
}
func TestActivityOriginalClosedIntervalAndGenerationBoundaries(t *testing.T) {
	for _, mode := range []string{"closed_epoch", "before_start", "at_end", "empty", "new_generation", "cancelled"} {
		t.Run(mode, func(t *testing.T) {
			db := pgtest.New(t, migrations.FS, migrations.Dir)
			ctx := context.Background()
			a, e := ac.NewSQLStore(db.Pool).EnsureDevelopmentAccount(ctx, accountcredentials.DevelopmentRoleMember)
			if e != nil {
				t.Fatal(e)
			}
			s := NewSQLStore(db.Pool)
			s.ConfigureActivities("https://athena.test")
			in := activityFixture(t, db.Pool, a.ID, 1)
			if _, e = db.Pool.Exec(ctx, `UPDATE trader_sync_collector_epochs SET ended_at=clock_timestamp(),reason='test_closed'`); e != nil {
				t.Fatal(e)
			}
			switch mode {
			case "before_start":
				_, e = db.Pool.Exec(ctx, `UPDATE trader_sync_monitor_intervals SET effective_at=$2 WHERE subscription_id=$1`, in.Candidate.SubscriptionID, in.Confirmation.SettledAt.Add(time.Second))
			case "at_end":
				_, e = db.Pool.Exec(ctx, `UPDATE trader_sync_monitor_intervals SET ended_at=$2 WHERE subscription_id=$1`, in.Candidate.SubscriptionID, in.Confirmation.SettledAt)
			case "empty":
				_, e = db.Pool.Exec(ctx, `UPDATE trader_sync_monitor_intervals SET effective_at=$2,ended_at=$3 WHERE subscription_id=$1`, in.Candidate.SubscriptionID, in.Confirmation.SettledAt.Add(time.Second), in.Confirmation.SettledAt)
			case "new_generation":
				_, e = db.Pool.Exec(ctx, `UPDATE trader_sync_subscriptions SET activation_generation=2 WHERE id=$1`, in.Candidate.SubscriptionID)
			case "cancelled":
				_, e = db.Pool.Exec(ctx, `UPDATE trader_sync_subscriptions SET desired_state='cancelled' WHERE id=$1`, in.Candidate.SubscriptionID)
			}
			if e != nil {
				t.Fatal(e)
			}
			_, created, e := s.Project(ctx, in)
			if e != nil || created != (mode == "closed_epoch") {
				t.Fatal(created, e)
			}
		})
	}
}

func TestActivityPayloadUsesExistingNonConflictingPartialAtDeadline(t *testing.T) {
	db := pgtest.New(t, migrations.FS, migrations.Dir)
	ctx := context.Background()
	owner, e := ac.NewSQLStore(db.Pool).EnsureDevelopmentAccount(ctx, accountcredentials.DevelopmentRoleMember)
	if e != nil {
		t.Fatal(e)
	}
	s := NewSQLStore(db.Pool)
	s.ConfigureActivities("https://athena.test")
	in := activityFixture(t, db.Pool, owner.ID, 1)
	if _, e = db.Pool.Exec(ctx, `INSERT INTO telegram_bindings(account_id,telegram_user_id,telegram_chat_id,telegram_display_name,revision)VALUES($1,123,123,'test',1)`, owner.ID); e != nil {
		t.Fatal(e)
	}
	known := tm.TradeMetadata{Relationship: "AND", LegsEvidence: tm.Evidence{Availability: "available"}, Legs: []tm.ComboLeg{{PositionID: "1", Market: tm.MarketRef{Evidence: tm.Evidence{Availability: "available"}, ID: "known"}}, {PositionID: "2", Market: tm.MarketRef{Evidence: tm.Evidence{Availability: "unavailable"}}}}}
	if e = s.SaveMetadata(ctx, activity.MetadataKey(in.Trade, in.Confirmation.BlockHash.Hex()), known); e != nil {
		t.Fatal(e)
	}
	id, _, e := s.Project(ctx, in)
	if e != nil {
		t.Fatal(e)
	}
	var raw []byte
	if e = db.Pool.QueryRow(ctx, `SELECT payload FROM account_notification_deliveries WHERE activity_id=$1`, id).Scan(&raw); e != nil {
		t.Fatal(e)
	}
	if !bytes.Contains(raw, []byte("Leg metadata: 1/2")) {
		t.Fatal("frozen payload lost already known partial", string(raw))
	}
}

func TestActivityHundredRelationsSharedAndDistinctTargets(t *testing.T) {
	for _, distinct := range []bool{false, true} {
		t.Run(fmt.Sprint("distinct=", distinct), func(t *testing.T) {
			db := pgtest.New(t, migrations.FS, migrations.Dir)
			ctx := context.Background()
			first, e := ac.NewSQLStore(db.Pool).EnsureDevelopmentAccount(ctx, accountcredentials.DevelopmentRoleMember)
			if e != nil {
				t.Fatal(e)
			}
			s := NewSQLStore(db.Pool)
			s.ConfigureActivities("https://athena.test")
			for i := 0; i < 10; i++ {
				owner := first.ID
				if i > 0 {
					owner = activityOtherOwner(t, db.Pool, first.ID)
				}
				for j := 0; j < 10; j++ {
					n := j + 1
					if distinct {
						n = i*10 + j + 1
					}
					wallet := common.BytesToAddress([]byte{byte(n)})
					in := activityWalletFixture(t, db.Pool, owner, i*10+j+1, wallet)
					if _, c, e := s.Project(ctx, in); e != nil || !c {
						t.Fatal(i, j, c, e)
					}
				}
			}
			var activities, owners, targets int
			if e = db.Pool.QueryRow(ctx, `SELECT count(*),count(DISTINCT owner_id),(SELECT count(DISTINCT wallet) FROM trader_sync_subscriptions) FROM trader_sync_activities`).Scan(&activities, &owners, &targets); e != nil {
				t.Fatal(e)
			}
			want := 10
			if distinct {
				want = 100
			}
			if activities != 100 || owners != 10 || targets != want {
				t.Fatal(activities, owners, targets)
			}
		})
	}
}
func TestActivityConcurrentDuplicatePlansOneQualification(t *testing.T) {
	db := pgtest.New(t, migrations.FS, migrations.Dir)
	ctx := context.Background()
	owner, e := ac.NewSQLStore(db.Pool).EnsureDevelopmentAccount(ctx, accountcredentials.DevelopmentRoleMember)
	if e != nil {
		t.Fatal(e)
	}
	s := NewSQLStore(db.Pool)
	s.ConfigureActivities("https://athena.test")
	in := activityFixture(t, db.Pool, owner.ID, 1)
	if _, e = db.Pool.Exec(ctx, `INSERT INTO telegram_bindings(account_id,telegram_user_id,telegram_chat_id,telegram_display_name,revision)VALUES($1,123,123,'test',1)`, owner.ID); e != nil {
		t.Fatal(e)
	}
	type result struct {
		id      int64
		created bool
		e       error
	}
	done := make(chan result, 2)
	for range 2 {
		go func() { id, c, e := s.Project(ctx, in); done <- result{id, c, e} }()
	}
	a, b := <-done, <-done
	if a.e != nil || b.e != nil || a.id != b.id || a.created == b.created {
		t.Fatal(a, b)
	}
	var activities, memberships, deliveries int
	if e = db.Pool.QueryRow(ctx, `SELECT (SELECT count(*) FROM trader_sync_activities),(SELECT count(*) FROM trader_sync_alert_memberships),(SELECT count(*) FROM account_notification_deliveries)`).Scan(&activities, &memberships, &deliveries); e != nil || activities != 1 || memberships != 1 || deliveries != 1 {
		t.Fatal(activities, memberships, deliveries, e)
	}
}
func TestProjectionSourcesPersistentCheckOrderDoesNotStarveFreshCandidates(t *testing.T) {
	db := pgtest.New(t, migrations.FS, migrations.Dir)
	ctx := context.Background()
	owner, e := ac.NewSQLStore(db.Pool).EnsureDevelopmentAccount(ctx, accountcredentials.DevelopmentRoleMember)
	if e != nil {
		t.Fatal(e)
	}
	s := NewSQLStore(db.Pool)
	first := activityFixture(t, db.Pool, owner.ID, 1)
	for i := 2; i <= 5; i++ {
		activitySource(t, db.Pool, owner.ID, first.Candidate.SubscriptionID, first.Candidate.AttemptID, first.Trade.Wallet, first.Confirmation.SettledAt, i)
	}
	rows, e := s.ProjectionSources(ctx, 2)
	if e != nil || len(rows) != 2 {
		t.Fatal(rows, e)
	}
	for _, r := range rows {
		if e = s.SaveProjectionEvidence(ctx, r.ID, tm.CanonicalEvidence{Status: "unverified", Reason: "provider_unavailable", CheckedAt: time.Now()}, nil); e != nil {
			t.Fatal(e)
		}
	}
	restarted := NewSQLStore(db.Pool)
	later, e := restarted.ProjectionSources(ctx, 2)
	if e != nil || len(later) != 2 || later[0].ID <= rows[1].ID {
		t.Fatal("old failed source starved later candidates across restart", rows, later, e)
	}
	var pending int
	if e = db.Pool.QueryRow(ctx, `SELECT count(*) FROM trader_sync_source_candidates WHERE disposition='pending'`).Scan(&pending); e != nil || pending != 5 {
		t.Fatal(pending, e)
	}
}

func TestRemovedAnomalyOnlyFillsFirstConfirmedConflictHash(t *testing.T) {
	db := pgtest.New(t, migrations.FS, migrations.Dir)
	ctx := context.Background()
	owner, e := ac.NewSQLStore(db.Pool).EnsureDevelopmentAccount(ctx, accountcredentials.DevelopmentRoleMember)
	if e != nil {
		t.Fatal(e)
	}
	s := NewSQLStore(db.Pool)
	s.ConfigureActivities("https://athena.test")
	first := activityFixture(t, db.Pool, owner.ID, 1)
	if _, _, e = s.Project(ctx, first); e != nil {
		t.Fatal(e)
	}
	if _, e = db.Pool.Exec(ctx, `UPDATE trader_sync_source_records SET removed=true WHERE id=$1`, first.Candidate.SourceID); e != nil {
		t.Fatal(e)
	}
	if e = s.SaveProjectionEvidence(ctx, first.Candidate.SourceID, first.Confirmation, &first.Trade); e != nil {
		t.Fatal(e)
	}
	var at time.Time
	if e = db.Pool.QueryRow(ctx, `SELECT detected_at FROM trader_sync_finality_anomalies`).Scan(&at); e != nil {
		t.Fatal(e)
	}
	for i, hash := range []common.Hash{common.HexToHash("0xbb"), common.HexToHash("0xcc")} {
		in := activitySource(t, db.Pool, owner.ID, first.Candidate.SubscriptionID, first.Candidate.AttemptID, first.Trade.Wallet, first.Confirmation.SettledAt, i+2)
		in.Confirmation.BlockHash = hash
		if _, e = db.Pool.Exec(ctx, `UPDATE trader_sync_source_records SET block_hash=$2 WHERE id=$1`, in.Candidate.SourceID, hash.Bytes()); e != nil {
			t.Fatal(e)
		}
		if id, c, e := s.Project(ctx, in); e != nil || c || id != 0 {
			t.Fatal("quarantined transaction published", id, c, e)
		}
	}
	var known, published []byte
	var reason string
	var detected time.Time
	if e = db.Pool.QueryRow(ctx, `SELECT conflicting_block_hash,published_block_hash,reason,detected_at FROM trader_sync_finality_anomalies`).Scan(&known, &published, &reason, &detected); e != nil {
		t.Fatal(e)
	}
	if common.BytesToHash(known) != common.HexToHash("0xbb") || common.BytesToHash(published) != first.Confirmation.BlockHash || reason != "removed" || !detected.Equal(at) {
		t.Fatal("anomaly failed one-way evidence enrichment", common.BytesToHash(known), reason, detected, at)
	}
	var count int
	if e = db.Pool.QueryRow(ctx, `SELECT count(*) FROM trader_sync_activities`).Scan(&count); e != nil || count != 1 {
		t.Fatal(count, e)
	}
}

func TestOnlyConfirmedDifferentBlockChallengesPublishedTransaction(t *testing.T) {
	db := pgtest.New(t, migrations.FS, migrations.Dir)
	ctx := context.Background()
	owner, e := ac.NewSQLStore(db.Pool).EnsureDevelopmentAccount(ctx, accountcredentials.DevelopmentRoleMember)
	if e != nil {
		t.Fatal(e)
	}
	s := NewSQLStore(db.Pool)
	s.ConfigureActivities("https://athena.test")
	first := activityFixture(t, db.Pool, owner.ID, 1)
	if _, _, e = s.Project(ctx, first); e != nil {
		t.Fatal(e)
	}
	next := activitySource(t, db.Pool, owner.ID, first.Candidate.SubscriptionID, first.Candidate.AttemptID, first.Trade.Wallet, first.Confirmation.SettledAt, 2)
	next.Confirmation.BlockHash = common.HexToHash("0xbb")
	if _, e = db.Pool.Exec(ctx, `UPDATE trader_sync_source_records SET block_hash=$2 WHERE id=$1`, next.Candidate.SourceID, next.Confirmation.BlockHash.Bytes()); e != nil {
		t.Fatal(e)
	}
	if e = s.SaveProjectionEvidence(ctx, next.Candidate.SourceID, tm.CanonicalEvidence{Status: "invalid", Reason: "noncanonical_block", CheckedAt: time.Now()}, nil); e != nil {
		t.Fatal(e)
	}
	var count int
	if e = db.Pool.QueryRow(ctx, `SELECT count(*) FROM trader_sync_finality_anomalies`).Scan(&count); e != nil || count != 0 {
		t.Fatalf("unconfirmed discarded fork falsely challenged published block: %d %v", count, e)
	}
	sibling := activitySource(t, db.Pool, owner.ID, first.Candidate.SubscriptionID, first.Candidate.AttemptID, first.Trade.Wallet, first.Confirmation.SettledAt, 4)
	if e = s.SaveProjectionEvidence(ctx, sibling.Candidate.SourceID, tm.CanonicalEvidence{Status: "invalid", Reason: "log_not_in_receipt", CheckedAt: time.Now()}, nil); e != nil {
		t.Fatal(e)
	}
	if e = db.Pool.QueryRow(ctx, `SELECT count(*) FROM trader_sync_finality_anomalies`).Scan(&count); e != nil || count != 0 {
		t.Fatalf("unpublished invalid sibling log falsely invalidated published source: %d %v", count, e)
	}
	later := activitySource(t, db.Pool, owner.ID, first.Candidate.SubscriptionID, first.Candidate.AttemptID, first.Trade.Wallet, first.Confirmation.SettledAt, 3)
	later.Confirmation.BlockHash = common.HexToHash("0xcc")
	if _, e = db.Pool.Exec(ctx, `UPDATE trader_sync_source_records SET block_hash=$2 WHERE id=$1`, later.Candidate.SourceID, later.Confirmation.BlockHash.Bytes()); e != nil {
		t.Fatal(e)
	}
	if e = s.SaveProjectionEvidence(ctx, later.Candidate.SourceID, later.Confirmation, &later.Trade); e != nil {
		t.Fatal(e)
	}
	var hash []byte
	if e = db.Pool.QueryRow(ctx, `SELECT conflicting_block_hash FROM trader_sync_finality_anomalies`).Scan(&hash); e != nil || common.BytesToHash(hash) != later.Confirmation.BlockHash {
		t.Fatal("confirmed source did not register anomaly independently of later owner eligibility", common.BytesToHash(hash), e)
	}
}
