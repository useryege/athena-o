package server

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"net/url"
	"strings"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/useryege/athena/internal/accountaccess"
	"github.com/useryege/athena/internal/accountcredentials"
	"github.com/useryege/athena/internal/authregistration"
	"github.com/useryege/athena/internal/server/walletsecrethttp"
	walletapiclient "github.com/useryege/athena/internal/wallet/apiclient"
	"github.com/useryege/athena/internal/walletsecret"
	utilsession "github.com/useryege/athena/util/session"
)

func newWalletSecretHTTPHandler(server *AthenaServer) (*walletsecrethttp.Handler, error) {
	if server == nil || server.WalletClientset == nil {
		return nil, status.Error(codes.Internal, "wallet client is not configured")
	}
	return walletsecrethttp.NewHandler(
		server.authenticateWalletSecretHTTP,
		server.walletSecretMgr,
		func(ctx context.Context, accountID string, walletID int64) (string, error) {
			response, err := server.WalletClientset.Wallet().RevealWalletPrivateKey(ctx, &walletapiclient.RevealWalletPrivateKeyRequest{
				Id:                 walletID,
				RequesterAccountId: accountID,
			})
			if err != nil {
				return "", err
			}
			return response.GetPrivateKey(), nil
		},
		server.validWalletSecretOrigin,
	)
}

func registerWalletSecretHandlers(mux *http.ServeMux, handler *walletsecrethttp.Handler) {
	if handler == nil {
		return
	}
	// net/http wildcards occupy a complete path segment, so this registration
	// owns only one POST resource below /wallets. The handler still validates
	// the exact `{id}:revealPrivateKey` suffix before authenticating or calling
	// Wallet, while collection routes continue to the gRPC gateway.
	mux.Handle("POST /api/v1/wallets/{secretResource}", traceHTTP(http.HandlerFunc(handler.Reveal)))
}

func (server *AthenaServer) authenticateWalletSecretHTTP(request *http.Request) (context.Context, accountcredentials.AuthenticatedCredential, error) {
	if server.DisableAuth {
		if !requestIsLoopback(request) {
			return request.Context(), accountcredentials.AuthenticatedCredential{}, walletsecret.ErrLoginSessionRequired
		}
		developmentAccountID, err := server.developmentAccountIDFromHTTPRequest(request, false)
		if err != nil {
			return request.Context(), accountcredentials.AuthenticatedCredential{}, err
		}
		ctx := withDisabledAuthClaims(request.Context(), developmentAccountID)
		credential, ok := utilsession.AuthenticatedCredentialFromContext(ctx)
		if !ok {
			return ctx, accountcredentials.AuthenticatedCredential{}, walletsecret.ErrLoginSessionRequired
		}
		access, err := server.accessController.Get(credential.AccountID)
		if err != nil {
			return ctx, accountcredentials.AuthenticatedCredential{}, err
		}
		if err := server.accessController.Authorize(credential.AccountID, accountaccess.RequireModule(accountaccess.ModuleWallet, accountaccess.AccessLevelReadWrite)); err != nil {
			return ctx, accountcredentials.AuthenticatedCredential{}, err
		}
		credential.AccessRevision = access.Revision
		ctx = utilsession.WithAuthenticatedCredential(ctx, credential)
		return ctx, credential, nil
	}

	claims, credential, err := server.authenticateRealmLoginCookie(request, false)
	if err != nil {
		return request.Context(), accountcredentials.AuthenticatedCredential{}, walletsecret.ErrLoginSessionRequired
	}
	if err := server.accessController.Authorize(credential.AccountID, accountaccess.RequireModule(accountaccess.ModuleWallet, accountaccess.AccessLevelReadWrite)); err != nil {
		return request.Context(), accountcredentials.AuthenticatedCredential{}, err
	}
	ctx := context.WithValue(request.Context(), "claims", claims) //nolint:staticcheck
	ctx = utilsession.WithAuthenticatedCredential(ctx, credential)
	return ctx, credential, nil
}

func (server *AthenaServer) developmentWalletSecretLease(w http.ResponseWriter, r *http.Request) {
	observed := false
	accountID := ""
	defer func() {
		if !observed {
			observeWormAuthorization(r.Context(), "account", accountID, "WALLET_REVEAL_AUTHORIZE", "DEVELOPMENT", "authorization_failed", false, operationLogHTTPStatus(w))
		}
	}()
	walletsecret.SetSecretResponseHeaders(w)
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", "POST")
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	if !server.DisableAuth || !requestIsLoopback(r) || !server.validWalletSecretOrigin(r) {
		walletsecret.WriteError(w, walletsecret.ErrLoginSessionRequired)
		return
	}
	ctx, credential, err := server.authenticateWalletSecretHTTP(r)
	if err != nil {
		walletsecret.WriteError(w, err)
		return
	}
	accountID = credential.AccountID
	expiresAt, err := server.walletSecretMgr.Issue(ctx, w, credential)
	if err != nil {
		walletsecret.WriteError(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(struct {
		ExpiresAt int64 `json:"expiresAt"`
	}{ExpiresAt: expiresAt.Unix()})
	observeWormAuthorization(r.Context(), "account", accountID, "WALLET_REVEAL_AUTHORIZE", "DEVELOPMENT", "authorization_verified", true, operationLogHTTPStatus(w))
	observed = true
}

func requestIsLoopback(request *http.Request) bool {
	host, _, err := net.SplitHostPort(strings.TrimSpace(request.RemoteAddr))
	if err != nil {
		host = strings.Trim(strings.TrimSpace(request.RemoteAddr), "[]")
	}
	address := net.ParseIP(host)
	return address != nil && address.IsLoopback()
}

func (server *AthenaServer) validWalletSecretOrigin(request *http.Request) bool {
	origin := strings.TrimSpace(request.Header.Get("Origin"))
	if origin == "" {
		return false
	}
	if !server.DisableAuth {
		return authregistration.ConstantTimeEqual(origin, server.walletSecretPublicOrigin)
	}
	parsed, err := url.Parse(origin)
	if err != nil || parsed.Scheme != "http" || parsed.Host == "" || parsed.Path != "" || parsed.RawQuery != "" || parsed.Fragment != "" {
		return false
	}
	host := net.ParseIP(parsed.Hostname())
	return strings.EqualFold(parsed.Hostname(), "localhost") || (host != nil && host.IsLoopback())
}
