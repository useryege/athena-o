//go:build integration

package store

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/useryege/athena/internal/operationlog/event"
	"github.com/useryege/athena/internal/operationlog/ingest"
	"github.com/useryege/athena/internal/operationlog/schema"
	"github.com/useryege/athena/internal/testutil/pgtest"
	"sync"
	"testing"
	"time"
)

var ctx = context.Background()

func ptr[T any](v T) *T { return &v }
func newStore(t *testing.T) (*Store, *pgtest.DB) {
	t.Helper()
	db := pgtest.NewUnmigrated(t)
	if err := schema.Up(ctx, db.DSN); err != nil {
		t.Fatal(err)
	}
	s, err := Open(ctx, db.DSN)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(s.Close)
	return s, db
}
func fixture() event.Event {
	now := time.Date(2026, 9, 18, 1, 0, 0, 123456789, time.UTC)
	return event.Event{SchemaVersion: 1, EventID: uuid.NewString(), OperationID: uuid.NewString(), RequestID: uuid.NewString(), ProducerID: uuid.NewString(), Phase: event.Start, StartedAt: now, OccurredAt: now, Actor: event.Actor{Role: "UNKNOWN", Realm: "UNKNOWN", CredentialKind: "UNAUTHENTICATED"}, ActionCode: "account.access.update", ModuleCode: "account", Outcome: event.Unknown, Observation: event.StartOnly, Resources: []event.Resource{}, Effect: []string{}, Details: event.Details{}, ResourcesComplete: true}
}
func finish(e event.Event) event.Event {
	e.EventID = uuid.NewString()
	e.Phase = event.Finish
	e.Observation = event.FinishOnly
	e.Outcome = event.Succeeded
	e.OccurredAt = e.StartedAt.Add(time.Second)
	e.DurationMs = ptr(int64(1000))
	return e
}
func appendOK(t *testing.T, s *Store, e event.Event) {
	t.Helper()
	if err := s.Append(ctx, e); err != nil {
		t.Fatal(err)
	}
}
func projectOK(t *testing.T, s *Store) Projection {
	t.Helper()
	r, err := s.Project(ctx)
	if err != nil {
		t.Fatal(err)
	}
	return r
}
func count(t *testing.T, s *Store, query string, args ...any) int64 {
	t.Helper()
	var n int64
	if err := s.pool.QueryRow(ctx, query, args...).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}
func view(t *testing.T, s *Store, op string, seq int64) (string, string, NormalizedView) {
	t.Helper()
	var outcome, observation string
	var b []byte
	if err := s.pool.QueryRow(ctx, `SELECT outcome,observation,detail FROM operation_log.entry_version WHERE operation_id=$1 AND visible_from_seq<=$2 AND (visible_to_seq IS NULL OR visible_to_seq>$2)`, op, seq).Scan(&outcome, &observation, &b); err != nil {
		t.Fatal(err)
	}
	var v NormalizedView
	if err := json.Unmarshal(b, &v); err != nil {
		t.Fatal(err)
	}
	if string(v.Outcome) != outcome || string(v.Observation) != observation {
		t.Fatal("column/detail disagreement")
	}
	return outcome, observation, v
}

// insertRaw holds the real inbox transaction open to model a producer with a late commit.
func insertRaw(t *testing.T, tx pgx.Tx, e event.Event, payload []byte) {
	t.Helper()
	if payload == nil {
		var err error
		payload, err = event.Canonical(e)
		if err != nil {
			t.Fatal(err)
		}
	}
	h := sha256.Sum256(payload)
	if _, err := tx.Exec(ctx, `INSERT INTO operation_log.event(event_id,operation_id,phase,producer_id,schema_version,occurred_at,payload,payload_hash) VALUES($1,$2,$3,$4,$5,$6,$7,$8)`, e.EventID, e.OperationID, string(e.Phase), e.ProducerID, e.SchemaVersion, e.OccurredAt, payload, h[:]); err != nil {
		t.Fatal(err)
	}
	if _, err := tx.Exec(ctx, `INSERT INTO operation_log.delivery(event_id) VALUES($1)`, e.EventID); err != nil {
		t.Fatal(err)
	}
}
func TestAppendIdempotencyAndAtomicity(t *testing.T) {
	s, _ := newStore(t)
	e := fixture()
	appendOK(t, s, e)
	appendOK(t, s, e)
	if n := count(t, s, `SELECT count(*) FROM operation_log.event`); n != 1 {
		t.Fatalf("duplicate events %d", n)
	}
	if n := count(t, s, `SELECT count(*) FROM operation_log.delivery`); n != 1 {
		t.Fatalf("deliveries %d", n)
	}
	conflict := e
	conflict.RequestID = uuid.NewString()
	if !errors.Is(s.Append(ctx, conflict), ingest.ErrConflict) {
		t.Fatal("same eventId conflict accepted")
	}
	conflict = e
	conflict.EventID = uuid.NewString()
	if !errors.Is(s.Append(ctx, conflict), ingest.ErrConflict) {
		t.Fatal("same operation/phase with different eventId accepted")
	}
	invalid := e
	invalid.EventID = uuid.NewString()
	invalid.SchemaVersion = 99
	if !errors.Is(s.Append(ctx, invalid), event.ErrInvalid) {
		t.Fatal("invalid accepted")
	}
	_, err := s.pool.Exec(ctx, `ALTER TABLE operation_log.delivery ADD CONSTRAINT reject_delivery CHECK(false) NOT VALID`)
	if err != nil {
		t.Fatal(err)
	}
	e = fixture()
	if s.Append(ctx, e) == nil {
		t.Fatal("delivery failure acknowledged")
	}
	if count(t, s, `SELECT count(*) FROM operation_log.event WHERE event_id=$1`, e.EventID) != 0 {
		t.Fatal("orphan event committed")
	}
}
func TestFoldBothArrivalOrdersAndStableHistory(t *testing.T) {
	for _, reverse := range []bool{false, true} {
		t.Run(map[bool]string{false: "start-first", true: "finish-first"}[reverse], func(t *testing.T) {
			s, _ := newStore(t)
			a := fixture()
			b := finish(a)
			b.Actor = event.Actor{AccountID: ptr(uuid.NewString()), UsernameSnapshot: ptr("Alice"), Role: "ADMINISTRATOR", Realm: "ADMIN", CredentialKind: "LOGIN_SESSION", IdentityVerified: true, IdentitySnapshotComplete: true}
			first, second := a, b
			if reverse {
				first, second = b, a
			}
			appendOK(t, s, first)
			p := projectOK(t, s)
			if p.PublishedSeq != 1 {
				t.Fatalf("first seq %+v", p)
			}
			o, obs, _ := view(t, s, a.OperationID, 1)
			if reverse && (o != "SUCCEEDED" || obs != "FINISH_ONLY") || !reverse && (o != "UNKNOWN" || obs != "START_ONLY") {
				t.Fatalf("first view %s %s", o, obs)
			}
			appendOK(t, s, second)
			projectOK(t, s)
			o, obs, v := view(t, s, a.OperationID, 2)
			if o != "SUCCEEDED" || obs != "COMPLETE" || v.Actor.AccountID == nil || *v.Actor.AccountID != *b.Actor.AccountID {
				t.Fatalf("complete %+v", v)
			}
			oldO, oldObs, _ := view(t, s, a.OperationID, 1)
			if o == "" || oldO != map[bool]string{true: "SUCCEEDED", false: "UNKNOWN"}[reverse] || oldObs != map[bool]string{true: "FINISH_ONLY", false: "START_ONLY"}[reverse] {
				t.Fatal("history changed")
			}
			if count(t, s, `SELECT cardinality(source_event_ids) FROM operation_log.entry_version WHERE operation_id=$1 AND visible_to_seq IS NULL`, a.OperationID) != 2 {
				t.Fatal("source events missing")
			}
			if v.FirstReceivedAt.IsZero() || v.LastReceivedAt.Before(v.FirstReceivedAt) {
				t.Fatal("receipt facts missing")
			}
		})
	}
}
func TestSameBatchFoldsOnceAndConflictStaysExcluded(t *testing.T) {
	s, _ := newStore(t)
	a := fixture()
	appendOK(t, s, a)
	appendOK(t, s, finish(a))
	r := projectOK(t, s)
	if r.Processed != 2 || count(t, s, `SELECT count(*) FROM operation_log.entry_version`) != 1 {
		t.Fatalf("not folded once %+v", r)
	}
	badStart := fixture()
	badStart.Actor = event.Actor{AccountID: ptr(uuid.NewString()), Role: "MEMBER", Realm: "MEMBER", CredentialKind: "LOGIN_SESSION", IdentityVerified: true}
	badFinish := finish(badStart)
	badFinish.Actor.AccountID = ptr(uuid.NewString())
	appendOK(t, s, badStart)
	appendOK(t, s, badFinish)
	r = projectOK(t, s)
	if r.Quarantined != 1 {
		t.Fatalf("conflict not quarantined %+v", r)
	}
	o, obs, _ := view(t, s, badStart.OperationID, 2)
	if o != "UNKNOWN" || obs != "START_ONLY" {
		t.Fatal("conflict contaminated view")
	}
	projectOK(t, s)
	if count(t, s, `SELECT last_seq FROM operation_log.publication`) != 2 {
		t.Fatal("idle or quarantined event republished")
	}
}
func TestLateCommitAndDualProjectors(t *testing.T) {
	s, db := newStore(t)
	late := fixture()
	tx, err := db.Pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback(ctx)
	insertRaw(t, tx, late, nil)
	early := fixture()
	appendOK(t, s, early)
	projectOK(t, s)
	if count(t, s, `SELECT count(*) FROM operation_log.entry_version WHERE operation_id=$1`, late.OperationID) != 0 {
		t.Fatal("uncommitted event visible")
	}
	if err = tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	s2, err := Open(ctx, db.DSN)
	if err != nil {
		t.Fatal(err)
	}
	defer s2.Close()
	var wg sync.WaitGroup
	errs := make(chan error, 2)
	for _, p := range []*Store{s, s2} {
		wg.Add(1)
		go func(p *Store) { defer wg.Done(); _, err := p.Project(ctx); errs <- err }(p)
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}
	if count(t, s, `SELECT count(*) FROM operation_log.entry_version`) != 2 || count(t, s, `SELECT last_seq FROM operation_log.publication`) != 2 {
		t.Fatal("late ingest lost or double published")
	}
	if count(t, s, `SELECT count(*) FROM operation_log.entry_version WHERE visible_from_seq<=1 AND (visible_to_seq IS NULL OR visible_to_seq>1)`) != 1 {
		t.Fatal("old snapshot admitted late commit")
	}
}
func TestLockedPublicationReturnsBusy(t *testing.T) {
	s, db := newStore(t)
	appendOK(t, s, fixture())
	tx, err := db.Pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback(ctx)
	if _, err = tx.Exec(ctx, `SELECT * FROM operation_log.publication FOR UPDATE`); err != nil {
		t.Fatal(err)
	}
	start := time.Now()
	r := projectOK(t, s)
	if !r.Busy || time.Since(start) > time.Second {
		t.Fatalf("lock not skipped %+v", r)
	}
}
func TestProjectRollbackRestartAndBacklog(t *testing.T) {
	s, db := newStore(t)
	for i := 0; i < 105; i++ {
		appendOK(t, s, fixture())
	}
	_, err := db.Pool.Exec(ctx, `CREATE FUNCTION operation_log.fail_publish() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'temporary failure' USING ERRCODE='40001'; END $$; CREATE TRIGGER fail_publish BEFORE UPDATE ON operation_log.publication FOR EACH ROW EXECUTE FUNCTION operation_log.fail_publish()`)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.Project(ctx); err == nil {
		t.Fatal("publication failure acknowledged")
	}
	if count(t, s, `SELECT count(*) FROM operation_log.entry_version`) != 0 || count(t, s, `SELECT count(*) FROM operation_log.delivery WHERE state<>'PENDING'`) != 0 || count(t, s, `SELECT last_seq FROM operation_log.publication`) != 0 {
		t.Fatal("partial publication or database fault quarantined data")
	}
	if _, err = db.Pool.Exec(ctx, `DROP TRIGGER fail_publish ON operation_log.publication; DROP FUNCTION operation_log.fail_publish()`); err != nil {
		t.Fatal(err)
	}
	s.Close()
	restarted, err := Open(ctx, db.DSN)
	if err != nil {
		t.Fatal(err)
	}
	defer restarted.Close()
	r := projectOK(t, restarted)
	if r.Processed != 100 || r.PublishedSeq != 1 {
		t.Fatalf("batch %+v", r)
	}
	r = projectOK(t, restarted)
	if r.Processed != 5 || r.PublishedSeq != 2 || count(t, restarted, `SELECT count(*) FROM operation_log.entry_version`) != 105 {
		t.Fatalf("restart backlog %+v", r)
	}
}
func TestInvalidAndFuturePendingSourcesExcluded(t *testing.T) {
	s, db := newStore(t)
	a := fixture()
	b := finish(a)
	tx, err := db.Pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	insertRaw(t, tx, a, nil)
	payload, err := event.Canonical(b)
	if err != nil {
		t.Fatal(err)
	}
	var obj map[string]any
	json.Unmarshal(payload, &obj)
	obj["actionCode"] = "unknown.action"
	payload, _ = json.Marshal(obj)
	insertRaw(t, tx, b, payload)
	if err = tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	r := projectOK(t, s)
	if r.Quarantined != 1 || r.Processed != 1 {
		t.Fatalf("invalid handling %+v", r)
	}
	_, obs, _ := view(t, s, a.OperationID, 1)
	if obs != "START_ONLY" {
		t.Fatal("invalid finish folded")
	}
	c := fixture()
	d := finish(c)
	appendOK(t, s, c)
	appendOK(t, s, d)
	if _, err = db.Pool.Exec(ctx, `UPDATE operation_log.delivery SET next_attempt_at=clock_timestamp()+interval '1 hour' WHERE event_id=$1`, d.EventID); err != nil {
		t.Fatal(err)
	}
	projectOK(t, s)
	_, obs, _ = view(t, s, c.OperationID, 2)
	if obs != "START_ONLY" {
		t.Fatal("unvalidated pending finish folded")
	}
}
func TestStatusMonotonicAndReadOnly(t *testing.T) {
	s, _ := newStore(t)
	now := time.Now().UTC()
	v := ingest.Status{ProducerID: uuid.NewString(), StartedAt: now, ObservedAt: now, SnapshotNo: 2, AttemptedEvents: 3, ConfirmedEvents: 2, InFlightEvents: 1, PersistenceReachable: ptr(true), LastConfirmedAt: &now}
	if err := s.PublishStatus(ctx, v); err != nil {
		t.Fatal(err)
	}
	v.SnapshotNo = 1
	v.ConfirmedEvents = 1
	v.InFlightEvents = 2
	if err := s.PublishStatus(ctx, v); err != nil {
		t.Fatal(err)
	}
	if count(t, s, `SELECT confirmed_events FROM operation_log.producer_status`) != 2 {
		t.Fatal("old status overwrote new")
	}
	if err := s.Read(ctx, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, `UPDATE operation_log.publication SET last_seq=123`)
		return err
	}); err == nil {
		t.Fatal("read transaction wrote")
	}
}
func TestProducerPoolRecoversMissingSchema(t *testing.T) {
	db := pgtest.NewUnmigrated(t)
	s, err := OpenProducer(ctx, db.DSN)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	if s.pool.Config().MaxConns != 4 {
		t.Fatal("unbounded producer pool")
	}
	e := fixture()
	if s.Append(ctx, e) == nil {
		t.Fatal("missing schema acknowledged")
	}
	if err = schema.Up(ctx, db.DSN); err != nil {
		t.Fatal(err)
	}
	appendOK(t, s, e)
	if count(t, s, `SELECT count(*) FROM operation_log.event`) != 1 {
		t.Fatal("producer failed to recover")
	}
}
