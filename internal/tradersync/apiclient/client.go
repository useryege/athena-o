package apiclient

import (
	"context"
	"io"
	"net"

	"github.com/useryege/athena/internal/tradersync/rpcconfig"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// NewClient allocates an owned, non-blocking connection. The remote service may
// be offline without preventing the API's other modules from starting.
func NewClient(cfg rpcconfig.Client) (TraderSyncServiceClient, io.Closer, error) {
	if err := cfg.Validate(); err != nil {
		return nil, nil, err
	}
	credentials, err := rpcconfig.ClientCredentials(cfg)
	if err != nil {
		return nil, nil, err
	}
	address := cfg.Address
	// A local plaintext target never depends on DNS resolving localhost safely.
	if cfg.Transport == "loopback-insecure" {
		host, port, _ := net.SplitHostPort(address)
		if host == "localhost" {
			address = net.JoinHostPort("127.0.0.1", port)
		}
	}
	connection, err := grpc.NewClient(address,
		grpc.WithTransportCredentials(credentials),
		grpc.WithPerRPCCredentials(serviceCredential{token: cfg.Token, secure: cfg.Transport == "tls"}),
		grpc.WithDisableRetry(),
		grpc.WithDisableServiceConfig(),
		grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(cfg.MaxMessageBytes), grpc.MaxCallSendMsgSize(cfg.MaxMessageBytes)),
		grpc.WithUnaryInterceptor(func(ctx context.Context, method string, req, reply any, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
			budget, err := rpcconfig.MethodBudget(method)
			if err != nil {
				return status.Error(codes.Unimplemented, "unknown Trader Sync method")
			}
			bounded, cancel := context.WithTimeout(ctx, budget)
			defer cancel()
			return invoker(bounded, method, req, reply, cc, opts...)
		}))
	if err != nil {
		return nil, nil, err
	}
	return NewTraderSyncServiceClient(connection), connection, nil
}

type serviceCredential struct {
	token  string
	secure bool
}

func (c serviceCredential) GetRequestMetadata(context.Context, ...string) (map[string]string, error) {
	return map[string]string{"authorization": "Bearer " + c.token}, nil
}
func (c serviceCredential) RequireTransportSecurity() bool { return c.secure }

// NewUnavailableClient reports a dependency failure uniformly for every method,
// without creating a transport or retaining a resource that needs closing.
func NewUnavailableClient(reason string) TraderSyncServiceClient {
	// Only known safe categories may cross this boundary; callers should log their
	// own configuration error separately without putting credentials in a status.
	switch reason {
	case "configuration_invalid", "connection_unavailable", "authentication_invalid":
	default:
		reason = "dependency_unavailable"
	}
	return NewTraderSyncServiceClient(unavailableConnection{err: status.Error(codes.Unavailable, "Trader Sync "+reason)})
}

type unavailableConnection struct{ err error }

func (c unavailableConnection) Invoke(context.Context, string, any, any, ...grpc.CallOption) error {
	return c.err
}
func (c unavailableConnection) NewStream(context.Context, *grpc.StreamDesc, string, ...grpc.CallOption) (grpc.ClientStream, error) {
	return nil, c.err
}
