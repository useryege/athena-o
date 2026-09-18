package transport

import (
	"context"
	"net"
	"strings"
	"testing"

	ipb "github.com/useryege/athena/internal/operationlog/apiclient"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
)

func TestAuthorizeRequiresBearerAndViewer(t *testing.T) {
	a := Authenticator{Token: "secret", CheckViewer: func(Viewer) error { return nil }}
	viewer := Viewer{AccountID: "00000000-0000-4000-8000-000000000001", Realm: "ADMIN", CredentialKind: "LOGIN_SESSION", SessionBinding: []byte("01234567890123456789012345678901"), AccessRevision: 1}
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("authorization", "Bearer secret"))
	if _, err := a.Authorize(ctx, viewer); err != nil {
		t.Fatal(err)
	}
	bad := metadata.NewIncomingContext(context.Background(), metadata.Pairs("authorization", "Bearer wrong"))
	if _, err := a.Authorize(bad, viewer); status.Code(err).String() != "Unauthenticated" {
		t.Fatalf("got %v", err)
	}
}

func TestUnaryInterceptorReturnsStableOversizeReason(t *testing.T) {
	interceptor := UnaryServerInterceptor("secret")
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("authorization", "Bearer secret"))
	req := &ipb.ListOperationLogsRequest{ActorQuery: strings.Repeat("x", 70<<10)}
	_, err := interceptor(ctx, req, &grpc.UnaryServerInfo{FullMethod: "/operationlog.internal.v1.OperationLogInternalService/ListOperationLogs"}, func(context.Context, interface{}) (interface{}, error) {
		t.Fatal("oversize request reached handler")
		return nil, nil
	})
	if status.Code(err).String() != "InvalidArgument" || !strings.Contains(err.Error(), ReasonMessageTooLarge) {
		t.Fatalf("err=%v", err)
	}
}
func TestMessageCapacity(t *testing.T) {
	if err := CheckMessageSize(make([]byte, 64<<10)); err != nil {
		t.Fatal(err)
	}
	if err := CheckMessageSize(make([]byte, 64<<10+1)); err == nil {
		t.Fatal("oversize accepted")
	}
}

type listenerService struct {
	ipb.UnimplementedOperationLogInternalServiceServer
	response *ipb.GetOperationLogRuntimeStatusResponse
}

func (s listenerService) GetOperationLogRuntimeStatus(context.Context, *ipb.GetOperationLogRuntimeStatusRequest) (*ipb.GetOperationLogRuntimeStatusResponse, error) {
	return s.response, nil
}

func TestUnaryInterceptorRealBufconnEnforcesBearerAndCapacity(t *testing.T) {
	const token = "01234567890123456789012345678901"
	listener := bufconn.Listen(256 * 1024)
	grpcServer := grpc.NewServer(
		grpc.MaxRecvMsgSize(128<<10),
		grpc.MaxSendMsgSize(128<<10),
		grpc.UnaryInterceptor(UnaryServerInterceptor(token)),
	)
	service := &listenerService{response: &ipb.GetOperationLogRuntimeStatusResponse{Status: &ipb.RuntimeStatus{ProjectionState: "READY"}}}
	ipb.RegisterOperationLogInternalServiceServer(grpcServer, service)
	go func() { _ = grpcServer.Serve(listener) }()
	t.Cleanup(func() { grpcServer.Stop(); _ = listener.Close() })
	dialer := func(context.Context, string) (net.Conn, error) { return listener.Dial() }
	conn, err := grpc.DialContext(context.Background(), "bufnet", grpc.WithContextDialer(dialer), grpc.WithInsecure())
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	client := ipb.NewOperationLogInternalServiceClient(conn)
	ctx := metadata.NewOutgoingContext(context.Background(), metadata.Pairs("authorization", "Bearer "+token))
	resp, err := client.GetOperationLogRuntimeStatus(ctx, &ipb.GetOperationLogRuntimeStatusRequest{})
	if err != nil || resp.GetStatus().GetProjectionState() != "READY" {
		t.Fatalf("authorized request failed: resp=%v err=%v", resp, err)
	}
	_, err = client.GetOperationLogRuntimeStatus(context.Background(), &ipb.GetOperationLogRuntimeStatusRequest{})
	if status.Code(err) != 16 || !strings.Contains(err.Error(), ReasonInternalAuthRequired) {
		t.Fatalf("missing bearer error=%v", err)
	}
	_, err = client.ListOperationLogs(ctx, &ipb.ListOperationLogsRequest{ActorQuery: strings.Repeat("x", 70<<10)})
	if status.Code(err) != 3 || !strings.Contains(err.Error(), ReasonMessageTooLarge) {
		t.Fatalf("oversize request error=%v", err)
	}
	service.response = &ipb.GetOperationLogRuntimeStatusResponse{Status: &ipb.RuntimeStatus{ServiceEpoch: strings.Repeat("x", 70<<10)}}
	_, err = client.GetOperationLogRuntimeStatus(ctx, &ipb.GetOperationLogRuntimeStatusRequest{})
	if status.Code(err) != 8 || !strings.Contains(err.Error(), ReasonMessageTooLarge) {
		t.Fatalf("oversize response error=%v", err)
	}
}
