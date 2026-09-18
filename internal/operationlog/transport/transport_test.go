package transport

import (
	"context"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"testing"
)

func TestAuthorizeRequiresBearerAndViewer(t *testing.T) {
	a := Authenticator{Token: "secret", CheckViewer: func(Viewer) error { return nil }}
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("authorization", "Bearer secret"))
	if _, err := a.Authorize(ctx, Viewer{}); err != nil {
		t.Fatal(err)
	}
	bad := metadata.NewIncomingContext(context.Background(), metadata.Pairs("authorization", "Bearer wrong"))
	if _, err := a.Authorize(bad, Viewer{}); status.Code(err).String() != "PermissionDenied" {
		t.Fatalf("got %v", err)
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
