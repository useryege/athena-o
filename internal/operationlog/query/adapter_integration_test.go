//go:build integration

package query_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/useryege/athena/internal/operationlog/event"
	q "github.com/useryege/athena/internal/operationlog/query"
	"github.com/useryege/athena/internal/operationlog/schema"
	"github.com/useryege/athena/internal/operationlog/store"
	"github.com/useryege/athena/internal/testutil/pgtest"
)

func TestAdapterReadsPublishedVersionsWithStableSnapshotAndKeyset(t *testing.T) {
	db := pgtest.NewUnmigrated(t)
	if err := schema.Up(context.Background(), db.DSN); err != nil {
		t.Fatal(err)
	}
	s, err := store.Open(context.Background(), db.DSN)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	now := time.Date(2026, 9, 18, 12, 0, 0, 0, time.UTC)
	aid := uuid.NewString()
	viewer := q.Viewer{AccountID: aid, Realm: "ADMIN", CredentialKind: "LOGIN_SESSION", SessionBinding: []byte("01234567890123456789012345678901"), AccessRevision: 1}
	codec, _ := q.NewCodec([]byte("01234567890123456789012345678901"), func() time.Time { return now })
	a := &q.Adapter{Reader: s, Codec: codec, Now: func() time.Time { return now }}
	for i := 0; i < 3; i++ {
		id := uuid.NewString()
		e := event.Event{SchemaVersion: 1, EventID: uuid.NewString(), OperationID: id, RequestID: uuid.NewString(), ProducerID: uuid.NewString(), Phase: event.Finish, StartedAt: now.Add(time.Duration(-i) * time.Minute), OccurredAt: now.Add(time.Duration(-i) * time.Minute), Actor: event.Actor{AccountID: &aid, UsernameSnapshot: ptr("Admin"), Role: "ADMINISTRATOR", Realm: "ADMIN", CredentialKind: "LOGIN_SESSION", IdentityVerified: true, IdentitySnapshotComplete: true}, ActionCode: "account.access.update", ModuleCode: "account", Outcome: event.Succeeded, Observation: event.FinishOnly, Resources: []event.Resource{}, Effect: []string{}, Details: event.Details{}, ResourcesComplete: true, DurationMs: ptr(int64(1))}
		if err := s.Append(context.Background(), e); err != nil {
			t.Fatal(err)
		}
	}
	if n := countRows(t, db.Pool); n != 3 {
		t.Fatalf("events=%d", n)
	}
	if _, err := s.Project(context.Background()); err != nil {
		t.Fatal(err)
	}
	p, err := a.List(context.Background(), q.Filter{PageSize: 2, ModuleCode: "account", From: ptr(now.Add(-time.Hour)), To: ptr(now.Add(time.Minute))}, viewer)
	if err != nil {
		t.Fatal(err)
	}
	if p.NextCursor == "" {
		_, e := codec.EncodeCursorPreserving(p.AppliedFilters, viewer, q.CursorPosition{StartedAt: p.Items[1].StartedAt, OperationID: p.Items[1].OperationID}, p.SnapshotSequence, p.SnapshotAt, time.Time{}, time.Time{}, now)
		t.Fatalf("cursor empty direct err=%v page=%+v", e, p)
	}
	if len(p.Items) != 2 || p.SnapshotSequence == 0 {
		t.Fatalf("page=%+v", p)
	}
	p2, err := a.List(context.Background(), q.Filter{PageSize: 2, ModuleCode: "account", From: ptr(now.Add(-time.Hour)), To: ptr(now.Add(time.Minute)), Cursor: p.NextCursor}, viewer)
	if err != nil {
		t.Fatal(err)
	}
	if len(p2.Items) != 1 {
		t.Fatalf("second page=%+v", p2)
	}
	if _, err := a.Get(context.Background(), p.Items[0].OperationID, viewer, p.SnapshotToken); err != nil {
		t.Fatal(err)
	}
}
func countRows(t *testing.T, p interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}) int {
	var n int
	_ = p.QueryRow(context.Background(), `SELECT count(*) FROM operation_log.event`).Scan(&n)
	return n
}
func ptr[T any](v T) *T { return &v }
