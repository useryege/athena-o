package server

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/useryege/athena/internal/operationlog/event"
	"github.com/useryege/athena/internal/operationlog/ingest"
)

func TestOperationLogHTTPRouteTableCoversCataloguedPositions(t *testing.T) {
	var expected int
	for _, entry := range event.Catalog() {
		if entry.Transport == "http" && entry.HTTPMethod != "" && entry.Route != "" {
			expected++
		}
	}
	if len(operationLogHTTPRoutes) != expected {
		t.Fatalf("route table entries=%d catalog HTTP entries=%d", len(operationLogHTTPRoutes), expected)
	}
	for _, route := range operationLogHTTPRoutes {
		path := strings.TrimSpace(strings.Split(route.entry.Route, " (")[0])
		path = strings.NewReplacer(
			"{id}", "00000000-0000-4000-8000-000000000001",
			"{runId}", "00000000-0000-4000-8000-000000000002",
			"{stepId}", "00000000-0000-4000-8000-000000000003",
			"{cashOutId}", "00000000-0000-4000-8000-000000000004",
			"{batchId}", "00000000-0000-4000-8000-000000000005",
			"{combinationId}", "00000000-0000-4000-8000-000000000006",
			"{planId}", "00000000-0000-4000-8000-000000000007",
			"{walletId}", "1",
		).Replace(path)
		request := httptest.NewRequest(route.entry.HTTPMethod, path, nil)
		if strings.Contains(route.entry.Route, "wallet.reveal.authorize") {
			request.AddCookie(&http.Cookie{Name: "athena.wallet-secret.google.state", Value: "state"})
		}
		if _, _, ok := operationLogHTTPRouteForRequest(request); !ok {
			t.Errorf("catalogued HTTP entry did not match: %s", route.entry.Entry)
		}
	}
}

func TestOperationLogHTTPCallbackSelectionUsesHandlerPrecedence(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/auth/google/callback?state=wcob.AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA", nil)
	entry, _, ok := operationLogHTTPRouteForRequest(request)
	if !ok || entry.ActionCode != "worm.cash_out_batch.authorize" {
		t.Fatalf("entry=%+v matched=%v", entry, ok)
	}
}

func TestOperationLogHTTPMiddlewareCapturesRouteAndStatusWithoutBody(t *testing.T) {
	sink := &operationLogTestSink{}
	server, cleanup := newOperationLogTestServer(sink)
	defer cleanup()

	handler := server.operationLogHTTPMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Context() == context.Background() {
			t.Fatal("request context was not derived")
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	request := httptest.NewRequest(http.MethodPut, "/api/v1/account/00000000-0000-4000-8000-000000000001/avatar", nil)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusNoContent {
		t.Fatalf("status=%d", response.Code)
	}
	events := sink.snapshot()
	if len(events) != 2 || events[0].Phase != event.Start || events[1].Phase != event.Finish {
		t.Fatalf("events=%+v", events)
	}
	finish := events[1]
	if finish.ActionCode != "account.avatar.upload" || finish.HTTPStatus == nil || *finish.HTTPStatus != http.StatusNoContent || len(finish.Resources) != 0 {
		t.Fatalf("finish=%+v", finish)
	}
	if finish.Outcome != event.Unknown {
		t.Fatalf("transport status must not fabricate business success: %+v", finish)
	}
}

func TestOperationLogHTTPMiddlewareSkipsGRPCWeb(t *testing.T) {
	sink := &operationLogTestSink{}
	server := &AthenaServer{operationLogProducer: ingest.New(sink)}
	defer func() { _ = server.operationLogProducer.Close(context.Background()) }()
	handler := server.operationLogHTTPMiddleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	request := httptest.NewRequest(http.MethodPut, "/api/v1/account/00000000-0000-4000-8000-000000000001/avatar", nil)
	request.Header.Set("Content-Type", "application/grpc-web+proto")
	handler.ServeHTTP(httptest.NewRecorder(), request)
	if events := sink.snapshot(); len(events) != 0 {
		t.Fatalf("grpc-web created native HTTP events: %+v", events)
	}
}

func TestOperationLogHTTPMiddlewareRecordsKnownHTTPFailure(t *testing.T) {
	sink := &operationLogTestSink{}
	server := &AthenaServer{operationLogProducer: ingest.New(sink)}
	defer func() { _ = server.operationLogProducer.Close(context.Background()) }()
	handler := server.operationLogHTTPMiddleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "denied", http.StatusForbidden)
	}))
	request := httptest.NewRequest(http.MethodPost, "/auth/worm-trading/development", nil)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	events := sink.snapshot()
	if len(events) != 2 || events[1].Phase != event.Finish || events[1].Outcome != event.Denied || events[1].HTTPStatus == nil || *events[1].HTTPStatus != http.StatusForbidden {
		t.Fatalf("events=%+v", events)
	}
}

func TestOperationLogHTTPMiddlewareLeavesUnavailableUnknown(t *testing.T) {
	sink := &operationLogTestSink{}
	server := &AthenaServer{operationLogProducer: ingest.New(sink)}
	defer func() { _ = server.operationLogProducer.Close(context.Background()) }()
	handler := server.operationLogHTTPMiddleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "unavailable", http.StatusServiceUnavailable)
	}))
	request := httptest.NewRequest(http.MethodPost, "/auth/worm-trading/development", nil)
	handler.ServeHTTP(httptest.NewRecorder(), request)
	events := sink.snapshot()
	if len(events) != 2 || events[1].Phase != event.Finish || events[1].Outcome != event.Unknown || events[1].HTTPStatus == nil || *events[1].HTTPStatus != http.StatusServiceUnavailable {
		t.Fatalf("events=%+v", events)
	}
}

type operationLogFailingResponseWriter struct {
	header http.Header
}

func (w *operationLogFailingResponseWriter) Header() http.Header { return w.header }
func (w *operationLogFailingResponseWriter) WriteHeader(int)     {}
func (w *operationLogFailingResponseWriter) Write([]byte) (int, error) {
	return 0, errors.New("response closed")
}

func TestOperationLogHTTPMiddlewareObservesResponseWriteFailure(t *testing.T) {
	sink := &operationLogTestSink{}
	server := &AthenaServer{operationLogProducer: ingest.New(sink)}
	defer func() { _ = server.operationLogProducer.Close(context.Background()) }()
	handler := server.operationLogHTTPMiddleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("body"))
	}))
	request := httptest.NewRequest(http.MethodPut, "/api/v1/account/00000000-0000-4000-8000-000000000001/avatar", nil)
	writer := &operationLogFailingResponseWriter{header: make(http.Header)}
	handler.ServeHTTP(writer, request)
	events := sink.snapshot()
	if len(events) != 2 || !events[1].ResponseWriteFailed {
		t.Fatalf("events=%+v", events)
	}
}
