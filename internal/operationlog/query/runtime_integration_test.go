//go:build integration

package query_test

import (
	"context"
	"testing"
	"time"

	"github.com/useryege/athena/internal/operationlog/query"
	"github.com/useryege/athena/internal/operationlog/schema"
	"github.com/useryege/athena/internal/operationlog/store"
	"github.com/useryege/athena/internal/testutil/pgtest"
)

func TestRuntimeStatusReadsPublicationAndPendingFacts(t *testing.T) {
	db := pgtest.NewUnmigrated(t)
	if err := schema.Up(context.Background(), db.DSN); err != nil {
		t.Fatal(err)
	}
	s, err := store.Open(context.Background(), db.DSN)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	s.SetQueryReady(true)
	a := &query.Adapter{RuntimeSource: s}
	st, err := a.Runtime(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if !st.QueryReady || st.ProjectionState != "READY" || st.PublicationSequence < 0 {
		t.Fatalf("runtime=%+v", st)
	}
	if st.CheckedAt.Before(time.Now().Add(-time.Minute)) {
		t.Fatalf("stale runtime=%+v", st)
	}
}
