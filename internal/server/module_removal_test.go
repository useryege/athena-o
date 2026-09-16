package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/health"
)

func TestRemovedModuleRegistrations(t *testing.T) {
	s := &AthenaServer{serviceSet: &AthenaServiceSet{HealthService: health.NewServer()}, log: log.NewEntry(log.New())}
	g := s.newGRPCServer()
	defer g.Stop()
	services := g.GetServiceInfo()
	for _, name := range []string{"sportslive.SportsLiveService", "sportshistory.SportsHistoryService", "worldcupcorners.WorldCupCornersService"} {
		require.NotContains(t, services, name)
	}
	for _, name := range []string{"wormmarkets.WormMarketsService", "wormtrading.WormTradingService", "wallet.WalletService", "account.AccountService", "grpc.health.v1.Health"} {
		require.Contains(t, services, name)
	}
	conn, err := grpc.NewClient("127.0.0.1:1", grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err)
	defer conn.Close()
	s.StaticAssetsDir = t.TempDir()
	h := s.newHTTPServer(context.Background(), 0, http.NotFoundHandler(), conn).Handler
	for _, path := range []string{"/api/v1/sports-live/status", "/api/v1/sports-history/status", "/api/v1/world-cup-corners/dataset"} {
		response := httptest.NewRecorder()
		h.ServeHTTP(response, httptest.NewRequest(http.MethodGet, path, nil))
		require.Equal(t, http.StatusNotFound, response.Code, path)
	}
}
