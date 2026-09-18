package apiclient

import (
	"context"
	"fmt"
	"strings"

	"github.com/useryege/athena/internal/operationlog/rpcconfig"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Config struct {
	Address, Token, Transport, CAFile, ServerName string
	MaxMessageBytes                               int
}
type serviceCredential struct {
	token  string
	secure bool
}

func (c serviceCredential) RequireTransportSecurity() bool { return c.secure }
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
	rc := rpcconfig.Client{Address: cfg.Address, Token: cfg.Token, Transport: cfg.Transport, CAFile: cfg.CAFile, ServerName: cfg.ServerName, MaxMessageBytes: cfg.MaxMessageBytes}
	if rc.Transport == "" {
		rc.Transport = "plaintext"
	}
	if err := rc.Validate(); err != nil {
		return nil, nil, err
	}
	creds, err := rpcconfig.ClientCredentials(rc)
	if err != nil {
		return nil, nil, err
	}
	conn, err := grpc.NewClient(cfg.Address, grpc.WithTransportCredentials(creds), grpc.WithPerRPCCredentials(serviceCredential{token: cfg.Token, secure: rc.Transport == "tls"}), grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(cfg.MaxMessageBytes), grpc.MaxCallSendMsgSize(cfg.MaxMessageBytes)))
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
