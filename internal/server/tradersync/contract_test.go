package tradersync

import (
	"context"
	"encoding/json"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	api "github.com/useryege/athena/pkg/apiclient/tradersync"
	app "github.com/useryege/athena/pkg/apis/application/v1alpha1"
	gu "github.com/useryege/athena/util/grpc"
	"google.golang.org/grpc"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

// This harness tests protobuf + the production JSON marshaler, not HandlerServer.
// The application's actual authentication policy is covered separately.
type contractServer struct {
	api.UnimplementedTraderSyncServiceServer
	mu      sync.Mutex
	creates []*api.CreateSubscriptionRequest
}

func (s *contractServer) CreateSubscription(ctx context.Context, r *api.CreateSubscriptionRequest) (*api.CreateSubscriptionResponse, error) {
	s.mu.Lock()
	s.creates = append(s.creates, r)
	s.mu.Unlock()
	return &api.CreateSubscriptionResponse{Subscription: &app.TraderSyncSubscription{ID: "subscription", Revision: 9007199254740993}}, nil
}
func (s *contractServer) ListActivities(ctx context.Context, r *api.ListActivitiesRequest) (*api.ListActivitiesResponse, error) {
	zero := "0"
	no := false
	_ = no
	return &api.ListActivitiesResponse{Activities: []*app.TraderSyncActivity{{ID: "9007199254740993", PositionID: "90071992547409931234567890", CollateralRaw: zero, SourceLocation: app.TraderSyncSourceLocation{ChainID: "137", LogIndex: "0"}, TargetDisplaySnapshot: app.TraderSyncTargetDisplay{DisplayName: app.TraderSyncStringField{Evidence: app.TraderSyncFieldEvidence{Availability: "available"}, Value: &zero}}}}, Page: &api.ActivityPageInfo{NextCursor: "next", RefreshCursor: "refresh", Snapshot: "snapshot", AsOf: "2026-09-11T00:00:00Z", HasNewer: true}}, nil
}
func contractGateway(t *testing.T, s api.TraderSyncServiceServer) *httptest.Server {
	t.Helper()
	lis, e := net.Listen("tcp", "127.0.0.1:0")
	if e != nil {
		t.Fatal(e)
	}
	g := grpc.NewServer()
	api.RegisterTraderSyncServiceServer(g, s)
	go g.Serve(lis)
	t.Cleanup(g.Stop)
	conn, e := grpc.Dial(lis.Addr().String(), grpc.WithInsecure())
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() { conn.Close() })
	mux := runtime.NewServeMux(runtime.WithMarshalerOption(runtime.MIMEWildcard, new(gu.JSONMarshaler)))
	if e = api.RegisterTraderSyncServiceHandler(context.Background(), mux, conn); e != nil {
		t.Fatal(e)
	}
	h := httptest.NewServer(mux)
	t.Cleanup(h.Close)
	return h
}
func TestGatewayCamelCaseAndPresence(t *testing.T) {
	s := &contractServer{}
	h := contractGateway(t, s)
	for _, body := range []string{`{"confirmationToken":"x","requestId":"a"}`, `{"confirmationToken":"x","requestId":"b","note":{"value":""}}`, `{"confirmationToken":"x","requestId":"c","note":null}`} {
		resp, e := h.Client().Post(h.URL+"/api/v1/trader-sync/subscriptions", "application/json", strings.NewReader(body))
		if e != nil {
			t.Fatal(e)
		}
		raw, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Fatal(resp.Status, string(raw))
		}
		if !strings.Contains(string(raw), `"revision":"9007199254740993"`) {
			t.Fatal("revision lost exact string", string(raw))
		}
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.creates) != 3 || s.creates[0].ConfirmationToken != "x" || s.creates[0].RequestId != "a" || s.creates[0].Note != nil || s.creates[1].Note == nil || s.creates[1].Note.Value != "" || s.creates[2].Note != nil {
		t.Fatalf("camelCase or wrapper presence lost: %+v", s.creates)
	}
	resp, e := h.Client().Get(h.URL + "/api/v1/trader-sync/activities?page.page_size=50")
	if e != nil {
		t.Fatal(e)
	}
	defer resp.Body.Close()
	var body map[string]json.RawMessage
	if e = json.NewDecoder(resp.Body).Decode(&body); e != nil {
		t.Fatal(e)
	}
	if !strings.Contains(string(body["activities"]), `"positionId":"90071992547409931234567890"`) {
		t.Fatal("position string changed", string(body["activities"]))
	}
	var page map[string]any
	if e = json.Unmarshal(body["page"], &page); e != nil {
		t.Fatal(e)
	}
	if page["refreshCursor"] != "refresh" || page["hasNewer"] != true || page["asOf"] != "2026-09-11T00:00:00Z" {
		t.Fatal("page camelCase lost", page)
	}
}
