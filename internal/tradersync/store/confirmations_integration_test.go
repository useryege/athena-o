//go:build integration

package store

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/useryege/athena/internal/accountstate/store/migrations"
	"github.com/useryege/athena/internal/testutil/pgtest"
	tsmodel "github.com/useryege/athena/internal/tradersync/types"
)

func TestConfirmationOwnerExpiryAndConsumption(t *testing.T) {
	db := pgtest.New(t, migrations.FS, migrations.Dir)
	ctx := context.Background()
	var dbname string
	if e := db.Pool.QueryRow(ctx, "select current_database()").Scan(&dbname); e != nil || !strings.HasPrefix(dbname, "athena_test_") {
		t.Fatal(dbname, e)
	}
	s := NewSQLStore(db.Pool)
	owner, other := uuid.NewString(), uuid.NewString()
	identity := tsmodel.Identity{ResolutionInput: "0x1111111111111111111111111111111111111111", Wallet: common.HexToAddress("0x1111111111111111111111111111111111111111"), Digest: sha256.Sum256([]byte("identity"))}
	token := sha256.Sum256([]byte("secret"))
	tx, e := db.Pool.Begin(ctx)
	if e != nil {
		t.Fatal(e)
	}
	if e = s.SaveConfirmationTx(ctx, tx, owner, identity, token[:], time.Now().Add(time.Minute)); e != nil {
		t.Fatal(e)
	}
	if e = tx.Commit(ctx); e != nil {
		t.Fatal(e)
	}
	run := func(fn func(pgx.Tx) error) error {
		tx, e := db.Pool.Begin(ctx)
		if e != nil {
			t.Fatal(e)
		}
		defer tx.Rollback(ctx)
		return fn(tx)
	}
	if e = run(func(tx pgx.Tx) error { _, e := s.ReadConfirmationTx(ctx, tx, other, token[:]); return e }); e == nil {
		t.Fatal("cross owner read")
	}
	if e = run(func(tx pgx.Tx) error {
		_, e := s.ConsumeConfirmationTx(ctx, tx, other, token[:], "request", identity.Digest[:])
		return e
	}); e == nil {
		t.Fatal("cross owner consumption")
	}
	if e = run(func(tx pgx.Tx) error {
		_, e := s.ConsumeConfirmationTx(ctx, tx, owner, token[:], "request", make([]byte, 32))
		return e
	}); e == nil {
		t.Fatal("identity mismatch")
	}
	if e = run(func(tx pgx.Tx) error {
		v, e := s.ReadConfirmationTx(ctx, tx, owner, token[:])
		if e == nil && v.Wallet != identity.Wallet {
			t.Fatal(v)
		}
		return e
	}); e != nil {
		t.Fatal(e)
	}
	// A rollback must restore the ability to consume.
	if e = run(func(tx pgx.Tx) error {
		_, e := s.ConsumeConfirmationTx(ctx, tx, owner, token[:], "rolled-back", identity.Digest[:])
		return e
	}); e != nil {
		t.Fatal(e)
	}
	tx, _ = db.Pool.Begin(ctx)
	if _, e = s.ConsumeConfirmationTx(ctx, tx, owner, token[:], "first", identity.Digest[:]); e != nil {
		t.Fatal(e)
	}
	if e = tx.Commit(ctx); e != nil {
		t.Fatal(e)
	}
	for _, request := range []string{"first", "second"} {
		if e = run(func(tx pgx.Tx) error {
			_, e := s.ConsumeConfirmationTx(ctx, tx, owner, token[:], request, identity.Digest[:])
			return e
		}); e == nil {
			t.Fatal("reconsumed", request)
		}
	}
	if e = run(func(tx pgx.Tx) error { _, e := s.ReadConfirmationTx(ctx, tx, owner, token[:]); return e }); e == nil {
		t.Fatal("read consumed")
	}
	token = sha256.Sum256([]byte("expired"))
	tx, _ = db.Pool.Begin(ctx)
	if e = s.SaveConfirmationTx(ctx, tx, owner, identity, token[:], time.Now().Add(-time.Second)); e != nil {
		t.Fatal(e)
	}
	tx.Commit(ctx)
	if e = run(func(tx pgx.Tx) error { _, e := s.ReadConfirmationTx(ctx, tx, owner, token[:]); return e }); e == nil {
		t.Fatal("read expired")
	}
	if e = run(func(tx pgx.Tx) error {
		_, e := s.ConsumeConfirmationTx(ctx, tx, owner, token[:], "expired", identity.Digest[:])
		return e
	}); e == nil {
		t.Fatal("consumed expired")
	}
}

func TestConfirmationSchemaProtectsDigestsWalletAndIdempotency(t *testing.T) {
	db := pgtest.New(t, migrations.FS, migrations.Dir)
	ctx := context.Background()
	owner := uuid.NewString()
	digest := sha256.Sum256([]byte("payload"))
	payload, _ := json.Marshal(map[string]string{"id": "result"})
	_, e := db.Pool.Exec(ctx, `INSERT INTO trader_sync_request_results(owner_id,operation,request_id,payload_digest,result_json) VALUES($1,'create','r',$2,$3)`, owner, digest[:], payload)
	if e != nil {
		t.Fatal(e)
	}
	if _, e = db.Pool.Exec(ctx, `INSERT INTO trader_sync_request_results(owner_id,operation,request_id,payload_digest,result_json) VALUES($1,'create','r',$2,$3)`, owner, digest[:], payload); e == nil {
		t.Fatal("duplicate request accepted")
	}
	if _, e = db.Pool.Exec(ctx, `INSERT INTO trader_sync_request_results(owner_id,operation,request_id,payload_digest,result_json) VALUES($1,'create','r',$2,$3)`, uuid.NewString(), digest[:], payload); e != nil {
		t.Fatal(e)
	}
	if _, e = db.Pool.Exec(ctx, `INSERT INTO trader_sync_request_results(owner_id,operation,request_id,payload_digest,result_json) VALUES($1,'create','bad',$2,$3)`, owner, []byte{1}, payload); e == nil {
		t.Fatal("short digest accepted")
	}
	if _, e = db.Pool.Exec(ctx, `INSERT INTO trader_sync_targets(wallet) VALUES($1)`, make([]byte, 20)); e != nil {
		t.Fatal(e)
	}
	if _, e = db.Pool.Exec(ctx, `INSERT INTO trader_sync_targets(wallet) VALUES($1)`, make([]byte, 20)); e == nil {
		t.Fatal("duplicate wallet accepted")
	}
}

func TestConfirmationIdentityResolutionInputRoundTrip(t *testing.T) {
	db := pgtest.New(t, migrations.FS, migrations.Dir)
	ctx := context.Background()
	s := NewSQLStore(db.Pool)
	owner := uuid.NewString()
	identity := tsmodel.Identity{Wallet: common.HexToAddress("0xbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"), Digest: sha256.Sum256([]byte("source-A-to-wallet-B"))}
	raw, e := json.Marshal(identity)
	if e != nil {
		t.Fatal(e)
	}
	var fields map[string]json.RawMessage
	json.Unmarshal(raw, &fields)
	fields["ResolutionInput"] = json.RawMessage(`"0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"`)
	raw, e = json.Marshal(fields)
	if e != nil {
		t.Fatal(e)
	}
	if e = json.Unmarshal(raw, &identity); e != nil {
		t.Fatal(e)
	}
	token := sha256.Sum256([]byte("source-bound-token"))
	tx, e := db.Pool.Begin(ctx)
	if e != nil {
		t.Fatal(e)
	}
	if e = s.SaveConfirmationTx(ctx, tx, owner, identity, token[:], time.Now().Add(time.Minute)); e != nil {
		t.Fatal(e)
	}
	if e = tx.Commit(ctx); e != nil {
		t.Fatal(e)
	}
	check := func(identity tsmodel.Identity) {
		t.Helper()
		raw, e := json.Marshal(identity)
		if e != nil {
			t.Fatal(e)
		}
		var got map[string]json.RawMessage
		if e = json.Unmarshal(raw, &got); e != nil {
			t.Fatal(e)
		}
		if string(got["ResolutionInput"]) != `"0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"` {
			t.Errorf("original resolution source lost in stored identity: %s", raw)
		}
	}
	tx, e = db.Pool.Begin(ctx)
	if e != nil {
		t.Fatal(e)
	}
	defer tx.Rollback(ctx)
	read, e := s.ReadConfirmationTx(ctx, tx, owner, token[:])
	if e != nil {
		t.Fatal(e)
	}
	check(read)
	consumed, e := s.ConsumeConfirmationTx(ctx, tx, owner, token[:], "create", identity.Digest[:])
	if e != nil {
		t.Fatal(e)
	}
	check(consumed)
}
