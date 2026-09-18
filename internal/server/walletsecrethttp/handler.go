package walletsecrethttp

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/useryege/athena/internal/accountcredentials"
	operationlogrecord "github.com/useryege/athena/internal/operationlog/record"
	"github.com/useryege/athena/internal/walletsecret"
)

// Authenticator validates only Athena's interactive login cookie, attaches its
// typed credential to the returned context, and enforces Wallet READ_WRITE.
type Authenticator func(request *http.Request) (context.Context, accountcredentials.AuthenticatedCredential, error)

// RevealPrivateKey is the narrow trusted adapter to the internal Wallet RPC.
// The handler always supplies the authenticated account UUID; no requester
// identity is accepted from public input.
type RevealPrivateKey func(ctx context.Context, accountID string, walletID int64) (string, error)

type OriginValidator func(request *http.Request) bool

type Handler struct {
	authenticate Authenticator
	leases       *walletsecret.Manager
	reveal       RevealPrivateKey
	validOrigin  OriginValidator
}

func NewHandler(authenticate Authenticator, leases *walletsecret.Manager, reveal RevealPrivateKey, validOrigin OriginValidator) (*Handler, error) {
	if authenticate == nil || leases == nil || reveal == nil || validOrigin == nil {
		return nil, fmt.Errorf("wallet-secret HTTP authenticator, lease manager, reveal adapter, and origin validator are required")
	}
	return &Handler{authenticate: authenticate, leases: leases, reveal: reveal, validOrigin: validOrigin}, nil
}

// Reveal handles the same-origin native HTTP private-key resource. Secret
// material never crosses the public protobuf or gRPC-web boundary.
func (h *Handler) Reveal(w http.ResponseWriter, r *http.Request) {
	walletsecret.SetSecretResponseHeaders(w)
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", "POST")
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	if !h.validOrigin(r) {
		walletsecret.WriteError(w, walletsecret.ErrReauthenticationRequired)
		return
	}
	walletID, err := requestWalletID(r)
	if err != nil {
		walletsecret.WriteError(w, err)
		return
	}
	ctx, credential, err := h.authenticate(r)
	if err != nil {
		walletsecret.WriteError(w, err)
		return
	}
	if err := h.leases.Validate(ctx, r, credential); err != nil {
		walletsecret.WriteError(w, err)
		return
	}
	privateKey, err := h.reveal(ctx, credential.AccountID, walletID)
	if err != nil {
		walletsecret.WriteError(w, err)
		return
	}
	if strings.TrimSpace(privateKey) == "" {
		walletsecret.WriteError(w, status.Error(codes.Internal, "wallet service returned empty private key"))
		return
	}
	operationlogrecord.CaptureResource(ctx, "wallet", strconv.FormatInt(walletID, 10))
	operationlogrecord.Commit(ctx, "WALLET_PRIVATE_KEY_REVEAL")
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	if err := json.NewEncoder(w).Encode(struct {
		PrivateKey string `json:"privateKey"`
	}{PrivateKey: privateKey}); err != nil {
		// Headers have already committed; never log or re-encode the secret.
		return
	}
}

func requestWalletID(r *http.Request) (int64, error) {
	raw := r.PathValue("id")
	if raw == "" {
		const prefix = "/api/v1/wallets/"
		const suffix = ":revealPrivateKey"
		path := r.URL.Path
		if !strings.HasPrefix(path, prefix) || !strings.HasSuffix(path, suffix) {
			return 0, status.Error(codes.NotFound, "wallet private-key resource not found")
		}
		raw = strings.TrimSuffix(strings.TrimPrefix(path, prefix), suffix)
		if raw == "" || strings.Contains(raw, "/") {
			return 0, status.Error(codes.NotFound, "wallet private-key resource not found")
		}
	}
	walletID, err := strconv.ParseInt(strings.TrimSpace(raw), 10, 64)
	if err != nil || walletID <= 0 {
		return 0, status.Error(codes.InvalidArgument, "wallet ID must be a positive integer")
	}
	return walletID, nil
}
