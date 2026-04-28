package blocksniffer

import (
	"github.com/useryege/athena/internal/blocksniffer/apiclient"
	"github.com/useryege/athena/internal/server/version"
	versionpkg "github.com/useryege/athena/pkg/apiclient/version"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
)

type BlockSnifferServer struct {
	service *Service
}

func NewServer() *BlockSnifferServer {
	return &BlockSnifferServer{
		service: NewService(),
	}
}

// CreateGRPC creates a new gRPC server.
func (a *BlockSnifferServer) CreateGRPC() *grpc.Server {
	server := grpc.NewServer(grpc.MaxRecvMsgSize(apiclient.MaxGRPCMessageSize))

	// register the version service to the gRPC server
	versionService := version.NewServer(nil, func() (bool, error) {
		return true, nil
	})
	versionpkg.RegisterVersionServiceServer(server, versionService)

	// register the block sniffer service to the gRPC server
	apiclient.RegisterBlockSnifferServiceServer(server, a.service)

	// register the health service to the gRPC server
	healthService := health.NewServer()
	grpc_health_v1.RegisterHealthServer(server, healthService)
	return server
}
