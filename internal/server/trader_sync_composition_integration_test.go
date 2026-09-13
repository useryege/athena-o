//go:build integration

package server

import (
	"context"
	"fmt"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"github.com/useryege/athena/internal/accountcredentials"
	"github.com/useryege/athena/internal/accountstate/schema"
	"github.com/useryege/athena/internal/accountstate/store/migrations"
	"github.com/useryege/athena/internal/notification"
	ns "github.com/useryege/athena/internal/notification/store"
	"github.com/useryege/athena/internal/testutil/pgtest"
	ts "github.com/useryege/athena/internal/tradersync"
	trpc "github.com/useryege/athena/internal/tradersync/apiclient"
	tsstore "github.com/useryege/athena/internal/tradersync/store"
	walletclient "github.com/useryege/athena/internal/wallet/apiclient"
	accountapi "github.com/useryege/athena/pkg/apiclient/account"
	bootstrapapi "github.com/useryege/athena/pkg/apiclient/appbootstrap"
	api "github.com/useryege/athena/pkg/apiclient/tradersync"
	"github.com/useryege/athena/util/telegram"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

type fixedNotificationProfile struct{}

func (fixedNotificationProfile) SyncProfile(context.Context) (*telegram.BotIdentity, error) {
	return &telegram.BotIdentity{ID: 1, Username: "fixture_bot"}, nil
}

func TestAPIAndNotificationStartWithoutCollectorConfiguration(t *testing.T) {
	db := pgtest.New(t, migrations.FS, migrations.Dir)
	ctx := context.Background()
	t.Setenv(schema.DSNEnv, db.DSN)
	t.Setenv("ATHENA_JWT_SECRET", strings.Repeat("j", 32))
	t.Setenv("ATHENA_URL", "http://localhost:4000")
	for _, key := range []string{"ATHENA_TRADER_SYNC_HTTP_URL", "ATHENA_TRADER_SYNC_WSS_URL", "ATHENA_TRADER_SYNC_CURSOR_HMAC_KEY"} {
		t.Setenv(key, "broken")
	}
	// Invalid TLS/token/address is a local facade outage; construction still succeeds.
	t.Setenv("ATHENA_TRADER_SYNC_SERVER_ADDRESS", "bad address")
	t.Setenv("ATHENA_TRADER_SYNC_INTERNAL_AUTH_TOKEN", "")
	t.Setenv("ATHENA_TRADER_SYNC_TLS_CA_FILE", "/missing-ca-fixture")
	objects := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		fmt.Fprint(w, `<ListBucketResult xmlns="http://s3.amazonaws.com/doc/2006-03-01/"><IsTruncated>false</IsTruncated></ListBucketResult>`)
	}))
	defer objects.Close()
	t.Setenv(accountAvatarEndpointEnv, objects.URL)
	t.Setenv(accountAvatarAccessKeyEnv, "fixture")
	t.Setenv(accountAvatarSecretKeyEnv, "fixture")
	wallet, err := walletclient.NewWalletClientset("127.0.0.1:1", strings.Repeat("w", 32))
	if err != nil {
		t.Fatal(err)
	}
	defer wallet.Close()
	redisClient := redis.NewClient(&redis.Options{Addr: "127.0.0.1:1"})
	defer redisClient.Close()
	server, err := NewServer(ctx, AthenaServerOpts{DisableAuth: true, ListenHost: "127.0.0.1", StaticAssetsDir: t.TempDir(), RedisClient: redisClient, WalletClientset: wallet})
	if err != nil {
		t.Fatal(err)
	}
	defer server.Close()
	server.serviceSet = newAthenaServiceSet(server)
	member := server.developmentAccountIDs[accountcredentials.ApplicationRealmMember]
	memberCtx := context.WithValue(ctx, "claims", jwt.MapClaims{"sub": member})
	for _, id := range []string{member, strings.ToUpper(member)} {
		actor, err := server.resolveTraderSyncActor(context.WithValue(ctx, "claims", jwt.MapClaims{"sub": id}))
		if err != nil || actor.AccountId != member || actor.Realm != trpc.ApplicationRealm_APPLICATION_REALM_MEMBER {
			t.Fatal("canonical persisted actor", actor, err)
		}
	}
	admin := server.developmentAccountIDs[accountcredentials.ApplicationRealmAdmin]
	actor, err := server.resolveTraderSyncActor(context.WithValue(ctx, "claims", jwt.MapClaims{"sub": admin}))
	if err != nil || actor.Realm != trpc.ApplicationRealm_APPLICATION_REALM_ADMIN {
		t.Fatal(actor, err)
	}
	for _, id := range []string{"invalid", uuid.NewString()} {
		_, err = server.resolveTraderSyncActor(context.WithValue(ctx, "claims", jwt.MapClaims{"sub": id}))
		if status.Code(err) != codes.Unavailable {
			t.Fatal("untrusted actor became public auth error", err)
		}
	}
	_, err = server.resolveTraderSyncActor(ctx)
	if status.Code(err) != codes.Unauthenticated {
		t.Fatal(err)
	}
	_, err = server.serviceSet.TraderSyncService.ListSubscriptions(memberCtx, &api.ListSubscriptionsRequest{})
	if status.Code(err) != codes.Unavailable {
		t.Fatal(err)
	}
	if result, err := server.serviceSet.AppBootstrapService.GetAppBootstrap(memberCtx, &bootstrapapi.GetAppBootstrapRequest{}); err != nil || result.GetSession().GetUserInfo().GetAccountId() != member {
		t.Fatal("bootstrap unavailable", result, err)
	}
	if result, err := server.serviceSet.AccountService.GetAccount(memberCtx, &accountapi.GetAccountRequest{Id: member}); err != nil || result == nil {
		t.Fatal("account unavailable", err)
	}

	notificationStore, err := ns.NewSQLStoreSource()(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer notificationStore.Close()
	notificationPool, err := notificationStore.BorrowPool()
	if err != nil {
		t.Fatal(err)
	}
	traderPool, err := schema.ConnectVerified(ctx, db.DSN)
	if err != nil {
		t.Fatal(err)
	}
	defer traderPool.Close()
	pools := []*pgxpool.Pool{server.accountStateStore.Pool(), notificationPool, traderPool}
	var identity string
	seenPID := map[int32]bool{}
	for i, pool := range pools {
		for j := 0; j < i; j++ {
			if pool == pools[j] {
				t.Fatal("processes shared a pool")
			}
		}
		var current string
		var pid int32
		if err = pool.QueryRow(ctx, `SELECT current_database()||':'||(SELECT oid::text FROM pg_database WHERE datname=current_database())||':'||inet_server_addr()::text||':'||inet_server_port()::text,pg_backend_pid()`).Scan(&current, &pid); err != nil {
			t.Fatal(err)
		}
		if i == 0 {
			identity = current
		} else if current != identity {
			t.Fatal("processes point to different database identities")
		}
		if seenPID[pid] {
			t.Fatal("processes shared backend connection")
		}
		seenPID[pid] = true
	}
	telegramHTTP := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case strings.HasSuffix(r.URL.Path, "getWebhookInfo"):
			fmt.Fprint(w, `{"ok":true,"result":{"url":""}}`)
		case strings.HasSuffix(r.URL.Path, "getUpdates"):
			select {
			case <-r.Context().Done():
				return
			case <-time.After(20 * time.Millisecond):
			}
			fmt.Fprint(w, `{"ok":true,"result":[]}`)
		case strings.HasSuffix(r.URL.Path, "sendMessage"):
			fmt.Fprint(w, `{"ok":true,"result":{"message_id":42}}`)
		default:
			t.Errorf("unexpected fixture Telegram operation %s", r.URL.Path)
			w.WriteHeader(500)
		}
	}))
	defer telegramHTTP.Close()
	telegramClient, err := telegram.NewClient(telegram.Config{BotToken: "fixture", BaseURL: telegramHTTP.URL})
	if err != nil {
		t.Fatal(err)
	}
	notifier := notification.NewService(notificationStore, notification.NewTelegramSender(telegramClient, map[string]string{"test": "-123"}), fixedNotificationProfile{}, notification.NewTelegramPoller(notificationStore, telegramClient))
	if err = notifier.ConfigureSummaries(notificationPool, "http://localhost:4000"); err != nil {
		t.Fatal(err)
	}
	if err = notifier.ConfigureSummaries(traderPool, "http://localhost:4000"); err == nil {
		t.Fatal("borrowed pool mismatch accepted")
	}
	if err = notifier.Start(ctx); err != nil {
		t.Fatal(err)
	}
	defer notifier.Stop()
	// The facade reaches the real service through gRPC on its own database pool.
	composition, err := newGatewayServiceFixture(ctx, ts.Config{HTTPURL: "http://127.0.0.1:1", WebSocketURL: "ws://127.0.0.1:1", SiteURL: "http://localhost:4000", CursorHMACKey: "fixture-cursor"}, traderPool, tsstore.NewSQLStore(traderPool))
	if err != nil {
		t.Fatal(err)
	}
	defer composition.Close()
	internalClient, internalServer := newGatewayInternalClient(t, composition.service)
	server.TraderSyncClient = internalClient
	server.serviceSet = newAthenaServiceSet(server)
	adminCtx := context.WithValue(ctx, "claims", jwt.MapClaims{"sub": admin})
	if _, err = server.serviceSet.TraderSyncService.GetTraderSyncRuntimeStatus(adminCtx, &api.GetTraderSyncRuntimeStatusRequest{}); err != nil {
		t.Fatal("live internal service", err)
	}
	internalServer.Stop()
	if err = composition.Close(); err != nil {
		t.Fatal(err)
	}
	traderPool.Close()
	if _, err = server.serviceSet.TraderSyncService.GetTraderSyncRuntimeStatus(adminCtx, &api.GetTraderSyncRuntimeStatusRequest{}); status.Code(err) != codes.Unavailable {
		t.Fatal("stopped internal service", err)
	}
	if _, err = notificationPool.Exec(ctx, `INSERT INTO system_notification_topics(telegram_chat,label,message_thread_id) VALUES('test','fixture',1)`); err != nil {
		t.Fatal(err)
	}
	var deliveryID int64
	if err = notificationPool.QueryRow(ctx, `INSERT INTO system_notification_deliveries(source,severity,body,channel,status,telegram_chat,topic_label,payload,payload_digest) VALUES('test','info','still alive','telegram','pending','test','fixture',convert_to(json_build_object('format','html','text','still alive','messageThreadId',0)::text,'UTF8'),sha256(convert_to(json_build_object('format','html','text','still alive','messageThreadId',0)::text,'UTF8'))) RETURNING id`).Scan(&deliveryID); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(5 * time.Second)
	for {
		var state string
		if err = notificationPool.QueryRow(ctx, `SELECT status FROM system_notification_deliveries WHERE id=$1`, deliveryID).Scan(&state); err != nil {
			t.Fatal(err)
		}
		if state == "sent" {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("Notification stopped with Trader Sync", state)
		}
		time.Sleep(25 * time.Millisecond)
	}
	if _, err = server.serviceSet.AppBootstrapService.GetAppBootstrap(memberCtx, &bootstrapapi.GetAppBootstrapRequest{}); err != nil {
		t.Fatal(err)
	}
	if _, err = server.serviceSet.AccountService.GetAccount(memberCtx, &accountapi.GetAccountRequest{Id: member}); err != nil {
		t.Fatal(err)
	}
}
