//go:build integration

package server

import (
	"encoding/json"
	"github.com/ethereum/go-ethereum/common"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/useryege/athena/internal/accountstate/txgate"
	tm "github.com/useryege/athena/internal/tradersync/types"
	api "github.com/useryege/athena/pkg/apiclient/tradersync"
	"net/url"
	"reflect"
	"strconv"
	"testing"
	"time"
)

func TestTraderSyncActivityCursorsAcrossRealGatewayAndQueries(t *testing.T) {
	copyFilter := func(v url.Values) url.Values {
		r := url.Values{}
		for k, values := range v {
			r[k] = append([]string(nil), values...)
		}
		return r
	}
	h := newTraderSyncGatewayHarness(t, nil)
	ctx := h.ctx
	now := time.Now().UTC().Truncate(time.Second)
	if _, e := h.db.Pool.Exec(ctx, `INSERT INTO trader_sync_collector_epochs(id,fencing_token) VALUES(1,1)`); e != nil {
		t.Fatal(e)
	}
	baseline := func(sub string) string {
		id := uuid.NewString()
		if _, e := h.db.Pool.Exec(ctx, `INSERT INTO trader_sync_baseline_attempts(id,owner_id,subscription_id,activation_generation,expected_revision,state,effective_at) SELECT $1,owner_id,id,activation_generation,revision,'succeeded',$3 FROM trader_sync_subscriptions WHERE id=$2`, id, sub, now.Add(-4*time.Hour)); e != nil {
			t.Fatal(e)
		}
		if _, e := h.db.Pool.Exec(ctx, `INSERT INTO trader_sync_monitor_intervals(owner_id,subscription_id,baseline_attempt_id,activation_generation,collector_epoch,filter_revision,expected_revision,registered_high,candidate_effective_at,effective_at) SELECT owner_id,id,$1,activation_generation,1,1,revision,0,$3,$3 FROM trader_sync_subscriptions WHERE id=$2`, id, sub, now.Add(-4*time.Hour)); e != nil {
			t.Fatal(e)
		}
		return id
	}
	attempt := baseline(h.subscription)
	var otherSub string
	if e := h.db.Pool.QueryRow(ctx, `SELECT id::text FROM trader_sync_subscriptions WHERE owner_id=$1 AND id<>$2`, h.owner, h.subscription).Scan(&otherSub); e != nil {
		t.Fatal(e)
	}
	otherAttempt := baseline(otherSub)
	ids := []string{}
	for i := 1; i <= 16; i++ {
		if i == 2 {
			if _, e := h.db.Pool.Exec(ctx, `INSERT INTO telegram_bindings(account_id,telegram_user_id,telegram_chat_id,telegram_display_name,revision) VALUES($1,123,123,'fixture',1)`, h.owner); e != nil {
				t.Fatal(e)
			}
		}
		projection := gatewayProjection(t, h.db.Pool, h.owner, h.subscription, attempt, common.BytesToAddress(h.wallet), i)
		projection.Confirmation.SettledAt = now.Add(-time.Minute)
		id, created, e := h.traderStore.Project(ctx, projection)
		if e != nil || !created {
			t.Fatal(created, e)
		}
		ids = append(ids, strconv.FormatInt(id, 10))
	}
	var batch tm.SummaryBatch
	if e := txgate.WithAccountTx(ctx, h.db.Pool, h.owner, func(tx pgx.Tx) error {
		var e error
		batch, e = h.traderStore.FreezeSummaryTx(ctx, tx, h.owner, 1, 123)
		return e
	}); e != nil {
		t.Fatal(e)
	}
	if len(batch.Parts) == 0 {
		t.Fatal("summary fixture empty")
	}
	path := func(filter url.Values, token, refresh string) string {
		query := copyFilter(filter)
		if token != "" {
			query.Set("page.cursor", token)
		}
		if refresh != "" {
			query.Set("refresh_cursor", refresh)
		}
		return "/api/v1/trader-sync/activities?" + query.Encode()
	}
	get := func(filter url.Values, token, refresh string) *api.ListActivitiesResponse {
		raw := h.get("member", path(filter, token, refresh), 200)
		var page api.ListActivitiesResponse
		if e := json.Unmarshal(raw, &page); e != nil {
			t.Fatal(e)
		}
		if page.Page == nil || page.Page.RefreshCursor == "" || page.Page.Snapshot == "" {
			t.Fatal("missing actual cursor", string(raw))
		}
		return &page
	}
	listIDs := func(page *api.ListActivitiesResponse) []string {
		out := make([]string, 0, len(page.Activities))
		for _, a := range page.Activities {
			out = append(out, a.ID)
		}
		return out
	}
	assertPage := func(page *api.ListActivitiesResponse, want []string, newer bool) {
		t.Helper()
		if !reflect.DeepEqual(listIDs(page), want) || page.Page.HasNewer != newer {
			t.Fatalf("IDs/newer got %v/%v want %v/%v", listIDs(page), page.Page.HasNewer, want, newer)
		}
	}
	filter := url.Values{"page.page_size": {"2"}, "subscription_id": {h.subscription}, "from": {now.Add(-3 * time.Hour).Format(time.RFC3339)}, "to": {now.Format(time.RFC3339)}}
	first := get(filter, "", "")
	assertPage(first, []string{ids[15], ids[14]}, false)
	second := get(filter, first.Page.NextCursor, "")
	assertPage(second, []string{ids[13], ids[12]}, false)
	// Consume every returned next token: no hand-created lower/upper/snapshot values.
	seen := append(listIDs(first), listIDs(second)...)
	current := second
	for current.Page.NextCursor != "" {
		current = get(filter, current.Page.NextCursor, "")
		seen = append(seen, listIDs(current)...)
	}
	wantAll := make([]string, len(ids))
	for i := range ids {
		wantAll[i] = ids[len(ids)-1-i]
	}
	if !reflect.DeepEqual(seen, wantAll) {
		t.Fatal("next boundary skipped/duplicated", seen, wantAll)
	}
	emptyFilter := copyFilter(filter)
	emptyFilter.Set("from", now.Add(-2*time.Hour).Format(time.RFC3339))
	emptyFilter.Set("to", now.Add(-time.Hour).Format(time.RFC3339))
	empty := get(emptyFilter, "", "")
	assertPage(empty, []string{}, false)
	batchFilter := copyFilter(filter)
	batchFilter.Set("summary_batch_id", strconv.FormatInt(batch.ID, 10))
	batchPage := get(batchFilter, "", "")
	assertPage(batchPage, []string{ids[15], ids[14]}, false)
	batchNext := get(batchFilter, batchPage.Page.NextCursor, "")
	assertPage(batchNext, []string{ids[13], ids[12]}, false)

	// Explicit query fixtures: newly committed larger IDs may carry older recorded
	// and settled UTC. This is not historical scanning or a Project mutation.
	sourceIndex := 100
	clone := func(sub, baselineID string, settled time.Time, rollback bool) string {
		sourceIndex++
		source := gatewayProjection(t, h.db.Pool, h.owner, sub, baselineID, common.BytesToAddress(h.wallet), sourceIndex)
		tx, e := h.db.Pool.Begin(ctx)
		if e != nil {
			t.Fatal(e)
		}
		defer tx.Rollback(ctx)
		var id int64
		if e = tx.QueryRow(ctx, `INSERT INTO trader_sync_activities SELECT (jsonb_populate_record(NULL::trader_sync_activities,to_jsonb(a)||jsonb_build_object('id',nextval('trader_sync_activity_id_seq'),'source_record_id',$2::bigint,'subscription_id',$3::uuid,'interval_id',(SELECT id FROM trader_sync_monitor_intervals WHERE subscription_id=$3::uuid LIMIT 1),'recorded_at',$4::timestamptz,'settled_at',$5::timestamptz))).* FROM trader_sync_activities a WHERE id=$1 RETURNING id`, ids[0], source.Candidate.SourceID, sub, now.Add(-3*time.Hour), settled).Scan(&id); e != nil {
			t.Fatal(e)
		}
		if rollback {
			e = tx.Rollback(ctx)
		} else {
			e = tx.Commit(ctx)
		}
		if e != nil {
			t.Fatal(e)
		}
		return strconv.FormatInt(id, 10)
	}
	clone(h.subscription, attempt, now.Add(-90*time.Minute), true)
	assertPage(get(filter, "", first.Page.RefreshCursor), []string{ids[15], ids[14]}, false)
	assertPage(get(emptyFilter, "", empty.Page.RefreshCursor), []string{}, false)
	clone(otherSub, otherAttempt, now.Add(-90*time.Minute), false) // wrong subscription
	clone(h.subscription, attempt, now.Add(time.Hour), false)      // outside to
	clone(h.subscription, attempt, now.Add(-4*time.Hour), false)   // before from
	assertPage(get(filter, "", first.Page.RefreshCursor), []string{ids[15], ids[14]}, false)
	assertPage(get(emptyFilter, "", empty.Page.RefreshCursor), []string{}, false)
	latestID := clone(h.subscription, attempt, now.Add(-90*time.Minute), false)
	refreshed := get(filter, "", first.Page.RefreshCursor)
	assertPage(refreshed, []string{ids[15], ids[14]}, true)
	if refreshed.Page.NextCursor == "" {
		t.Fatal("refresh dropped older page continuation")
	}
	assertPage(get(filter, refreshed.Page.NextCursor, ""), []string{ids[13], ids[12]}, true)
	assertPage(get(emptyFilter, "", empty.Page.RefreshCursor), []string{}, true)
	assertPage(get(emptyFilter, "", ""), []string{latestID}, false)
	assertPage(get(filter, "", ""), []string{latestID, ids[15]}, false)
	// A sealed batch cannot acquire new memberships. New matching owner/time rows
	// outside that frozen batch do not set its hasNewer or change its original page.
	assertPage(get(batchFilter, "", batchPage.Page.RefreshCursor), []string{ids[15], ids[14]}, false)
	for name, change := range map[string]func(url.Values){
		"page size":    func(v url.Values) { v.Set("page.page_size", "3") },
		"subscription": func(v url.Values) { v.Set("subscription_id", otherSub) },
		"batch":        func(v url.Values) { v.Set("summary_batch_id", strconv.FormatInt(batch.ID, 10)) },
		"from":         func(v url.Values) { v.Set("from", now.Add(-5*time.Hour).Format(time.RFC3339)) },
		"to":           func(v url.Values) { v.Set("to", now.Add(time.Hour).Format(time.RFC3339)) },
	} {
		t.Run(name, func(t *testing.T) {
			changed := copyFilter(filter)
			change(changed)
			h.get("member", path(changed, first.Page.NextCursor, ""), 400)
			h.get("member", path(changed, "", first.Page.RefreshCursor), 400)
		})
	}
	h.get("other", path(filter, first.Page.NextCursor, ""), 400)
	h.get("other", path(filter, "", first.Page.RefreshCursor), 400)
	h.get("member", path(filter, first.Page.Snapshot, ""), 400)
	h.get("member", path(filter, "", first.Page.NextCursor), 400)
	h.get("member", path(filter, first.Page.NextCursor+"x", ""), 400)
	invalid := copyFilter(filter)
	invalid.Set("page.page_size", "101")
	h.get("member", path(invalid, "", ""), 400)
	for i := 0; i < 100; i++ {
		clone(h.subscription, attempt, now.Add(-90*time.Minute), false)
	}
	defaults := copyFilter(filter)
	defaults.Del("page.page_size")
	if len(get(defaults, "", "").Activities) != 50 {
		t.Fatal("default page size is not 50")
	}
	maximum := copyFilter(filter)
	maximum.Set("page.page_size", "100")
	if len(get(maximum, "", "").Activities) != 100 {
		t.Fatal("maximum page size is not 100")
	}
}
