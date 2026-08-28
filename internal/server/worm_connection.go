package server

import (
	"context"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/useryege/athena/common"
	"github.com/useryege/athena/internal/accountaccess"
	"github.com/useryege/athena/internal/accountcredentials"
	walletapiclient "github.com/useryege/athena/internal/wallet/apiclient"
	"github.com/useryege/athena/internal/walletsecret"
	wormtradingapiclient "github.com/useryege/athena/internal/wormtrading/apiclient"
	httputil "github.com/useryege/athena/util/http"
	utilsession "github.com/useryege/athena/util/session"
)

const (
	wormConnectionPathPrefix    = "/api/v1/worm-trading/wallet-connections/"
	wormReconnectSuffix         = ":reconnect"
	developmentWormPublicOrigin = "http://localhost:4000"
)

type wormConnectionAction string

const (
	wormConnectionConnect    wormConnectionAction = "connect"
	wormConnectionReconnect  wormConnectionAction = "reconnect"
	wormConnectionDisconnect wormConnectionAction = "disconnect"
)

type wormConnectionResponse struct {
	WalletID    int64  `json:"walletId"`
	Address     string `json:"address"`
	State       string `json:"state"`
	WarningCode string `json:"warningCode,omitempty"`
	ConnectedAt int64  `json:"connectedAt,omitempty"`
}

func registerWormConnectionHandlers(mux *http.ServeMux, server *AthenaServer) {
	if mux == nil || server == nil || server.wormCredentialMgr == nil || server.WalletClientset == nil || server.WormTradingClientset == nil {
		return
	}
	handler := traceHTTP(http.HandlerFunc(server.manageWormConnection))
	mux.Handle("POST "+wormConnectionPathPrefix+"{connectionResource}", handler)
	mux.Handle("DELETE "+wormConnectionPathPrefix+"{connectionResource}", handler)
}

func (server *AthenaServer) manageWormConnection(w http.ResponseWriter, request *http.Request) {
	walletsecret.SetSecretResponseHeaders(w)
	if !server.validWormConnectionOrigin(request) {
		walletsecret.WriteError(w, walletsecret.ErrWormReauthenticationRequired)
		return
	}
	walletID, action, err := wormConnectionTarget(request)
	if err != nil {
		walletsecret.WriteError(w, err)
		return
	}
	ctx, credential, err := server.authenticateWormConnectionHTTP(request)
	if err != nil {
		walletsecret.WriteError(w, err)
		return
	}
	if err := server.wormCredentialMgr.Validate(ctx, request, credential); err != nil {
		walletsecret.WriteError(w, err)
		return
	}

	walletResponse, err := server.WalletClientset.Wallet().GetWallet(ctx, &walletapiclient.GetWalletRequest{
		Id:                 walletID,
		RequesterAccountId: credential.AccountID,
	})
	if err != nil {
		walletsecret.WriteError(w, sanitizeWormConnectionDependencyError(err, "Wallet"))
		return
	}
	wallet := walletResponse.GetItem()
	if wallet == nil || wallet.ID != walletID || wallet.WalletType != "SOLANA" || strings.TrimSpace(wallet.Address) == "" {
		walletsecret.WriteError(w, status.Error(codes.FailedPrecondition, "the selected wallet is not an owned Solana wallet"))
		return
	}

	var connection *wormtradingapiclient.WormWalletConnection
	if action == wormConnectionDisconnect {
		response, callErr := server.WormTradingClientset.WormTrading().DisconnectWormWallet(ctx, &wormtradingapiclient.DisconnectWormWalletRequest{
			WalletId: walletID,
			Address:  wallet.Address,
		})
		if callErr != nil {
			walletsecret.WriteError(w, sanitizeWormConnectionDependencyError(callErr, "Worm Trading"))
			return
		}
		connection = response.GetConnection()
	} else {
		connection, err = server.completeWormConnection(ctx, credential.AccountID, walletID, wallet.Address, action == wormConnectionReconnect)
		if err != nil {
			walletsecret.WriteError(w, err)
			return
		}
	}
	if connection == nil || connection.GetWalletId() != walletID || connection.GetAddress() != wallet.Address || strings.TrimSpace(connection.GetState()) == "" {
		walletsecret.WriteError(w, status.Error(codes.Internal, "Worm Trading returned an invalid connection result"))
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(wormConnectionResponse{
		WalletID:    connection.GetWalletId(),
		Address:     connection.GetAddress(),
		State:       connection.GetState(),
		WarningCode: connection.GetWarningCode(),
		ConnectedAt: connection.GetConnectedAt(),
	})
}

func (server *AthenaServer) completeWormConnection(ctx context.Context, accountID string, walletID int64, address string, reconnect bool) (*wormtradingapiclient.WormWalletConnection, error) {
	prepared, err := server.WormTradingClientset.WormTrading().PrepareWormWalletConnection(ctx, &wormtradingapiclient.PrepareWormWalletConnectionRequest{
		WalletId:  walletID,
		Address:   address,
		Reconnect: reconnect,
	})
	if err != nil {
		return nil, sanitizeWormConnectionDependencyError(err, "Worm Trading")
	}
	if prepared.GetAttemptId() == "" || prepared.GetWalletId() != walletID || prepared.GetAddress() != address ||
		prepared.GetNonce() == "" || prepared.GetMessage() == "" || len(prepared.GetMessageSha256()) != 32 {
		return nil, status.Error(codes.Internal, "Worm Trading returned an invalid credential challenge")
	}

	signed, err := server.WalletClientset.Wallet().SignWormAuthChallenge(ctx, &walletapiclient.SignWormAuthChallengeRequest{
		Id:                 walletID,
		RequesterAccountId: accountID,
		ExpectedAddress:    address,
		Nonce:              prepared.GetNonce(),
		Message:            prepared.GetMessage(),
	})
	if err != nil {
		return nil, sanitizeWormConnectionDependencyError(err, "Wallet")
	}
	signature, err := hex.DecodeString(signed.GetSignature())
	if err != nil || len(signature) != 64 {
		return nil, status.Error(codes.Internal, "Wallet returned an invalid Worm challenge signature")
	}
	messageDigest, err := hex.DecodeString(signed.GetMessageSha256())
	if err != nil || len(messageDigest) != 32 || subtle.ConstantTimeCompare(messageDigest, prepared.GetMessageSha256()) != 1 {
		return nil, status.Error(codes.Internal, "Wallet returned a mismatched Worm challenge digest")
	}

	completed, err := server.WormTradingClientset.WormTrading().CompleteWormWalletConnection(ctx, &wormtradingapiclient.CompleteWormWalletConnectionRequest{
		AttemptId:     prepared.GetAttemptId(),
		WalletId:      walletID,
		Address:       address,
		MessageSha256: messageDigest,
		Signature:     signature,
	})
	if err != nil {
		return nil, sanitizeWormConnectionDependencyError(err, "Worm Trading")
	}
	return completed.GetConnection(), nil
}

func (server *AthenaServer) authenticateWormConnectionHTTP(request *http.Request) (context.Context, accountcredentials.AuthenticatedCredential, error) {
	if server.DisableAuth {
		if !requestIsLoopback(request) {
			return request.Context(), accountcredentials.AuthenticatedCredential{}, walletsecret.ErrWormLoginSessionRequired
		}
		ctx := withDisabledAuthClaims(request.Context(), server.developmentAccountID)
		credential, ok := utilsession.AuthenticatedCredentialFromContext(ctx)
		if !ok || !credential.IsInteractiveLogin() {
			return ctx, accountcredentials.AuthenticatedCredential{}, walletsecret.ErrWormLoginSessionRequired
		}
		access, err := server.accessController.Get(credential.AccountID)
		if err != nil {
			return ctx, accountcredentials.AuthenticatedCredential{}, err
		}
		if err := server.accessController.Authorize(credential.AccountID, accountaccess.RequireModule(accountaccess.ModuleWormTrading, accountaccess.AccessLevelReadWrite)); err != nil {
			return ctx, accountcredentials.AuthenticatedCredential{}, err
		}
		credential.AccessRevision = access.Revision
		ctx = utilsession.WithAuthenticatedCredential(ctx, credential)
		return ctx, credential, nil
	}

	token, err := httputil.JoinCookies(common.AuthCookieName, request.Cookies())
	if err != nil || token == "" {
		return request.Context(), accountcredentials.AuthenticatedCredential{}, walletsecret.ErrWormLoginSessionRequired
	}
	claims, credential, err := server.sessionMgr.AuthenticateToken(token)
	if err != nil || !credential.IsInteractiveLogin() {
		return request.Context(), accountcredentials.AuthenticatedCredential{}, walletsecret.ErrWormLoginSessionRequired
	}
	if err := server.accessController.Authorize(credential.AccountID, accountaccess.RequireModule(accountaccess.ModuleWormTrading, accountaccess.AccessLevelReadWrite)); err != nil {
		return request.Context(), accountcredentials.AuthenticatedCredential{}, err
	}
	ctx := context.WithValue(request.Context(), "claims", claims) //nolint:staticcheck
	ctx = utilsession.WithAuthenticatedCredential(ctx, credential)
	return ctx, credential, nil
}

func (server *AthenaServer) developmentWormCredentialLease(w http.ResponseWriter, request *http.Request) {
	walletsecret.SetSecretResponseHeaders(w)
	if request.Method != http.MethodPost {
		w.Header().Set("Allow", "POST")
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	if !server.DisableAuth || !requestIsLoopback(request) || !server.validWormConnectionOrigin(request) {
		walletsecret.WriteError(w, walletsecret.ErrWormLoginSessionRequired)
		return
	}
	ctx, credential, err := server.authenticateWormConnectionHTTP(request)
	if err != nil {
		walletsecret.WriteError(w, err)
		return
	}
	expiresAt, err := server.wormCredentialMgr.Issue(ctx, w, credential)
	if err != nil {
		walletsecret.WriteError(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(struct {
		ExpiresAt int64 `json:"expiresAt"`
	}{ExpiresAt: expiresAt.Unix()})
}

func (server *AthenaServer) validWormConnectionOrigin(request *http.Request) bool {
	expectedOrigin := server.walletSecretPublicOrigin
	if server.DisableAuth {
		expectedOrigin = developmentWormPublicOrigin
	}
	origin := strings.TrimSpace(request.Header.Get("Origin"))
	return origin != "" && expectedOrigin != "" && subtle.ConstantTimeCompare([]byte(origin), []byte(expectedOrigin)) == 1
}

func wormConnectionTarget(request *http.Request) (int64, wormConnectionAction, error) {
	resource := strings.TrimSpace(request.PathValue("connectionResource"))
	if resource == "" && strings.HasPrefix(request.URL.Path, wormConnectionPathPrefix) {
		resource = strings.TrimPrefix(request.URL.Path, wormConnectionPathPrefix)
	}
	if resource == "" || strings.Contains(resource, "/") {
		return 0, "", status.Error(codes.NotFound, "Worm wallet connection resource not found")
	}
	action := wormConnectionConnect
	switch request.Method {
	case http.MethodPost:
		if strings.HasSuffix(resource, wormReconnectSuffix) {
			action = wormConnectionReconnect
			resource = strings.TrimSuffix(resource, wormReconnectSuffix)
		}
	case http.MethodDelete:
		if strings.HasSuffix(resource, wormReconnectSuffix) {
			return 0, "", status.Error(codes.NotFound, "Worm wallet connection resource not found")
		}
		action = wormConnectionDisconnect
	default:
		return 0, "", status.Error(codes.NotFound, "Worm wallet connection resource not found")
	}
	walletID, err := strconv.ParseInt(resource, 10, 64)
	if err != nil || walletID <= 0 {
		return 0, "", status.Error(codes.InvalidArgument, "wallet ID must be a positive integer")
	}
	return walletID, action, nil
}

func sanitizeWormConnectionDependencyError(err error, dependency string) error {
	if status.Code(err) == codes.Aborted && strings.Contains(strings.ToLower(status.Convert(err).Message()), "outcome is unknown") {
		return walletsecret.ErrWormConnectOutcomeUnknown
	}
	switch status.Code(err) {
	case codes.InvalidArgument, codes.FailedPrecondition, codes.AlreadyExists, codes.Aborted, codes.NotFound, codes.ResourceExhausted:
		return err
	case codes.Unauthenticated, codes.PermissionDenied:
		return status.Errorf(codes.FailedPrecondition, "%s rejected the Worm wallet connection operation", dependency)
	case codes.DeadlineExceeded, codes.Unavailable:
		return status.Errorf(codes.Unavailable, "%s is unavailable", dependency)
	default:
		return status.Errorf(codes.Internal, "%s could not complete the Worm wallet connection operation", dependency)
	}
}
