package txgate

import (
	"context"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"testing"
)

func TestCanonicalAccountIDRejectsNonCanonicalUUIDs(t *testing.T) {
	for _, value := range []string{
		"x5b0ddd23-bfc2-4c86-af84-28a984188f5cy",
		"00000000-0000-0000-0000-000000000000",
		"5b0ddd23-bfc2-4c86-af84-28a984188f5z",
	} {
		if _, err := canonicalAccountID(value); err == nil {
			t.Fatalf("canonicalAccountID(%q) succeeded", value)
		}
	}
}

func TestCanonicalAccountIDNormalizesUppercaseUUID(t *testing.T) {
	got, err := canonicalAccountID("5B0DDD23-BFC2-4C86-AF84-28A984188F5C")
	if err != nil {
		t.Fatalf("canonicalAccountID: %v", err)
	}
	if got != "5b0ddd23-bfc2-4c86-af84-28a984188f5c" {
		t.Fatalf("canonicalAccountID=%q", got)
	}
}

func TestWithAccountTxRejectsNilConcretePool(t *testing.T) {
	var pool *pgxpool.Pool
	err := WithAccountTx(context.Background(), pool, "5b0ddd23-bfc2-4c86-af84-28a984188f5c", func(pgx.Tx) error { t.Fatal("nil pool invoked callback"); return nil })
	if err == nil {
		t.Fatal("nil pool accepted")
	}
}
