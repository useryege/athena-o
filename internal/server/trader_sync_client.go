package server

import (
	"context"
	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/useryege/athena/internal/accountcredentials"
	trpc "github.com/useryege/athena/internal/tradersync/apiclient"
	"github.com/useryege/athena/internal/tradersync/rpcconfig"
	"github.com/useryege/athena/util/session"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"io"
)

func newTraderSyncClient(lookup func(string) (string, bool)) (trpc.TraderSyncServiceClient, io.Closer) {
	cfg, err := rpcconfig.LoadClient(lookup)
	if err == nil {
		var client trpc.TraderSyncServiceClient
		var closer io.Closer
		client, closer, err = trpc.NewClient(cfg)
		if err == nil {
			return client, closer
		}
	}
	// Configuration errors may contain paths; log only the fixed dependency category.
	log.WithField("reason", "client_configuration_invalid").Warn("Trader Sync facade unavailable")
	return trpc.NewUnavailableClient("client_configuration_invalid"), nil
}

func (server *AthenaServer) resolveTraderSyncActor(ctx context.Context) (*trpc.Actor, error) {
	rawID := session.AccountID(ctx)
	if rawID == "" {
		return nil, status.Error(codes.Unauthenticated, "account identity required")
	}
	id, err := uuid.Parse(rawID)
	if err != nil || id == uuid.Nil || server.credentialMgr == nil {
		return nil, status.Error(codes.Unavailable, "Trader Sync actor identity unavailable")
	}
	account, err := server.credentialMgr.Get(id.String())
	if err != nil {
		return nil, status.Error(codes.Unavailable, "Trader Sync actor account unavailable")
	}
	var realm trpc.ApplicationRealm
	switch account.ApplicationRealm() {
	case accountcredentials.ApplicationRealmMember:
		realm = trpc.ApplicationRealm_APPLICATION_REALM_MEMBER
	case accountcredentials.ApplicationRealmAdmin:
		realm = trpc.ApplicationRealm_APPLICATION_REALM_ADMIN
	default:
		return nil, status.Error(codes.Unavailable, "Trader Sync actor realm unavailable")
	}
	return &trpc.Actor{AccountId: id.String(), Realm: realm}, nil
}
