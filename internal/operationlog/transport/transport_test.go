package transport

import (
	"context"
	"strings"

	ipb "github.com/useryege/athena/internal/operationlog/apiclient"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"testing"
)

func TestAuthorizeRequiresBearerAndViewer(t *testing.T) {
	a := Authenticator{Token: "secret", CheckViewer: func(Viewer) error { return nil }}
	viewer := Viewer{AccountID: "00000000-0000-4000-8000-000000000001", Realm: "ADMIN", CredentialKind: "LOGIN_SESSION", SessionBinding: []byte("01234567890123456789012345678901"), AccessRevision: 1}
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("authorization", "Bearer secret"))
	if _, err := a.Authorize(ctx, viewer); err != nil {
		t.Fatal(err)
	}
	bad := metadata.NewIncomingContext(context.Background(), metadata.Pairs("authorization", "Bearer wrong"))
	if _, err := a.Authorize(bad, viewer); status.Code(err).String() != "PermissionDenied" {
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
