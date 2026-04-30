package application

import (
	"github.com/useryege/athena/internal/application/apiclient"
	"github.com/useryege/athena/internal/server/version"
	versionpkg "github.com/useryege/athena/pkg/apiclient/version"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
)

type ProjectControllerServer struct {
	service *Service
}

func NewServer() *ProjectControllerServer {
	return &ProjectControllerServer{
		service: NewService(),
	}
}

// CreateGRPC creates a new gRPC server.
func (a *ProjectControllerServer) CreateGRPC() *grpc.Server {
	server := grpc.NewServer(grpc.MaxRecvMsgSize(apiclient.MaxGRPCMessageSize))

	// register the version service to the gRPC server
	versionService := version.NewServer(nil, func() (bool, error) {
		return true, nil
	})
	versionpkg.RegisterVersionServiceServer(server, versionService)

	// register the project controller service to the gRPC server
	apiclient.RegisterProjectControllerServiceServer(server, a.service)

	// register the health service to the gRPC server
	healthService := health.NewServer()
	grpc_health_v1.RegisterHealthServer(server, healthService)
	return server
}
