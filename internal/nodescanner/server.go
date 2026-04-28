package nodescanner

import (
	"github.com/useryege/athena/internal/nodescanner/apiclient"
	"github.com/useryege/athena/internal/server/version"
	versionpkg "github.com/useryege/athena/pkg/apiclient/version"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
)

type NodeScannerServer struct {
	service *Service
}

func NewServer() *NodeScannerServer {
	return &NodeScannerServer{
		service: NewService(),
	}
}

// CreateGRPC creates a new gRPC server.
func (n *NodeScannerServer) CreateGRPC() *grpc.Server {
	server := grpc.NewServer(grpc.MaxRecvMsgSize(apiclient.MaxGRPCMessageSize))

	versionService := version.NewServer(nil, func() (bool, error) {
		return true, nil
	})
	versionpkg.RegisterVersionServiceServer(server, versionService)

	apiclient.RegisterNodeScannerServiceServer(server, n.service)

	healthService := health.NewServer()
	grpc_health_v1.RegisterHealthServer(server, healthService)
	return server
}
