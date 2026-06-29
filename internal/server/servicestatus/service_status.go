package servicestatus

import (
	"context"
	"sync"
	"time"

	servicestatuspkg "github.com/useryege/athena/pkg/apiclient/servicestatus"
	"google.golang.org/grpc/health/grpc_health_v1"
)

const healthCheckTimeout = time.Second

type healthChecker interface {
	CheckHealth(ctx context.Context) (grpc_health_v1.HealthCheckResponse_ServingStatus, error)
}

type namedHealthChecker struct {
	name    string
	checker healthChecker
}

type Server struct {
	checkers []namedHealthChecker
}

func NewServer(
	notification healthChecker,
	wallet healthChecker,
	worm healthChecker,
	wormPoly healthChecker,
	polymarket healthChecker,
	tokenAPI healthChecker,
) *Server {
	return &Server{checkers: []namedHealthChecker{
		{name: "notification", checker: notification},
		{name: "wallet", checker: wallet},
		{name: "worm", checker: worm},
		{name: "worm-poly", checker: wormPoly},
		{name: "polymarket", checker: polymarket},
		{name: "token-api", checker: tokenAPI},
	}}
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
