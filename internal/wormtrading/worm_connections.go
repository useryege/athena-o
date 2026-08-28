package wormtrading

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"math"
	"strings"
	"time"
	"unicode"

	"github.com/google/uuid"
	"github.com/mr-tron/base58/base58"
	"github.com/useryege/athena/internal/wormtrading/apiclient"
	wormstore "github.com/useryege/athena/internal/wormtrading/store"
	"github.com/useryege/athena/util/worm"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	maxWormChallengeNonceBytes   = 512
	maxWormChallengeMessageBytes = 1024
	maxWormChallengeLifetime     = 60 * 60
)

func (s *Service) PrepareWormWalletConnection(
	ctx context.Context,
	req *apiclient.PrepareWormWalletConnectionRequest,
) (*apiclient.PrepareWormWalletConnectionResponse, error) {
	if err := s.requireCredentialCapability(); err != nil {
		return nil, err
	}
	walletID, address, err := normalizeWormWalletReference(req.GetWalletId(), req.GetAddress())
	if err != nil {
		return nil, err
	}
	unlock := s.walletOperationLock(walletID)
	defer unlock()

	client, err := s.wormClientFactory.NewUnauthenticatedClient()
	if err != nil {
		return nil, status.Error(codes.Internal, "create Worm API client")
	}
	attemptCtx, cancel := context.WithTimeout(ctx, s.wormAPIAttemptTimeout)
	challenge, requestErr := client.CreateAuthChallenge(attemptCtx, worm.CreateAuthChallengeRequest{WalletAddress: address})
	cancel()
	s.wormCapabilities.recordWormResult(requestErr)
	if requestErr != nil {
		return nil, wormRPCError("prepare Worm wallet connection", requestErr)
	}

	nonce, message, expiresAt, digest, err := validateWormAuthChallenge(address, challenge)
	if err != nil {
		s.wormCapabilities.recordWormResult(err)
		return nil, status.Error(codes.Unavailable, "Worm returned an invalid authentication challenge")
	}
	kind := wormstore.ConnectionAttemptKindConnect
	if req.GetReconnect() {
		kind = wormstore.ConnectionAttemptKindReconnect
	}
	now := timeNowUTC()
	attempt, err := s.credentialStore.PrepareConnectionAttempt(ctx, wormstore.PrepareConnectionAttemptRequest{
		AttemptID:        uuid.NewString(),
		WalletID:         walletID,
		Address:          address,
		Kind:             kind,
		Nonce:            nonce,
		ChallengeMessage: message,
		MessageDigest:    digest[:],
		ExpiresAt:        expiresAt,
		Now:              now,
	})
	s.recordCredentialStoreResult(err)
	if err != nil {
		return nil, connectionStoreRPCError("persist Worm connection challenge", err)
	}
	if attempt == nil || attempt.ID == "" || attempt.WalletID != walletID || attempt.Address != address ||
		!bytes.Equal(attempt.MessageDigest, digest[:]) || attempt.ChallengeMessage != message || attempt.Nonce != nonce {
		return nil, status.Error(codes.Internal, "Worm credential store returned a mismatched connection attempt")
	}

	return &apiclient.PrepareWormWalletConnectionResponse{
		AttemptId:     attempt.ID,
		WalletId:      walletID,
		Address:       address,
		Nonce:         nonce,
		Message:       message,
		MessageSha256: append([]byte(nil), digest[:]...),
		ExpiresAt:     attempt.ExpiresAt.Unix(),
	}, nil
}

func (s *Service) CompleteWormWalletConnection(
	ctx context.Context,
	req *apiclient.CompleteWormWalletConnectionRequest,
) (*apiclient.CompleteWormWalletConnectionResponse, error) {
	if err := s.requireCredentialCapability(); err != nil {
		return nil, err
	}
	attemptID := strings.TrimSpace(req.GetAttemptId())
	if attemptID == "" {
		return nil, status.Error(codes.InvalidArgument, "attempt_id is required")
	}
	if _, err := uuid.Parse(attemptID); err != nil {
		return nil, status.Error(codes.InvalidArgument, "attempt_id is invalid")
	}
	walletID, address, err := normalizeWormWalletReference(req.GetWalletId(), req.GetAddress())
	if err != nil {
		return nil, err
	}
	if len(req.GetMessageSha256()) != sha256.Size {
		return nil, status.Error(codes.InvalidArgument, "message_sha256 must contain 32 bytes")
	}
	if len(req.GetSignature()) != ed25519.SignatureSize {
		return nil, status.Error(codes.InvalidArgument, "signature must contain 64 bytes")
	}
	unlock := s.walletOperationLock(walletID)
	defer unlock()

	now := timeNowUTC()
	attempt, err := s.credentialStore.BeginConnectionAttemptCompletion(ctx, attemptID, now)
	s.recordCredentialStoreResult(err)
	if err != nil {
		return nil, connectionStoreRPCError("begin Worm connection completion", err)
	}
	if attempt == nil || attempt.WalletID != walletID || attempt.Address != address {
		s.failConnectionAttempt(attemptID, "ATTEMPT_MISMATCH")
		return nil, status.Error(codes.PermissionDenied, "Worm connection attempt does not match the wallet")
	}
	if !bytes.Equal(attempt.MessageDigest, req.GetMessageSha256()) {
		s.failConnectionAttempt(attemptID, "MESSAGE_DIGEST_MISMATCH")
		return nil, status.Error(codes.InvalidArgument, "Worm challenge digest does not match")
	}
	expectedDigest := sha256.Sum256([]byte(attempt.ChallengeMessage))
	if !bytes.Equal(expectedDigest[:], attempt.MessageDigest) ||
		attempt.ChallengeMessage != expectedWormChallengeMessage(address, attempt.Nonce) {
		s.failConnectionAttempt(attemptID, "STORED_CHALLENGE_INVALID")
		return nil, status.Error(codes.Internal, "stored Worm authentication challenge is invalid")
	}
	if err := verifyWormChallengeSignature(address, attempt.ChallengeMessage, req.GetSignature()); err != nil {
		s.failConnectionAttempt(attemptID, "SIGNATURE_INVALID")
		return nil, status.Error(codes.PermissionDenied, "Worm authentication signature is invalid")
	}

	client, err := s.wormClientFactory.NewUnauthenticatedClient()
	if err != nil {
		s.failConnectionAttempt(attemptID, "CLIENT_UNAVAILABLE")
		return nil, status.Error(codes.Internal, "create Worm API client")
	}
	attemptCtx, cancel := context.WithTimeout(ctx, s.wormAPIAttemptTimeout)
	credential, requestErr := client.CreateAPIKey(attemptCtx, worm.CreateAPIKeyRequest{
		WalletAddress: address,
		Message:       attempt.ChallengeMessage,
		Signature:     hex.EncodeToString(req.GetSignature()),
		Nonce:         attempt.Nonce,
	})
	cancel()
	s.wormCapabilities.recordWormResult(requestErr)
	if requestErr != nil {
		if wormCreateOutcomeIsUnknown(requestErr) {
			s.markConnectionAttemptOutcomeUnknown(attemptID)
			return nil, status.Error(codes.Aborted, "Worm credential creation outcome is unknown")
		}
		s.failConnectionAttempt(attemptID, classifyWormError(requestErr))
		return nil, wormRPCError("complete Worm wallet connection", requestErr)
	}
	if credential == nil || strings.TrimSpace(credential.APIKey) == "" || strings.TrimSpace(credential.Secret) == "" {
		s.markConnectionAttemptOutcomeUnknown(attemptID)
		return nil, status.Error(codes.Aborted, "Worm credential creation outcome is unknown")
	}

	apiKeyCiphertext, apiSecretCiphertext, err := s.credentialCipher.encrypt(walletID, credential.APIKey, credential.Secret)
	if err != nil {
		if s.bestEffortRevokeCredential(credential.APIKey, credential.Secret) {
			s.failConnectionAttempt(attemptID, "CREDENTIAL_ENCRYPTION_FAILED")
		} else {
			s.markConnectionAttemptOutcomeUnknown(attemptID)
		}
		return nil, status.Error(codes.Internal, "persist Worm credential")
	}
	snapshot, err := s.credentialStore.ActivateCredential(ctx, wormstore.ActivateCredentialRequest{
		AttemptID:           attemptID,
		APIKeyCiphertext:    apiKeyCiphertext,
		APISecretCiphertext: apiSecretCiphertext,
		Now:                 timeNowUTC(),
	})
	s.recordCredentialStoreResult(err)
	if err != nil {
		if errors.Is(err, wormstore.ErrTransactionOutcomeUnknown) {
			s.markConnectionAttemptOutcomeUnknown(attemptID)
			return nil, status.Error(codes.Aborted, "Worm credential creation outcome is unknown")
		}
		if s.bestEffortRevokeCredential(credential.APIKey, credential.Secret) {
			s.failConnectionAttempt(attemptID, "CREDENTIAL_PERSIST_FAILED")
		} else {
			s.markConnectionAttemptOutcomeUnknown(attemptID)
		}
		return nil, status.Error(codes.Internal, "persist Worm credential")
	}
	if snapshot == nil || snapshot.WalletID != walletID || snapshot.StoredAddress != address {
		return nil, status.Error(codes.Internal, "Worm credential store returned a mismatched connection")
	}

	return &apiclient.CompleteWormWalletConnectionResponse{
		Connection: wormWalletConnectionFromStore(*snapshot),
	}, nil
}

func (s *Service) DisconnectWormWallet(
	ctx context.Context,
	req *apiclient.DisconnectWormWalletRequest,
) (*apiclient.DisconnectWormWalletResponse, error) {
	if err := s.requireCredentialCapability(); err != nil {
		return nil, err
	}
	walletID, address, err := normalizeWormWalletReference(req.GetWalletId(), req.GetAddress())
	if err != nil {
		return nil, err
	}
	unlock := s.walletOperationLock(walletID)
	defer unlock()

	for {
		credential, beginErr := s.credentialStore.BeginDisconnect(ctx, walletID, address, timeNowUTC())
		s.recordCredentialStoreResult(beginErr)
		if errors.Is(beginErr, wormstore.ErrConnectionNotConnected) || errors.Is(beginErr, wormstore.ErrCredentialNotFound) {
			snapshot, snapshotErr := s.credentialStore.GetWalletConnectionSnapshot(ctx, walletID, address)
			s.recordCredentialStoreResult(snapshotErr)
			if snapshotErr != nil {
				return nil, connectionStoreRPCError("load disconnected Worm wallet", snapshotErr)
			}
			return &apiclient.DisconnectWormWalletResponse{Connection: wormWalletConnectionFromStore(*snapshot)}, nil
		}
		if beginErr != nil {
			return nil, connectionStoreRPCError("begin Worm wallet disconnect", beginErr)
		}
		if credential == nil {
			return nil, status.Error(codes.Internal, "Worm credential store returned an empty credential")
		}
		if err := s.revokeStoredCredential(ctx, *credential, true); err != nil {
			return nil, err
		}
	}
}

func (s *Service) revokeStoredCredential(ctx context.Context, credential wormstore.StoredCredential, disconnect bool) error {
	apiKey, apiSecret, err := s.credentialCipher.decrypt(credential.WalletID, credential.APIKeyCiphertext, credential.APISecretCiphertext)
	if err != nil {
		connectionState := wormstore.ConnectionStateConnected
		if disconnect {
			connectionState = wormstore.ConnectionStateRevocationRequired
		}
		s.markCredentialRevocationFailed(
			credential,
			connectionState,
			wormstore.CredentialStateRevocationRequired,
			wormErrorCredentialUnavailable,
		)
		return status.Error(codes.Internal, "stored Worm credential is unavailable")
	}
	client, err := s.wormClientFactory.NewAuthenticatedClient(apiKey, apiSecret)
	if err != nil {
		connectionState := wormstore.ConnectionStateConnected
		if disconnect {
			connectionState = wormstore.ConnectionStateRevocationRequired
		}
		s.markCredentialRevocationFailed(
			credential,
			connectionState,
			wormstore.CredentialStateRevocationRequired,
			wormErrorCredentialUnavailable,
		)
		return status.Error(codes.Internal, "stored Worm credential is unavailable")
	}
	attemptCtx, cancel := context.WithTimeout(ctx, s.wormAPIAttemptTimeout)
	_, revokeErr := client.RevokeAPIKey(attemptCtx, apiKey)
	cancel()
	s.wormCapabilities.recordWormResult(revokeErr)
	if revokeErr == nil || isWormNotFoundError(revokeErr) {
		markCtx, markCancel := s.persistenceContext()
		err = s.credentialStore.MarkCredentialRevoked(markCtx, credential.WalletID, credential.ID, timeNowUTC())
		markCancel()
		s.recordCredentialStoreResult(err)
		if err != nil {
			return status.Error(codes.Internal, "persist Worm credential revocation")
		}
		return nil
	}

	connectionState := wormstore.ConnectionStateConnected
	if disconnect {
		connectionState = wormstore.ConnectionStateDisconnecting
	}
	if disconnect && isWormAuthenticationError(revokeErr) {
		connectionState = wormstore.ConnectionStateRevocationRequired
	}
	credentialState := wormstore.CredentialStateRevocationRequired
	if !disconnect && wormRevocationCanRetry(revokeErr) {
		credentialState = wormstore.CredentialStatePendingRevocation
	}
	warningCode := classifyWormError(revokeErr)
	if isWormAuthenticationError(revokeErr) {
		warningCode = wormWarningRevocationRequired
	}
	s.markCredentialRevocationFailed(credential, connectionState, credentialState, warningCode)
	return wormRPCError("revoke Worm wallet credential", revokeErr)
}

func (s *Service) bestEffortRevokeCredential(apiKey, apiSecret string) bool {
	ctx, cancel := context.WithTimeout(context.Background(), s.wormAPIAttemptTimeout)
	defer cancel()
	client, err := s.wormClientFactory.NewAuthenticatedClient(apiKey, apiSecret)
	if err != nil {
		return false
	}
	_, err = client.RevokeAPIKey(ctx, apiKey)
	return err == nil || isWormNotFoundError(err)
}

func (s *Service) failConnectionAttempt(attemptID, failureCode string) {
	ctx, cancel := s.persistenceContext()
	defer cancel()
	if err := s.credentialStore.FailConnectionAttempt(ctx, attemptID, failureCode, timeNowUTC()); err != nil {
		s.recordCredentialStoreResult(err)
	}
}

func (s *Service) markConnectionAttemptOutcomeUnknown(attemptID string) {
	ctx, cancel := s.persistenceContext()
	defer cancel()
	if err := s.credentialStore.MarkConnectionAttemptOutcomeUnknown(ctx, attemptID, wormErrorConnectOutcomeUnknown, timeNowUTC()); err != nil {
		s.recordCredentialStoreResult(err)
	}
}

func (s *Service) markCredentialRevocationFailed(
	credential wormstore.StoredCredential,
	connectionState wormstore.ConnectionState,
	credentialState wormstore.CredentialState,
	warning string,
) {
	ctx, cancel := s.persistenceContext()
	defer cancel()
	if err := s.credentialStore.MarkCredentialRevocationFailed(
		ctx,
		credential.WalletID,
		credential.ID,
		connectionState,
		credentialState,
		warning,
		timeNowUTC(),
	); err != nil {
		s.recordCredentialStoreResult(err)
	}
}

func wormRevocationCanRetry(err error) bool {
	switch classifyWormError(err) {
	case wormErrorUnavailable, wormErrorRateLimited, wormErrorInvalidResponse, wormErrorTimeout, wormErrorCancelled:
		return true
	default:
		return false
	}
}

func (s *Service) persistenceContext() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), s.wormAPIAttemptTimeout)
}

func validateWormAuthChallenge(address string, challenge *worm.AuthChallenge) (string, string, time.Time, [sha256.Size]byte, error) {
	if challenge == nil {
		return "", "", time.Time{}, [sha256.Size]byte{}, errors.New("empty Worm authentication challenge")
	}
	nonce := challenge.Nonce
	if nonce == "" || nonce != strings.TrimSpace(nonce) || len(nonce) > maxWormChallengeNonceBytes || containsControlCharacter(nonce) {
		return "", "", time.Time{}, [sha256.Size]byte{}, errors.New("invalid Worm authentication nonce")
	}
	message := challenge.Message
	if message == "" || len(message) > maxWormChallengeMessageBytes || containsControlCharacter(message) ||
		message != expectedWormChallengeMessage(address, nonce) {
		return "", "", time.Time{}, [sha256.Size]byte{}, errors.New("invalid Worm authentication message")
	}
	if challenge.ExpiresInSeconds <= 0 || challenge.ExpiresInSeconds > maxWormChallengeLifetime ||
		challenge.ExpiresInSeconds > math.MaxInt64/int64(time.Second) {
		return "", "", time.Time{}, [sha256.Size]byte{}, errors.New("invalid Worm authentication challenge expiry")
	}
	digest := sha256.Sum256([]byte(message))
	return nonce, message, timeNowUTC().Add(time.Duration(challenge.ExpiresInSeconds) * time.Second), digest, nil
}

func expectedWormChallengeMessage(address, nonce string) string {
	return "Create Worm API credential | Wallet: " + address + " | Nonce: " + nonce
}

func verifyWormChallengeSignature(address, message string, signature []byte) error {
	publicKey, err := base58.Decode(address)
	if err != nil || len(publicKey) != ed25519.PublicKeySize || base58.Encode(publicKey) != address {
		return errors.New("invalid Solana wallet address")
	}
	if len(signature) != ed25519.SignatureSize || !ed25519.Verify(ed25519.PublicKey(publicKey), []byte(message), signature) {
		return errors.New("invalid Ed25519 signature")
	}
	return nil
}

func normalizeWormWalletReference(walletID int64, address string) (int64, string, error) {
	if walletID <= 0 {
		return 0, "", status.Error(codes.InvalidArgument, "wallet_id must be positive")
	}
	if address == "" || address != strings.TrimSpace(address) {
		return 0, "", status.Error(codes.InvalidArgument, "address must be a canonical Solana address")
	}
	decoded, err := base58.Decode(address)
	if err != nil || len(decoded) != ed25519.PublicKeySize || base58.Encode(decoded) != address {
		return 0, "", status.Error(codes.InvalidArgument, "address must be a canonical Solana address")
	}
	return walletID, address, nil
}

func containsControlCharacter(value string) bool {
	for _, character := range value {
		if unicode.IsControl(character) {
			return true
		}
	}
	return false
}

func wormWalletConnectionFromStore(snapshot wormstore.WalletConnectionSnapshot) *apiclient.WormWalletConnection {
	address := snapshot.StoredAddress
	if address == "" {
		address = snapshot.RequestedAddress
	}
	connectedAt := int64(0)
	if !snapshot.ConnectedAt.IsZero() {
		connectedAt = snapshot.ConnectedAt.Unix()
	}
	return &apiclient.WormWalletConnection{
		WalletId:    snapshot.WalletID,
		Address:     address,
		State:       string(snapshot.State),
		WarningCode: snapshot.WarningCode,
		ConnectedAt: connectedAt,
	}
}

func wormCreateOutcomeIsUnknown(err error) bool {
	switch classifyWormError(err) {
	case wormErrorUnavailable, wormErrorInvalidResponse, wormErrorTimeout, wormErrorCancelled:
		return true
	default:
		return false
	}
}

func wormRPCError(operation string, err error) error {
	if err == nil {
		return nil
	}
	switch classifyWormError(err) {
	case wormErrorCancelled:
		return status.FromContextError(context.Canceled).Err()
	case wormErrorTimeout:
		return status.Error(codes.DeadlineExceeded, operation+" timed out")
	case wormErrorRateLimited:
		return status.Error(codes.ResourceExhausted, operation+" was rate limited")
	case wormErrorReconnectRequired:
		return status.Error(codes.FailedPrecondition, operation+" requires reconnection")
	case wormErrorRejected:
		return status.Error(codes.FailedPrecondition, operation+" was rejected")
	case wormErrorInvalidResponse:
		return status.Error(codes.Unavailable, operation+" returned an invalid response")
	default:
		return status.Error(codes.Unavailable, operation+" is unavailable")
	}
}

func connectionStoreRPCError(operation string, err error) error {
	switch {
	case errors.Is(err, wormstore.ErrWalletConnectionNotFound), errors.Is(err, wormstore.ErrConnectionAttemptNotFound):
		return status.Error(codes.NotFound, operation+": record not found")
	case errors.Is(err, wormstore.ErrWalletAddressMismatch):
		return status.Error(codes.PermissionDenied, operation+": wallet address mismatch")
	case errors.Is(err, wormstore.ErrConnectionAlreadyConnected), errors.Is(err, wormstore.ErrConnectionOperationActive):
		return status.Error(codes.AlreadyExists, operation+": conflicting connection state")
	case errors.Is(err, wormstore.ErrConnectionNotConnected), errors.Is(err, wormstore.ErrConnectionAttemptState), errors.Is(err, wormstore.ErrCredentialNotFound):
		return status.Error(codes.FailedPrecondition, operation+": invalid connection state")
	case errors.Is(err, wormstore.ErrConnectionAttemptExpired):
		return status.Error(codes.DeadlineExceeded, operation+": connection challenge expired")
	case errors.Is(err, wormstore.ErrCredentialOutcomeUnknown), errors.Is(err, wormstore.ErrTransactionOutcomeUnknown):
		return status.Error(codes.Aborted, "Worm credential creation outcome is unknown")
	case errors.Is(err, context.Canceled), errors.Is(err, context.DeadlineExceeded):
		return status.FromContextError(err).Err()
	default:
		return status.Error(codes.Internal, operation)
	}
}

func timeNowUTC() time.Time {
	return time.Now().UTC()
}
