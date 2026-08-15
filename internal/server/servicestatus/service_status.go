package servicestatus

import (
	"context"
	"fmt"
	"net"
	"strings"
	"sync"
	"time"

	etherscangatewaypkg "github.com/useryege/athena/pkg/apiclient/etherscangateway"
	servicestatuspkg "github.com/useryege/athena/pkg/apiclient/servicestatus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

const healthCheckTimeout = time.Second
const etherscanGatewayStatusTimeout = 2 * time.Second
const etherscanGatewayDefaultPort = "6776"
const etherscanGatewayAuthorizationMetadataKey = "authorization"

type healthChecker interface {
	CheckHealth(ctx context.Context) (grpc_health_v1.HealthCheckResponse_ServingStatus, error)
}

type namedHealthChecker struct {
	name    string
	checker healthChecker
}

type Server struct {
	checkers                     []namedHealthChecker
	etherscanGatewayIPs          []string
	etherscanGatewayAuthTokenRaw string
	etherscanAPIKeysRaw          string
	etherscanGatewayProbeAddress string
	probeMu                      sync.Mutex
	latestProbeRun               *servicestatuspkg.EtherscanGatewayProbeRun
}

func NewServer(
	notification healthChecker,
	wallet healthChecker,
	marketRadar healthChecker,
	sportsLive healthChecker,
	sportsHistory healthChecker,
	managedOO healthChecker,
	wormMarkets healthChecker,
	fifaMarketDashboard healthChecker,
	tokenAPI healthChecker,
	etherscanGatewayIPs string,
	etherscanGatewayAuthToken string,
	etherscanAPIKeys string,
	etherscanGatewayProbeAddress string,
) *Server {
	return &Server{
		checkers: []namedHealthChecker{
			{name: "notification", checker: notification},
			{name: "wallet", checker: wallet},
			{name: "market-radar", checker: marketRadar},
			{name: "sports-live", checker: sportsLive},
			{name: "sports-history", checker: sportsHistory},
			{name: "managed-oo", checker: managedOO},
			{name: "worm-markets", checker: wormMarkets},
			{name: "fifa-market-dashboard", checker: fifaMarketDashboard},
			{name: "token-api", checker: tokenAPI},
		},
		etherscanGatewayIPs:          normalizeList(etherscanGatewayIPs),
		etherscanGatewayAuthTokenRaw: etherscanGatewayAuthToken,
		etherscanAPIKeysRaw:          etherscanAPIKeys,
		etherscanGatewayProbeAddress: strings.TrimSpace(etherscanGatewayProbeAddress),
	}
}

func (s *Server) ListServiceStatuses(ctx context.Context, _ *servicestatuspkg.ListServiceStatusesRequest) (*servicestatuspkg.ListServiceStatusesResponse, error) {
	items := make([]*servicestatuspkg.ServiceStatus, len(s.checkers))
	var wg sync.WaitGroup

	for i, checker := range s.checkers {
		wg.Add(1)
		go func(index int, target namedHealthChecker) {
			defer wg.Done()

			checkCtx, cancel := context.WithTimeout(ctx, healthCheckTimeout)
			defer cancel()

			status, err := target.checker.CheckHealth(checkCtx)
			item := &servicestatuspkg.ServiceStatus{
				Name:   target.name,
				Status: status.String(),
			}
			if err != nil {
				item.Status = "UNREACHABLE"
				item.ErrorMessage = err.Error()
			}
			items[index] = item
		}(i, checker)
	}

	wg.Wait()
	return &servicestatuspkg.ListServiceStatusesResponse{
		Items:     items,
		CheckedAt: time.Now().Unix(),
	}, nil
}

func (s *Server) ListEtherscanGatewayStatuses(
	ctx context.Context,
	_ *servicestatuspkg.ListEtherscanGatewayStatusesRequest,
) (*servicestatuspkg.ListEtherscanGatewayStatusesResponse, error) {
	items := make([]*servicestatuspkg.EtherscanGatewayStatus, len(s.etherscanGatewayIPs))
	var wg sync.WaitGroup

	for i, ip := range s.etherscanGatewayIPs {
		wg.Add(1)
		go func(index int, gatewayIP string) {
			defer wg.Done()
			items[index] = s.checkEtherscanGatewayStatus(ctx, gatewayIP)
		}(i, ip)
	}

	wg.Wait()
	return &servicestatuspkg.ListEtherscanGatewayStatusesResponse{
		Items:     items,
		CheckedAt: time.Now().Unix(),
	}, nil
}

func (s *Server) checkEtherscanGatewayStatus(ctx context.Context, gatewayIP string) *servicestatuspkg.EtherscanGatewayStatus {
	now := time.Now()
	item := &servicestatuspkg.EtherscanGatewayStatus{
		Address:   net.JoinHostPort(gatewayIP, etherscanGatewayDefaultPort),
		Status:    "checking",
		CheckedAt: now.Unix(),
	}

	if net.ParseIP(gatewayIP) == nil {
		item.Address = gatewayIP
		item.Status = "configuration_error"
		item.ErrorMessage = fmt.Sprintf("ETHERSCAN_GATEWAY_IPS entry %q must be a pure IP address", gatewayIP)
		return item
	}

	authToken := strings.TrimSpace(s.etherscanGatewayAuthTokenRaw)
	if authToken == "" {
		item.Status = "configuration_error"
		item.ErrorMessage = "ATHENA_ETHERSCAN_GATEWAY_AUTH_TOKEN is required"
		return item
	}

	checkCtx, cancel := context.WithTimeout(ctx, etherscanGatewayStatusTimeout)
	defer cancel()
	checkCtx = metadata.AppendToOutgoingContext(
		checkCtx,
		etherscanGatewayAuthorizationMetadataKey,
		"Bearer "+authToken,
	)

	start := time.Now()
	conn, err := grpc.NewClient(item.Address, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		item.Status = "configuration_error"
		item.ErrorMessage = err.Error()
		return item
	}
	defer conn.Close()

	response, err := etherscangatewaypkg.NewEtherscanGatewayServiceClient(conn).GetEtherscanGatewayStatus(
		checkCtx,
		&etherscangatewaypkg.GetEtherscanGatewayStatusRequest{},
	)
	item.LatencyMs = time.Since(start).Milliseconds()
	item.CheckedAt = time.Now().Unix()
	if err != nil {
		applyEtherscanGatewayError(item, err)
		return item
	}

	item.Reachable = true
	item.Started = response.GetStarted()
	item.Status = response.GetStatus()
	item.EtherscanBaseUrl = response.GetEtherscanBaseUrl()
	return item
}

func applyEtherscanGatewayError(item *servicestatuspkg.EtherscanGatewayStatus, err error) {
	item.ErrorMessage = err.Error()
	st, ok := status.FromError(err)
	if !ok {
		item.Status = "error"
		return
	}

	switch st.Code() {
	case codes.Unauthenticated, codes.PermissionDenied:
		item.Reachable = true
		item.Status = "authentication_failed"
	case codes.FailedPrecondition, codes.InvalidArgument:
		item.Reachable = true
		item.Status = "configuration_error"
	case codes.DeadlineExceeded, codes.Unavailable:
		item.Status = "unreachable"
	default:
		item.Reachable = true
		item.Status = "error"
	}
}

func normalizeList(value string) []string {
	value = strings.Trim(strings.TrimSpace(value), `'"`)
	rawItems := strings.FieldsFunc(value, func(r rune) bool {
		return r == ',' || r == '\n' || r == '\r' || r == '\t' || r == ' '
	})
	items := make([]string, 0, len(rawItems))
	for _, item := range rawItems {
		item = strings.Trim(strings.TrimSpace(item), `'"`)
		if item != "" {
			items = append(items, item)
		}
	}
	return items
}
