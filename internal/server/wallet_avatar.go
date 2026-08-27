package server

import (
	"context"
	"fmt"
	"net/http"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"github.com/useryege/athena/internal/accountaccess"
	"github.com/useryege/athena/internal/server/walletavatarhttp"
	walletapiclient "github.com/useryege/athena/internal/wallet/apiclient"
	"github.com/useryege/athena/pkg/apis/application/v1alpha1"
	utilsession "github.com/useryege/athena/util/session"
)

type walletAvatarRepository struct {
	client walletapiclient.WalletServiceClient
}

func newWalletAvatarHandler(ctx context.Context, server *AthenaServer) (*walletavatarhttp.Handler, error) {
	if server.WalletClientset == nil || server.WalletClientset.Wallet() == nil {
		return nil, fmt.Errorf("wallet client is required for wallet avatars")
	}
	objects, maxBytes, err := newPrivateAvatarStore(ctx)
	if err != nil {
		return nil, err
	}
	return walletavatarhttp.NewHandler(
		&walletAvatarRepository{client: server.WalletClientset.Wallet()},
		objects,
		server.authenticateWalletAvatarHTTP,
		maxBytes,
		server.log,
	)
}

func (server *AthenaServer) authenticateWalletAvatarHTTP(request *http.Request, write bool) (context.Context, string, error) {
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
		return authenticated, "", err
	}
	accountID := utilsession.AccountID(authenticated)
	if accountID == "" {
		return authenticated, "", status.Error(codes.Unauthenticated, "authenticated account ID is missing")
	}
	level := accountaccess.AccessLevelRead
	if write {
		level = accountaccess.AccessLevelReadWrite
	}
	if err := server.authorizeAccount(accountID, accountaccess.RequireModule(accountaccess.ModuleWallet, level)); err != nil {
		return authenticated, "", err
	}
	return authenticated, accountID, nil
}

func (r *walletAvatarRepository) GetWalletAvatar(ctx context.Context, ownerAccountID string, walletID int64) (walletavatarhttp.Wallet, error) {
	response, err := r.client.GetWalletAvatarMetadata(ctx, &walletapiclient.GetWalletAvatarMetadataRequest{
		Id: walletID, RequesterAccountId: ownerAccountID,
	})
	if err != nil {
		return walletavatarhttp.Wallet{}, err
	}
	return walletAvatarFromResponse(response.GetItem(), response.GetMetadata()), nil
}

func (r *walletAvatarRepository) ReplaceWalletAvatar(ctx context.Context, ownerAccountID string, walletID int64, avatar walletavatarhttp.AvatarMetadata, expectedRevision uint64) (walletavatarhttp.Wallet, error) {
	response, err := r.client.ReplaceWalletAvatarMetadata(ctx, &walletapiclient.ReplaceWalletAvatarMetadataRequest{
		Id:                 walletID,
		ExpectedRevision:   expectedRevision,
		ObjectKey:          avatar.ObjectKey,
		ContentType:        avatar.ContentType,
		Etag:               avatar.ETag,
		SizeBytes:          avatar.SizeBytes,
		RequesterAccountId: ownerAccountID,
	})
	if err != nil {
		return walletavatarhttp.Wallet{}, err
	}
	return walletavatarhttp.Wallet{Item: response.GetItem(), Avatar: avatar}, nil
}

func (r *walletAvatarRepository) ResetWalletAvatar(ctx context.Context, ownerAccountID string, walletID int64, expectedRevision uint64) (walletavatarhttp.Wallet, error) {
	response, err := r.client.ResetWalletAvatarMetadata(ctx, &walletapiclient.ResetWalletAvatarMetadataRequest{
		Id: walletID, ExpectedRevision: expectedRevision, RequesterAccountId: ownerAccountID,
	})
	if err != nil {
		return walletavatarhttp.Wallet{}, err
	}
	return walletavatarhttp.Wallet{Item: response.GetItem()}, nil
}

func (r *walletAvatarRepository) ListWalletAvatarObjectKeys(ctx context.Context) ([]string, error) {
	response, err := r.client.ListWalletAvatarObjectKeys(ctx, &walletapiclient.ListWalletAvatarObjectKeysRequest{})
	if err != nil {
		return nil, err
	}
	return append([]string(nil), response.GetObjectKeys()...), nil
}

func walletAvatarFromResponse(item *v1alpha1.WalletItem, metadata *walletapiclient.WalletAvatarMetadata) walletavatarhttp.Wallet {
	wallet := walletavatarhttp.Wallet{Item: item}
	if metadata != nil {
		wallet.Avatar = walletavatarhttp.AvatarMetadata{
			ObjectKey: metadata.GetObjectKey(), ContentType: metadata.GetContentType(),
			ETag: metadata.GetEtag(), SizeBytes: metadata.GetSizeBytes(),
		}
	}
	return wallet
}

func registerWalletAvatarHandlers(mux *http.ServeMux, handler *walletavatarhttp.Handler) {
	if handler == nil {
		return
	}
	mux.Handle("PUT /api/v1/wallets/{id}/avatar", traceHTTP(http.HandlerFunc(handler.Upload)))
	mux.Handle("GET /api/v1/wallets/{id}/avatar", traceHTTP(http.HandlerFunc(handler.Download)))
	mux.Handle("DELETE /api/v1/wallets/{id}/avatar", traceHTTP(http.HandlerFunc(handler.Delete)))
}
