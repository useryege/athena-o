package apiclient

import (
	"context"
	"fmt"
	"strings"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Config struct {
	Address         string
	Token           string
	MaxMessageBytes int
}
type serviceCredential struct{ token string }

func (serviceCredential) RequireTransportSecurity() bool { return false }
func (c serviceCredential) GetRequestMetadata(context.Context, ...string) (map[string]string, error) {
	return map[string]string{"authorization": "Bearer " + c.token}, nil
}
func NewClient(cfg Config) (OperationLogInternalServiceClient, func() error, error) {
	if strings.TrimSpace(cfg.Address) == "" || strings.TrimSpace(cfg.Token) == "" {
		return nil, nil, fmt.Errorf("operation-log client configuration is incomplete")
	}
	if cfg.MaxMessageBytes <= 0 {
		cfg.MaxMessageBytes = 64 << 10
	}
	conn, err := grpc.NewClient(cfg.Address, grpc.WithInsecure(), grpc.WithPerRPCCredentials(serviceCredential{cfg.Token}), grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(cfg.MaxMessageBytes), grpc.MaxCallSendMsgSize(cfg.MaxMessageBytes)))
	if err != nil {
		return nil, nil, err
	}
	return NewOperationLogInternalServiceClient(conn), conn.Close, nil
}
func NewUnavailableClient(reason string) OperationLogInternalServiceClient {
	return unavailableClient{err: status.Error(codes.Unavailable, "operation log "+reason)}
}

type unavailableClient struct{ err error }

func (c unavailableClient) ListOperationLogs(context.Context, *ListOperationLogsRequest, ...grpc.CallOption) (*ListOperationLogsResponse, error) {
	return nil, c.err
}
func (c unavailableClient) GetOperationLog(context.Context, *GetOperationLogRequest, ...grpc.CallOption) (*GetOperationLogResponse, error) {
	return nil, c.err
}
func (c unavailableClient) GetOperationLogRuntimeStatus(context.Context, *GetOperationLogRuntimeStatusRequest, ...grpc.CallOption) (*GetOperationLogRuntimeStatusResponse, error) {
	return nil, c.err
}
