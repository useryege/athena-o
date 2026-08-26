package server

import (
	"context"
	"net/http"
	"os"
	"strings"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"google.golang.org/grpc/metadata"

	"github.com/useryege/athena/internal/accountaccess"
	"github.com/useryege/athena/internal/accountavatar"
	"github.com/useryege/athena/internal/server/accountavatarhttp"
	"github.com/useryege/athena/util/env"
	utilsession "github.com/useryege/athena/util/session"
)

const (
	accountAvatarEndpointEnv     = "ATHENA_ACCOUNT_AVATAR_S3_ENDPOINT"
	accountAvatarRegionEnv       = "ATHENA_ACCOUNT_AVATAR_S3_REGION"
	accountAvatarBucketEnv       = "ATHENA_ACCOUNT_AVATAR_S3_BUCKET"
	accountAvatarAccessKeyEnv    = "ATHENA_ACCOUNT_AVATAR_S3_ACCESS_KEY_ID"
	accountAvatarSecretKeyEnv    = "ATHENA_ACCOUNT_AVATAR_S3_SECRET_ACCESS_KEY"
	accountAvatarPathStyleEnv    = "ATHENA_ACCOUNT_AVATAR_S3_PATH_STYLE"
	accountAvatarMaxBytesEnv     = "ATHENA_ACCOUNT_AVATAR_MAX_BYTES"
	defaultAccountAvatarRegion   = "us-east-1"
	defaultAccountAvatarBucket   = "athena-account-avatars"
	defaultAccountAvatarMaxBytes = 2 * 1024 * 1024
)

func newAccountAvatarHandler(ctx context.Context, server *AthenaServer) (*accountavatarhttp.Handler, error) {
	config := accountavatar.Config{
		Endpoint:     strings.TrimSpace(os.Getenv(accountAvatarEndpointEnv)),
		Region:       stringEnv(accountAvatarRegionEnv, defaultAccountAvatarRegion),
		Bucket:       stringEnv(accountAvatarBucketEnv, defaultAccountAvatarBucket),
		AccessKey:    strings.TrimSpace(os.Getenv(accountAvatarAccessKeyEnv)),
		SecretKey:    os.Getenv(accountAvatarSecretKeyEnv),
		UsePathStyle: env.ParseBoolFromEnv(accountAvatarPathStyleEnv, true),
	}
	store, err := accountavatar.NewStore(ctx, config)
	if err != nil {
		return nil, err
	}
	maxBytes := int64(env.ParseNumFromEnv(
		accountAvatarMaxBytesEnv,
		defaultAccountAvatarMaxBytes,
		1,
		defaultAccountAvatarMaxBytes,
	))
	return accountavatarhttp.NewHandler(
		server.accountCenter,
		store,
		server.authenticateAccountAvatarHTTP,
		maxBytes,
		server.log,
	)
}

func stringEnv(name, fallback string) string {
	value := strings.TrimSpace(os.Getenv(name))
	if value == "" {
		return fallback
	}
	return value
}

func (server *AthenaServer) authenticateAccountAvatarHTTP(request *http.Request, targetAccountID string) (context.Context, error) {
	md := metadata.MD{}
	if authorization := request.Header.Get("Authorization"); authorization != "" {
		md.Set("authorization", authorization)
	}
	if cookie := request.Header.Get("Cookie"); cookie != "" {
		md.Set("grpcgateway-cookie", cookie)
	}
	ctx := metadata.NewIncomingContext(request.Context(), md)
	authenticated, err := server.Authenticate(ctx)
	if err != nil {
		return authenticated, err
	}
	accountID := utilsession.GetUserIdentifier(authenticated)
	if accountID == targetAccountID {
		return authenticated, nil
	}
	if err := server.authorizeAccount(accountID, accountaccess.RequirementAdministrator); err != nil {
		return authenticated, err
	}
	return authenticated, nil
}

func registerAccountAvatarHandlers(mux *http.ServeMux, handler *accountavatarhttp.Handler) {
	if handler == nil {
		return
	}
	mux.Handle("PUT /api/v1/account/{id}/avatar", traceHTTP(http.HandlerFunc(handler.Upload)))
	mux.Handle("GET /api/v1/account/{id}/avatar", traceHTTP(http.HandlerFunc(handler.Download)))
	mux.Handle("DELETE /api/v1/account/{id}/avatar", traceHTTP(http.HandlerFunc(handler.Delete)))
}

func traceHTTP(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		propagator := otel.GetTextMapPropagator()
		ctx := propagator.Extract(r.Context(), propagation.HeaderCarrier(r.Header))
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
