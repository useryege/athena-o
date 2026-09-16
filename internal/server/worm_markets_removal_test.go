package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	serverservicestatus "github.com/useryege/athena/internal/server/servicestatus"
	servicestatuspkg "github.com/useryege/athena/pkg/apiclient/servicestatus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
)

type servingHealthChecker struct{}

func (servingHealthChecker) CheckHealth(context.Context) (grpc_health_v1.HealthCheckResponse_ServingStatus, error) {
	return grpc_health_v1.HealthCheckResponse_SERVING, nil
}

func TestWormMarketsPublicAPIIsRemovedAndTradingRemains(t *testing.T) {
	s := &AthenaServer{serviceSet: &AthenaServiceSet{HealthService: health.NewServer()}, log: log.NewEntry(log.New())}
	g := s.newGRPCServer()
	defer g.Stop()
	require.NotContains(t, g.GetServiceInfo(), "wormmarkets.WormMarketsService")
	require.Contains(t, g.GetServiceInfo(), "wormtrading.WormTradingService")

	conn, err := grpc.NewClient("127.0.0.1:1", grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err)
	defer conn.Close()
	s.StaticAssetsDir = t.TempDir()
	h := s.newHTTPServer(context.Background(), 0, http.NotFoundHandler(), conn).Handler
	for _, path := range []string{
		"/api/v1/worm-markets/status",
		"/api/v1/worm-markets/events/11111111111111111111111111111111",
		"/api/v1/worm-markets/events",
	} {
		response := httptest.NewRecorder()
		h.ServeHTTP(response, httptest.NewRequest(http.MethodGet, path, nil))
		require.Equal(t, http.StatusNotFound, response.Code, path)
	}
	response := httptest.NewRecorder()
	h.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/v1/worm-trading/wallet-balances", nil))
	require.NotEqual(t, http.StatusNotFound, response.Code)
}

func TestWormMarketsServiceStatusIsRemovedAndTradingRemains(t *testing.T) {
	checker := servingHealthChecker{}
	statusServer := serverservicestatus.NewServer(checker, checker, checker, checker, checker, checker, checker, "", "", "", "")
	response, err := statusServer.ListServiceStatuses(context.Background(), &servicestatuspkg.ListServiceStatusesRequest{})
	require.NoError(t, err)
	var names []string
	for _, item := range response.GetItems() {
		names = append(names, item.GetName())
	}
	require.NotContains(t, names, "worm-markets")
	require.Contains(t, names, "worm-trading")
}
