package worm

import (
	"bytes"
	"crypto/ed25519"
	"encoding/hex"
	"strings"
	"testing"

	solana "github.com/gagliardetto/solana-go"
)

func TestSignLiveWormWebTransactionLegacyUsesSignatureWhenSlotZeroIsEmpty(t *testing.T) {
	walletPrivateKey := deterministicWormWebPrivateKey(1)
	otherPrivateKey := deterministicWormWebPrivateKey(2)
	transaction := newWormWebSigningTransaction(
		t,
		solana.MessageVersionLegacy,
		solana.PrivateKey(otherPrivateKey).PublicKey(),
		solana.PrivateKey(walletPrivateKey).PublicKey(),
	)

	result, err := signLiveWormWebTransaction(walletPrivateKey, encodeWormWebSigningTransaction(t, transaction))
	if err != nil {
		t.Fatalf("sign legacy Worm Web transaction: %v", err)
	}
	if result.finalizeMode != liveWormWebFinalizeModeSignature {
		t.Fatalf("finalize mode = %q, want %q", result.finalizeMode, liveWormWebFinalizeModeSignature)
	}
	if result.transactionVersion != "legacy" {
		t.Fatalf("transaction version = %q, want legacy", result.transactionVersion)
	}
	if result.requiredSignatures != 2 {
		t.Fatalf("required signatures = %d, want 2", result.requiredSignatures)
	}
	if result.signerIndex != 1 {
		t.Fatalf("wallet signer index = %d, want 1", result.signerIndex)
	}
	if result.signatureHex == "" {
		t.Fatal("wallet signature is empty")
	}
	if result.signedTransactionHex != "" {
		t.Fatal("legacy transaction with an empty slot 0 unexpectedly returned a signed transaction")
	}

	messageBytes := marshalWormWebSigningMessage(t, &transaction.Message)
	walletSignature := decodeWormWebSigningSignature(t, result.signatureHex)
	if !solana.PrivateKey(walletPrivateKey).PublicKey().Verify(messageBytes, walletSignature) {
		t.Fatal("returned wallet signature does not verify against the legacy message")
	}
}

func TestSignLiveWormWebTransactionLegacyUsesSignedTransactionWhenOtherSignerIsValid(t *testing.T) {
	walletPrivateKey := deterministicWormWebPrivateKey(3)
	otherPrivateKey := deterministicWormWebPrivateKey(4)
	transaction := newWormWebSigningTransaction(
		t,
		solana.MessageVersionLegacy,
		solana.PrivateKey(otherPrivateKey).PublicKey(),
		solana.PrivateKey(walletPrivateKey).PublicKey(),
	)
	messageBytes := marshalWormWebSigningMessage(t, &transaction.Message)
	otherSignature, err := solana.PrivateKey(otherPrivateKey).Sign(messageBytes)
	if err != nil {
		t.Fatalf("sign legacy message with other signer: %v", err)
	}
	transaction.Signatures[0] = otherSignature

	result, err := signLiveWormWebTransaction(walletPrivateKey, encodeWormWebSigningTransaction(t, transaction))
	if err != nil {
		t.Fatalf("sign legacy Worm Web transaction: %v", err)
	}
	if result.finalizeMode != liveWormWebFinalizeModeSignedTransaction {
		t.Fatalf("finalize mode = %q, want %q", result.finalizeMode, liveWormWebFinalizeModeSignedTransaction)
	}
	if result.transactionVersion != "legacy" {
		t.Fatalf("transaction version = %q, want legacy", result.transactionVersion)
	}
	if result.requiredSignatures != 2 {
		t.Fatalf("required signatures = %d, want 2", result.requiredSignatures)
	}
	if result.signerIndex != 1 {
		t.Fatalf("wallet signer index = %d, want 1", result.signerIndex)
	}
	if result.signedTransactionHex == "" {
		t.Fatal("complete legacy transaction did not return a signed transaction")
	}

	signedTransaction := decodeWormWebSigningTransaction(t, result.signedTransactionHex)
	if signedTransaction.Message.GetVersion() != solana.MessageVersionLegacy {
		t.Fatalf("serialized transaction version = %d, want legacy", signedTransaction.Message.GetVersion())
	}
	if len(signedTransaction.Signatures) != 2 {
		t.Fatalf("serialized signature count = %d, want 2", len(signedTransaction.Signatures))
	}
	if !signedTransaction.Signatures[0].Equals(otherSignature) {
		t.Fatal("serialized transaction did not preserve the other signer's signature")
	}
	if err := signedTransaction.VerifySignatures(); err != nil {
		t.Fatalf("verify complete legacy transaction signatures: %v", err)
	}
	if result.signatureHex != hex.EncodeToString(signedTransaction.Signatures[1][:]) {
		t.Fatal("returned wallet signature does not match signed transaction slot 1")
	}
}

func TestSignLiveWormWebTransactionV0UsesPartialSignedTransactionWhenOtherSlotIsEmpty(t *testing.T) {
	walletPrivateKey := deterministicWormWebPrivateKey(5)
	otherPrivateKey := deterministicWormWebPrivateKey(6)
	transaction := newWormWebSigningTransaction(
		t,
		solana.MessageVersionV0,
		solana.PrivateKey(otherPrivateKey).PublicKey(),
		solana.PrivateKey(walletPrivateKey).PublicKey(),
	)

	result, err := signLiveWormWebTransaction(walletPrivateKey, encodeWormWebSigningTransaction(t, transaction))
	if err != nil {
		t.Fatalf("sign v0 Worm Web transaction: %v", err)
	}
	if result.finalizeMode != liveWormWebFinalizeModeSignedTransaction {
		t.Fatalf("finalize mode = %q, want %q", result.finalizeMode, liveWormWebFinalizeModeSignedTransaction)
	}
	if result.transactionVersion != "v0" {
		t.Fatalf("transaction version = %q, want v0", result.transactionVersion)
	}
	if result.requiredSignatures != 2 {
		t.Fatalf("required signatures = %d, want 2", result.requiredSignatures)
	}
	if result.signerIndex != 1 {
		t.Fatalf("wallet signer index = %d, want 1", result.signerIndex)
	}
	if result.signedTransactionHex == "" {
		t.Fatal("partially signed v0 transaction did not return a signed transaction")
	}

	signedTransaction := decodeWormWebSigningTransaction(t, result.signedTransactionHex)
	if signedTransaction.Message.GetVersion() != solana.MessageVersionV0 {
		t.Fatalf("serialized transaction version = %d, want v0", signedTransaction.Message.GetVersion())
	}
	if len(signedTransaction.Signatures) != 2 {
		t.Fatalf("serialized signature count = %d, want 2", len(signedTransaction.Signatures))
	}
	if !signedTransaction.Signatures[0].IsZero() {
		t.Fatal("partially signed v0 transaction unexpectedly populated signature slot 0")
	}
	messageBytes := marshalWormWebSigningMessage(t, &signedTransaction.Message)
	if !solana.PrivateKey(walletPrivateKey).PublicKey().Verify(messageBytes, signedTransaction.Signatures[1]) {
		t.Fatal("serialized v0 wallet signature does not verify")
	}
	if result.signatureHex != hex.EncodeToString(signedTransaction.Signatures[1][:]) {
		t.Fatal("returned wallet signature does not match signed transaction slot 1")
	}
}

func TestSignLiveWormWebTransactionRejectsWalletOutsideRequiredSigners(t *testing.T) {
	walletPrivateKey := deterministicWormWebPrivateKey(7)
	firstRequiredSigner := deterministicWormWebPrivateKey(8)
	secondRequiredSigner := deterministicWormWebPrivateKey(9)
	transaction := newWormWebSigningTransaction(
		t,
		solana.MessageVersionLegacy,
		solana.PrivateKey(firstRequiredSigner).PublicKey(),
		solana.PrivateKey(secondRequiredSigner).PublicKey(),
	)

	result, err := signLiveWormWebTransaction(walletPrivateKey, encodeWormWebSigningTransaction(t, transaction))
	if err == nil {
		t.Fatal("expected wallet outside required signers to fail")
	}
	if result != nil {
		t.Fatal("wallet outside required signers unexpectedly returned a signing result")
	}
	if !strings.Contains(err.Error(), "is not a required signer") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestVerifyLiveWormWebWalletSignatureRejectsInvalidSignature(t *testing.T) {
	walletPrivateKey := deterministicWormWebPrivateKey(10)
	otherPrivateKey := deterministicWormWebPrivateKey(11)
	messageBytes := []byte("deterministic Worm Web signing fixture")
	invalidSignature, err := solana.PrivateKey(otherPrivateKey).Sign(messageBytes)
	if err != nil {
		t.Fatalf("create invalid wallet signature fixture: %v", err)
	}

	err = verifyLiveWormWebWalletSignature(
		solana.PrivateKey(walletPrivateKey).PublicKey(),
		messageBytes,
		invalidSignature,
		1,
	)
	if err == nil {
		t.Fatal("expected invalid wallet signature to fail")
	}
	if err.Error() != "Worm Web transaction wallet signature verification failed" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func deterministicWormWebPrivateKey(seedByte byte) ed25519.PrivateKey {
	return ed25519.NewKeyFromSeed(bytes.Repeat([]byte{seedByte}, ed25519.SeedSize))
}

func newWormWebSigningTransaction(
	t *testing.T,
	version solana.MessageVersion,
	requiredSigners ...solana.PublicKey,
) *solana.Transaction {
	t.Helper()
	if len(requiredSigners) == 0 || len(requiredSigners) > 255 {
		t.Fatalf("invalid required signer fixture count: %d", len(requiredSigners))
	}
	transaction := &solana.Transaction{
		Signatures: make([]solana.Signature, len(requiredSigners)),
		Message: solana.Message{
			AccountKeys: append(solana.PublicKeySlice(nil), requiredSigners...),
			Header: solana.MessageHeader{
				NumRequiredSignatures: uint8(len(requiredSigners)),
			},
		},
	}
	transaction.Message.RecentBlockhash[0] = byte(version) + 1
	if _, err := transaction.Message.SetVersion(version); err != nil {
		t.Fatalf("set transaction message version: %v", err)
	}
	return transaction
}

func encodeWormWebSigningTransaction(t *testing.T, transaction *solana.Transaction) string {
	t.Helper()
	encoded, err := transaction.MarshalBinary()
	if err != nil {
		t.Fatalf("marshal Worm Web signing fixture: %v", err)
	}
	return hex.EncodeToString(encoded)
}

func decodeWormWebSigningTransaction(t *testing.T, encoded string) *solana.Transaction {
	t.Helper()
	raw, err := hex.DecodeString(encoded)
	if err != nil {
		t.Fatalf("decode signed Worm Web transaction hex: %v", err)
	}
	transaction, err := solana.TransactionFromBytes(raw)
	if err != nil {
		t.Fatalf("parse signed Worm Web transaction: %v", err)
	}
	return transaction
}

func marshalWormWebSigningMessage(t *testing.T, message *solana.Message) []byte {
	t.Helper()
	encoded, err := message.MarshalBinary()
	if err != nil {
		t.Fatalf("marshal Worm Web signing message: %v", err)
	}
	return encoded
}

func decodeWormWebSigningSignature(t *testing.T, encoded string) solana.Signature {
	t.Helper()
	raw, err := hex.DecodeString(encoded)
	if err != nil {
		t.Fatalf("decode Worm Web signature hex: %v", err)
	}
	if len(raw) != len(solana.Signature{}) {
		t.Fatalf("decoded Worm Web signature length = %d, want %d", len(raw), len(solana.Signature{}))
	}
	return solana.SignatureFromBytes(raw)
}
