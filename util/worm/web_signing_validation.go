package worm

import (
	"bytes"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"

	solana "github.com/gagliardetto/solana-go"
)

// Solana transactions must fit in one 1,232-byte wire packet. Reject a larger
// provider transaction before asking a wallet to sign or dispatching Finalize.
const webMaximumPositionTransactionSize = 1232

type webPositionTransactionDescriptor struct {
	rawTransaction     []byte
	message            []byte
	digest             [sha256.Size]byte
	version            solana.MessageVersion
	versionName        string
	requiredSignatures int
	signerIndex        int
	signerPublicKey    solana.PublicKey
}

func validateWebSignInSignature(walletAddress, message, encodedSignature string) (string, error) {
	publicKey, err := canonicalWebSolanaPublicKey(walletAddress, "wallet address")
	if err != nil {
		return "", err
	}
	signature, canonicalSignature, err := decodeWebSignature(encodedSignature)
	if err != nil {
		return "", fmt.Errorf("decode Worm Web sign-in signature: %w", err)
	}
	if !publicKey.Verify([]byte(message), signature) {
		return "", errors.New("Worm Web sign-in signature verification failed")
	}
	return canonicalSignature, nil
}

func inspectWebPositionTransaction(walletAddress, transactionHex string) (*webPositionTransactionDescriptor, error) {
	publicKey, err := canonicalWebSolanaPublicKey(walletAddress, "wallet address")
	if err != nil {
		return nil, err
	}
	rawTransaction, err := decodeWebPositionTransaction(transactionHex)
	if err != nil {
		return nil, err
	}
	transaction, err := solana.TransactionFromBytes(rawTransaction)
	if err != nil {
		return nil, errors.New("Worm Web transaction is not a valid Solana transaction")
	}
	if err := transaction.Sanitize(); err != nil {
		return nil, errors.New("Worm Web transaction is structurally invalid")
	}
	canonicalTransaction, err := transaction.MarshalBinary()
	if err != nil {
		return nil, errors.New("marshal Worm Web transaction")
	}
	if !bytes.Equal(canonicalTransaction, rawTransaction) {
		return nil, errors.New("Worm Web transaction is not canonically encoded")
	}

	version := transaction.Message.GetVersion()
	versionName := ""
	switch version {
	case solana.MessageVersionLegacy:
		versionName = "legacy"
	case solana.MessageVersionV0:
		versionName = "v0"
	default:
		return nil, errors.New("Worm Web transaction uses an unsupported Solana message version")
	}

	requiredSignatures := int(transaction.Message.Header.NumRequiredSignatures)
	if requiredSignatures <= 0 || requiredSignatures > len(transaction.Message.AccountKeys) {
		return nil, errors.New("Worm Web transaction contains an invalid required signer set")
	}
	if len(transaction.Signatures) != requiredSignatures {
		return nil, errors.New("Worm Web transaction contains an invalid signature count")
	}

	signerIndex := -1
	for index, candidate := range transaction.Message.AccountKeys[:requiredSignatures] {
		if candidate.Equals(publicKey) {
			signerIndex = index
			break
		}
	}
	if signerIndex < 0 {
		return nil, errors.New("wallet is not a required signer for Worm Web transaction")
	}

	message, err := transaction.Message.MarshalBinary()
	if err != nil {
		return nil, errors.New("marshal Worm Web transaction message")
	}
	if _, err := validateWebRequiredSignatures(transaction, message, requiredSignatures); err != nil {
		return nil, err
	}
	return &webPositionTransactionDescriptor{
		rawTransaction:     bytes.Clone(rawTransaction),
		message:            bytes.Clone(message),
		digest:             sha256.Sum256(rawTransaction),
		version:            version,
		versionName:        versionName,
		requiredSignatures: requiredSignatures,
		signerIndex:        signerIndex,
		signerPublicKey:    publicKey,
	}, nil
}

func validateWebPositionFinalizePayload(
	descriptor *webPositionTransactionDescriptor,
	payload WebPositionFinalizePayload,
) (WebPositionFinalizePayload, error) {
	if descriptor == nil {
		return WebPositionFinalizePayload{}, errors.New("Worm Web transaction descriptor is required")
	}
	value := strings.TrimSpace(payload.Value)
	if value == "" {
		return WebPositionFinalizePayload{}, errors.New("Worm Web finalize payload is required")
	}

	var walletSignature solana.Signature
	switch payload.Mode {
	case WebFinalizeModeSignature:
		signature, _, err := decodeWebSignature(value)
		if err != nil {
			return WebPositionFinalizePayload{}, fmt.Errorf("decode Worm Web finalize signature: %w", err)
		}
		walletSignature = signature
	case WebFinalizeModeSignedTransaction:
		signedBytes, err := decodeWebPositionTransaction(value)
		if err != nil {
			return WebPositionFinalizePayload{}, fmt.Errorf("decode signed Worm Web transaction: %w", err)
		}
		signedTransaction, err := solana.TransactionFromBytes(signedBytes)
		if err != nil {
			return WebPositionFinalizePayload{}, errors.New("signed Worm Web payload is not a valid Solana transaction")
		}
		if err := signedTransaction.Sanitize(); err != nil {
			return WebPositionFinalizePayload{}, errors.New("signed Worm Web transaction is structurally invalid")
		}
		canonicalSigned, err := signedTransaction.MarshalBinary()
		if err != nil || !bytes.Equal(canonicalSigned, signedBytes) {
			return WebPositionFinalizePayload{}, errors.New("signed Worm Web transaction is not canonically encoded")
		}
		message, err := signedTransaction.Message.MarshalBinary()
		if err != nil || !bytes.Equal(message, descriptor.message) {
			return WebPositionFinalizePayload{}, errors.New("signed Worm Web transaction message changed")
		}
		if signedTransaction.Message.GetVersion() != descriptor.version ||
			len(signedTransaction.Signatures) != descriptor.requiredSignatures ||
			descriptor.signerIndex >= len(signedTransaction.Signatures) {
			return WebPositionFinalizePayload{}, errors.New("signed Worm Web transaction signer metadata changed")
		}
		walletSignature = signedTransaction.Signatures[descriptor.signerIndex]
	default:
		return WebPositionFinalizePayload{}, fmt.Errorf("unsupported Worm Web finalize mode %q", payload.Mode)
	}

	expected, err := buildWebPositionFinalizePayload(descriptor, walletSignature)
	if err != nil {
		return WebPositionFinalizePayload{}, err
	}
	if expected.Mode != payload.Mode {
		return WebPositionFinalizePayload{}, fmt.Errorf(
			"Worm Web signer selected finalize mode %q, expected %q",
			payload.Mode,
			expected.Mode,
		)
	}
	if !strings.EqualFold(expected.Value, value) {
		return WebPositionFinalizePayload{}, errors.New("Worm Web finalize payload changed the provider transaction")
	}
	return expected, nil
}

func buildWebPositionFinalizePayload(
	descriptor *webPositionTransactionDescriptor,
	walletSignature solana.Signature,
) (WebPositionFinalizePayload, error) {
	if walletSignature.IsZero() || !descriptor.signerPublicKey.Verify(descriptor.message, walletSignature) {
		return WebPositionFinalizePayload{}, errors.New("Worm Web transaction wallet signature verification failed")
	}
	transaction, err := solana.TransactionFromBytes(descriptor.rawTransaction)
	if err != nil || len(transaction.Signatures) != descriptor.requiredSignatures {
		return WebPositionFinalizePayload{}, errors.New("reparse Worm Web transaction")
	}
	transaction.Signatures[descriptor.signerIndex] = walletSignature

	complete, err := validateWebRequiredSignatures(
		transaction,
		descriptor.message,
		descriptor.requiredSignatures,
	)
	if err != nil {
		return WebPositionFinalizePayload{}, err
	}
	if descriptor.version == solana.MessageVersionLegacy {
		if !complete {
			return WebPositionFinalizePayload{
				Mode:  WebFinalizeModeSignature,
				Value: hex.EncodeToString(walletSignature[:]),
			}, nil
		}
	}

	signedTransaction, err := transaction.MarshalBinary()
	if err != nil {
		return WebPositionFinalizePayload{}, errors.New("marshal signed Worm Web transaction")
	}
	return WebPositionFinalizePayload{
		Mode:  WebFinalizeModeSignedTransaction,
		Value: hex.EncodeToString(signedTransaction),
	}, nil
}

func validateWebRequiredSignatures(
	transaction *solana.Transaction,
	message []byte,
	requiredSignatures int,
) (bool, error) {
	if transaction == nil || len(transaction.Signatures) != requiredSignatures ||
		requiredSignatures > len(transaction.Message.AccountKeys) {
		return false, errors.New("Worm Web transaction contains an invalid signer set")
	}
	complete := true
	for index, signature := range transaction.Signatures {
		if signature.IsZero() {
			complete = false
			continue
		}
		if !transaction.Message.AccountKeys[index].Verify(message, signature) {
			return false, errors.New("Worm Web transaction contains an invalid required signature")
		}
	}
	return complete, nil
}

func decodeWebPositionTransaction(transactionHex string) ([]byte, error) {
	encoded := strings.TrimSpace(transactionHex)
	if strings.HasPrefix(encoded, "0x") || strings.HasPrefix(encoded, "0X") {
		encoded = encoded[2:]
	}
	if encoded == "" {
		return nil, errors.New("Worm Web transaction is required")
	}
	if len(encoded) > webMaximumPositionTransactionSize*2 {
		return nil, fmt.Errorf("Worm Web transaction exceeds %d bytes", webMaximumPositionTransactionSize)
	}
	raw, err := hex.DecodeString(encoded)
	if err != nil || len(raw) == 0 {
		return nil, errors.New("Worm Web transaction must be hexadecimal")
	}
	return raw, nil
}

func decodeWebSignature(encodedSignature string) (solana.Signature, string, error) {
	encoded := strings.TrimSpace(encodedSignature)
	if strings.HasPrefix(encoded, "0x") || strings.HasPrefix(encoded, "0X") {
		encoded = encoded[2:]
	}
	if len(encoded) != ed25519.SignatureSize*2 {
		return solana.Signature{}, "", errors.New("signature must contain 64 hexadecimal bytes")
	}
	raw, err := hex.DecodeString(encoded)
	if err != nil || len(raw) != ed25519.SignatureSize {
		return solana.Signature{}, "", errors.New("signature must contain 64 hexadecimal bytes")
	}
	signature := solana.SignatureFromBytes(raw)
	if signature.IsZero() {
		return solana.Signature{}, "", errors.New("signature must not be empty")
	}
	return signature, hex.EncodeToString(raw), nil
}

func canonicalWebSolanaPublicKey(value, field string) (solana.PublicKey, error) {
	if value == "" || strings.TrimSpace(value) != value {
		return solana.PublicKey{}, fmt.Errorf("Worm Web %s must be canonical", field)
	}
	publicKey, err := solana.PublicKeyFromBase58(value)
	if err != nil || publicKey.IsZero() || publicKey.String() != value {
		return solana.PublicKey{}, fmt.Errorf("Worm Web %s must be a canonical Solana public key", field)
	}
	return publicKey, nil
}

// signWebPositionTransactionWithPrivateKey exists for package-local live
// probes. SubmitWebMarketPosition itself receives only the capability-scoped
// WebMarketPositionSigner interface and never accepts private key material.
func signWebPositionTransactionWithPrivateKey(
	privateKey ed25519.PrivateKey,
	request WebPositionTransactionSigningRequest,
) (WebPositionTransactionSigningResponse, error) {
	if err := solana.PrivateKey(privateKey).Validate(); err != nil {
		return WebPositionTransactionSigningResponse{}, errors.New("invalid Solana private key")
	}
	wallet, err := canonicalWebSolanaPublicKey(request.WalletAddress, "wallet address")
	if err != nil {
		return WebPositionTransactionSigningResponse{}, err
	}
	if !solana.PrivateKey(privateKey).PublicKey().Equals(wallet) {
		return WebPositionTransactionSigningResponse{}, errors.New("Solana private key does not match wallet address")
	}
	descriptor, err := inspectWebPositionTransaction(request.WalletAddress, request.TransactionHex)
	if err != nil {
		return WebPositionTransactionSigningResponse{}, err
	}
	if descriptor.digest != request.TransactionSHA256 {
		return WebPositionTransactionSigningResponse{}, errors.New("Worm Web transaction digest changed before signing")
	}
	signatureBytes := ed25519.Sign(privateKey, descriptor.message)
	signature := solana.SignatureFromBytes(signatureBytes)
	payload, err := buildWebPositionFinalizePayload(descriptor, signature)
	if err != nil {
		return WebPositionTransactionSigningResponse{}, err
	}
	return WebPositionTransactionSigningResponse{Payload: payload}, nil
}
