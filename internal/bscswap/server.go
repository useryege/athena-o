package bscswap

import (
	"github.com/useryege/athena/internal/bscswap/apiclient"
	commonapiclient "github.com/useryege/athena/pkg/apiclient"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
)

type GRPCServer struct {
	server *grpc.Server
	health *health.Server
}

func NewGRPCServer(service *Service) *GRPCServer {
	healthService := health.NewServer()
	healthService.SetServingStatus("", grpc_health_v1.HealthCheckResponse_NOT_SERVING)
	server := grpc.NewServer(
		grpc.MaxRecvMsgSize(commonapiclient.MaxGRPCMessageSize),
		grpc.MaxSendMsgSize(commonapiclient.MaxGRPCMessageSize),
	)
	apiclient.RegisterBscSwapTransactionServiceServer(server, service)
	grpc_health_v1.RegisterHealthServer(server, healthService)
	return &GRPCServer{server: server, health: healthService}
}

func (s *GRPCServer) Server() *grpc.Server { return s.server }

func (s *GRPCServer) SetServing(serving bool) {
	status := grpc_health_v1.HealthCheckResponse_NOT_SERVING
	if serving {
		status = grpc_health_v1.HealthCheckResponse_SERVING
	}
	s.health.SetServingStatus("", status)
}
