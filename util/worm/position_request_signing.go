package worm

import (
	"encoding/hex"
	"errors"
	"fmt"
	"strings"

	solana "github.com/gagliardetto/solana-go"
)

type positionRequestMessageSignature struct {
	signatureHex       string
	transactionVersion string
	requiredSignatures int
	signerIndex        int
	signerPublicKey    string
}

// SignPositionRequestMessage signs the Solana transaction returned by Worm.
func SignPositionRequestMessage(privateKeyText, message string) (string, error) {
	result, err := signPositionRequestMessage(privateKeyText, message)
	if err != nil {
		return "", err
	}
	return result.signatureHex, nil
}

func signPositionRequestMessage(privateKeyText, message string) (*positionRequestMessageSignature, error) {
	privateKey, err := parseSolanaPrivateKey(privateKeyText)
	if err != nil {
		return nil, err
	}

	encodedTransaction := strings.TrimSpace(message)
	if encodedTransaction == "" {
		return nil, errors.New("worm position request message is required")
	}
	if strings.HasPrefix(encodedTransaction, "0x") || strings.HasPrefix(encodedTransaction, "0X") {
		encodedTransaction = encodedTransaction[2:]
	}

	transactionBytes, err := hex.DecodeString(encodedTransaction)
	if err != nil {
		return nil, fmt.Errorf("decode worm position request transaction: %w", err)
	}
	transaction, err := solana.TransactionFromBytes(transactionBytes)
	if err != nil {
		return nil, fmt.Errorf("parse worm position request transaction: %w", err)
	}

	solanaPrivateKey := solana.PrivateKey(privateKey)
	signerPublicKey := solanaPrivateKey.PublicKey()
	requiredSignatures := int(transaction.Message.Header.NumRequiredSignatures)
	if requiredSignatures <= 0 || requiredSignatures > len(transaction.Message.AccountKeys) {
		return nil, fmt.Errorf(
			"invalid worm position request signer set: required=%d account_keys=%d",
			requiredSignatures,
			len(transaction.Message.AccountKeys),
		)
	}

	signerIndex := -1
	for index, accountKey := range transaction.Message.AccountKeys[:requiredSignatures] {
		if accountKey.Equals(signerPublicKey) {
			signerIndex = index
			break
		}
	}
	if signerIndex < 0 {
		return nil, fmt.Errorf("wallet %s is not a required signer for worm position request transaction", signerPublicKey)
	}

	signatures, err := transaction.PartialSign(func(key solana.PublicKey) *solana.PrivateKey {
		if key.Equals(signerPublicKey) {
			return &solanaPrivateKey
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("sign worm position request transaction: %w", err)
	}
	if signerIndex >= len(signatures) {
		return nil, fmt.Errorf("worm position request signature slot %d is missing", signerIndex)
	}

	messageBytes, err := transaction.Message.MarshalBinary()
	if err != nil {
		return nil, fmt.Errorf("marshal worm position request transaction message: %w", err)
	}
	signature := signatures[signerIndex]
	if !signerPublicKey.Verify(messageBytes, signature) {
		return nil, errors.New("worm position request transaction signature verification failed")
	}
	if _, err := transaction.MarshalBinary(); err != nil {
		return nil, fmt.Errorf("marshal signed worm position request transaction: %w", err)
	}

	return &positionRequestMessageSignature{
		signatureHex:       hex.EncodeToString(signature[:]),
		transactionVersion: positionRequestTransactionVersion(transaction.Message.GetVersion()),
		requiredSignatures: requiredSignatures,
		signerIndex:        signerIndex,
		signerPublicKey:    signerPublicKey.String(),
	}, nil
}

func positionRequestTransactionVersion(version solana.MessageVersion) string {
	switch version {
	case solana.MessageVersionLegacy:
		return "legacy"
	case solana.MessageVersionV0:
		return "v0"
	default:
		return fmt.Sprintf("unknown(%d)", version)
	}
}
