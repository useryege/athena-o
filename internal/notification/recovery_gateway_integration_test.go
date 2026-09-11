//go:build integration

package notification

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/useryege/athena/internal/accountstate/store/migrations"
	"github.com/useryege/athena/internal/notification/apiclient"
	ns "github.com/useryege/athena/internal/notification/store"
	facade "github.com/useryege/athena/internal/server/notification"
	"github.com/useryege/athena/internal/testutil/pgtest"
	api "github.com/useryege/athena/pkg/apiclient/notification"
	gu "github.com/useryege/athena/util/grpc"
	utiltelegram "github.com/useryege/athena/util/telegram"
	"google.golang.org/grpc"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// The two real protobuf hops exercise internal bearer authentication, the actual
// runtime method/facade, and production JSON. Public administrator authorization
// remains the server policy's separate test; this harness does not replace it.
func TestRecoveryRuntimeRealGatewayEvidence(t *testing.T) {
	db := pgtest.New(t, migrations.FS, migrations.Dir)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	store := ns.NewSQLStore(db.Pool)
	probe := &pollingProbe{requests: make(chan utiltelegram.PollUpdatesRequest, 8), updates: make(chan []utiltelegram.Update)}
	poller := NewTelegramPoller(store, probe)
	token := strings.Repeat("r", 32)
	internal, e := NewServer(ServerOpts{Store: store, Sender: NewTelegramSender(probe, nil), ProfileSyncer: recoveryProfile{}, Poller: poller, InternalAuthToken: token, SiteURL: "https://athena.test"})
	if e != nil {
		t.Fatal(e)
	}
	service := internal.service
	lis, e := net.Listen("tcp", "127.0.0.1:0")
	if e != nil {
		t.Fatal(e)
	}
	ig := internal.CreateGRPC()
	go ig.Serve(lis)
	t.Cleanup(ig.Stop)
	clients, e := apiclient.NewNotificationClientset(lis.Addr().String(), token)
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() { clients.Close() })
	publicLis, e := net.Listen("tcp", "127.0.0.1:0")
	if e != nil {
		t.Fatal(e)
	}
	public := grpc.NewServer()
	api.RegisterNotificationServiceServer(public, facade.NewServer(clients))
	go public.Serve(publicLis)
	t.Cleanup(public.Stop)
	conn, e := grpc.Dial(publicLis.Addr().String(), grpc.WithInsecure())
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() { conn.Close() })
	mux := runtime.NewServeMux(runtime.WithMarshalerOption(runtime.MIMEWildcard, new(gu.JSONMarshaler)))
	if e = api.RegisterNotificationServiceHandler(ctx, mux, conn); e != nil {
		t.Fatal(e)
	}
	httpServer := httptest.NewServer(mux)
	t.Cleanup(httpServer.Close)
	capture := func(name, top, state, remaining, elapsed string) {
		t.Helper()
		response, e := httpServer.Client().Get(httpServer.URL + "/api/v1/admin/notification-runtime/status")
		if e != nil {
			t.Fatal(e)
		}
		raw, e := io.ReadAll(response.Body)
		response.Body.Close()
		if e != nil || response.StatusCode != http.StatusOK {
			t.Fatal(response.Status, string(raw), e)
		}
		var body map[string]json.RawMessage
		if e = json.Unmarshal(raw, &body); e != nil {
			t.Fatal(e)
		}
		if string(body["status"]) != fmt.Sprintf("%q", top) {
			t.Fatalf("%s top: %s", name, raw)
		}
		if state == "" {
			if _, ok := body["recovery"]; ok {
				t.Fatalf("invented recovery: %s", raw)
			}
		} else {
			var recovery map[string]json.RawMessage
			if e = json.Unmarshal(body["recovery"], &recovery); e != nil {
				t.Fatal(e)
			}
			for key, want := range map[string]string{"state": state, "clockSource": "sender_monotonic", "elapsedMillis": elapsed} {
				if string(recovery[key]) != fmt.Sprintf("%q", want) {
					t.Fatalf("%s %s: %s", name, key, raw)
				}
			}
			if remaining == "" {
				if _, ok := recovery["remainingMillis"]; ok {
					t.Fatalf("unknown remaining serialized as known: %s", raw)
				}
			} else if string(recovery["remainingMillis"]) != fmt.Sprintf("%q", remaining) {
				t.Fatalf("%s remaining: %s", name, raw)
			}
			if _, ok := recovery["startedAt"]; !ok {
				t.Fatal("missing real entry UTC", string(raw))
			}
		}
		if dir := os.Getenv("ATHENA_TASK13_GATEWAY_FIXTURE_DIR"); dir != "" {
			if e = os.MkdirAll(dir, 0755); e != nil {
				t.Fatal(e)
			}
			if e = os.WriteFile(filepath.Join(dir, name+".json"), raw, 0644); e != nil {
				t.Fatal(e)
			}
		}
		t.Logf("%s %s", name, raw)
	}
	capture("notification-01-not-entered", "stopped", "", "", "")
	clock := &manualDispatchClock{now: time.Now()}
	session, e := store.AcquireSender(ctx, uuid.New())
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(session.Close)
	service.startStopMu.Lock()
	service.started = true
	service.startStopMu.Unlock()
	service.recovery.begin(clock)
	capture("notification-02-initializing", "recovering", "initializing", "", "0")
	if _, e = recoverStartupBudget(ctx, store, NewBudget(20, time.Second, 20, time.Minute), clock, session.FirstStart(), &service.recovery); e != nil {
		t.Fatal(e)
	}
	if e = poller.Start(ctx); e != nil {
		t.Fatal(e)
	}
	t.Cleanup(poller.Stop)
	nextPoll(t, probe)
	capture("notification-03-first-completed", "running", "completed", "0", "0")
	if e = session.Finish(ctx); e != nil {
		t.Fatal(e)
	}
	session.Close()
	next, e := store.AcquireSender(ctx, uuid.New())
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(next.Close)
	service.recovery.begin(clock)
	waitCtx, stop := context.WithCancel(ctx)
	defer stop()
	done := make(chan error, 1)
	go func() {
		_, e := recoverStartupBudget(waitCtx, store, NewBudget(20, time.Second, 20, time.Minute), clock, next.FirstStart(), &service.recovery)
		done <- e
	}()
	waitClockTimer(t, clock)
	clock.advance(5 * time.Second)
	capture("notification-04-waiting", "recovering", "waiting", "55000", "5000")
	stop()
	select {
	case e = <-done:
		if e != context.Canceled {
			t.Fatal(e)
		}
	case <-ctx.Done():
		t.Fatal("recovery did not join")
	}
	capture("notification-05-cancelled", "degraded", "cancelled", "55000", "5000")
	service.runtimeFailed.Store(true)
	capture("notification-06-fatal", "failed", "cancelled", "55000", "5000")
	service.runtimeFailed.Store(false)
	service.startStopMu.Lock()
	service.started = false
	service.startStopMu.Unlock()
	capture("notification-07-stopped", "stopped", "cancelled", "55000", "5000")
	service.recovery.begin(clock)
	service.recovery.finish("failed", "retry_after_read_failed")
	service.startStopMu.Lock()
	service.started = true
	service.startStopMu.Unlock()
	capture("notification-08-recovery-failed", "failed", "failed", "", "0")
	if e = next.Finish(ctx); e != nil {
		t.Fatal(e)
	}
}
