//go:build integration

package transport

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/useryege/athena/internal/accountcredentials"
	accountstore "github.com/useryege/athena/internal/accountstate/store"
	"github.com/useryege/athena/internal/accountstate/store/migrations"
	"github.com/useryege/athena/internal/testutil/pgtest"
	ts "github.com/useryege/athena/internal/tradersync"
	trpc "github.com/useryege/athena/internal/tradersync/apiclient"
	"github.com/useryege/athena/internal/tradersync/store"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func TestBufconnDatabaseAuthorityOverridesValidActor(t *testing.T) {
	db := pgtest.New(t, migrations.FS, migrations.Dir)
	ctx := context.Background()
	account, err := accountstore.NewSQLStore(db.Pool).EnsureDevelopmentAccount(ctx, accountcredentials.DevelopmentRoleMember)
	require.NoError(t, err)
	collector := &ts.Collector{}
	subscriptions, err := ts.NewSubscriptionService(db.Pool, store.NewSQLStore(db.Pool), &ts.TargetResolver{}, collector)
	require.NoError(t, err)
	service, err := ts.NewService(ts.Config{HTTPURL: "https://node.example", WebSocketURL: "wss://node.example", SiteURL: "https://athena.test", CursorHMACKey: "cursor-test-key"}, ts.Dependencies{Pool: db.Pool, Resolver: &ts.TargetResolver{}, Subscriptions: subscriptions, Collector: collector, Projector: &ts.Projector{}, Directory: &ts.DirectoryRefresher{}})
	require.NoError(t, err)
	client, _ := bufClient(t, NewServer(service), func() bool { return true })
	ctx = metadata.NewOutgoingContext(ctx, metadata.Pairs("authorization", "Bearer "+authToken))
	actor := &trpc.Actor{AccountId: account.ID, Realm: trpc.ApplicationRealm_APPLICATION_REALM_MEMBER}
	// A member row claiming administrator in its Actor is still rejected by DB authority.
	_, err = client.GetTraderSyncRuntimeStatus(ctx, &trpc.GetTraderSyncRuntimeStatusRequest{Actor: &trpc.Actor{AccountId: account.ID, Realm: trpc.ApplicationRealm_APPLICATION_REALM_ADMIN}})
	require.Equal(t, codes.PermissionDenied, status.Code(err))
	require.Empty(t, status.Convert(err).Details())
	// The same authenticated identity becomes ineligible immediately after grant revocation.
	_, err = db.Pool.Exec(ctx, `UPDATE account_module_access SET access_level='none' WHERE account_id=$1 AND module='trader_sync'`, account.ID)
	require.NoError(t, err)
	_, err = client.GetActivity(ctx, &trpc.GetActivityRequest{Actor: actor, ActivityId: "1"})
	require.Equal(t, codes.PermissionDenied, status.Code(err))
	require.Empty(t, status.Convert(err).Details())
	_, err = client.CreateSubscription(ctx, &trpc.CreateSubscriptionRequest{Actor: actor, ConfirmationToken: "unused", RequestId: "bdcbe504-c2e7-4d61-8580-fb99e7341d38"})
	require.Equal(t, codes.PermissionDenied, status.Code(err))
}
