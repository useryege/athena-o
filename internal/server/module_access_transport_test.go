package server

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/useryege/athena/internal/accountcredentials"
	modulehandler "github.com/useryege/athena/internal/server/moduleaccess"
	modulepb "github.com/useryege/athena/pkg/apiclient/moduleaccess"
	utilsession "github.com/useryege/athena/util/session"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/improbable-eng/grpc-web/go/grpcweb"
	"github.com/stretchr/testify/require"
	"github.com/useryege/athena/internal/accountaccess"
	"github.com/useryege/athena/internal/moduleaccess"
	solanapb "github.com/useryege/athena/pkg/apiclient/solana"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
)

type admissionSolana struct {
	solanapb.UnimplementedSolanaServiceServer
	traderAuthIdentity
}

func TestModuleAccessRealGRPCGatewayAndWeb(t *testing.T) {
	access := accountaccess.Access{LoginEnabled: true, Modules: accountaccess.NoModuleAccess(), Revision: 1}
	access.Modules[accountaccess.ModuleSolana] = accountaccess.AccessLevelRead
	controller, err := accountaccess.NewController(context.Background(), traderAuthStore{"member": access})
	require.NoError(t, err)
	s := &AthenaServer{accessController: controller, moduleAccessStore: &moduleAdmissionStore{}}
	s.StaticAssetsDir = t.TempDir()
	g := grpc.NewServer(grpc.UnaryInterceptor(s.unaryAuthInterceptor))
	solanapb.RegisterSolanaServiceServer(g, &admissionSolana{traderAuthIdentity: "member"})
	listener := bufconn.Listen(1024 * 1024)
	go g.Serve(listener)
	t.Cleanup(g.Stop)
	conn, err := grpc.NewClient("passthrough:///test", grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) { return listener.Dial() }), grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err)
	defer conn.Close()
	_, err = solanapb.NewSolanaServiceClient(conn).ListProjects(context.Background(), &solanapb.ListProjectsRequest{})
	require.Equal(t, moduleaccess.ClosedReason, status.Convert(err).Message())
	h := s.newHTTPServer(context.Background(), 0, grpcweb.WrapServer(g), conn).Handler
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest("GET", "http://localhost/api/v1/solana/projects", nil))
	require.Equal(t, 503, w.Code, w.Body.String())
	require.Equal(t, moduleaccess.ClosedReason, w.Header().Get("X-Athena-Error-Reason"))
	require.Contains(t, w.Header().Get("Cache-Control"), "no-store")
	var body map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	require.Contains(t, w.Body.String(), "athena.module_access")
	require.Contains(t, w.Body.String(), "module_key")
	request := httptest.NewRequest("POST", "http://localhost/solana.SolanaService/ListProjects", bytes.NewReader([]byte{0, 0, 0, 0, 0}))
	request.Header.Set("Content-Type", "application/grpc-web+proto")
	request.Header.Set("X-Grpc-Web", "1")
	w = httptest.NewRecorder()
	h.ServeHTTP(w, request)
	raw, err := io.ReadAll(w.Result().Body)
	require.NoError(t, err)
	require.True(t, bytes.Contains(raw, []byte("grpc-status: 14")) || w.Header().Get("Grpc-Status") == "14", "body=%s headers=%v status=%d", raw, w.Header(), w.Code)
	require.Contains(t, string(raw)+w.Header().Get("Grpc-Message"), moduleaccess.ClosedReason)
}

type transportSettingsStore struct {
	moduleaccess.Store
	values []moduleaccess.Setting
	actor  string
}

func (s *transportSettingsStore) ListModuleAccessSettings(context.Context) ([]moduleaccess.Setting, error) {
	return s.values, nil
}
func (s *transportSettingsStore) UpdateModuleAccessSetting(_ context.Context, key moduleaccess.Key, open bool, actor string) (moduleaccess.Setting, error) {
	s.actor = actor
	return moduleaccess.Setting{Key: key, Open: open, UpdatedByAccountID: actor, UpdatedByUsername: "local-admin", UpdatedAt: time.Date(2026, 9, 17, 0, 0, 0, 0, time.UTC)}, nil
}

type transportSettingsService struct {
	*modulehandler.Server
	credential accountcredentials.AuthenticatedCredential
}

func (s *transportSettingsService) AuthFuncOverride(ctx context.Context, _ string) (context.Context, error) {
	ctx = withDisabledAuthClaims(ctx, s.credential.AccountID)
	return utilsession.WithAuthenticatedCredential(ctx, s.credential), nil
}
func TestModuleAccessSettingsRealJSONContractAndAPIKeyWrite(t *testing.T) {
	access := accountaccess.Access{Administrator: true, LoginEnabled: true, Modules: accountaccess.NoModuleAccess(), Revision: 1}
	controller, err := accountaccess.NewController(context.Background(), traderAuthStore{"admin": access})
	require.NoError(t, err)
	store := &transportSettingsStore{}
	for _, key := range moduleaccess.Keys() {
		store.values = append(store.values, moduleaccess.Setting{Key: key})
	}
	s := &AthenaServer{accessController: controller, moduleAccessStore: store}
	s.StaticAssetsDir = t.TempDir()
	service := &transportSettingsService{Server: modulehandler.NewServer(store), credential: accountcredentials.AuthenticatedCredential{AccountID: "admin", JTI: "session", Capability: accountcredentials.CapabilityDevelopment}}
	g := grpc.NewServer(grpc.UnaryInterceptor(s.unaryAuthInterceptor))
	modulepb.RegisterModuleAccessServiceServer(g, service)
	listener := bufconn.Listen(1024 * 1024)
	go g.Serve(listener)
	t.Cleanup(g.Stop)
	conn, err := grpc.NewClient("passthrough:///settings", grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) { return listener.Dial() }), grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err)
	defer conn.Close()
	h := s.newHTTPServer(context.Background(), 0, grpcweb.WrapServer(g), conn).Handler
	request := func(method, path, body string) *httptest.ResponseRecorder {
		r := httptest.NewRequest(method, "http://localhost"+path, strings.NewReader(body))
		r.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		return w
	}
	for path, field := range map[string]string{"/api/v1/module-access-states": "states", "/api/v1/admin/module-access-settings": "settings"} {
		w := request("GET", path, "")
		require.Equal(t, 200, w.Code, w.Body.String())
		require.Contains(t, w.Header().Get("Cache-Control"), "no-store")
		var response map[string]any
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &response))
		rows := response[field].([]any)
		require.Len(t, rows, 6)
		row := rows[0].(map[string]any)
		require.Equal(t, map[string]any{"module_key": "trader_sync", "state": float64(2)}, row)
	}
	w := request("PUT", "/api/v1/admin/module-access-settings/worm", `{"state":1}`)
	require.Equal(t, 200, w.Code, w.Body.String())
	require.JSONEq(t, `{"setting":{"module_key":"worm","state":1,"updated_by_account_id":"admin","updated_by_username":"local-admin","updated_at":"2026-09-17T00:00:00Z"}}`, w.Body.String())
	require.Equal(t, "admin", store.actor)
	service.credential.Capability = accountcredentials.CapabilityAPIKey
	w = request("PUT", "/api/v1/admin/module-access-settings/worm", `{"state":2}`)
	require.Equal(t, 403, w.Code, w.Body.String())
}

// A real HTTP -> gateway -> gRPC call is held only after the production
// interceptor admitted it. Closing admission rejects a later call without
// cancelling work already inside the handler. No trade is submitted.
type acceptedWorkStore struct {
	moduleaccess.Store
	open atomic.Bool
}

func (s *acceptedWorkStore) GetModuleAccessSetting(_ context.Context, key moduleaccess.Key) (moduleaccess.Setting, error) {
	return moduleaccess.Setting{Key: key, Open: s.open.Load()}, nil
}
func (s *acceptedWorkStore) UpdateModuleAccessSetting(_ context.Context, key moduleaccess.Key, open bool, actor string) (moduleaccess.Setting, error) {
	s.open.Store(open)
	return moduleaccess.Setting{Key: key, Open: open, UpdatedByAccountID: actor}, nil
}

type acceptedWorkSolana struct {
	solanapb.UnimplementedSolanaServiceServer
	traderAuthIdentity
	entered chan struct{}
	release chan struct{}
	calls   atomic.Int32
}

func (s *acceptedWorkSolana) ListProjects(ctx context.Context, _ *solanapb.ListProjectsRequest) (*solanapb.ListProjectsResponse, error) {
	s.calls.Add(1)
	close(s.entered)
	select {
	case <-s.release:
		return &solanapb.ListProjectsResponse{}, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}
func TestModuleAccessClosingPreservesWorkAlreadyAdmittedOverHTTP(t *testing.T) {
	access := accountaccess.Access{Administrator: true, LoginEnabled: true, Modules: accountaccess.NoModuleAccess(), Revision: 1}
	member := accountaccess.Access{LoginEnabled: true, Modules: accountaccess.NoModuleAccess(), Revision: 1}
	member.Modules[accountaccess.ModuleSolana] = accountaccess.AccessLevelRead
	controller, err := accountaccess.NewController(context.Background(), traderAuthStore{"admin": access, "member": member})
	require.NoError(t, err)
	store := &acceptedWorkStore{}
	store.open.Store(true)
	server := &AthenaServer{accessController: controller, moduleAccessStore: store}
	server.StaticAssetsDir = t.TempDir()
	business := &acceptedWorkSolana{traderAuthIdentity: "member", entered: make(chan struct{}), release: make(chan struct{})}
	var once sync.Once
	release := func() { once.Do(func() { close(business.release) }) }
	defer release()
	g := grpc.NewServer(grpc.UnaryInterceptor(server.unaryAuthInterceptor))
	solanapb.RegisterSolanaServiceServer(g, business)
	modulepb.RegisterModuleAccessServiceServer(g, &transportSettingsService{Server: modulehandler.NewServer(store), credential: accountcredentials.AuthenticatedCredential{AccountID: "admin", JTI: "session", Capability: accountcredentials.CapabilityDevelopment}})
	listener := bufconn.Listen(1024 * 1024)
	go g.Serve(listener)
	t.Cleanup(g.Stop)
	conn, err := grpc.NewClient("passthrough:///accepted", grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) { return listener.Dial() }), grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err)
	defer conn.Close()
	web := httptest.NewServer(server.newHTTPServer(context.Background(), 0, grpcweb.WrapServer(g), conn).Handler)
	defer web.Close()
	client := &http.Client{Timeout: 5 * time.Second}
	type outcome struct {
		status int
		err    error
	}
	first := make(chan outcome, 1)
	go func() {
		response, err := client.Get(web.URL + "/api/v1/solana/projects")
		if err != nil {
			first <- outcome{err: err}
			return
		}
		defer response.Body.Close()
		_, err = io.Copy(io.Discard, response.Body)
		first <- outcome{response.StatusCode, err}
	}()
	select {
	case <-business.entered:
	case <-time.After(3 * time.Second):
		t.Fatal("first request never passed admission")
	}
	req, err := http.NewRequest("PUT", web.URL+"/api/v1/admin/module-access-settings/solana", strings.NewReader(`{"state":2}`))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	response, err := client.Do(req)
	require.NoError(t, err)
	response.Body.Close()
	require.Equal(t, 200, response.StatusCode)
	response, err = client.Get(web.URL + "/api/v1/solana/projects")
	require.NoError(t, err)
	response.Body.Close()
	require.Equal(t, 503, response.StatusCode)
	require.Equal(t, moduleaccess.ClosedReason, response.Header.Get("X-Athena-Error-Reason"))
	require.Equal(t, int32(1), business.calls.Load())
	select {
	case got := <-first:
		t.Fatalf("accepted work completed before release: %+v", got)
	default:
	}
	release()
	select {
	case got := <-first:
		require.NoError(t, got.err)
		require.Equal(t, 200, got.status)
	case <-time.After(3 * time.Second):
		t.Fatal("accepted work did not finish")
	}
}
