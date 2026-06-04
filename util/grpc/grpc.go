package grpc

import (
	"context"
	"fmt"
	"net"
	"runtime/debug"

	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/recovery"
	"github.com/sirupsen/logrus"
	"golang.org/x/net/proxy"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/status"

	"github.com/useryege/athena/common"
)

// LoggerRecoveryHandler return a handler for recovering from panics and returning error
func LoggerRecoveryHandler(log *logrus.Entry) recovery.RecoveryHandlerFunc {
	return func(p any) (err error) {
		log.Errorf("Recovered from panic: %+v\n%s", p, debug.Stack())
		return status.Errorf(codes.Internal, "%s", p)
	}
}

// BlockingNewClient is a helper method to dial the given address and block until
// the returned plain-text connection is ready.
// Lifted from: https://github.com/fullstorydev/grpcurl/blob/master/grpcurl.go
func BlockingNewClient(ctx context.Context, network, address string, opts ...grpc.DialOption) (*grpc.ClientConn, error) {
	proxyDialer := proxy.FromEnvironment()
	rawConn, err := proxyDialer.Dial(network, address)
	if err != nil {
		return nil, fmt.Errorf("error dial proxy: %w", err)
	}

	customDialer := func(_ context.Context, _ string) (net.Conn, error) {
		return rawConn, nil
	}

	opts = append(opts,
		grpc.WithContextDialer(customDialer),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithKeepaliveParams(keepalive.ClientParameters{Time: common.GetGRPCKeepAliveTime()}),
	)

	conn, err := grpc.NewClient("passthrough:"+address, opts...)
	if err != nil {
		return nil, fmt.Errorf("grpc.NewClient failed: %w", err)
	}

	conn.Connect()
	if err := waitForReady(ctx, conn); err != nil {
		return nil, fmt.Errorf("gRPC connection not ready: %w", err)
	}

	return conn, nil
}

func waitForReady(ctx context.Context, conn *grpc.ClientConn) error {
	for {
		state := conn.GetState()
		if state == connectivity.Ready {
			return nil
		}
		if !conn.WaitForStateChange(ctx, state) {
			return ctx.Err() // context timeout or cancellation
		}
	}
}
