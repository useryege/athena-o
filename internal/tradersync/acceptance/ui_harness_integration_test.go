//go:build integration && uiharness

package acceptance

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	notification "github.com/useryege/athena/internal/notification"
	napiclient "github.com/useryege/athena/internal/notification/apiclient"
	ns "github.com/useryege/athena/internal/notification/store"
	nfacade "github.com/useryege/athena/internal/server/notification"
	traderstore "github.com/useryege/athena/internal/tradersync/store"
	napi "github.com/useryege/athena/pkg/apiclient/notification"
	telegram "github.com/useryege/athena/util/telegram"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/useryege/athena/internal/accountaccess"
	"github.com/useryege/athena/internal/accountcenter"
	"github.com/useryege/athena/internal/accountcredentials"
	ac "github.com/useryege/athena/internal/accountstate/store"
	server "github.com/useryege/athena/internal/server"
	accounts "github.com/useryege/athena/internal/server/account"
	bootstrap "github.com/useryege/athena/internal/server/appbootstrap"
	sessions "github.com/useryege/athena/internal/server/session"
	settingsserver "github.com/useryege/athena/internal/server/settings"
	facade "github.com/useryege/athena/internal/server/tradersync"
	accountapi "github.com/useryege/athena/pkg/apiclient/account"
	bootstrapapi "github.com/useryege/athena/pkg/apiclient/appbootstrap"
	sessionapi "github.com/useryege/athena/pkg/apiclient/session"
	api "github.com/useryege/athena/pkg/apiclient/tradersync"
	httphelper "github.com/useryege/athena/util/http"
	sessionmgr "github.com/useryege/athena/util/session"
	"github.com/useryege/athena/util/settings"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type UIHarnessInfo struct {
	BaseURL, PathPrefix, MemberAState, MemberBState, AdminState string
	ControlURL                                                  string
	Database                                                    string
	Owners                                                      []string
	HTMLHashes                                                  map[string]string
}
type uiRevocations struct {
	mu     sync.Mutex
	values map[string]bool
}

func (s *uiRevocations) Init(context.Context) {}
func (s *uiRevocations) RevokeToken(_ context.Context, id string, _ time.Duration) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.values[id] = true
	return nil
}
func (s *uiRevocations) IsTokenRevoked(id string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.values[id]
}

func (h *harness) StartUI(t *testing.T, distDir string) UIHarnessInfo {
	t.Helper()
	dir := os.Getenv("ATHENA_UI_E2E_DIR")
	if dir == "" {
		t.Fatal("ATHENA_UI_E2E_DIR is required")
	}
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatal(err)
	}
	prefix := strings.TrimSuffix(os.Getenv("ATHENA_UI_E2E_PATH_PREFIX"), "/")
	if prefix != "" && prefix != "/athena" {
		t.Fatal("only root and /athena deployments are supported")
	}
	store := ac.NewSQLStore(h.db.Pool)
	store.SetAccessChangeHook(traderstore.NewSQLStore(h.db.Pool).ApplyAccessChangeTx)
	if err := store.RequireAccessChangeHook(); err != nil {
		t.Fatal(err)
	}
	var identities []accountcredentials.Account
	for i, name := range []string{"browsera", "browserb", "admin"} {
		realm := accountcredentials.ApplicationRealmMember
		if i == 2 {
			realm = accountcredentials.ApplicationRealmAdmin
		}
		identity, _, err := store.RegisterExternalAccount(h.ctx, accountcredentials.IdentityProviderGoogle, "ui-"+name, name+"@test.invalid", name, realm)
		if err != nil {
			t.Fatal(err)
		}
		identities = append(identities, identity)
		if i < 2 {
			if _, err = h.db.Pool.Exec(h.ctx, `UPDATE account_module_access SET access_level='read_write' WHERE account_id=$1 AND module='trader_sync'`, identity.ID); err != nil {
				t.Fatal(err)
			}
			h.ownerIDs = append(h.ownerIDs, identity.ID)
		}
	}
	codec, err := accountcredentials.NewJWTCodec([]byte("task20-isolated-test-session-signing-key-2026"))
	if err != nil {
		t.Fatal(err)
	}
	credentials, err := accountcredentials.NewCredentialManager(h.ctx, store, codec)
	if err != nil {
		t.Fatal(err)
	}
	access, err := accountaccess.NewController(h.ctx, store)
	if err != nil {
		t.Fatal(err)
	}
	center, err := accountcenter.NewManager(store)
	if err != nil {
		t.Fatal(err)
	}
	sm := sessionmgr.NewSessionManager(credentials, codec, &uiRevocations{values: map[string]bool{}}, access)
	adapter, err := server.NewTraderSyncUIHarnessAdapter(credentials, sm, access, distDir, prefix)
	if err != nil {
		t.Fatal(err)
	}
	config, err := settings.NewSettingsManagerFromEnv(h.ctx)
	if err != nil {
		t.Fatal(err)
	}
	if err = h.notifier.Stop(); err != nil {
		t.Fatal(err)
	}
	h.notifier = nil
	telegramClient, err := telegram.NewClient(telegram.Config{BotToken: "acceptance", BaseURL: h.telegramHTTP.URL})
	if err != nil {
		t.Fatal(err)
	}
	nserver, err := notification.NewServer(notification.ServerOpts{Store: ns.NewSQLStore(h.db.Pool), Sender: notification.NewTelegramSender(telegramClient, nil), ProfileSyncer: notification.NewTelegramProfileSyncer(telegramClient, telegram.BotProfileConfig{Name: "Acceptance"}), Poller: notification.NewTelegramPoller(ns.NewSQLStore(h.db.Pool), telegramClient), InternalAuthToken: "task20-loopback-internal-authentication", SiteURL: "https://athena.test"})
	if err != nil {
		t.Fatal(err)
	}
	nl, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	ng := nserver.CreateGRPC()
	ndone := make(chan error, 1)
	go func() { ndone <- ng.Serve(nl) }()
	nc, err := napiclient.NewNotificationClientset(nl.Addr().String(), "task20-loopback-internal-authentication")
	if err != nil {
		t.Fatal(err)
	}
	if err = nserver.Start(h.ctx); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := nserver.Stop(); err != nil {
			t.Error(err)
		}
		nc.Close()
		ng.Stop()
		if err := <-ndone; err != nil {
			t.Error(err)
		}
	})

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	gs := grpc.NewServer(grpc.UnaryInterceptor(adapter.Unary))
	api.RegisterTraderSyncServiceServer(gs, facade.New(h.service))
	napi.RegisterNotificationServiceServer(gs, nfacade.NewServer(nc))
	bootstrapapi.RegisterAppBootstrapServiceServer(gs, bootstrap.NewServer(settingsserver.NewProjector(config), access, center, credentials, adapter.Authenticator))
	sessionapi.RegisterSessionServiceServer(gs, sessions.NewServer(adapter.Authenticator, access, center, credentials))
	accountapi.RegisterAccountServiceServer(gs, accounts.NewServer(credentials, access, center, store))
	done := make(chan error, 1)
	go func() { done <- gs.Serve(listener) }()
	conn, err := grpc.NewClient(listener.Addr().String(), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatal(err)
	}
	for _, register := range []func(context.Context, *grpc.ClientConn) error{
		func(c context.Context, g *grpc.ClientConn) error {
			return napi.RegisterNotificationServiceHandler(c, adapter.Gateway, g)
		},
		func(c context.Context, g *grpc.ClientConn) error {
			return api.RegisterTraderSyncServiceHandler(c, adapter.Gateway, g)
		},
		func(c context.Context, g *grpc.ClientConn) error {
			return bootstrapapi.RegisterAppBootstrapServiceHandler(c, adapter.Gateway, g)
		},
		func(c context.Context, g *grpc.ClientConn) error {
			return sessionapi.RegisterSessionServiceHandler(c, adapter.Gateway, g)
		},
		func(c context.Context, g *grpc.ClientConn) error {
			return accountapi.RegisterAccountServiceHandler(c, adapter.Gateway, g)
		},
	} {
		if err = register(h.ctx, conn); err != nil {
			t.Fatal(err)
		}
	}
	var transportMu sync.Mutex
	dropCreate := false
	holdPath := ""
	heldReady := false
	var release chan struct{}
	mux := http.NewServeMux()
	mux.Handle("/api/", adapter.API)
	mux.Handle("/", adapter.Static)
	var handler http.Handler = mux
	if prefix != "" {
		root := http.NewServeMux()
		root.Handle(prefix+"/", http.StripPrefix(prefix, mux))
		handler = root
	}
	realHandler := handler
	handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		transportMu.Lock()
		held := holdPath != "" && r.URL.Path == prefix+holdPath
		var waitRelease chan struct{}
		if held {
			holdPath = ""
			waitRelease = release
		}
		drop := dropCreate && r.Method == "POST" && r.URL.Path == prefix+"/api/v1/trader-sync/subscriptions"
		if drop {
			dropCreate = false
		}
		transportMu.Unlock()
		if held {
			rec := httptest.NewRecorder()
			realHandler.ServeHTTP(rec, r)
			transportMu.Lock()
			heldReady = true
			transportMu.Unlock()
			select {
			case <-waitRelease:
			case <-h.ctx.Done():
				return
			}
			for k, v := range rec.Header() {
				w.Header()[k] = v
			}
			w.WriteHeader(rec.Code)
			w.Write(rec.Body.Bytes())
			return
		}
		if !drop {
			realHandler.ServeHTTP(w, r)
			return
		}
		rec := httptest.NewRecorder()
		realHandler.ServeHTTP(rec, r)
		t.Logf("UI_CREATE_TRANSPORT dropped=%v status=%d bytes=%d", drop, rec.Code, rec.Body.Len())
		if rec.Code == 200 {
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("Content-Length", strconv.Itoa(rec.Body.Len()))
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("{"))
			if f, ok := w.(http.Flusher); ok {
				f.Flush()
			}
			if hijacker, ok := w.(http.Hijacker); ok {
				conn, _, err := hijacker.Hijack()
				if err == nil {
					conn.Close()
					return
				}
			}
		}
		for k, v := range rec.Header() {
			w.Header()[k] = v
		}
		w.WriteHeader(rec.Code)
		w.Write(rec.Body.Bytes())
	})
	web := httptest.NewServer(handler)
	t.Cleanup(func() {
		web.Close()
		conn.Close()
		gs.Stop()
		if err := <-done; err != nil {
			t.Errorf("UI gRPC join: %v", err)
		}
	})
	info := UIHarnessInfo{BaseURL: web.URL, PathPrefix: prefix, Owners: h.ownerIDs, HTMLHashes: adapter.HTMLHashes}
	if err = h.db.Pool.QueryRow(h.ctx, "SELECT current_database()").Scan(&info.Database); err != nil {
		t.Fatal(err)
	}
	paths := []*string{&info.MemberAState, &info.MemberBState, &info.AdminState}
	for i, identity := range identities {
		realm := identity.ApplicationRealm()
		token, err := sm.CreateExternalLogin(h.ctx, identity.ID, realm, identity.IdentityProvider, identity.IdentitySubject, identity.VerifiedEmail, 7200, uuid.NewString())
		if err != nil {
			t.Fatal(err)
		}
		rec := httptest.NewRecorder()
		if err = httphelper.SetTokenCookie(token, realm, prefix, false, rec); err != nil {
			t.Fatal(err)
		}
		var cookies []map[string]any
		for _, cookie := range rec.Result().Cookies() {
			cookies = append(cookies, map[string]any{"name": cookie.Name, "value": cookie.Value, "domain": "127.0.0.1", "path": cookie.Path, "expires": -1, "httpOnly": cookie.HttpOnly, "secure": cookie.Secure, "sameSite": "Lax"})
		}
		*paths[i] = filepath.Join(dir, fmt.Sprintf("session-%d.json", i))
		writeUIJSON(t, *paths[i], map[string]any{"cookies": cookies, "origins": []any{}})
	}

	controls := http.NewServeMux()
	controls.HandleFunc("POST /hold-read", func(w http.ResponseWriter, r *http.Request) {
		transportMu.Lock()
		defer transportMu.Unlock()
		if release != nil {
			http.Error(w, "hold already armed", 409)
			return
		}
		holdPath = "/api/v1/trader-sync/activities"
		release = make(chan struct{})
		heldReady = false
		w.WriteHeader(204)
	})
	controls.HandleFunc("GET /hold-read", func(w http.ResponseWriter, r *http.Request) {
		transportMu.Lock()
		defer transportMu.Unlock()
		json.NewEncoder(w).Encode(map[string]bool{"ready": heldReady})
	})
	controls.HandleFunc("POST /release-read", func(w http.ResponseWriter, r *http.Request) {
		transportMu.Lock()
		defer transportMu.Unlock()
		if release != nil {
			close(release)
			release = nil
		}
		w.WriteHeader(204)
	})
	t.Cleanup(func() {
		transportMu.Lock()
		if release != nil {
			close(release)
			release = nil
		}
		transportMu.Unlock()
	})

	controls.HandleFunc("POST /drop-create", func(w http.ResponseWriter, r *http.Request) {
		transportMu.Lock()
		dropCreate = true
		transportMu.Unlock()
		w.WriteHeader(204)
	})
	controls.HandleFunc("POST /expire-confirmations", func(w http.ResponseWriter, r *http.Request) {
		_, err := h.db.Pool.Exec(r.Context(), `UPDATE trader_sync_target_confirmations SET expires_at=clock_timestamp()-interval '1 second'`)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		w.WriteHeader(204)
	})
	controls.HandleFunc("POST /telegram-update", func(w http.ResponseWriter, r *http.Request) {
		var input struct {
			Text string
			Chat int64
		}
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil || input.Chat < 1000 || input.Chat > 1002 {
			http.Error(w, "invalid loopback bot update", 400)
			return
		}
		select {
		case h.telegramUpdates <- []any{map[string]any{"update_id": time.Now().UnixNano() / 1000, "message": map[string]any{"message_id": 1, "date": time.Now().Unix(), "chat": map[string]any{"id": input.Chat, "type": "private"}, "from": map[string]any{"id": input.Chat, "first_name": "Acceptance"}, "text": input.Text}}}:
			w.WriteHeader(204)
		case <-r.Context().Done():
			return
		}
	})
	controls.HandleFunc("POST /send-fault", func(w http.ResponseWriter, r *http.Request) {
		var input struct{ Fault string }
		json.NewDecoder(r.Body).Decode(&input)
		if input.Fault != "" && input.Fault != "failed" && input.Fault != "timeout" {
			http.Error(w, "unknown fault", 400)
			return
		}
		h.mu.Lock()
		h.sendFault = input.Fault
		h.mu.Unlock()
		w.WriteHeader(204)
	})

	controls.HandleFunc("POST /push", func(w http.ResponseWriter, r *http.Request) {
		var input struct {
			Wallet string
			Count  int
		}
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			http.Error(w, err.Error(), 400)
			return
		}
		if !common.IsHexAddress(input.Wallet) || input.Count < 1 || input.Count > 150 {
			http.Error(w, "invalid controlled replay", 400)
			return
		}
		deadline := time.Now().Add(15 * time.Second)
		for {
			var ready bool
			err := h.db.Pool.QueryRow(r.Context(), `SELECT count(*)>0 AND bool_and(observation_state='healthy' AND effective_at<=clock_timestamp()) FROM trader_sync_subscriptions WHERE desired_state='enabled' AND wallet=decode($1,'hex')`, strings.TrimPrefix(strings.ToLower(input.Wallet), "0x")).Scan(&ready)
			if err != nil {
				http.Error(w, err.Error(), 500)
				return
			}
			if ready {
				break
			}
			if time.Now().After(deadline) {
				http.Error(w, "real baseline/effective boundary not ready", 409)
				return
			}
			select {
			case <-r.Context().Done():
				return
			case <-time.After(20 * time.Millisecond):
			}
		}
		for i := 0; i < input.Count; i++ {
			h.Push(h.Replay(common.HexToAddress(input.Wallet), i%len(h.fixtures)))
		}
		json.NewEncoder(w).Encode(h.Stats())
	})
	controls.HandleFunc("POST /grant", func(w http.ResponseWriter, r *http.Request) {
		var input struct {
			Owner   string
			Enabled bool
		}
		json.NewDecoder(r.Body).Decode(&input)
		if input.Owner != h.ownerIDs[0] && input.Owner != h.ownerIDs[1] {
			http.Error(w, "unknown test owner", 400)
			return
		}
		next, err := access.Get(input.Owner)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		level := accountaccess.AccessLevelNone
		if input.Enabled {
			level = accountaccess.AccessLevelReadWrite
		}
		next.Modules[accountaccess.ModuleTraderSync] = level
		updated, err := access.Update(r.Context(), input.Owner, next, next.Revision)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		json.NewEncoder(w).Encode(updated)
	})
	controls.HandleFunc("GET /snapshot", func(w http.ResponseWriter, r *http.Request) { json.NewEncoder(w).Encode(h.Stats()) })
	controls.HandleFunc("GET /sql", func(w http.ResponseWriter, r *http.Request) {
		rows, err := h.db.Pool.Query(r.Context(), `SELECT json_build_object('id',id,'accountId',owner_id,'wallet','0x'||encode(wallet,'hex'),'desiredState',desired_state,'observationState',observation_state,'generation',activation_generation) FROM trader_sync_subscriptions ORDER BY created_at`)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		defer rows.Close()
		values := []json.RawMessage{}
		for rows.Next() {
			var b []byte
			if err := rows.Scan(&b); err != nil {
				http.Error(w, err.Error(), 500)
				return
			}
			values = append(values, b)
		}
		json.NewEncoder(w).Encode(values)
	})
	controls.HandleFunc("GET /admin-sql", func(w http.ResponseWriter, r *http.Request) {
		const query = `WITH links AS (
          SELECT a.subscription_id,d.id,d.status FROM trader_sync_activities a JOIN account_notification_deliveries d ON d.activity_id=a.id
          UNION
          SELECT a.subscription_id,d.id,d.status FROM trader_sync_activities a JOIN trader_sync_summary_part_items i ON i.activity_id=a.id JOIN trader_sync_summary_parts p ON p.id=i.part_id JOIN account_notification_deliveries d ON d.id=p.delivery_id
        ) SELECT json_build_object('subscriptionId',s.id,'activityCount',(SELECT count(*)::text FROM trader_sync_activities a WHERE a.subscription_id=s.id),'deliveryCounts',
          (SELECT json_build_object('total',count(*)::text,'pending',count(*) FILTER(WHERE status='pending')::text,'sending',count(*) FILTER(WHERE status='sending')::text,'sent',count(*) FILTER(WHERE status='sent')::text,'failed',count(*) FILTER(WHERE status='failed')::text,'unknown',count(*) FILTER(WHERE status='unknown')::text,'cancelled',count(*) FILTER(WHERE status='cancelled')::text) FROM links WHERE subscription_id=s.id))
          FROM trader_sync_subscriptions s ORDER BY s.id`
		rows, err := h.db.Pool.Query(r.Context(), query)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		defer rows.Close()
		values := []json.RawMessage{}
		for rows.Next() {
			var b []byte
			if err := rows.Scan(&b); err != nil {
				http.Error(w, err.Error(), 500)
				return
			}
			values = append(values, b)
		}
		if err := rows.Err(); err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		json.NewEncoder(w).Encode(values)
	})
	control := httptest.NewServer(controls)
	t.Cleanup(control.Close)
	info.ControlURL = control.URL
	return info
}
func writeUIJSON(t *testing.T, path string, value any) {
	t.Helper()
	b, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(path, b, 0600); err != nil {
		t.Fatal(err)
	}
}
func TestUIHarness(t *testing.T) {
	for _, key := range []string{"ATHENA_UI_E2E_DIR", "ATHENA_UI_DIST", "ATHENA_TEST_PG_ADMIN_DSN"} {
		if os.Getenv(key) == "" {
			t.Fatalf("%s is required", key)
		}
	}
	h := newHarness(t, 0, 0)
	info := h.StartUI(t, os.Getenv("ATHENA_UI_DIST"))
	dir := os.Getenv("ATHENA_UI_E2E_DIR")
	writeUIJSON(t, filepath.Join(dir, "harness.json"), info)
	t.Logf("UI_READY pid=%d database=%s base=%s prefix=%s stop=%s", os.Getpid(), info.Database, info.BaseURL, info.PathPrefix, filepath.Join(dir, "stop"))
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			if _, err := os.Stat(filepath.Join(dir, "stop")); err == nil {
				return
			}
		case err := <-h.runDone:
			h.runDone = nil
			t.Fatalf("trader runtime ended before stop: %v", err)
		}
	}
}
