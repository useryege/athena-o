//go:build integration

package commands

import (
	"bytes"
	"context"
	"fmt"
	"net"
	"os"
	"os/exec"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/useryege/athena/internal/accountcredentials"
	accountstore "github.com/useryege/athena/internal/accountstate/store"
	"github.com/useryege/athena/internal/accountstate/store/migrations"
	"github.com/useryege/athena/internal/accountstate/txgate"
	"github.com/useryege/athena/internal/testutil/pgtest"
	"github.com/useryege/athena/internal/tradersync"
	trpc "github.com/useryege/athena/internal/tradersync/apiclient"
	"github.com/useryege/athena/internal/tradersync/rpcconfig"
	"github.com/useryege/athena/internal/tradersync/store"
	"github.com/useryege/athena/internal/tradersync/transport"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func TestRuntimeDatabaseHelperProcess(t *testing.T) {
	if os.Getenv("ATHENA_TEST_RUNTIME_DB_HELPER") != "1" {
		return
	}
	cfg := tradersync.Config{AccountStateDSN: os.Getenv("ATHENA_TEST_RUNTIME_DSN"), ShutdownTimeout: time.Second, HTTPURL: "http://127.0.0.1:1", ProxyURL: "http://127.0.0.1:1", WebSocketURL: "ws://127.0.0.1:1", SiteURL: "https://athena.test", CursorHMACKey: "fixture-cursor-key"}
	rpcCfg := rpcconfig.Server{ListenAddress: os.Getenv("ATHENA_TEST_RUNTIME_ADDRESS"), Transport: "loopback-insecure", Token: "0123456789abcdef0123456789abcdef", MaxMessageBytes: 1024}
	accepted := make(chan struct{})
	blocked := make(chan struct{})
	r, err := tradersync.NewRuntime(cfg, rpcCfg, tradersync.RuntimeDependencies{NewRPCServer: func(*tradersync.Service, func() bool) *grpc.Server {
		return grpc.NewServer(grpc.UnaryInterceptor(func(context.Context, any, *grpc.UnaryServerInfo, grpc.UnaryHandler) (any, error) {
			close(accepted)
			<-blocked
			return nil, nil
		}))
	}})
	require.NoError(t, err)
	require.NoError(t, r.Start(context.Background()))
	conn, err := grpc.NewClient(rpcCfg.ListenAddress, grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err)
	go healthpb.NewHealthClient(conn).Check(context.Background(), &healthpb.HealthCheckRequest{Service: tradersync.HealthServiceName})
	<-accepted
	fmt.Println("accepted RPC remains blocked during shutdown")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_ = runLifecycle(ctx, r, time.Second)
	os.Exit(0)
}
func TestRuntimeProcessWatchdogRetainsDatabaseOwnerUntilExit(t *testing.T) {
	db := pgtest.New(t, migrations.FS, migrations.Dir)
	l, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	address := l.Addr().String()
	require.NoError(t, l.Close())
	child := exec.Command(os.Args[0], "-test.run=^TestRuntimeDatabaseHelperProcess$")
	child.Env = append(os.Environ(), "ATHENA_TEST_RUNTIME_DB_HELPER=1", "ATHENA_TEST_RUNTIME_DSN="+db.DSN, "ATHENA_TEST_RUNTIME_ADDRESS="+address)
	var logs bytes.Buffer
	child.Stdout = &logs
	child.Stderr = &logs
	require.NoError(t, child.Start())
	started := time.Now()
	// Observe the real session lock while the child owns accepted RPC work.
	require.Eventually(t, func() bool {
		var active bool
		err := db.Pool.QueryRow(context.Background(), `SELECT owner_id IS NOT NULL FROM trader_sync_runtime_control WHERE singleton`).Scan(&active)
		return err == nil && active
	}, time.Second, 10*time.Millisecond)
	_, err = store.NewSQLStore(db.Pool).AcquireRuntimeSession(context.Background())
	require.ErrorContains(t, err, "another Trader Sync runtime")
	err = child.Wait()
	require.Error(t, err)
	require.Less(t, time.Since(started), 5*time.Second)
	require.Contains(t, logs.String(), "shutdown deadline exceeded")
	require.Contains(t, logs.String(), "accepted RPC remains blocked")
	successor, err := store.NewSQLStore(db.Pool).AcquireRuntimeSession(context.Background())
	require.NoError(t, err)
	require.NoError(t, successor.CloseAfterWorkers(context.Background()))
	t.Log(logs.String())
}

func TestRuntimeRecoveryServesNotServingBeforePublishingService(t *testing.T) {
	for _, mode := range []string{"publish", "owner_loss"} {
		t.Run(mode, func(t *testing.T) {
			db := pgtest.New(t, migrations.FS, migrations.Dir)
			ctx := context.Background()
			account, err := accountstore.NewSQLStore(db.Pool).EnsureDevelopmentAccount(ctx, accountcredentials.DevelopmentRoleMember)
			require.NoError(t, err)
			var sub string
			require.NoError(t, db.Pool.QueryRow(ctx, `INSERT INTO trader_sync_subscriptions(owner_id,wallet,desired_state,observation_state,target_display) VALUES($1,decode(repeat('66',20),'hex'),'enabled','pending_baseline','{"displayName":{"availability":"unavailable","reasonCode":"fixture_not_queried","source":"fixture"},"avatar":{"availability":"unavailable","reasonCode":"fixture_not_queried","source":"fixture"},"profileURL":{"availability":"unavailable","reasonCode":"fixture_not_queried","source":"fixture"}}'::jsonb) RETURNING id`, account.ID).Scan(&sub))
			_, err = db.Pool.Exec(ctx, `INSERT INTO trader_sync_baseline_attempts(owner_id,subscription_id,activation_generation,expected_revision) VALUES($1,$2,1,1)`, account.ID, sub)
			require.NoError(t, err)
			blocker, err := txgate.AcquireAccountSession(ctx, db.Pool, account.ID)
			require.NoError(t, err)
			defer blocker.Release(ctx)
			l, err := net.Listen("tcp", "127.0.0.1:0")
			require.NoError(t, err)
			address := l.Addr().String()
			require.NoError(t, l.Close())
			cfg := tradersync.Config{AccountStateDSN: db.DSN, ShutdownTimeout: time.Second, HTTPURL: "http://127.0.0.1:1", ProxyURL: "http://127.0.0.1:1", WebSocketURL: "ws://127.0.0.1:1", SiteURL: "https://athena.test", CursorHMACKey: "fixture-cursor-key"}
			rpcCfg := rpcconfig.Server{ListenAddress: address, Transport: "loopback-insecure", Token: "0123456789abcdef0123456789abcdef", MaxMessageBytes: 1024}
			r, err := tradersync.NewRuntime(cfg, rpcCfg, tradersync.RuntimeDependencies{NewRPCServer: func(s *tradersync.Service, ready func() bool) *grpc.Server {
				server := grpc.NewServer(grpc.UnaryInterceptor(transport.NewUnaryInterceptor(rpcCfg.Token, ready)))
				trpc.RegisterTraderSyncServiceServer(server, transport.NewServer(s))
				return server
			}})
			require.NoError(t, err)
			startCtx, cancel := context.WithCancel(ctx)
			defer cancel()
			started := make(chan error, 1)
			go func() { started <- r.Start(startCtx) }()
			t.Cleanup(func() {
				cleanup, stop := context.WithTimeout(context.Background(), 5*time.Second)
				defer stop()
				_ = r.Shutdown(cleanup)
			})
			require.Eventually(t, func() bool {
				var owned bool
				err := db.Pool.QueryRow(ctx, `SELECT owner_id IS NOT NULL FROM trader_sync_runtime_control WHERE singleton`).Scan(&owned)
				return err == nil && owned
			}, time.Second, 10*time.Millisecond)
			require.False(t, r.Ready())
			conn, err := grpc.NewClient(address, grpc.WithTransportCredentials(insecure.NewCredentials()))
			require.NoError(t, err)
			defer conn.Close()
			bounded, stop := context.WithTimeout(ctx, 200*time.Millisecond)
			defer stop()
			health, err := healthpb.NewHealthClient(conn).Check(bounded, &healthpb.HealthCheckRequest{Service: tradersync.HealthServiceName})
			require.NoError(t, err)
			require.Equal(t, healthpb.HealthCheckResponse_NOT_SERVING, health.Status)
			actor := &trpc.Actor{AccountId: account.ID, Realm: trpc.ApplicationRealm_APPLICATION_REALM_MEMBER}
			authCtx, stopAuth := context.WithTimeout(ctx, time.Second)
			defer stopAuth()
			auth := metadata.AppendToOutgoingContext(authCtx, "authorization", "Bearer "+rpcCfg.Token)
			adminActor := &trpc.Actor{AccountId: account.ID, Realm: trpc.ApplicationRealm_APPLICATION_REALM_ADMIN}
			for _, call := range []struct {
				method  string
				request any
			}{
				{"ResolveTarget", &trpc.ResolveTargetRequest{Actor: actor}},
				{"CreateSubscription", &trpc.CreateSubscriptionRequest{Actor: actor}},
				{"ListSubscriptions", &trpc.ListSubscriptionsRequest{Actor: actor}},
				{"GetSubscription", &trpc.GetSubscriptionRequest{Actor: actor}},
				{"PauseSubscription", &trpc.PauseSubscriptionRequest{Actor: actor}},
				{"ResumeSubscription", &trpc.ResumeSubscriptionRequest{Actor: actor}},
				{"CancelSubscription", &trpc.CancelSubscriptionRequest{Actor: actor}},
				{"UpdateTargetNote", &trpc.UpdateTargetNoteRequest{Actor: actor}},
				{"ListActivities", &trpc.ListActivitiesRequest{Actor: actor}},
				{"GetActivity", &trpc.GetActivityRequest{Actor: actor}},
				{"ListSubscriptionHistory", &trpc.ListSubscriptionHistoryRequest{Actor: actor}},
				{"GetSummaryBatch", &trpc.GetSummaryBatchRequest{Actor: actor}},
				{"ListSummaryParts", &trpc.ListSummaryPartsRequest{Actor: actor}},
				{"ListSubscriptionSummaries", &trpc.ListSubscriptionSummariesRequest{Actor: adminActor}},
				{"GetSubscriptionSummary", &trpc.GetSubscriptionSummaryRequest{Actor: adminActor}},
				{"GetTraderSyncRuntimeStatus", &trpc.GetTraderSyncRuntimeStatusRequest{Actor: adminActor}},
			} {
				err := conn.Invoke(auth, "/"+tradersync.HealthServiceName+"/"+call.method, call.request, &trpc.BoolValue{})
				require.Equal(t, codes.Unavailable, status.Code(err), call.method)
			}
			if mode == "owner_loss" {
				_, err = db.Pool.Exec(ctx, `SELECT pg_terminate_backend(pid) FROM pg_locks WHERE locktype='advisory' AND granted AND classid=((hashtextextended('athena:trader-sync:collector',0)>>32)&4294967295)::oid AND objid=(hashtextextended('athena:trader-sync:collector',0)&4294967295)::oid`)
				require.NoError(t, err)
				select {
				case err = <-started:
					require.ErrorIs(t, err, store.ErrRuntimeFenced)
				case <-time.After(2 * time.Second):
					t.Fatal("ownership loss left recovery blocked behind the account lock")
				}
				require.False(t, r.Ready())
				return
			}
			require.NoError(t, blocker.Release(ctx))
			require.NoError(t, <-started)
			require.True(t, r.Ready())
			serving, stopServing := context.WithTimeout(ctx, time.Second)
			defer stopServing()
			health, err = healthpb.NewHealthClient(conn).Check(serving, &healthpb.HealthCheckRequest{Service: tradersync.HealthServiceName})
			require.NoError(t, err)
			require.Equal(t, healthpb.HealthCheckResponse_SERVING, health.Status)
			_, err = trpc.NewTraderSyncServiceClient(conn).ListSubscriptions(metadata.AppendToOutgoingContext(serving, "authorization", "Bearer "+rpcCfg.Token), &trpc.ListSubscriptionsRequest{Actor: actor})
			require.NoError(t, err)

		})
	}
}
