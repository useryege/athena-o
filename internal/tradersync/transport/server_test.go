package transport

import (
	"context"
	"net"
	"reflect"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	ts "github.com/useryege/athena/internal/tradersync"
	trpc "github.com/useryege/athena/internal/tradersync/apiclient"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
)

func TestAllHandlersRequireActorBeforeAccessingService(t *testing.T) {
	server := NewServer(nil)
	typ := reflect.TypeOf((*trpc.TraderSyncServiceServer)(nil)).Elem()
	require.Equal(t, 16, typ.NumMethod())
	for i := 0; i < typ.NumMethod(); i++ {
		method := typ.Method(i)
		t.Run(method.Name, func(t *testing.T) {
			request := reflect.New(method.Type.In(1).Elem())
			result := reflect.ValueOf(server).MethodByName(method.Name).Call([]reflect.Value{reflect.ValueOf(context.Background()), request})
			require.True(t, result[0].IsNil())
			assertReason(t, result[1].Interface().(error), codes.InvalidArgument, "ACTOR_INVALID")
		})
	}
}
func TestServerPreservesBusinessParameterValidation(t *testing.T) {
	server := NewServer(&ts.Service{})
	_, err := server.ResolveTarget(context.Background(), &trpc.ResolveTargetRequest{Actor: member(), Input: " \t"})
	require.Equal(t, codes.InvalidArgument, status.Code(err))
	_, err = server.GetActivity(context.Background(), &trpc.GetActivityRequest{Actor: member(), ActivityId: "01"})
	require.Equal(t, codes.InvalidArgument, status.Code(err))
	_, err = server.UpdateTargetNote(context.Background(), &trpc.UpdateTargetNoteRequest{Actor: member(), Wallet: "0x0000000000000000000000000000000000000000"})
	require.Equal(t, codes.InvalidArgument, status.Code(err))
	_, err = server.ListActivities(context.Background(), &trpc.ListActivitiesRequest{Actor: member(), From: "bad"})
	require.Equal(t, codes.InvalidArgument, status.Code(err))
	_, err = server.ListActivities(context.Background(), &trpc.ListActivitiesRequest{Actor: member(), SummaryBatchId: "01"})
	require.Equal(t, codes.InvalidArgument, status.Code(err))
	_, err = server.ListSubscriptionSummaries(context.Background(), &trpc.ListSubscriptionSummariesRequest{Actor: &trpc.Actor{AccountId: memberID, Realm: 2}, Wallet: "not-wallet"})
	require.Equal(t, codes.InvalidArgument, status.Code(err))
	at, err := parseTimeFilter("2026-09-13T10:03:04.000000005+08:00")
	require.NoError(t, err)
	require.Equal(t, "2026-09-13T02:03:04.000000005Z", at.Format(time.RFC3339Nano))
}
func bufClient(t *testing.T, service trpc.TraderSyncServiceServer, ready func() bool) (trpc.TraderSyncServiceClient, *grpc.ClientConn) {
	t.Helper()
	listener := bufconn.Listen(1024 * 1024)
	server := grpc.NewServer(grpc.UnaryInterceptor(NewUnaryInterceptor(authToken, ready)))
	trpc.RegisterTraderSyncServiceServer(server, service)
	healthpb.RegisterHealthServer(server, health.NewServer())
	go func() { _ = server.Serve(listener) }()
	t.Cleanup(server.Stop)
	connection, err := grpc.NewClient("passthrough:///bufconn", grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) { return listener.Dial() }))
	require.NoError(t, err)
	t.Cleanup(func() { _ = connection.Close() })
	return trpc.NewTraderSyncServiceClient(connection), connection
}
func TestBufconnAuthReasonsAndHealthBoundary(t *testing.T) {
	client, connection := bufClient(t, NewServer(&ts.Service{}), func() bool { return true })
	for _, tc := range []struct {
		token  string
		actor  *trpc.Actor
		code   codes.Code
		reason string
	}{{"", member(), codes.Unauthenticated, "SERVICE_AUTH_MISSING"}, {"Bearer wrong", member(), codes.Unauthenticated, "SERVICE_AUTH_INVALID"}, {"Bearer " + authToken, nil, codes.InvalidArgument, "ACTOR_INVALID"}} {
		ctx := metadata.NewOutgoingContext(context.Background(), metadata.Pairs("authorization", tc.token))
		_, err := client.GetActivity(ctx, &trpc.GetActivityRequest{Actor: tc.actor})
		assertReason(t, err, tc.code, tc.reason)
	}
	response, err := healthpb.NewHealthClient(connection).Check(context.Background(), &healthpb.HealthCheckRequest{})
	require.NoError(t, err)
	require.Equal(t, healthpb.HealthCheckResponse_SERVING, response.Status)
	ctx := metadata.NewOutgoingContext(context.Background(), metadata.Pairs("authorization", "Bearer "+authToken))
	_, err = client.GetActivity(ctx, &trpc.GetActivityRequest{Actor: member(), ActivityId: "01"})
	require.Equal(t, codes.InvalidArgument, status.Code(err))
	require.Empty(t, status.Convert(err).Details(), "business error must not acquire internal contract reason")
}

type cancellationServer struct {
	trpc.UnimplementedTraderSyncServiceServer
	entered chan time.Time
	done    chan error
}

func (s *cancellationServer) GetActivity(ctx context.Context, _ *trpc.GetActivityRequest) (*trpc.GetActivityResponse, error) {
	deadline, _ := ctx.Deadline()
	s.entered <- deadline
	<-ctx.Done()
	s.done <- ctx.Err()
	return nil, status.FromContextError(ctx.Err()).Err()
}
func TestBufconnServerPropagatesCancellationAndShortDeadline(t *testing.T) {
	service := &cancellationServer{entered: make(chan time.Time, 1), done: make(chan error, 1)}
	client, _ := bufClient(t, service, func() bool { return true })
	ctx, cancel := context.WithTimeout(metadata.NewOutgoingContext(context.Background(), metadata.Pairs("authorization", "Bearer "+authToken)), 50*time.Millisecond)
	defer cancel()
	want, _ := ctx.Deadline()
	result := make(chan error, 1)
	go func() { _, err := client.GetActivity(ctx, &trpc.GetActivityRequest{Actor: member()}); result <- err }()
	select {
	case got := <-service.entered:
		require.WithinDuration(t, want, got, 10*time.Millisecond)
	case <-time.After(time.Second):
		t.Fatal("handler not entered")
	}
	cancel()
	select {
	case err := <-result:
		require.Equal(t, codes.Canceled, status.Code(err))
	case <-time.After(time.Second):
		t.Fatal("client cancellation blocked")
	}
	select {
	case err := <-service.done:
		require.ErrorIs(t, err, context.Canceled)
	case <-time.After(time.Second):
		t.Fatal("handler did not receive cancellation")
	}
}
