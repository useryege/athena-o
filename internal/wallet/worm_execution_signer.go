package wallet

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/useryege/athena/internal/wallet/apiclient"
	utilcrypto "github.com/useryege/athena/util/crypto"
	utilworm "github.com/useryege/athena/util/worm"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	wormWebIssuedAtPrefix = "\nIssued At: "
	wormWebIssuedAtLayout = "2006-01-02T15:04:05.000Z"
)

func (s *Service) SignWormWebSignInMessage(
	ctx context.Context,
	req *apiclient.SignWormWebSignInMessageRequest,
) (*apiclient.SignWormWebSignInMessageResponse, error) {
	runID, stepID, intentDigest, err := validateExecutionSigningBinding(
		req.GetExecutionRunId(),
		req.GetExecutionStepId(),
		req.GetIntentSha256(),
	)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	if err := validateWormWebSignInMessage(req.GetExpectedAddress(), req.GetNonce(), req.GetMessage()); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	messageBytes := []byte(req.GetMessage())
	digest := sha256.Sum256(messageBytes)
	if err := validateExpectedSHA256(req.GetExpectedMessageSha256(), digest); err != nil {
		return nil, status.Error(codes.InvalidArgument, "expected_message_sha256 does not match message")
	}

	privateKey, recordAddress, err := s.executionSolanaPrivateKey(
		ctx,
		req.GetId(),
		req.GetRequesterAccountId(),
		req.GetExpectedAddress(),
	)
	if err != nil {
		return nil, err
	}
	defer clear(privateKey)
	if recordAddress != req.GetExpectedAddress() {
		return nil, status.Error(codes.Internal, "wallet address validation changed during signing")
	}

	signature := ed25519.Sign(privateKey, messageBytes)
	if !ed25519.Verify(privateKey.Public().(ed25519.PublicKey), messageBytes, signature) {
		return nil, status.Error(codes.Internal, "failed to verify Worm Web sign-in signature")
	}
	return &apiclient.SignWormWebSignInMessageResponse{
		Signature:       hex.EncodeToString(signature),
		MessageSha256:   bytes.Clone(digest[:]),
		ExecutionRunId:  runID,
		ExecutionStepId: stepID,
		IntentSha256:    intentDigest,
	}, nil
}

func (s *Service) SignWormPositionRequestTransaction(
	ctx context.Context,
	req *apiclient.SignWormPositionRequestTransactionRequest,
) (*apiclient.SignWormPositionRequestTransactionResponse, error) {
	if req.GetPositionRequestId() <= 0 {
		return nil, status.Error(codes.InvalidArgument, "position_request_id must be positive")
	}
	runID, stepID, intentDigest, err := validateExecutionSigningBinding(
		req.GetExecutionRunId(),
		req.GetExecutionStepId(),
		req.GetIntentSha256(),
	)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	if len(req.GetExpectedTransactionSha256()) != sha256.Size {
		return nil, status.Error(codes.InvalidArgument, "expected_transaction_sha256 must contain exactly 32 bytes")
	}
	var expectedTransactionDigest [sha256.Size]byte
	copy(expectedTransactionDigest[:], req.GetExpectedTransactionSha256())
	privateKey, _, err := s.executionSolanaPrivateKey(
		ctx,
		req.GetId(),
		req.GetRequesterAccountId(),
		req.GetExpectedAddress(),
	)
	if err != nil {
		return nil, err
	}
	defer clear(privateKey)

	signed, metadata, err := utilworm.BuildWebPositionTransactionSigningResponse(
		utilworm.WebPositionTransactionSigningRequest{
			WalletAddress:     req.GetExpectedAddress(),
			PositionRequestID: req.GetPositionRequestId(),
			TransactionHex:    req.GetTransactionHex(),
			TransactionSHA256: expectedTransactionDigest,
		},
		func(message []byte) ([]byte, error) {
			signature := ed25519.Sign(privateKey, message)
			if !ed25519.Verify(privateKey.Public().(ed25519.PublicKey), message, signature) {
				return nil, errors.New("failed to verify Worm Web transaction signature")
			}
			return signature, nil
		},
	)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	response := &apiclient.SignWormPositionRequestTransactionResponse{
		TransactionSha256:  bytes.Clone(metadata.TransactionSHA256[:]),
		TransactionVersion: metadata.TransactionVersion,
		RequiredSignatures: int32(metadata.RequiredSignatures),
		SignerIndex:        int32(metadata.WalletSignerIndex),
		ExecutionRunId:     runID,
		ExecutionStepId:    stepID,
		IntentSha256:       intentDigest,
		PositionRequestId:  req.GetPositionRequestId(),
	}
	switch signed.Payload.Mode {
	case utilworm.WebFinalizeModeSignature:
		response.FinalizePayload = &apiclient.SignWormPositionRequestTransactionResponse_Signature{
			Signature: signed.Payload.Value,
		}
	case utilworm.WebFinalizeModeSignedTransaction:
		response.FinalizePayload = &apiclient.SignWormPositionRequestTransactionResponse_SignedTransaction{
			SignedTransaction: signed.Payload.Value,
		}
	default:
		return nil, status.Error(codes.Internal, "Worm Web transaction selected an invalid finalize mode")
	}
	return response, nil
}

func validateExecutionSigningBinding(runID, stepID string, intentDigest []byte) (string, string, []byte, error) {
	canonicalRunID, err := canonicalExecutionSigningID(runID, "execution_run_id")
	if err != nil {
		return "", "", nil, err
	}
	canonicalStepID, err := canonicalExecutionSigningID(stepID, "execution_step_id")
	if err != nil {
		return "", "", nil, err
	}
	if len(intentDigest) != sha256.Size {
		return "", "", nil, errors.New("intent_sha256 must contain exactly 32 bytes")
	}
	return canonicalRunID, canonicalStepID, bytes.Clone(intentDigest), nil
}

func canonicalExecutionSigningID(value, field string) (string, error) {
	if value == "" || strings.TrimSpace(value) != value {
		return "", fmt.Errorf("%s must be a canonical UUID", field)
	}
	parsed, err := uuid.Parse(value)
	if err != nil || parsed == uuid.Nil || parsed.String() != value {
		return "", fmt.Errorf("%s must be a canonical UUID", field)
	}
	return parsed.String(), nil
}

func (s *Service) executionSolanaPrivateKey(
	ctx context.Context,
	id int64,
	requesterAccountID string,
	expectedAddress string,
) (ed25519.PrivateKey, string, error) {
	record, err := s.walletRecord(ctx, id, requesterAccountID)
	if err != nil {
		return nil, "", err
	}
	if record.WalletType != walletTypeSolana {
		return nil, "", status.Error(codes.FailedPrecondition, "wallet must be a Solana wallet")
	}
	if expectedAddress == "" {
		return nil, "", status.Error(codes.InvalidArgument, "expected_address is required")
	}
	if expectedAddress != record.Address {
		return nil, "", status.Error(codes.InvalidArgument, "expected_address does not match wallet")
	}
	if len(s.encryptionKey) == 0 {
		return nil, "", status.Error(codes.FailedPrecondition, "wallet encryption key is required")
	}

	privateKeyText, err := utilcrypto.Decrypt(record.PrivateKeyCiphertext, s.encryptionKey)
	if err != nil {
		return nil, "", status.Error(codes.Internal, "failed to decrypt wallet key material")
	}
	defer clear(privateKeyText)
	privateKey, err := parseSolanaPrivateKey(string(privateKeyText))
	if err != nil {
		return nil, "", status.Error(codes.Internal, "wallet key material is invalid")
	}
	if solanaWalletAddress(privateKey) != record.Address {
		clear(privateKey)
		return nil, "", status.Error(codes.Internal, "wallet key material does not match stored address")
	}
	return privateKey, record.Address, nil
}

func validateWormWebSignInMessage(address, nonce, message string) error {
	if nonce == "" || !utf8.ValidString(nonce) || len(nonce) > maxWormAuthNonceBytes {
		return errors.New("nonce is invalid")
	}
	for _, value := range nonce {
		if unicode.IsControl(value) {
			return errors.New("nonce is invalid")
		}
	}
	if !utf8.ValidString(message) {
		return errors.New("message must be valid UTF-8")
	}
	issuedAtIndex := strings.LastIndex(message, wormWebIssuedAtPrefix)
	if issuedAtIndex < 0 {
		return errors.New("message does not match Worm Web sign-in challenge")
	}
	issuedAtText := message[issuedAtIndex+len(wormWebIssuedAtPrefix):]
	if strings.TrimSpace(issuedAtText) != issuedAtText || strings.ContainsAny(issuedAtText, "\r\n") {
		return errors.New("message does not match Worm Web sign-in challenge")
	}
	issuedAt, err := time.Parse(wormWebIssuedAtLayout, issuedAtText)
	if err != nil {
		return errors.New("message contains an invalid Worm Web issued-at value")
	}
	if message != utilworm.BuildWebSignInMessage(address, nonce, issuedAt) {
		return errors.New("message does not match Worm Web sign-in challenge")
	}
	return nil
}

func validateExpectedSHA256(expected []byte, actual [sha256.Size]byte) error {
	if len(expected) != sha256.Size {
		return errors.New("expected digest must contain exactly 32 bytes")
	}
	if subtle.ConstantTimeCompare(expected, actual[:]) != 1 {
		return errors.New("expected digest does not match")
	}
	return nil
}
