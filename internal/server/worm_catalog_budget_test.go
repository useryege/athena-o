package server

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	trading "github.com/useryege/athena/internal/wormtrading/apiclient"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type budgetCatalogHTTPClient struct {
	trading.WormTradingServiceClient
	invoke func(context.Context) error
}

func (c budgetCatalogHTTPClient) GetOrderEventCatalog(ctx context.Context, _ *trading.GetOrderEventCatalogRequest, _ ...grpc.CallOption) (*trading.GetOrderEventCatalogResponse, error) {
	return nil, c.invoke(ctx)
}
func (c budgetCatalogHTTPClient) CreateMarketCombination(ctx context.Context, _ *trading.CreateMarketCombinationRequest, _ ...grpc.CallOption) (*trading.CreateMarketCombinationResponse, error) {
	return nil, c.invoke(ctx)
}
func (c budgetCatalogHTTPClient) UpdateMarketCombination(ctx context.Context, _ *trading.UpdateMarketCombinationRequest, _ ...grpc.CallOption) (*trading.UpdateMarketCombinationResponse, error) {
	return nil, c.invoke(ctx)
}

func TestWormCatalogHTTPTransportBudget(t *testing.T) {
	for _, method := range []string{http.MethodGet, http.MethodPost, http.MethodPut} {
		for _, mode := range []string{"transport cap", "earlier caller deadline", "caller cancellation"} {
			t.Run(method+"/"+mode, func(t *testing.T) {
				s, _ := newCatalogHTTPServer(t)
				path := "/api/v1/worm-trading/combinations"
				body := `{"name":"Selection","expectedRevision":1,"items":[{"eventConditionId":"` + httpCatalogEventID + `","marketConditionId":"` + httpCatalogEventID + `","side":"YES"}]}`
				if method == http.MethodGet {
					path = "/api/v1/worm-trading/events/" + httpCatalogEventID
					body = ""
				}
				if method == http.MethodPut {
					path += "/965f7c56-5e65-450d-8faf-8bd9b918aeeb"
				}
				if method == http.MethodPost {
					body = strings.Replace(body, `"expectedRevision":1,`, "", 1)
				}
				req := catalogHTTPRequest(method, path, body)
				ctx, cancel := context.WithCancel(req.Context())
				defer cancel()
				if mode == "earlier caller deadline" {
					var deadlineCancel context.CancelFunc
					ctx, deadlineCancel = context.WithTimeout(ctx, 30*time.Millisecond)
					defer deadlineCancel()
				}
				start := time.Now()
				calls := 0
				s.WormTradingClientset = catalogHTTPClientset{client: budgetCatalogHTTPClient{invoke: func(rpcCtx context.Context) error {
					calls++
					deadline, bounded := rpcCtx.Deadline()
					require.True(t, bounded, "RPC must have a finite transport deadline before entering Trading")
					if callerDeadline, ok := ctx.Deadline(); ok {
						require.Equal(t, callerDeadline, deadline)
					} else {
						require.False(t, deadline.After(time.Now().Add(time.Minute)), "transport cap exceeds 60 seconds")
						require.False(t, deadline.Before(start.Add(59*time.Second)), "default transport cap should allow the 45 second service budget")
					}
					md, _ := metadata.FromOutgoingContext(rpcCtx)
					require.Equal(t, []string{httpCatalogAccountID}, md.Get("x-athena-account-id"))
					if mode != "earlier caller deadline" {
						cancel()
					}
					<-rpcCtx.Done()
					return status.FromContextError(rpcCtx.Err()).Err()
				}}}
				mux := http.NewServeMux()
				registerWormCombinationHandlers(mux, s)
				response := httptest.NewRecorder()
				mux.ServeHTTP(response, req.WithContext(ctx))
				require.Equal(t, 1, calls, "write requests must not be retried")
				require.Equal(t, http.StatusInternalServerError, response.Code)
				want := codes.Canceled
				if mode == "earlier caller deadline" {
					want = codes.DeadlineExceeded
				}
				require.Contains(t, response.Body.String(), fmt.Sprintf(`"code":%d`, want))
			})
		}
	}
}
