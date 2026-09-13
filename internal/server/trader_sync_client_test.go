package server

import (
	"context"
	trpc "github.com/useryege/athena/internal/tradersync/apiclient"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestTraderSyncInvalidClientConfigurationIsLocalFailure(t *testing.T) {
	for _, cfg := range []map[string]string{{}, {"ATHENA_TRADER_SYNC_GRPC_TRANSPORT": "wrong"}, {"ATHENA_TRADER_SYNC_GRPC_TRANSPORT": "loopback-insecure", "ATHENA_TRADER_SYNC_INTERNAL_AUTH_TOKEN": strings.Repeat("t", 32), "ATHENA_TRADER_SYNC_SERVER_ADDRESS": "remote:8122"}} {
		client, closer := newTraderSyncClient(func(key string) (string, bool) { v, ok := cfg[key]; return v, ok })
		if client == nil || closer != nil {
			t.Fatal("invalid configuration must return resource-free unavailable client")
		}
		_, e := client.GetTraderSyncRuntimeStatus(context.Background(), &trpc.GetTraderSyncRuntimeStatusRequest{})
		if status.Code(e) != codes.Unavailable {
			t.Fatal(e)
		}
		s := &AthenaServer{AthenaServerOpts: AthenaServerOpts{TraderSyncClient: client}}
		s.available.Store(true)
		if e = s.healthCheck(httptest.NewRequest("GET", "/healthz", nil)); e != nil {
			t.Fatal("dependency failure killed API", e)
		}
	}
}
func TestTraderSyncClientIgnoresCollectorConfiguration(t *testing.T) {
	cfg := map[string]string{"ATHENA_TRADER_SYNC_GRPC_TRANSPORT": "loopback-insecure", "ATHENA_TRADER_SYNC_INTERNAL_AUTH_TOKEN": strings.Repeat("t", 32), "ATHENA_TRADER_SYNC_SERVER_ADDRESS": "127.0.0.1:1", "ATHENA_TRADER_SYNC_HTTP_URL": "broken", "ATHENA_TRADER_SYNC_WSS_URL": "broken", "ATHENA_TRADER_SYNC_CURSOR_HMAC_KEY": ""}
	client, closer := newTraderSyncClient(func(k string) (string, bool) { v, ok := cfg[k]; return v, ok })
	if client == nil || closer == nil {
		t.Fatal("collector-only config prevented nonblocking API client creation")
	}
	closer.Close()
}

type countingCloser struct{ count int }

func (c *countingCloser) Close() error { c.count++; return nil }
func TestAPICloseOnlyOwnsCreatedTraderSyncConnection(t *testing.T) {
	c := &countingCloser{}
	s := &AthenaServer{traderSyncCloser: c}
	if e := s.Close(); e != nil {
		t.Fatal(e)
	}
	if e := s.Close(); e != nil {
		t.Fatal(e)
	}
	if c.count != 1 {
		t.Fatal("owned connection closed more than once", c.count)
	}
	borrowed := &AthenaServer{AthenaServerOpts: AthenaServerOpts{TraderSyncClient: trpc.NewUnavailableClient("dependency_unavailable")}}
	if e := borrowed.Close(); e != nil {
		t.Fatal(e)
	}
}
