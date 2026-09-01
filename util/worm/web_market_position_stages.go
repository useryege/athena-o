package worm

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"

	solana "github.com/gagliardetto/solana-go"
)

const (
	webMaximumProviderStateBytes = 100
	webMaximumProviderTxIDBytes  = 200
)

// WebPositionTransactionSigner signs only one exact Worm-returned transaction.
// Authentication is deliberately separate so durable callers can bind the two
// capabilities to different checkpoints while WebMarketPositionSigner embeds
// both for the convenience SubmitWebMarketPosition orchestration.
type WebPositionTransactionSigner interface {
	SignWebPositionTransaction(
		ctx context.Context,
		request WebPositionTransactionSigningRequest,
	) (WebPositionTransactionSigningResponse, error)
}

// WebAuthenticatedSession is an opaque, in-memory Worm Web JWT session. It can
// be cached by durable callers, but its token cannot be serialized or logged
// through this package's public API.
type WebAuthenticatedSession struct {
	walletAddress string
	accessToken   string
}

// WebMarketPositionOpenCommand is an immutable, prepared market/1x Open. Its
// provider request and wallet scope are intentionally private; callers persist
// only RequestSHA256 before dispatching it.
type WebMarketPositionOpenCommand struct {
	walletAddress string
	request       WebMarketPositionOpenRequest
	requestSHA256 [sha256.Size]byte
}

// RequestSHA256 returns the stable digest of the exact Open request body.
func (c *WebMarketPositionOpenCommand) RequestSHA256() [sha256.Size]byte {
	if c == nil {
		return [sha256.Size]byte{}
	}
	return c.requestSHA256
}

// WebPositionRequestObservation is the normalized, non-secret part of a Worm
// Web request response. The provider transaction remains private and transient
// so it can only flow into InspectWebPositionRequestTransaction and the signer.
type WebPositionRequestObservation struct {
	PositionRequestID  int64
	ProviderState      string
	ProviderOrderState string
	FundingTxID        *string
	RefundTxID         *string

	walletAddress  string
	transactionHex string
}

// IsCompleted reports the Web provider's diagnostic completed state. It is not
// proof that an Open Position exists.
func (o *WebPositionRequestObservation) IsCompleted() bool {
	return o != nil && o.ProviderState == "completed"
}

// IsAccepted reports the same Web-request acceptance evidence used by the
// convenience submit flow. Durable execution must still use Open Positions as
// its completion authority.
func (o *WebPositionRequestObservation) IsAccepted() bool {
	if o == nil {
		return false
	}
	if o.ProviderState == "completed" {
		return true
	}
	return o.ProviderState == "processing" &&
		(o.ProviderOrderState == "created" || o.ProviderOrderState == "opened")
}

// IsTerminalFailure reports an explicit failed/cancelled Web request.
func (o *WebPositionRequestObservation) IsTerminalFailure() bool {
	return o != nil && (o.ProviderState == "failed" || o.ProviderState == "cancelled")
}

// WebPositionTransactionMetadata contains the safe, persistable identity and
// signer layout of one canonical Worm-returned Solana transaction.
type WebPositionTransactionMetadata struct {
	PositionRequestID  int64
	TransactionSHA256  [sha256.Size]byte
	TransactionVersion string
	RequiredSignatures int
	WalletSignerIndex  int
	SignerPublicKey    string
}

// WebPositionFinalizeCommand is an immutable prepared Finalize. Its signature
// or signed transaction is intentionally private; callers persist only the
// request digest and safe metadata before dispatch.
type WebPositionFinalizeCommand struct {
	walletAddress string
	request       WebPositionFinalizeRequest
	requestSHA256 [sha256.Size]byte
	metadata      WebPositionTransactionMetadata
}

// RequestSHA256 returns the stable digest of the exact Finalize request shape
// used by Athena's durable mutation checkpoint.
func (c *WebPositionFinalizeCommand) RequestSHA256() [sha256.Size]byte {
	if c == nil {
		return [sha256.Size]byte{}
	}
	return c.requestSHA256
}

// PositionRequestID returns the request identity bound to this command.
func (c *WebPositionFinalizeCommand) PositionRequestID() int64 {
	if c == nil {
		return 0
	}
	return c.request.PositionRequestID
}

// FinalizeMode returns the validated, fixed payload mode selected before the
// mutation is dispatched.
func (c *WebPositionFinalizeCommand) FinalizeMode() WebFinalizeMode {
	if c == nil {
		return ""
	}
	return c.request.Payload.Mode
}

// TransactionMetadata returns the safe signer metadata bound to the command.
func (c *WebPositionFinalizeCommand) TransactionMetadata() WebPositionTransactionMetadata {
	if c == nil {
		return WebPositionTransactionMetadata{}
	}
	return c.metadata
}

// AuthenticateWebWallet performs exactly one Challenge -> SignIn exchange and
// returns an opaque session. It performs no retry and never exposes the JWT.
func AuthenticateWebWallet(
	ctx context.Context,
	client WebSignInClient,
	signer WebSignInSigner,
	walletAddress string,
) (*WebAuthenticatedSession, error) {
	if client == nil {
		return nil, errors.New("Worm Web sign-in client is required")
	}
	if signer == nil {
		return nil, errors.New("Worm Web sign-in signer is required")
	}
	if _, err := canonicalWebSolanaPublicKey(walletAddress, "wallet address"); err != nil {
		return nil, err
	}
	accessToken, err := authenticateWebWallet(ctx, client, signer, walletAddress)
	if err != nil {
		return nil, err
	}
	return &WebAuthenticatedSession{
		walletAddress: walletAddress,
		accessToken:   accessToken,
	}, nil
}

// PrepareWebMarketPositionOpen validates and freezes one market/1x Open and
// computes the exact request digest before any provider mutation is dispatched.
func PrepareWebMarketPositionOpen(
	request WebMarketPositionSubmitRequest,
) (*WebMarketPositionOpenCommand, error) {
	if _, err := canonicalWebSolanaPublicKey(request.WalletAddress, "wallet address"); err != nil {
		return nil, err
	}
	if _, err := canonicalWebSolanaPublicKey(request.MarketConditionID, "market condition id"); err != nil {
		return nil, err
	}
	providerRequest := WebMarketPositionOpenRequest{
		MarketConditionID: request.MarketConditionID,
		Funds:             request.Funds,
		IsYes:             request.IsYes,
		Leverage:          1,
	}
	if err := validateWebMarketPositionOpenRequest(providerRequest); err != nil {
		return nil, err
	}
	digest, err := digestWebRequest(providerRequest)
	if err != nil {
		return nil, fmt.Errorf("digest Worm Web market-position Open: %w", err)
	}
	return &WebMarketPositionOpenCommand{
		walletAddress: request.WalletAddress,
		request:       providerRequest,
		requestSHA256: digest,
	}, nil
}

// DispatchWebMarketPositionOpen sends exactly one prepared Open mutation. It
// never authenticates, retries, polls, signs, or finalizes.
func DispatchWebMarketPositionOpen(
	ctx context.Context,
	client WebMarketPositionOpenClient,
	session *WebAuthenticatedSession,
	command *WebMarketPositionOpenCommand,
) (*WebPositionRequestObservation, error) {
	if client == nil {
		return nil, errors.New("Worm Web market-position Open client is required")
	}
	if err := validateWebSession(session); err != nil {
		return nil, err
	}
	if command == nil {
		return nil, errors.New("Worm Web market-position Open command is required")
	}
	if command.walletAddress != session.walletAddress {
		return nil, errors.New("Worm Web market-position Open session wallet mismatch")
	}
	if err := validateWebMarketPositionOpenRequest(command.request); err != nil {
		return nil, err
	}
	expectedDigest, err := digestWebRequest(command.request)
	if err != nil || subtle.ConstantTimeCompare(expectedDigest[:], command.requestSHA256[:]) != 1 {
		return nil, errors.New("Worm Web market-position Open command changed after preparation")
	}

	providerRequest, dispatchErr := client.OpenMarketPosition(ctx, session.accessToken, command.request)
	observation, observationErr := normalizeWebPositionRequest(
		providerRequest,
		session.walletAddress,
		0,
	)
	if observationErr != nil {
		observationErr = &WebResponseError{err: observationErr}
	}
	return observation, errors.Join(dispatchErr, observationErr)
}

// ObserveWebPositionRequest performs exactly one safe request GET. Polling,
// retry timing, reauthentication, and durable recovery remain caller policy.
func ObserveWebPositionRequest(
	ctx context.Context,
	client WebPositionRequestClient,
	session *WebAuthenticatedSession,
	requestID int64,
) (*WebPositionRequestObservation, error) {
	if client == nil {
		return nil, errors.New("Worm Web position-request client is required")
	}
	if err := validateWebSession(session); err != nil {
		return nil, err
	}
	if requestID <= 0 {
		return nil, errors.New("Worm Web position request id must be positive")
	}
	providerRequest, observeErr := client.GetPositionRequest(ctx, session.accessToken, requestID)
	observation, observationErr := normalizeWebPositionRequest(
		providerRequest,
		session.walletAddress,
		requestID,
	)
	if observationErr != nil {
		observationErr = &WebResponseError{err: observationErr}
	}
	return observation, errors.Join(observeErr, observationErr)
}

// InspectWebPositionRequestTransaction validates the transient Worm Solana
// transaction and returns only safe metadata suitable for durable storage.
func InspectWebPositionRequestTransaction(
	observation *WebPositionRequestObservation,
) (WebPositionTransactionMetadata, error) {
	metadata, descriptor, err := inspectWebPositionRequestObservation(observation)
	if descriptor != nil {
		clear(descriptor.rawTransaction)
		clear(descriptor.message)
	}
	return metadata, err
}

func inspectWebPositionRequestObservation(
	observation *WebPositionRequestObservation,
) (WebPositionTransactionMetadata, *webPositionTransactionDescriptor, error) {
	if observation == nil || observation.PositionRequestID <= 0 {
		return WebPositionTransactionMetadata{}, nil,
			errors.New("Worm Web position request observation is required")
	}
	descriptor, err := inspectWebPositionTransaction(
		observation.walletAddress,
		observation.transactionHex,
	)
	if err != nil {
		return WebPositionTransactionMetadata{}, nil, err
	}
	return webPositionTransactionMetadata(observation.PositionRequestID, descriptor), descriptor, nil
}

// PrepareWebPositionFinalize asks the injected signer to sign one exact
// provider transaction, validates the returned payload, and freezes Finalize
// plus its digest. It dispatches no provider mutation.
func PrepareWebPositionFinalize(
	ctx context.Context,
	signer WebPositionTransactionSigner,
	observation *WebPositionRequestObservation,
	expectedTransactionSHA256 [sha256.Size]byte,
) (*WebPositionFinalizeCommand, error) {
	if signer == nil {
		return nil, errors.New("Worm Web position transaction signer is required")
	}
	metadata, descriptor, err := inspectWebPositionRequestObservation(observation)
	if err != nil {
		return nil, err
	}
	defer clear(descriptor.rawTransaction)
	defer clear(descriptor.message)
	if subtle.ConstantTimeCompare(
		metadata.TransactionSHA256[:],
		expectedTransactionSHA256[:],
	) != 1 {
		return nil, errors.New("Worm Web transaction digest changed before signing")
	}
	signed, err := signer.SignWebPositionTransaction(ctx, WebPositionTransactionSigningRequest{
		WalletAddress:     observation.walletAddress,
		PositionRequestID: observation.PositionRequestID,
		TransactionHex:    observation.transactionHex,
		TransactionSHA256: metadata.TransactionSHA256,
	})
	if err != nil {
		return nil, err
	}
	finalizePayload, err := validateWebPositionFinalizePayload(descriptor, signed.Payload)
	if err != nil {
		return nil, err
	}
	providerRequest := WebPositionFinalizeRequest{
		PositionRequestID: observation.PositionRequestID,
		Payload:           finalizePayload,
	}
	requestDigest, err := digestWebRequest(providerRequest)
	if err != nil {
		return nil, fmt.Errorf("digest Worm Web position Finalize: %w", err)
	}
	return &WebPositionFinalizeCommand{
		walletAddress: observation.walletAddress,
		request:       providerRequest,
		requestSHA256: requestDigest,
		metadata:      metadata,
	}, nil
}

// DispatchWebPositionFinalize sends exactly one prepared Finalize mutation. It
// never retries, switches payload mode, or performs a status GET.
func DispatchWebPositionFinalize(
	ctx context.Context,
	client WebPositionFinalizeClient,
	session *WebAuthenticatedSession,
	command *WebPositionFinalizeCommand,
) (*WebPositionRequestObservation, error) {
	if client == nil {
		return nil, errors.New("Worm Web position Finalize client is required")
	}
	if err := validateWebSession(session); err != nil {
		return nil, err
	}
	if command == nil || command.request.PositionRequestID <= 0 {
		return nil, errors.New("Worm Web position Finalize command is required")
	}
	if command.walletAddress != session.walletAddress {
		return nil, errors.New("Worm Web position Finalize session wallet mismatch")
	}
	expectedDigest, err := digestWebRequest(command.request)
	if err != nil || subtle.ConstantTimeCompare(expectedDigest[:], command.requestSHA256[:]) != 1 {
		return nil, errors.New("Worm Web position Finalize command changed after preparation")
	}

	providerRequest, dispatchErr := client.FinalizePosition(ctx, session.accessToken, command.request)
	observation, observationErr := normalizeWebPositionRequest(
		providerRequest,
		session.walletAddress,
		command.request.PositionRequestID,
	)
	if observationErr != nil {
		observationErr = &WebResponseError{err: observationErr}
	}
	return observation, errors.Join(dispatchErr, observationErr)
}

// BuildWebPositionTransactionSigningResponse centralizes the exact Solana
// parsing, signer-slot verification, message signing, and Finalize payload
// selection used by both live probes and Athena Wallet. The callback receives
// only canonical transaction message bytes and must return one Ed25519
// signature; private key material never enters util/worm.
func BuildWebPositionTransactionSigningResponse(
	request WebPositionTransactionSigningRequest,
	signMessage func(message []byte) ([]byte, error),
) (WebPositionTransactionSigningResponse, WebPositionTransactionMetadata, error) {
	if request.PositionRequestID <= 0 {
		return WebPositionTransactionSigningResponse{}, WebPositionTransactionMetadata{},
			errors.New("Worm Web position request id must be positive")
	}
	if signMessage == nil {
		return WebPositionTransactionSigningResponse{}, WebPositionTransactionMetadata{},
			errors.New("Worm Web transaction message signer is required")
	}
	descriptor, err := inspectWebPositionTransaction(request.WalletAddress, request.TransactionHex)
	if err != nil {
		return WebPositionTransactionSigningResponse{}, WebPositionTransactionMetadata{}, err
	}
	defer clear(descriptor.rawTransaction)
	defer clear(descriptor.message)
	if subtle.ConstantTimeCompare(descriptor.digest[:], request.TransactionSHA256[:]) != 1 {
		return WebPositionTransactionSigningResponse{}, WebPositionTransactionMetadata{},
			errors.New("Worm Web transaction digest changed before signing")
	}
	messageForSigner := bytes.Clone(descriptor.message)
	rawSignature, err := signMessage(messageForSigner)
	clear(messageForSigner)
	if err != nil {
		return WebPositionTransactionSigningResponse{}, WebPositionTransactionMetadata{}, err
	}
	defer clear(rawSignature)
	if len(rawSignature) != ed25519.SignatureSize {
		return WebPositionTransactionSigningResponse{}, WebPositionTransactionMetadata{},
			errors.New("Worm Web transaction signer returned an invalid Ed25519 signature")
	}
	signature := solana.SignatureFromBytes(rawSignature)
	payload, err := buildWebPositionFinalizePayload(
		descriptor,
		signature,
	)
	if err != nil {
		return WebPositionTransactionSigningResponse{}, WebPositionTransactionMetadata{}, err
	}
	metadata := webPositionTransactionMetadata(request.PositionRequestID, descriptor)
	return WebPositionTransactionSigningResponse{Payload: payload}, metadata, nil
}

func validateWebSession(session *WebAuthenticatedSession) error {
	if session == nil {
		return errors.New("Worm Web authenticated session is required")
	}
	if _, err := canonicalWebSolanaPublicKey(session.walletAddress, "session wallet address"); err != nil {
		return err
	}
	if session.accessToken == "" || strings.TrimSpace(session.accessToken) != session.accessToken {
		return errors.New("Worm Web authenticated session is invalid")
	}
	return nil
}

func normalizeWebPositionRequest(
	request *WebPositionRequest,
	walletAddress string,
	expectedID int64,
) (*WebPositionRequestObservation, error) {
	if request == nil {
		return nil, errors.New("Worm Web position request response is missing")
	}
	observation := &WebPositionRequestObservation{
		PositionRequestID: int64(request.ID),
		walletAddress:     walletAddress,
		transactionHex:    request.Message,
	}
	if observation.PositionRequestID <= 0 {
		return observation, errors.New("Worm Web position request response has no positive id")
	}
	if expectedID > 0 && observation.PositionRequestID != expectedID {
		return observation, fmt.Errorf(
			"Worm Web position request id mismatch: got %d, want %d",
			observation.PositionRequestID,
			expectedID,
		)
	}

	providerState, err := normalizeWebProviderStateStrict(request.State)
	if err != nil {
		return observation, fmt.Errorf("normalize Worm Web provider state: %w", err)
	}
	observation.ProviderState = providerState
	providerOrderState, err := normalizeWebProviderStateStrict(request.OrderState)
	if err != nil {
		return observation, fmt.Errorf("normalize Worm Web provider order state: %w", err)
	}
	observation.ProviderOrderState = providerOrderState
	fundingTxID, err := normalizeWebProviderTxID(request.FundingTxID)
	if err != nil {
		return observation, fmt.Errorf("normalize Worm Web funding transaction id: %w", err)
	}
	observation.FundingTxID = fundingTxID
	refundTxID, err := normalizeWebProviderTxID(request.RefundTxID)
	if err != nil {
		return observation, fmt.Errorf("normalize Worm Web refund transaction id: %w", err)
	}
	observation.RefundTxID = refundTxID
	return observation, nil
}

func normalizeWebProviderStateStrict(value string) (string, error) {
	if !utf8.ValidString(value) || containsWebControl(value) {
		return "", errors.New("value contains invalid text")
	}
	normalized := strings.ToLower(strings.TrimSpace(value))
	if len(normalized) > webMaximumProviderStateBytes {
		return "", fmt.Errorf("value exceeds %d bytes", webMaximumProviderStateBytes)
	}
	return normalized, nil
}

func normalizeWebProviderTxID(value *string) (*string, error) {
	if value == nil {
		return nil, nil
	}
	if !utf8.ValidString(*value) || containsWebControl(*value) {
		return nil, errors.New("value contains invalid text")
	}
	normalized := strings.TrimSpace(*value)
	if normalized == "" {
		return nil, nil
	}
	if len(normalized) > webMaximumProviderTxIDBytes {
		return nil, fmt.Errorf("value exceeds %d bytes", webMaximumProviderTxIDBytes)
	}
	return &normalized, nil
}

func containsWebControl(value string) bool {
	for _, candidate := range value {
		if unicode.IsControl(candidate) {
			return true
		}
	}
	return false
}

func webPositionTransactionMetadata(
	requestID int64,
	descriptor *webPositionTransactionDescriptor,
) WebPositionTransactionMetadata {
	if descriptor == nil {
		return WebPositionTransactionMetadata{}
	}
	return WebPositionTransactionMetadata{
		PositionRequestID:  requestID,
		TransactionSHA256:  descriptor.digest,
		TransactionVersion: descriptor.versionName,
		RequiredSignatures: descriptor.requiredSignatures,
		WalletSignerIndex:  descriptor.signerIndex,
		SignerPublicKey:    descriptor.signerPublicKey.String(),
	}
}

func digestWebRequest(request any) ([sha256.Size]byte, error) {
	encoded, err := json.Marshal(request)
	if err != nil {
		return [sha256.Size]byte{}, err
	}
	return sha256.Sum256(encoded), nil
}
