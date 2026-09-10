//go:build integration

package tradersync

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/useryege/athena/internal/accountstate/store/migrations"
	"github.com/useryege/athena/internal/testutil/pgtest"
	tsmodel "github.com/useryege/athena/internal/tradersync/types"
	"github.com/useryege/athena/util/polymarket"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type routedHTTP struct {
	base *url.URL
	next http.RoundTripper
}

func (r routedHTTP) RoundTrip(req *http.Request) (*http.Response, error) {
	c := req.Clone(req.Context())
	u := *req.URL
	u.Scheme, u.Host = r.base.Scheme, r.base.Host
	c.URL = &u
	return r.next.RoundTrip(c)
}

func TestResolverOwnerContextGrantAndEvidence(t *testing.T) {
	db := pgtest.New(t, migrations.FS, migrations.Dir)
	ctx := context.Background()
	owner := uuid.NewString()
	wallet := "0x1111111111111111111111111111111111111111"
	var denied atomic.Bool
	var changed atomic.Bool
	var renamed atomic.Bool
	var probe sync.Mutex
	var revokeOnHTTP atomic.Bool
	var auxUnavailable atomic.Bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Holding the account advisory lock while doing HTTP would make this unavailable.
		probe.Lock()
		tx, e := db.Pool.Begin(ctx)
		if e != nil {
			t.Error(e)
			return
		}
		var free bool
		e = tx.QueryRow(ctx, `SELECT pg_try_advisory_xact_lock(hashtextextended('athena:account:' || $1::text,0))`, owner).Scan(&free)
		tx.Rollback(ctx)
		probe.Unlock()
		if e != nil || !free {
			t.Error("external HTTP held account gate", e)
		}
		if revokeOnHTTP.Swap(false) {
			denied.Store(true)
		}
		if auxUnavailable.Load() && r.URL.Path != "/public-profile" {
			w.WriteHeader(503)
			return
		}
		switch r.URL.Path {
		case "/public-profile":
			addr := wallet
			if changed.Load() {
				addr = "0x2222222222222222222222222222222222222222"
			}
			name := "Alpha"
			if renamed.Load() {
				name = "Beta"
			}
			json.NewEncoder(w).Encode(map[string]any{"proxyWallet": addr, "name": name, "verifiedBadge": false, "createdAt": "2000-01-01T00:00:00Z"})
		case "/v1/user-stats":
			w.Write([]byte(`{"joinDate":"2024-06-19T21:35:45.083000Z","largestWin":0}`))
		case "/traded":
			w.Write([]byte(`{"user":"` + wallet + `","traded":9007199254740993}`))
		case "/value":
			w.Write([]byte(`[{"user":"` + wallet + `","value":0}]`))
		case "/user-pnl":
			if r.URL.Query().Get("interval") == "1w" {
				w.WriteHeader(503)
				return
			}
			w.Write([]byte(`[{"t":1700000000,"p":-9007199254740993.25},{"t":1789000000,"p":0}]`))
		default:
			t.Error(r.URL)
			w.WriteHeader(500)
		}
	}))
	defer server.Close()
	u, _ := url.Parse(server.URL)
	adapter := polymarket.NewProfileAdapter(routedHTTP{u, server.Client().Transport})
	grant := func(ctx context.Context, tx pgx.Tx, id string) error {
		otherTx, err := db.Pool.Begin(ctx)
		if err != nil {
			return err
		}
		var lockFree bool
		err = otherTx.QueryRow(ctx, `SELECT pg_try_advisory_xact_lock(hashtextextended('athena:account:' || $1::text,0))`, owner).Scan(&lockFree)
		otherTx.Rollback(ctx)
		if err != nil || lockFree {
			t.Error("grant not inside account gate", err)
		}
		if id != owner {
			return status.Error(codes.PermissionDenied, "wrong owner")
		}
		if denied.Load() {
			return status.Error(codes.PermissionDenied, "revoked")
		}
		return nil
	}
	contextCall := func(ctx context.Context, tx pgx.Tx, id string, w common.Address) (tsmodel.ResolutionContext, error) {
		if w.Hex() != common.HexToAddress(wallet).Hex() {
			t.Fatal(w)
		}
		return tsmodel.ResolutionContext{SavedNote: &tsmodel.TargetNote{Wallet: w, Note: "", Revision: 2}, Existing: &tsmodel.ExistingSubscription{ID: "existing", Status: "paused", Revision: 3}, Quota: tsmodel.Quota{Used: 2, Limit: 10}}, nil
	}
	if _, e := NewTargetResolver(db.Pool, adapter, nil, contextCall); e == nil {
		t.Fatal("nil grant accepted")
	}
	if _, e := NewTargetResolver(db.Pool, adapter, grant, nil); e == nil {
		t.Fatal("nil context accepted")
	}
	r, e := NewTargetResolver(db.Pool, adapter, grant, contextCall)
	if e != nil {
		t.Fatal(e)
	}
	resolved, e := r.Resolve(ctx, owner, wallet)
	if e != nil {
		t.Fatal(e)
	}
	if resolved.Context.SavedNote == nil || resolved.Context.SavedNote.Note != "" || resolved.Context.SavedNote.Revision != 2 || resolved.Context.Existing.Status != "paused" || resolved.Context.Quota.Used != 2 {
		t.Fatal(resolved.Context)
	}
	c := resolved.Card
	if len(c.PnL) != 6 || c.DefaultPeriod != "1Y" || c.UsageNotice == "" || c.Predictions.Value == nil || *c.Predictions.Value != "9007199254740993" || c.PositionValue.Value == nil || *c.PositionValue.Value != "0" || c.JoinedAt.Value == nil || *c.JoinedAt.Value != "2024-06-19T21:35:45.083000Z" {
		t.Fatal(c)
	}
	if c.Avatar.Value != nil || c.Verified.Value == nil || *c.Verified.Value != "false" || c.PnL["ALL"].Curve.Availability != "available" || c.PnL["ALL"].Amount.Value != nil || c.PnL["1W"].Curve.Availability != "unavailable" {
		t.Fatal(c)
	}
	for _, field := range []tsmodel.Scalar{c.Avatar, c.DisplayName, c.Verified, c.JoinedAt, c.PositionValue, c.LargestWin, c.Predictions} {
		if field.Source == "" || field.QueriedAt.IsZero() {
			t.Fatal(field)
		}
	}
	token, e := base64.RawURLEncoding.DecodeString(resolved.Token)
	if e != nil || len(token) != 32 {
		t.Fatal(resolved.Token, e)
	}
	digest := sha256.Sum256(token)
	var raw []byte
	var expires, now time.Time
	if e = db.Pool.QueryRow(ctx, `SELECT card_json,expires_at,clock_timestamp() FROM trader_sync_target_confirmations WHERE owner_id=$1 AND token_digest=$2`, owner, digest[:]).Scan(&raw, &expires, &now); e != nil {
		t.Fatal(e)
	}
	if expires.Sub(now) > 5*time.Minute || expires.Sub(now) < 4*time.Minute || !expires.Equal(resolved.ExpiresAt) {
		t.Fatal(expires, now, resolved.ExpiresAt)
	}
	var stored tsmodel.ConfirmationCard
	if e = json.Unmarshal(raw, &stored); e != nil || stored.Predictions.Value == nil || *stored.Predictions.Value != "9007199254740993" {
		t.Fatal(string(raw), e)
	}
	renamed.Store(true)
	if e = r.Revalidate(ctx, c.Identity); e != nil {
		t.Fatal("display rename changed identity", e)
	}
	changed.Store(true)
	if e = r.Revalidate(ctx, c.Identity); status.Code(e) != codes.FailedPrecondition {
		t.Fatal("wallet changed", e)
	}
	changed.Store(false)
	revokeOnHTTP.Store(true)
	if _, e = r.Resolve(ctx, owner, wallet); status.Code(e) != codes.PermissionDenied {
		t.Fatal(e)
	}
	var count int
	if e = db.Pool.QueryRow(ctx, `SELECT count(*) FROM trader_sync_target_confirmations`).Scan(&count); e != nil || count != 1 {
		t.Fatal(count, e)
	}
	denied.Store(false)
	r, e = NewTargetResolver(db.Pool, adapter, grant, func(context.Context, pgx.Tx, string, common.Address) (tsmodel.ResolutionContext, error) {
		return tsmodel.ResolutionContext{Quota: tsmodel.Quota{Limit: 10}}, nil
	})
	if e != nil {
		t.Fatal(e)
	}
	auxUnavailable.Store(true)
	resolved, e = r.Resolve(ctx, owner, wallet)
	if e != nil || resolved.Context.SavedNote != nil {
		t.Fatal(resolved, e)
	}
	if resolved.Card.JoinedAt.Value != nil || resolved.Card.PositionValue.Value != nil || resolved.Card.Predictions.Value != nil || resolved.Card.LargestWin.Value != nil {
		t.Fatal("auxiliary failure synthesized values", resolved.Card)
	}
	for _, period := range []string{"1D", "1W", "1M", "1Y", "YTD", "ALL"} {
		view, ok := resolved.Card.PnL[period]
		if !ok || view.Amount.Availability != "unavailable" || view.Curve.Availability != "unavailable" || view.Curve.Source == "" || view.Curve.QueriedAt.IsZero() {
			t.Fatal(period, view)
		}
	}
	if _, e = r.Resolve(ctx, strings.ReplaceAll(owner, "-", ""), wallet); e == nil {
		t.Fatal("invalid UUID accepted")
	}
}

func TestResolverRevalidatesPersistedOriginalAddress(t *testing.T) {
	db := pgtest.New(t, migrations.FS, migrations.Dir)
	ctx := context.Background()
	owner := uuid.NewString()
	const addressA = "0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	const walletB = "0xbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
	const walletC = "0xcccccccccccccccccccccccccccccccccccccccc"
	const addressD = "0xdddddddddddddddddddddddddddddddddddddddd"
	var mode atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.URL.Path != "/public-profile" {
			w.WriteHeader(503)
			return
		}
		address := req.URL.Query().Get("address")
		wallet := walletB
		if address == addressA {
			switch mode.Load() {
			case 1:
				wallet = walletC
			case 2:
				w.WriteHeader(404)
				return
			}
		}
		if address == walletC {
			wallet = walletC
		}
		name, avatar := "Before", "https://example.org/before.png"
		if mode.Load() == 3 {
			name, avatar = "After", "https://example.org/after.png"
		}
		json.NewEncoder(w).Encode(map[string]string{"proxyWallet": wallet, "name": name, "profileImage": avatar})
	}))
	defer server.Close()
	u, _ := url.Parse(server.URL)
	r, e := NewTargetResolver(db.Pool, polymarket.NewProfileAdapter(routedHTTP{u, server.Client().Transport}), func(context.Context, pgx.Tx, string) error { return nil }, func(context.Context, pgx.Tx, string, common.Address) (tsmodel.ResolutionContext, error) {
		return tsmodel.ResolutionContext{}, nil
	})
	if e != nil {
		t.Fatal(e)
	}
	resolved, e := r.Resolve(ctx, owner, " 0xAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA ")
	if e != nil {
		t.Fatal(e)
	}
	token, e := base64.RawURLEncoding.DecodeString(resolved.Token)
	if e != nil {
		t.Fatal(e)
	}
	digest := sha256.Sum256(token)
	read := func() tsmodel.Identity {
		t.Helper()
		tx, e := db.Pool.Begin(ctx)
		if e != nil {
			t.Fatal(e)
		}
		defer tx.Rollback(ctx)
		identity, e := r.store.ReadConfirmationTx(ctx, tx, owner, digest[:])
		if e != nil {
			t.Fatal(e)
		}
		return identity
	}
	persisted := read()
	raw, e := json.Marshal(persisted)
	if e != nil {
		t.Fatal(e)
	}
	var fields map[string]json.RawMessage
	if e = json.Unmarshal(raw, &fields); e != nil {
		t.Fatal(e)
	}
	var input string
	_ = json.Unmarshal(fields["ResolutionInput"], &input)
	if input != addressA {
		t.Errorf("persistent original query missing: got %q", input)
	}
	for _, tt := range []struct {
		name string
		mode int32
		want codes.Code
	}{{"stable_mapping", 0, codes.OK}, {"display_changes", 3, codes.OK}, {"A_maps_to_C_while_B_still_maps_to_B", 1, codes.FailedPrecondition}, {"A_unavailable_while_B_still_maps_to_B", 2, codes.FailedPrecondition}} {
		t.Run(tt.name, func(t *testing.T) {
			mode.Store(tt.mode)
			if e := r.Revalidate(ctx, read()); status.Code(e) != tt.want {
				t.Fatalf("revalidation = %v, want %s", e, tt.want)
			}
		})
	}
	mode.Store(0)
	other, e := r.Resolve(ctx, owner, addressD)
	if e != nil {
		t.Fatal(e)
	}
	if other.Card.Identity.Digest == persisted.Digest {
		t.Error("different original mappings share the same identity digest")
	}
	if _, e = db.Pool.Exec(ctx, `UPDATE trader_sync_target_confirmations SET identity_json=identity_json-'ResolutionInput' WHERE token_digest=$1`, digest[:]); e != nil {
		t.Fatal(e)
	}
	if e = r.Revalidate(ctx, read()); status.Code(e) != codes.FailedPrecondition {
		t.Errorf("missing source revalidation = %v", e)
	}
	if _, e = db.Pool.Exec(ctx, `UPDATE trader_sync_target_confirmations SET identity_json=jsonb_set(identity_json,'{ResolutionInput}',to_jsonb($2::text)) WHERE token_digest=$1`, digest[:], addressD); e != nil {
		t.Fatal(e)
	}
	if e = r.Revalidate(ctx, read()); status.Code(e) != codes.FailedPrecondition {
		t.Errorf("changed source with unchanged wallet revalidation = %v", e)
	}
}

func TestResolverRevalidatesOriginalRedirectURL(t *testing.T) {
	db := pgtest.New(t, migrations.FS, migrations.Dir)
	ctx := context.Background()
	html, e := os.ReadFile("../../util/polymarket/testdata/profile/profile.html")
	if e != nil {
		t.Fatal(e)
	}
	public, e := os.ReadFile("../../util/polymarket/testdata/profile/public-profile.json")
	if e != nil {
		t.Fatal(e)
	}
	var originalUnavailable atomic.Bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/@GCR":
			if originalUnavailable.Load() {
				w.WriteHeader(404)
				return
			}
			http.Redirect(w, r, "https://polymarket.com/@gcr", 302)
		case "/@gcr":
			w.Write(html)
		case "/public-profile":
			w.Write(public)
		default:
			w.WriteHeader(503)
		}
	}))
	defer server.Close()
	u, _ := url.Parse(server.URL)
	resolver, e := NewTargetResolver(db.Pool, polymarket.NewProfileAdapter(routedHTTP{u, server.Client().Transport}), func(context.Context, pgx.Tx, string) error { return nil }, func(context.Context, pgx.Tx, string, common.Address) (tsmodel.ResolutionContext, error) {
		return tsmodel.ResolutionContext{}, nil
	})
	if e != nil {
		t.Fatal(e)
	}
	resolved, e := resolver.Resolve(ctx, uuid.NewString(), "https://www.polymarket.com/@GCR")
	if e != nil {
		t.Fatal(e)
	}
	if e = resolver.Revalidate(ctx, resolved.Card.Identity); e != nil {
		t.Fatal("stable redirect mapping", e)
	}
	originalUnavailable.Store(true)
	if e = resolver.Revalidate(ctx, resolved.Card.Identity); status.Code(e) != codes.FailedPrecondition {
		t.Fatalf("original URL disappeared but canonical remained valid: %v", e)
	}
}
