//go:build integration

package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	facade "github.com/useryege/athena/internal/server/tradersync"
	trpc "github.com/useryege/athena/internal/tradersync/apiclient"
	"github.com/useryege/athena/internal/tradersync/transport"
	api "github.com/useryege/athena/pkg/apiclient/tradersync"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
)

// Block only the first successful response of each write, AFTER the real handler
// committed. The caller learns no result; subsequent calls still traverse the
// same authenticated internal server and real PostgreSQL transaction code.
func TestTraderSyncCommittedInternalResponseLossReplaysExactlyOnce(t *testing.T) {
	h := newTraderSyncGatewayHarness(t, nil)
	var resolved struct {
		Target struct {
			ConfirmationToken string `json:"confirmationToken"`
		} `json:"target"`
	}
	require.NoError(t, json.Unmarshal(h.postAs("member", "/api/v1/trader-sync/targets:resolve", fmt.Sprintf(`{"input":%q}`, h.profileWallet), 200), &resolved))
	require.NotEmpty(t, resolved.Target.ConfirmationToken)
	const token = "lost-response-internal-token-0123456789"
	committed := make(chan any, 1)
	var mu sync.Mutex
	counts := map[string]int{}
	listener := bufconn.Listen(1024 * 1024)
	internal := grpc.NewServer(grpc.ChainUnaryInterceptor(transport.NewUnaryInterceptor(token, func() bool { return true }), func(ctx context.Context, request any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		mu.Lock()
		counts[info.FullMethod]++
		first := counts[info.FullMethod] == 1
		mu.Unlock()
		response, err := handler(ctx, request)
		if err == nil && first {
			committed <- response
			<-ctx.Done()
			return nil, status.FromContextError(ctx.Err()).Err()
		}
		return response, err
	}))
	trpc.RegisterTraderSyncServiceServer(internal, transport.NewServer(h.composition.service))
	go internal.Serve(listener)
	t.Cleanup(internal.Stop)
	connection, err := grpc.DialContext(context.Background(), "passthrough:///lost-response", grpc.WithInsecure(), grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) { return listener.Dial() }), grpc.WithUnaryInterceptor(func(ctx context.Context, method string, req, reply any, cc *grpc.ClientConn, invoke grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		return invoke(metadata.AppendToOutgoingContext(ctx, "authorization", "Bearer "+token), method, req, reply, cc, opts...)
	}))
	require.NoError(t, err)
	t.Cleanup(func() { connection.Close() })
	public := facade.New(trpc.NewTraderSyncServiceClient(connection), func(context.Context) (*trpc.Actor, error) {
		return &trpc.Actor{AccountId: h.owner, Realm: trpc.ApplicationRealm_APPLICATION_REALM_MEMBER}, nil
	})
	wallet := common.HexToAddress(h.profileWallet).Bytes()
	type databaseState struct {
		Subscriptions                        int
		Revision, Generation, NoteRevision   int64
		State, Note                          string
		Results                              int
		SubscriptionUpdatedAt, NoteUpdatedAt time.Time
	}
	snapshot := func() databaseState {
		t.Helper()
		var s databaseState
		require.NoError(t, h.db.Pool.QueryRow(h.ctx, `SELECT count(*) FROM trader_sync_subscriptions WHERE owner_id=$1 AND wallet=$2`, h.owner, wallet).Scan(&s.Subscriptions))
		require.NoError(t, h.db.Pool.QueryRow(h.ctx, `SELECT revision,activation_generation,desired_state,updated_at FROM trader_sync_subscriptions WHERE owner_id=$1 AND wallet=$2`, h.owner, wallet).Scan(&s.Revision, &s.Generation, &s.State, &s.SubscriptionUpdatedAt))
		require.NoError(t, h.db.Pool.QueryRow(h.ctx, `SELECT note,revision,updated_at FROM trader_sync_target_notes WHERE owner_id=$1 AND wallet=$2`, h.owner, wallet).Scan(&s.Note, &s.NoteRevision, &s.NoteUpdatedAt))
		require.NoError(t, h.db.Pool.QueryRow(h.ctx, `SELECT count(*) FROM trader_sync_request_results WHERE owner_id=$1`, h.owner).Scan(&s.Results))
		return s
	}
	// Cancellation is triggered by confirmed handler completion, never a sleep.
	lost := func(call func(context.Context) (any, error)) any {
		t.Helper()
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		result := make(chan error, 1)
		go func() { _, err := call(ctx); result <- err }()
		var response any
		select {
		case response = <-committed:
		case err := <-result:
			t.Fatalf("write failed before commit: %v", err)
		case <-ctx.Done():
			t.Fatal("internal commit not observed")
		}
		// A separate pool connection observes the committed row before the response
		// is discarded, ruling out a transport failure before transaction commit.
		before := snapshot()
		require.Equal(t, 1, before.Subscriptions)
		select {
		case err := <-result:
			t.Fatalf("blocked response escaped: %v", err)
		default:
		}
		cancel()
		select {
		case err := <-result:
			require.Equal(t, codes.Canceled, status.Code(err))
		case <-time.After(5 * time.Second):
			t.Fatal("facade ignored cancellation")
		}
		return response
	}
	create := &api.CreateSubscriptionRequest{ConfirmationToken: resolved.Target.ConfirmationToken, RequestId: uuid.NewString(), Note: &api.TraderSyncNoteInput{Value: "initial"}}
	first := lost(func(ctx context.Context) (any, error) { return public.CreateSubscription(ctx, create) }).(*trpc.CreateSubscriptionResponse)
	before := snapshot()
	require.Equal(t, int64(1), before.Revision)
	require.Equal(t, int64(1), before.Generation)
	require.Equal(t, int64(1), before.NoteRevision)
	require.Equal(t, "initial", before.Note)
	require.Equal(t, 1, before.Results)
	profileRequests := h.profileRequests.Load()
	replay, err := public.CreateSubscription(h.ctx, create)
	require.NoError(t, err)
	require.Equal(t, first.Subscription.Id, replay.Subscription.ID)
	require.Equal(t, uint64(1), replay.Subscription.Revision)
	require.Equal(t, before, snapshot())
	require.Equal(t, profileRequests, h.profileRequests.Load())
	changedCreate := *create
	changedCreate.Note = &api.TraderSyncNoteInput{Value: "different"}
	_, err = public.CreateSubscription(h.ctx, &changedCreate)
	require.Equal(t, codes.AlreadyExists, status.Code(err))
	require.Equal(t, before, snapshot())

	pause := &api.PauseSubscriptionRequest{SubscriptionId: replay.Subscription.ID, ExpectedRevision: 1, RequestId: uuid.NewString()}
	paused := lost(func(ctx context.Context) (any, error) { return public.PauseSubscription(ctx, pause) }).(*trpc.PauseSubscriptionResponse)
	before = snapshot()
	require.Equal(t, int64(2), before.Revision)
	require.Equal(t, "paused", before.State)
	require.Equal(t, 2, before.Results)
	pauseReplay, err := public.PauseSubscription(h.ctx, pause)
	require.NoError(t, err)
	require.Equal(t, paused.Subscription.Revision, pauseReplay.Subscription.Revision)
	require.Equal(t, before, snapshot())
	changedPause := *pause
	changedPause.ExpectedRevision = 2
	_, err = public.PauseSubscription(h.ctx, &changedPause)
	require.Equal(t, codes.AlreadyExists, status.Code(err))
	require.Equal(t, before, snapshot())

	note := &api.UpdateTargetNoteRequest{Wallet: h.profileWallet, Note: "updated", ExpectedRevision: 1, RequestId: uuid.NewString()}
	noted := lost(func(ctx context.Context) (any, error) { return public.UpdateTargetNote(ctx, note) }).(*trpc.UpdateTargetNoteResponse)
	before = snapshot()
	require.Equal(t, int64(2), before.NoteRevision)
	require.Equal(t, "updated", before.Note)
	require.Equal(t, 3, before.Results)
	noteReplay, err := public.UpdateTargetNote(h.ctx, note)
	require.NoError(t, err)
	require.Equal(t, noted.Note.Revision, noteReplay.Note.Revision)
	require.Equal(t, before, snapshot())
	changedNote := *note
	changedNote.Note = "different"
	_, err = public.UpdateTargetNote(h.ctx, &changedNote)
	require.Equal(t, codes.AlreadyExists, status.Code(err))
	require.Equal(t, before, snapshot())
	mu.Lock()
	defer mu.Unlock()
	for _, method := range []string{"CreateSubscription", "PauseSubscription", "UpdateTargetNote"} {
		require.Equal(t, 3, counts["/tradersync.internal.v1.TraderSyncService/"+method], "first lost response, one identical replay, one rejected conflicting replay")
	}
	t.Log("real PG commit observed before response cancellation; create, revision mutation and note each applied once; conflicting request IDs rejected")
}
