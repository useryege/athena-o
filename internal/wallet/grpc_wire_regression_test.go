package wallet

import (
	"context"
	"errors"
	"net"
	"testing"
	"time"

	"github.com/golang/protobuf/ptypes/empty"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"github.com/useryege/athena/internal/wallet/apiclient"
	walletstore "github.com/useryege/athena/internal/wallet/store"
)

const (
	grpcRegressionInternalToken = "wallet-internal-regression-token-12345"
	grpcRegressionSignerToken   = "wallet-signer-regression-token-123456"
	grpcRegressionBlockMethod   = "/athena.test.BlockingService/Block"
)

func TestGRPCWireHealthStatusAndAuthentication(t *testing.T) {
	address, conn := startGRPCRegressionServer(t, nil)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Health is intentionally exempt from Wallet's bearer interceptor.
	health, err := grpc_health_v1.NewHealthClient(conn).Check(ctx, &grpc_health_v1.HealthCheckRequest{})
	if err != nil {
		t.Fatalf("unauthenticated health RPC: %v", err)
	}
	if health.GetStatus() != grpc_health_v1.HealthCheckResponse_SERVING {
		t.Fatalf("health status = %s, want SERVING", health.GetStatus())
	}

	clientset, err := apiclient.NewWalletClientset(address, grpcRegressionInternalToken)
	if err != nil {
		t.Fatalf("create Wallet clientset: %v", err)
	}
	t.Cleanup(func() { _ = clientset.Close() })
	walletStatus, err := clientset.Wallet().GetWalletStatus(ctx, &apiclient.GetWalletStatusRequest{})
	if err != nil {
		t.Fatalf("authenticated Wallet RPC: %v", err)
	}
	if !walletStatus.Started || walletStatus.Status != "running" {
		t.Fatalf("Wallet status = %+v, want started/running", walletStatus)
	}

	for _, tc := range []struct {
		name   string
		header string
	}{
		{name: "missing"},
		{name: "malformed", header: grpcRegressionInternalToken},
		{name: "wrong bearer", header: "Bearer wrong-wallet-internal-token-12345"},
		{name: "signer capability", header: "Bearer " + grpcRegressionSignerToken},
	} {
		t.Run(tc.name, func(t *testing.T) {
			callCtx := ctx
			if tc.header != "" {
				callCtx = metadata.AppendToOutgoingContext(ctx, "authorization", tc.header)
			}
			_, callErr := apiclient.NewWalletServiceClient(conn).GetWalletStatus(callCtx, &apiclient.GetWalletStatusRequest{})
			if status.Code(callErr) != codes.Unauthenticated {
				t.Fatalf("Wallet RPC error = %v, want Unauthenticated", callErr)
			}
		})
	}
}

func TestGRPCWireInFlightCancellationAndDeadline(t *testing.T) {
	for _, tc := range []struct {
		name             string
		deadline         time.Duration
		wantCode         codes.Code
		wantServerErrors []error
	}{
		{name: "cancel", deadline: 5 * time.Second, wantCode: codes.Canceled, wantServerErrors: []error{context.Canceled}},
		// For a deadline, the server's timer and the client's stream cancel race.
		{name: "deadline", deadline: time.Second, wantCode: codes.DeadlineExceeded, wantServerErrors: []error{context.Canceled, context.DeadlineExceeded}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			probe := &grpcRegressionBlockingProbe{
				entered:  make(chan struct{}, 1),
				observed: make(chan error, 1),
			}
			_, conn := startGRPCRegressionServer(t, probe)
			ctx, cancel := context.WithTimeout(context.Background(), tc.deadline)
			defer cancel()
			ctx = metadata.AppendToOutgoingContext(ctx, "authorization", "Bearer "+grpcRegressionInternalToken)

			callResult := make(chan error, 1)
			go func() {
				callResult <- conn.Invoke(ctx, grpcRegressionBlockMethod, &empty.Empty{}, &empty.Empty{})
			}()
			select {
			case <-probe.entered:
				// The RPC reached the server before cancellation or expiry.
			case <-ctx.Done():
				t.Fatalf("RPC did not enter the server before context ended: %v", ctx.Err())
			}
			if tc.wantCode == codes.Canceled {
				cancel()
			}
			select {
			case err := <-callResult:
				if status.Code(err) != tc.wantCode {
					t.Fatalf("client RPC error = %v, want %s", err, tc.wantCode)
				}
			case <-time.After(3 * time.Second):
				t.Fatal("client RPC did not finish after context ended")
			}
			select {
			case err := <-probe.observed:
				matched := false
				for _, want := range tc.wantServerErrors {
					matched = matched || errors.Is(err, want)
				}
				if !matched {
					t.Fatalf("server context error = %v, want one of %v", err, tc.wantServerErrors)
				}
			case <-time.After(3 * time.Second):
				t.Fatal("server did not observe the in-flight context ending")
			}
		})
	}
}

func startGRPCRegressionServer(t *testing.T, probe *grpcRegressionBlockingProbe) (string, *grpc.ClientConn) {
	t.Helper()
	server, err := NewServer(ServerOpts{
		Store:                        walletstore.NewSQLStore(nil),
		EncryptionKey:                []byte("regression-only-encryption-key"),
		InternalAuthToken:            grpcRegressionInternalToken,
		WormExecutionSignerAuthToken: grpcRegressionSignerToken,
	})
	if err != nil {
		t.Fatalf("create Wallet server: %v", err)
	}
	if err := server.Start(); err != nil {
		t.Fatalf("start Wallet service: %v", err)
	}
	grpcServer := server.CreateGRPC()
	if probe != nil {
		grpcServer.RegisterService(&grpcRegressionBlockingServiceDesc, probe)
	}
	listener, err := (&net.ListenConfig{}).Listen(context.Background(), "tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen for Wallet gRPC: %v", err)
	}
	serveResult := make(chan error, 1)
	go func() { serveResult <- grpcServer.Serve(listener) }()
	conn, err := grpc.NewClient(listener.Addr().String(), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		grpcServer.Stop()
		t.Fatalf("create gRPC client: %v", err)
	}
	t.Cleanup(func() {
		_ = conn.Close()
		grpcServer.Stop()
		if err := <-serveResult; err != nil {
			t.Errorf("serve Wallet gRPC: %v", err)
		}
		if err := server.Stop(); err != nil {
			t.Errorf("stop Wallet service: %v", err)
		}
	})
	return listener.Addr().String(), conn
}

type grpcRegressionBlockingService interface {
	Block(context.Context, *empty.Empty) (*empty.Empty, error)
}

type grpcRegressionBlockingProbe struct {
	entered  chan struct{}
	observed chan error
}

func (p *grpcRegressionBlockingProbe) Block(ctx context.Context, _ *empty.Empty) (*empty.Empty, error) {
	p.entered <- struct{}{}
	<-ctx.Done()
	p.observed <- ctx.Err()
	return nil, ctx.Err()
}

var grpcRegressionBlockingServiceDesc = grpc.ServiceDesc{
	ServiceName: "athena.test.BlockingService",
	HandlerType: (*grpcRegressionBlockingService)(nil),
	Methods: []grpc.MethodDesc{{
		MethodName: "Block",
		Handler: func(srv any, ctx context.Context, dec func(any) error, interceptor grpc.UnaryServerInterceptor) (any, error) {
			req := &empty.Empty{}
			if err := dec(req); err != nil {
				return nil, err
			}
			info := &grpc.UnaryServerInfo{Server: srv, FullMethod: grpcRegressionBlockMethod}
			return interceptor(ctx, req, info, func(ctx context.Context, req any) (any, error) {
				return srv.(grpcRegressionBlockingService).Block(ctx, req.(*empty.Empty))
			})
		},
	}},
}
