package application

import (
	"context"

	"github.com/ethereum/go-ethereum/ethclient"
	applicationpkg "github.com/useryege/athena/internal/application/apiclient"
	"github.com/useryege/athena/internal/server/version"
	versionpkg "github.com/useryege/athena/pkg/apiclient/version"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"

	log "github.com/sirupsen/logrus"
)

type ApplicationServer struct {
	ApplicationServerOpts
	service *Service
}

type ApplicationServerOpts struct {
	NodeClient *ethclient.Client
}

func NewServer(opts ApplicationServerOpts) *ApplicationServer {
	return &ApplicationServer{
		ApplicationServerOpts: opts,
		service:               NewService(opts.NodeClient),
	}
}

// CreateGRPC creates a new gRPC server.
func (a *ApplicationServer) CreateGRPC() *grpc.Server {
	server := grpc.NewServer(grpc.MaxRecvMsgSize(applicationpkg.MaxGRPCMessageSize))

	// register the version service to the gRPC server
	versionService := version.NewServer(nil, func() (bool, error) {
		return true, nil
	})
	versionpkg.RegisterVersionServiceServer(server, versionService)

	// register the project controller service to the gRPC server
	applicationpkg.RegisterApplicationServiceServer(server, a.service)

	// register the health service to the gRPC server
	healthService := health.NewServer()
	grpc_health_v1.RegisterHealthServer(server, healthService)
	return server
}

func (a *ApplicationServer) StartBackgroundServices() error {
	if err := a.service.chainwatcher.Start(); err != nil {
		return err
	}
	return nil
}

func (a *ApplicationServer) Shutdown(ctx context.Context) {
	err := a.service.chainwatcher.Stop()
	if err != nil {
		log.Errorf("failed to stop chainwatcher: %v", err)
	}
}
