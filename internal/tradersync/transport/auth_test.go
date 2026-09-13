package transport

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	trpc "github.com/useryege/athena/internal/tradersync/apiclient"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

const authToken = "0123456789abcdef0123456789abcdef"
const memberID = "b7b774b3-dcb5-4bf4-bc64-b0c523d263d7"
const methodPrefix = "/tradersync.internal.v1.TraderSyncService/"

func member() *trpc.Actor {
	return &trpc.Actor{AccountId: memberID, Realm: trpc.ApplicationRealm_APPLICATION_REALM_MEMBER}
}
func assertReason(t *testing.T, err error, code codes.Code, reason string) {
	t.Helper()
	require.Equal(t, code, status.Code(err))
	found := false
	for _, detail := range status.Convert(err).Details() {
		if info, ok := detail.(*errdetails.ErrorInfo); ok {
			found = info.Domain == "tradersync.internal.v1" && info.Reason == reason
		}
	}
	require.True(t, found, "missing reason %s: %v", reason, err)
}
func TestActorRejectsUnknownRealm(t *testing.T) {
	actor := member()
	actor.Realm = trpc.ApplicationRealm(99)
	assertReason(t, ValidateActor(actor), codes.InvalidArgument, "ACTOR_INVALID")
}
func TestActorRequiresCanonicalIdentityAndKnownRealm(t *testing.T) {
	require.NoError(t, ValidateActor(member()))
	for _, actor := range []*trpc.Actor{nil, {}, {AccountId: memberID}, {AccountId: "B7B774B3-DCB5-4BF4-BC64-B0C523D263D7", Realm: 1}, {AccountId: "00000000-0000-0000-0000-000000000000", Realm: 1}, {AccountId: " " + memberID, Realm: 1}, {AccountId: "urn:uuid:" + memberID, Realm: 1}} {
		assertReason(t, ValidateActor(actor), codes.InvalidArgument, "ACTOR_INVALID")
	}
}
func authContext(token string) context.Context {
	return metadata.NewIncomingContext(context.Background(), metadata.Pairs("authorization", token))
}
func TestUnaryAuthenticationRealmAndReadiness(t *testing.T) {
	for _, tc := range []struct {
		name, credential, method string
		actor                    *trpc.Actor
		ready                    bool
		code                     codes.Code
		reason                   string
	}{
		{"missing", "", "GetActivity", member(), true, codes.Unauthenticated, "SERVICE_AUTH_MISSING"},
		{"wrong", "Bearer wrong", "GetActivity", member(), true, codes.Unauthenticated, "SERVICE_AUTH_INVALID"},
		{"malformed", "Basic " + authToken, "GetActivity", member(), true, codes.Unauthenticated, "SERVICE_AUTH_INVALID"},
		{"bad actor", "Bearer " + authToken, "GetActivity", nil, true, codes.InvalidArgument, "ACTOR_INVALID"},
		{"member admin RPC", "Bearer " + authToken, "GetTraderSyncRuntimeStatus", member(), true, codes.PermissionDenied, ""},
		{"admin member RPC", "Bearer " + authToken, "GetActivity", &trpc.Actor{AccountId: memberID, Realm: 2}, true, codes.PermissionDenied, ""},
		{"not ready", "Bearer " + authToken, "GetActivity", member(), false, codes.Unavailable, ""},
		{"unknown method", "Bearer " + authToken, "Unknown", member(), true, codes.Unimplemented, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			called := false
			_, err := NewUnaryInterceptor(authToken, func() bool { return tc.ready })(authContext(tc.credential), &trpc.GetActivityRequest{Actor: tc.actor}, &grpc.UnaryServerInfo{FullMethod: methodPrefix + tc.method}, func(context.Context, any) (any, error) { called = true; return nil, nil })
			require.False(t, called)
			require.Equal(t, tc.code, status.Code(err))
			if tc.reason != "" {
				assertReason(t, err, tc.code, tc.reason)
			}
		})
	}
}
func TestUnaryBudgetsRespectParentAndHealthIsIndependent(t *testing.T) {
	interceptor := NewUnaryInterceptor(authToken, func() bool { return true })
	for _, tc := range []struct {
		method string
		budget time.Duration
	}{{"GetActivity", 5 * time.Second}, {"CreateSubscription", 15 * time.Second}} {
		start := time.Now()
		_, err := interceptor(authContext("Bearer "+authToken), &trpc.GetActivityRequest{Actor: member()}, &grpc.UnaryServerInfo{FullMethod: methodPrefix + tc.method}, func(ctx context.Context, _ any) (any, error) {
			deadline, ok := ctx.Deadline()
			require.True(t, ok)
			require.WithinDuration(t, start.Add(tc.budget), deadline, 100*time.Millisecond)
			return nil, nil
		})
		require.NoError(t, err)
	}
	ctx, cancel := context.WithTimeout(authContext("Bearer "+authToken), 50*time.Millisecond)
	defer cancel()
	want, _ := ctx.Deadline()
	_, err := interceptor(ctx, &trpc.GetActivityRequest{Actor: member()}, &grpc.UnaryServerInfo{FullMethod: methodPrefix + "GetActivity"}, func(ctx context.Context, _ any) (any, error) {
		got, _ := ctx.Deadline()
		require.Equal(t, want, got)
		return nil, nil
	})
	require.NoError(t, err)
	called := false
	_, err = NewUnaryInterceptor(authToken, func() bool { return false })(context.Background(), nil, &grpc.UnaryServerInfo{FullMethod: "/grpc.health.v1.Health/Check"}, func(context.Context, any) (any, error) { called = true; return nil, nil })
	require.NoError(t, err)
	require.True(t, called)
}
