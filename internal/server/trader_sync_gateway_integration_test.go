//go:build integration

package server

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	ethabi "github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	et "github.com/ethereum/go-ethereum/core/types"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/useryege/athena/internal/accountaccess"
	"github.com/useryege/athena/internal/accountcredentials"
	ac "github.com/useryege/athena/internal/accountstate/store"
	"github.com/useryege/athena/internal/accountstate/store/migrations"
	"github.com/useryege/athena/internal/accountstate/txgate"
	"github.com/useryege/athena/internal/notification/delivery"
	ns "github.com/useryege/athena/internal/notification/store"
	facade "github.com/useryege/athena/internal/server/tradersync"
	"github.com/useryege/athena/internal/testutil/pgtest"
	ts "github.com/useryege/athena/internal/tradersync"
	sourceabi "github.com/useryege/athena/internal/tradersync/abi"
	tsstore "github.com/useryege/athena/internal/tradersync/store"
	tm "github.com/useryege/athena/internal/tradersync/types"
	api "github.com/useryege/athena/pkg/apiclient/tradersync"
	app "github.com/useryege/athena/pkg/apis/application/v1alpha1"
	gu "github.com/useryege/athena/util/grpc"
	session "github.com/useryege/athena/util/session"
	"google.golang.org/grpc"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

type gatewayRevocations struct{ revoked map[string]bool }

func (*gatewayRevocations) Init(context.Context) {}
func (s *gatewayRevocations) RevokeToken(_ context.Context, id string, _ time.Duration) error {
	s.revoked[id] = true
	return nil
}
func (s *gatewayRevocations) IsTokenRevoked(id string) bool { return s.revoked[id] }

type traderSyncGatewayHarness struct {
	httpServer                         *httptest.Server
	tokens                             map[string]string
	ctx                                context.Context
	db                                 *pgtest.DB
	traderStore                        *tsstore.SQLStore
	composition                        *traderSyncRuntime
	controller                         *accountaccess.Controller
	ids                                map[string]string
	profileRequests                    *atomic.Int32
	profileWallet, owner, subscription string
	wallet                             []byte
	get                                func(string, string, int) []byte
	postAs                             func(string, string, string, int) []byte
}

func newTraderSyncGatewayHarness(t *testing.T, configure func(*ts.Config)) *traderSyncGatewayHarness {
	t.Helper()
	db := pgtest.New(t, migrations.FS, migrations.Dir)
	ctx := context.Background()
	accounts := ac.NewSQLStore(db.Pool)
	traderStore := tsstore.NewSQLStore(db.Pool)
	accounts.SetAccessChangeHook(traderStore.ApplyAccessChangeTx)
	codec, e := accountcredentials.NewJWTCodec([]byte(strings.Repeat("gateway-signing-key", 4)))
	if e != nil {
		t.Fatal(e)
	}
	credentials, e := accountcredentials.NewCredentialManager(ctx, accounts, codec)
	if e != nil {
		t.Fatal(e)
	}
	ids := map[string]string{}
	for _, name := range []string{"member", "other", "admin"} {
		realm := accountcredentials.ApplicationRealmMember
		if name == "admin" {
			realm = accountcredentials.ApplicationRealmAdmin
		}
		a, _, err := credentials.RegisterExternalAccount(ctx, accountcredentials.IdentityProviderGoogle, name+"-subject", name+"@example.test", name+"-gateway", realm)
		if err != nil {
			t.Fatal(err)
		}
		ids[name] = a.ID
	}
	controller, e := accountaccess.NewController(ctx, accounts)
	if e != nil {
		t.Fatal(e)
	}
	for _, name := range []string{"member", "other"} {
		access, err := controller.Get(ids[name])
		if err != nil {
			t.Fatal(err)
		}
		access.LoginEnabled = true
		access.APIKeyEnabled = true
		access.Modules[accountaccess.ModuleTraderSync] = accountaccess.AccessLevelReadWrite
		if _, err = controller.Update(ctx, ids[name], access, access.Revision); err != nil {
			t.Fatal(err)
		}
	}
	sessions := session.NewSessionManager(credentials, codec, &gatewayRevocations{revoked: map[string]bool{}}, controller)
	tokens := map[string]string{}
	for _, name := range []string{"member", "other", "admin"} {
		realm := accountcredentials.ApplicationRealmMember
		if name == "admin" {
			realm = accountcredentials.ApplicationRealmAdmin
		}
		tokens[name], e = sessions.CreateExternalLogin(ctx, ids[name], realm, accountcredentials.IdentityProviderGoogle, name+"-subject", name+"@example.test", 3600, uuid.NewString())
		if e != nil {
			t.Fatal(e)
		}
	}
	tokens["key"], e = credentials.IssueAPIKey(ctx, ids["member"], "gateway-key", 3600)
	if e != nil {
		t.Fatal(e)
	}
	cfg := ts.Config{HTTPURL: "http://127.0.0.1:1", WebSocketURL: "ws://127.0.0.1:1", SiteURL: "https://athena.test", CursorHMACKey: "gateway-cursor-key"}
	if configure != nil {
		configure(&cfg)
	}
	composition, e := newTraderSyncRuntime(ctx, cfg, db.Pool, traderStore)
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() { _ = composition.Close() })
	profileRequests := &atomic.Int32{}
	profileWallet := "0x00000000000000000000000000000000000000cc"
	profileHTTP := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		profileRequests.Add(1)
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/v1/rfq/combo-markets":
			w.WriteHeader(http.StatusServiceUnavailable) // bounded runtime test provider failure
		case "/public-profile":
			fmt.Fprintf(w, `{"proxyWallet":%q,"name":"Gateway Target","verifiedBadge":false}`, profileWallet)
		case "/v1/user-stats":
			fmt.Fprint(w, `{"joinDate":"2024-06-19T21:35:45.083000Z","largestWin":0}`)
		case "/traded":
			fmt.Fprintf(w, `{"user":%q,"traded":9007199254740993}`, profileWallet)
		case "/value":
			fmt.Fprintf(w, `[{"user":%q,"value":0}]`, profileWallet)
		case "/user-pnl":
			fmt.Fprint(w, `[{"t":1789000000,"p":0}]`)
		default:
			t.Errorf("unexpected profile request %s", r.URL)
			w.WriteHeader(500)
		}
	}))
	t.Cleanup(profileHTTP.Close)
	// Test-only routing keeps the actual ProfileAdapter/request paths while every
	// socket is loopback. No deployment transport disables certificate validation.
	profileURL, _ := url.Parse(profileHTTP.URL)
	composition.transport.DialContext = func(c context.Context, network, _ string) (net.Conn, error) {
		return (&net.Dialer{}).DialContext(c, network, profileURL.Host)
	}
	composition.transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}

	owner := ids["member"]
	subscription := uuid.NewString()
	wallet := make([]byte, 20)
	wallet[19] = 42
	missing := tm.Scalar{Evidence: tm.Evidence{Availability: "unavailable", ReasonCode: "not_observed", Source: "gateway_fixture", QueriedAt: time.Now().UTC()}}
	display, _ := json.Marshal(tm.TargetDisplay{DisplayName: missing, Avatar: missing, ProfileURL: missing})
	if _, e = db.Pool.Exec(ctx, `INSERT INTO trader_sync_subscriptions(id,owner_id,wallet,target_display,desired_state,observation_state,revision) VALUES($1,$2,$3,$4,'enabled','pending_baseline',9007199254740993)`, subscription, owner, wallet, display); e != nil {
		t.Fatal(e)
	}
	if _, e = db.Pool.Exec(ctx, `INSERT INTO trader_sync_target_notes(owner_id,wallet,note,revision) VALUES($1,$2,'private-note',1)`, owner, wallet); e != nil {
		t.Fatal(e)
	}
	secondWallet := append([]byte(nil), wallet...)
	secondWallet[19] = 43
	if _, e = db.Pool.Exec(ctx, `INSERT INTO trader_sync_subscriptions(owner_id,wallet,target_display,desired_state,observation_state) VALUES($1,$2,$3,'enabled','pending_baseline')`, owner, secondWallet, display); e != nil {
		t.Fatal(e)
	}
	athena := &AthenaServer{accessController: controller, credentialMgr: credentials, sessionMgr: sessions}
	listener, e := net.Listen("tcp", "127.0.0.1:0")
	if e != nil {
		t.Fatal(e)
	}
	grpcServer := grpc.NewServer(grpc.UnaryInterceptor(func(c context.Context, r any, i *grpc.UnaryServerInfo, h grpc.UnaryHandler) (any, error) {
		next, err := athena.authorizeGRPC(c, i.FullMethod, i.Server, r)
		if err != nil {
			return nil, err
		}
		return h(next, r)
	}))
	api.RegisterTraderSyncServiceServer(grpcServer, facade.New(composition.service))
	go grpcServer.Serve(listener)
	t.Cleanup(grpcServer.Stop)
	conn, e := grpc.Dial(listener.Addr().String(), grpc.WithInsecure())
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() { _ = conn.Close() })
	mux := runtime.NewServeMux(runtime.WithMarshalerOption(runtime.MIMEWildcard, new(gu.JSONMarshaler)))
	if e = api.RegisterTraderSyncServiceHandler(ctx, mux, conn); e != nil {
		t.Fatal(e)
	}
	httpServer := httptest.NewServer(mux)
	t.Cleanup(httpServer.Close)
	captureIndex := 0
	get := func(principal, path string, want int) []byte {
		t.Helper()
		request, _ := http.NewRequest(http.MethodGet, httpServer.URL+path, nil)
		if principal != "" {
			request.Header.Set("Authorization", "Bearer "+tokens[principal])
		}
		response, err := httpServer.Client().Do(request)
		if err != nil {
			t.Fatal(err)
		}
		defer response.Body.Close()
		raw, err := io.ReadAll(response.Body)
		if err != nil {
			t.Fatal(err)
		}
		if response.StatusCode != want {
			t.Fatalf("%s %s: status=%d body=%s", principal, path, response.StatusCode, raw)
		}
		if dir := os.Getenv("ATHENA_TRADER_SYNC_FIXTURE_DIR"); dir != "" && want == 200 {
			if err = os.MkdirAll(dir, 0755); err != nil {
				t.Fatal(err)
			}
			captureIndex++
			name := fmt.Sprintf("%02d-", captureIndex) + principal + "-" + strings.NewReplacer("/", "_", "?", "_", "=", "_", "&", "_").Replace(path) + ".json"
			if err = os.WriteFile(filepath.Join(dir, name), raw, 0644); err != nil {
				t.Fatal(err)
			}
		}
		return raw
	}
	postAs := func(principal, path, body string, want int) []byte {
		t.Helper()
		request, _ := http.NewRequest(http.MethodPost, httpServer.URL+path, strings.NewReader(body))
		request.Header.Set("Authorization", "Bearer "+tokens[principal])
		request.Header.Set("Content-Type", "application/json")
		response, err := httpServer.Client().Do(request)
		if err != nil {
			t.Fatal(err)
		}
		defer response.Body.Close()
		raw, _ := io.ReadAll(response.Body)
		if response.StatusCode != want {
			t.Fatalf("POST %s got %d: %s", path, response.StatusCode, raw)
		}
		return raw
	}
	return &traderSyncGatewayHarness{httpServer, tokens, ctx, db, traderStore, composition, controller, ids, profileRequests, profileWallet, owner, subscription, wallet, get, postAs}
}

func TestTraderSyncGatewayRealCredentialsAndOwnerPrivacy(t *testing.T) {
	h := newTraderSyncGatewayHarness(t, nil)
	ctx, db, traderStore, controller := h.ctx, h.db, h.traderStore, h.controller
	owner, subscription, wallet := h.owner, h.subscription, h.wallet
	profileRequests, profileWallet := h.profileRequests, h.profileWallet
	get, postAs := h.get, h.postAs
	httpServer, tokens := h.httpServer, h.tokens
	var e error
	post := func(path, body string, want int) []byte { return postAs("member", path, body, want) }
	base := "/api/v1/trader-sync/subscriptions"
	bad := post(base, fmt.Sprintf(`{"confirmationToken":"%%%%","requestId":%q}`, uuid.NewString()), 400)
	var tokenError struct {
		Code int `json:"code"`
	}
	if e = json.Unmarshal(bad, &tokenError); e != nil || tokenError.Code != 9 {
		t.Fatalf("bad token must be FailedPrecondition: %s", bad)
	}
	for _, principal := range []string{"member", "key"} {
		raw := get(principal, base+"/"+subscription, 200)
		if !strings.Contains(string(raw), `"revision":"9007199254740993"`) || !strings.Contains(string(raw), "private-note") {
			t.Fatal("own detail mapping lost", string(raw))
		}
		var body map[string]any
		if e = json.Unmarshal(raw, &body); e != nil {
			t.Fatal(e)
		}
	}
	get("other", base+"/"+subscription, 404)
	get("other", base+"/"+uuid.NewString(), 404)
	get("admin", base+"/"+subscription, 403)
	get("", base, 401)
	get("member", base+"?page.page_size=101", 400)
	// A signed cursor for one owner cannot be replayed as another principal.
	raw := get("member", base+"?page.page_size=1", 200)
	var listing struct {
		Page struct {
			NextCursor string `json:"nextCursor"`
		} `json:"page"`
	}
	if e = json.Unmarshal(raw, &listing); e != nil || listing.Page.NextCursor == "" {
		t.Fatal("missing bounded next cursor", string(raw), e)
	}
	get("other", base+"?page.page_size=1&page.cursor="+url.QueryEscape(listing.Page.NextCursor), 400)
	get("member", base+"?page.page_size=2&page.cursor="+url.QueryEscape(listing.Page.NextCursor), 400)
	adminRaw := get("admin", "/api/v1/admin/trader-sync/subscriptions/"+subscription, 200)
	if strings.Contains(string(adminRaw), "private-note") || strings.Contains(string(adminRaw), "targetDisplay") {
		t.Fatal("administrator received private projection", string(adminRaw))
	}
	get("member", "/api/v1/admin/trader-sync/subscriptions", 403)
	resolved := post("/api/v1/trader-sync/targets:resolve", fmt.Sprintf(`{"input":%q}`, profileWallet), 200)
	var resolvedBody struct {
		Target struct {
			ConfirmationToken string `json:"confirmationToken"`
			Verified          struct {
				Value *bool `json:"value"`
			} `json:"verified"`
			PositionValue struct {
				Value *string `json:"value"`
			} `json:"positionValue"`
		} `json:"target"`
	}
	if e = json.Unmarshal(resolved, &resolvedBody); e != nil || resolvedBody.Target.ConfirmationToken == "" || resolvedBody.Target.Verified.Value == nil || *resolvedBody.Target.Verified.Value || resolvedBody.Target.PositionValue.Value == nil || *resolvedBody.Target.PositionValue.Value != "0" {
		t.Fatal("resolved false/zero evidence lost", string(resolved), e)
	}
	createBody := fmt.Sprintf(`{"confirmationToken":%q,"requestId":%q}`, resolvedBody.Target.ConfirmationToken, uuid.NewString())
	otherToken := postAs("other", base, createBody, 400)
	if !strings.Contains(string(otherToken), `"code":9`) {
		t.Fatal("cross-owner token exposed", string(otherToken))
	}
	created := post(base, createBody, 200)
	if !strings.Contains(string(created), "Gateway Target") {
		t.Fatal("confirmed target display lost", string(created))
	}
	requestsBeforeReplay := profileRequests.Load()
	post(base, createBody, 200)
	if profileRequests.Load() != requestsBeforeReplay {
		t.Fatal("idempotent create revalidated externally")
	}
	if dir := os.Getenv("ATHENA_TRADER_SYNC_FIXTURE_DIR"); dir != "" {
		if e = os.WriteFile(filepath.Join(dir, "resolve-real-zero-false.json"), resolved, 0644); e != nil {
			t.Fatal(e)
		}
		if e = os.WriteFile(filepath.Join(dir, "create-real-confirmation.json"), created, 0644); e != nil {
			t.Fatal(e)
		}
	}
	noteRequest, _ := http.NewRequest(http.MethodPatch, httpServer.URL+"/api/v1/trader-sync/targets/"+profileWallet+"/note", strings.NewReader(fmt.Sprintf(`{"note":"新备注","expectedRevision":"0","requestId":%q}`, uuid.NewString())))
	noteRequest.Header.Set("Content-Type", "application/json")
	noteRequest.Header.Set("Authorization", "Bearer "+tokens["member"])
	noteResponse, err := httpServer.Client().Do(noteRequest)
	if err != nil {
		t.Fatal(err)
	}
	noteRaw, _ := io.ReadAll(noteResponse.Body)
	noteResponse.Body.Close()
	if noteResponse.StatusCode != 200 || !strings.Contains(string(noteRaw), "新备注") {
		t.Fatal("note mutation lost Unicode", noteResponse.Status, string(noteRaw))
	}
	saved := post("/api/v1/trader-sync/targets:resolve", fmt.Sprintf(`{"input":%q}`, profileWallet), 200)
	if !strings.Contains(string(saved), `"savedNote"`) || !strings.Contains(string(saved), `"existingSubscription"`) {
		t.Fatal("resolve lost owner context", string(saved))
	}
	if dir := os.Getenv("ATHENA_TRADER_SYNC_FIXTURE_DIR"); dir != "" {
		if e = os.WriteFile(filepath.Join(dir, "resolve-saved-note-existing.json"), saved, 0644); e != nil {
			t.Fatal(e)
		}
	}
	// Capture all three real summary states and a sent outcome with no start.
	attempt := uuid.NewString()
	if _, e = db.Pool.Exec(ctx, `INSERT INTO trader_sync_collector_epochs(id,fencing_token) VALUES(1,1)`); e != nil {
		t.Fatal(e)
	}
	if _, e = db.Pool.Exec(ctx, `INSERT INTO trader_sync_baseline_attempts(id,owner_id,subscription_id,activation_generation,expected_revision,state,effective_at) VALUES($1,$2,$3,1,9007199254740993,'succeeded',clock_timestamp()-interval '1 minute')`, attempt, owner, subscription); e != nil {
		t.Fatal(e)
	}
	if _, e = db.Pool.Exec(ctx, `INSERT INTO trader_sync_monitor_intervals(owner_id,subscription_id,baseline_attempt_id,activation_generation,collector_epoch,filter_revision,expected_revision,registered_high,candidate_effective_at,effective_at) VALUES($1,$2,$3,1,1,1,9007199254740993,0,clock_timestamp()-interval '1 minute',clock_timestamp()-interval '1 minute')`, owner, subscription, attempt); e != nil {
		t.Fatal(e)
	}
	activityIDs := []int64{}
	for i := 1; i <= 12; i++ {
		if i == 2 {
			if _, e = db.Pool.Exec(ctx, `INSERT INTO telegram_bindings(account_id,telegram_user_id,telegram_chat_id,telegram_display_name,revision) VALUES($1,123,123,'fixture',1)`, owner); e != nil {
				t.Fatal(e)
			}
		}
		id, created, err := traderStore.Project(ctx, gatewayProjection(t, db.Pool, owner, subscription, attempt, common.BytesToAddress(wallet), i))
		if err != nil || !created {
			t.Fatal(created, err)
		}
		activityIDs = append(activityIDs, id)
	}
	activityPath := func(id int64) string { return fmt.Sprintf("/api/v1/trader-sync/activities/%d", id) }
	historical := get("member", activityPath(activityIDs[0]), 200)
	if !strings.Contains(string(historical), "in_app_only") || !strings.Contains(string(historical), `"priceNumerator":"0"`) {
		t.Fatal("zero/missing metadata or historical mode lost", string(historical))
	}
	get("other", activityPath(activityIDs[0]), 404)
	ordinary, e := traderStore.ReadActivities(ctx, owner, tm.ActivityReadInput{Limit: 1, ActivityID: activityIDs[1]})
	if e != nil || ordinary.Activities[0].Delivery == nil {
		t.Fatal(e)
	}
	notificationStore := ns.NewSQLStore(db.Pool)
	permit, e := notificationStore.Authorize(ctx, delivery.Candidate{Ref: delivery.WorkRef{Kind: "account", ID: ordinary.Activities[0].Delivery.ID}, OwnerID: owner, ChatID: 123}, uuid.New(), nil)
	if e != nil {
		t.Fatal(e)
	}
	if e = notificationStore.RecordOutcome(ctx, permit, delivery.Outcome{Kind: "sent", MessageID: "fixture42"}, time.Now()); e != nil {
		t.Fatal(e)
	}
	sentRaw := get("member", activityPath(activityIDs[1]), 200)
	if !strings.Contains(string(sentRaw), `"status":"sent"`) || strings.Contains(string(sentRaw), `"startedAt"`) {
		t.Fatal("sent without start misrepresented", string(sentRaw))
	}
	waiting := get("member", activityPath(activityIDs[10]), 200)
	if !strings.Contains(string(waiting), `"phase":"waiting"`) {
		t.Fatal(string(waiting))
	}
	var batch tm.SummaryBatch
	if e = txgate.WithAccountTx(ctx, db.Pool, owner, func(tx pgx.Tx) error {
		var err error
		batch, err = traderStore.FreezeSummaryTx(ctx, tx, owner, 1, 123)
		return err
	}); e != nil {
		t.Fatal(e)
	}
	frozen := get("member", activityPath(activityIDs[10]), 200)
	if !strings.Contains(string(frozen), `"phase":"frozen"`) {
		t.Fatal(string(frozen))
	}
	get("member", fmt.Sprintf("/api/v1/trader-sync/summaries/%d", batch.ID), 200)
	get("member", fmt.Sprintf("/api/v1/trader-sync/summaries/%d/parts?page.page_size=1", batch.ID), 200)
	id, _, e := traderStore.Project(ctx, gatewayProjection(t, db.Pool, owner, subscription, attempt, common.BytesToAddress(wallet), 13))
	if e != nil {
		t.Fatal(e)
	}
	if _, e = notificationStore.DeleteTelegramBinding(ctx, owner); e != nil {
		t.Fatal(e)
	}
	cancelled := get("member", activityPath(id), 200)
	if !strings.Contains(string(cancelled), `"phase":"cancelled_before_freeze"`) {
		t.Fatal(string(cancelled))
	}
	pauseBody := fmt.Sprintf(`{"expectedRevision":"9007199254740993","requestId":%q}`, uuid.NewString())
	paused := post(base+"/"+subscription+":pause", pauseBody, 200)
	if !strings.Contains(string(paused), `"revision":"9007199254740994"`) || !strings.Contains(string(paused), `"pausedAt"`) {
		t.Fatal("pause lost exact revision or real end", string(paused))
	}
	replayed := post(base+"/"+subscription+":pause", pauseBody, 200)
	if !strings.Contains(string(replayed), `"revision":"9007199254740994"`) {
		t.Fatal("idempotent pause mutated revision")
	}
	post(base+"/"+subscription+":resume", fmt.Sprintf(`{"expectedRevision":"1","requestId":%q}`, uuid.NewString()), 409)
	resumed := post(base+"/"+subscription+":resume", fmt.Sprintf(`{"expectedRevision":"9007199254740994","requestId":%q}`, uuid.NewString()), 200)
	if !strings.Contains(string(resumed), `"revision":"9007199254740995"`) || strings.Contains(string(resumed), `"pausedAt"`) {
		t.Fatal("resume retained obsolete pause boundary", string(resumed))
	}
	cancelledSub := post(base+"/"+subscription+":cancel", fmt.Sprintf(`{"expectedRevision":"9007199254740995","requestId":%q}`, uuid.NewString()), 200)
	if !strings.Contains(string(cancelledSub), `"status":"cancelled"`) {
		t.Fatal(string(cancelledSub))
	}
	get("member", base+"/"+subscription+"/history?page.page_size=1", 200)
	get("admin", "/api/v1/admin/trader-sync/status", 200)
	get("admin", "/api/v1/admin/trader-sync/subscriptions?page.page_size=1", 200)
	get("member", "/api/v1/trader-sync/activities?page.page_size=1", 200)
	access, e := controller.Get(owner)
	if e != nil {
		t.Fatal(e)
	}
	// Persisted anomaly DTOs retain unknown replacement evidence and owner isolation.
	if _, e = db.Pool.Exec(ctx, `INSERT INTO trader_sync_finality_anomalies(chain_id,transaction_hash,published_block_hash,reason) VALUES(137,$1,$2,'source_removed')`, common.BigToHash(common.Big1).Bytes(), common.HexToHash("0xaa").Bytes()); e != nil {
		t.Fatal(e)
	}
	unknownConflict := get("member", activityPath(activityIDs[0]), 200)
	if !strings.Contains(string(unknownConflict), `"finalityAnomaly":{"reason":"source_removed"`) || strings.Contains(string(unknownConflict), `"conflictingBlockHash"`) {
		t.Fatal("unknown conflicting hash must remain absent", string(unknownConflict))
	}
	conflict := common.HexToHash("0xbb")
	if _, e = db.Pool.Exec(ctx, `UPDATE trader_sync_finality_anomalies SET conflicting_block_hash=$1 WHERE chain_id=137`, conflict.Bytes()); e != nil {
		t.Fatal(e)
	}
	knownConflict := get("member", activityPath(activityIDs[0]), 200)
	if !strings.Contains(string(knownConflict), `"conflictingBlockHash":"`+conflict.Hex()+`"`) {
		t.Fatal("known conflicting hash lost", string(knownConflict))
	}
	get("other", activityPath(activityIDs[0]), 404)
	get("admin", activityPath(activityIDs[0]), 403)
	// Credential switches are independent of the product grant, which stays intact.
	access.APIKeyEnabled = false
	if access, e = controller.Update(ctx, owner, access, access.Revision); e != nil {
		t.Fatal(e)
	}
	get("key", base+"/"+subscription, 401)
	get("member", base+"/"+subscription, 200)
	access.APIKeyEnabled = true
	access.LoginEnabled = false
	if access, e = controller.Update(ctx, owner, access, access.Revision); e != nil {
		t.Fatal(e)
	}
	get("member", base+"/"+subscription, 503)
	get("key", base+"/"+subscription, 503)
	access.LoginEnabled = true
	access.Modules[accountaccess.ModuleTraderSync] = accountaccess.AccessLevelNone
	if _, e = controller.Update(ctx, owner, access, access.Revision); e != nil {
		t.Fatal(e)
	}
	get("member", base+"/"+subscription, 403)
	get("key", base+"/"+subscription, 403)
}

// gatewayProjection inserts explicit synthetic source evidence; production Project
// forms the activity/ordinary/summary records. This does not verify a real chain.
func gatewayProjection(t *testing.T, pool *pgxpool.Pool, owner, sub, attempt string, wallet common.Address, index int) tm.Projection {
	t.Helper()
	ctx := context.Background()
	at := time.Now().UTC().Add(-time.Second)
	raw := et.Log{Topics: []common.Hash{}, Data: []byte{}, Address: common.HexToAddress("0xe111180000d2663c0091e4f400237545b87b996b"), BlockHash: common.HexToHash("0xaa"), TxHash: common.BigToHash(common.Big1), Index: uint(index), BlockNumber: 1}
	data, _ := json.Marshal(raw)
	var id int64
	if e := pool.QueryRow(ctx, `INSERT INTO trader_sync_source_records(chain_id,exchange_address,wallet,block_hash,transaction_hash,log_index,block_number,raw_json,collector_epoch,read_sequence,received_at,removed)VALUES(137,$1,$2,$3,$4,$5,1,$6,1,$5,$7,false) RETURNING id`, raw.Address.Bytes(), wallet.Bytes(), raw.BlockHash.Bytes(), raw.TxHash.Bytes(), index, data, at).Scan(&id); e != nil {
		t.Fatal(e)
	}
	if _, e := pool.Exec(ctx, `INSERT INTO trader_sync_source_candidates(source_record_id,owner_id,subscription_id,activation_generation,baseline_attempt_id,received_at)VALUES($1,$2,$3,1,$4,$5)`, id, owner, sub, attempt, at); e != nil {
		t.Fatal(e)
	}
	return tm.Projection{Candidate: tm.Candidate{SourceID: id, OwnerID: owner, SubscriptionID: sub, Generation: 1, AttemptID: attempt, ReceivedAt: at}, Trade: tm.Trade{Wallet: wallet, Exchange: raw.Address, Side: "BUY", PositionID: "90071992547409931234567890", CollateralRaw: "0", SharesRaw: "10000000", FeeRaw: "0", CollateralSymbol: "USDC", CollateralDecimals: 6, SharesDecimals: 6, SourceVersion: "polygon137-core-v2-ccc0596074f4", PriceNumerator: "0", PriceDenominator: "1"}, Confirmation: tm.CanonicalEvidence{Status: "confirmed", BlockHash: raw.BlockHash, SettledAt: at, CheckedAt: at}}
}

func TestTraderSyncRuntimeRawGaugesUseLiveQueueAndPersist(t *testing.T) {
	type wireWriter struct {
		conn *websocket.Conn
		mu   *sync.Mutex
	}
	ready := make(chan wireWriter, 1)
	up := websocket.Upgrader{}
	wss := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, e := up.Upgrade(w, r, nil)
		if e != nil {
			return
		}
		defer c.Close()
		mu := &sync.Mutex{}
		for {
			var req struct {
				ID     uint64 `json:"id"`
				Method string `json:"method"`
			}
			if c.ReadJSON(&req) != nil {
				return
			}
			result := any(true)
			if req.Method == "eth_chainId" {
				result = "0x89"
			}
			if req.Method == "eth_subscribe" {
				result = fmt.Sprint(req.ID)
			}
			mu.Lock()
			e = c.WriteJSON(map[string]any{"jsonrpc": "2.0", "id": req.ID, "result": result})
			mu.Unlock()
			if e != nil {
				return
			}
			if req.Method == "eth_chainId" {
				ready <- wireWriter{c, mu}
			}
		}
	}))
	defer wss.Close()
	h := newTraderSyncGatewayHarness(t, func(c *ts.Config) { c.WebSocketURL = "ws" + strings.TrimPrefix(wss.URL, "http") })
	runtimeStatus := func() app.TraderSyncRuntimeStatus {
		raw := h.get("admin", "/api/v1/admin/trader-sync/status", 200)
		var result struct {
			Status app.TraderSyncRuntimeStatus `json:"status"`
		}
		if e := json.Unmarshal(raw, &result); e != nil {
			t.Fatal(e)
		}
		return result.Status
	}
	find := func(result app.TraderSyncRuntimeStatus, name string) *app.TraderSyncRuntimeMetric {
		for i := range result.Metrics {
			if result.Metrics[i].Name == name {
				return &result.Metrics[i]
			}
		}
		return nil
	}
	unavailable := func(result app.TraderSyncRuntimeStatus) {
		v := find(result, "raw_observation_available")
		if v == nil || v.Value != "0" || v.Unit != "boolean" || v.Kind != "gauge" || find(result, "raw_queue_depth") != nil || find(result, "raw_persist_in_flight") != nil {
			t.Fatalf("unavailable raw evidence must omit values: %+v", result)
		}
	}
	unavailable(runtimeStatus())
	lock, e := h.db.Pool.Begin(h.ctx)
	if e != nil {
		t.Fatal(e)
	}
	defer lock.Rollback(h.ctx)
	if e = txgate.LockWallet(h.ctx, lock, common.BytesToAddress(h.wallet)); e != nil {
		t.Fatal(e)
	}
	ctx, cancel := context.WithCancel(h.ctx)
	defer cancel()
	done := make(chan error, 1)
	joined := false
	go func() { done <- h.composition.run(ctx) }()
	defer func() {
		if joined {
			return
		}
		cancel()
		select {
		case e := <-done:
			if e != nil {
				t.Error(e)
			}
		case <-time.After(7 * time.Second):
			t.Error("runtime did not join")
		}
	}()
	var writer wireWriter
	select {
	case writer = <-ready:
	case <-time.After(time.Second):
		t.Fatal("no session")
	}
	parsed, e := ethabi.JSON(bytes.NewReader(sourceabi.CoreExchange))
	if e != nil {
		t.Fatal(e)
	}
	raw := et.Log{Address: common.HexToAddress("0xe111180000d2663c0091e4f400237545b87b996b"), Topics: []common.Hash{parsed.Events["OrderFilled"].ID, {}, common.BytesToHash(h.wallet), {}}, Data: []byte{}, BlockNumber: 1, BlockHash: common.HexToHash("0xaa"), TxHash: common.HexToHash("0xbb")}
	for i := 0; i < 4; i++ {
		raw.Index = uint(i)
		writer.mu.Lock()
		e = writer.conn.WriteJSON(map[string]any{"jsonrpc": "2.0", "method": "eth_subscription", "params": map[string]any{"subscription": "fixture", "result": raw}})
		writer.mu.Unlock()
		if e != nil {
			t.Fatal(e)
		}
	}
	deadline := time.NewTimer(2 * time.Second)
	defer deadline.Stop()
	tick := time.NewTicker(10 * time.Millisecond)
	defer tick.Stop()
	var active app.TraderSyncRuntimeStatus
	for {
		active = runtimeStatus()
		depth, inflight := find(active, "raw_queue_depth"), find(active, "raw_persist_in_flight")
		if depth != nil && inflight != nil && depth.Value == "3" && inflight.Value == "1" {
			for _, v := range []*app.TraderSyncRuntimeMetric{depth, inflight} {
				if v.Kind != "gauge" || v.Unit != "raw_logs" || v.ServiceEpoch == nil || *v.ServiceEpoch != active.CollectorEpoch {
					t.Fatal("raw gauge scope", v)
				}
			}
			if find(active, "raw_observation_available").Value != "1" {
				t.Fatal("active observation unavailable")
			}
			break
		}
		select {
		case <-deadline.C:
			t.Fatalf("real blocked Persist/queued logs absent: %+v", active)
		case <-tick.C:
		}
	}
	// Deliberately change only the durable read-side epoch while the old local
	// receiver is still blocked. Both epochs are present; a boolean connected
	// check alone must not expose the previous session's raw gauges.
	var otherEpoch int64
	if e = h.db.Pool.QueryRow(h.ctx, `INSERT INTO trader_sync_collector_epochs(fencing_token) SELECT fencing_token FROM trader_sync_collector_control WHERE singleton RETURNING id`).Scan(&otherEpoch); e != nil {
		t.Fatal(e)
	}
	if _, e = h.db.Pool.Exec(h.ctx, `UPDATE trader_sync_collector_control SET active_epoch=$1 WHERE singleton`, otherEpoch); e != nil {
		t.Fatal(e)
	}
	mismatch := runtimeStatus()
	if !mismatch.CollectorConnected || mismatch.CollectorEpoch == active.CollectorEpoch {
		t.Fatal("epoch mismatch fixture invalid", mismatch)
	}
	unavailable(mismatch)
	if _, e = h.db.Pool.Exec(h.ctx, `UPDATE trader_sync_collector_control SET active_epoch=$1 WHERE singleton`, active.CollectorEpoch); e != nil {
		t.Fatal(e)
	}
	cancel()
	select {
	case e := <-done:
		joined = true
		if e != nil {
			t.Fatal(e)
		}
	case <-time.After(7 * time.Second):
		t.Fatal("runtime did not join")
	}
	unavailable(runtimeStatus())
}
