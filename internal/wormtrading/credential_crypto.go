package wormtrading

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"strings"

	utilcrypto "github.com/useryege/athena/util/crypto"
)

const minimumCredentialEncryptionPassphraseBytes = 32

type credentialCipher struct {
	key []byte
}

type credentialPlaintext struct {
	SchemaVersion int    `json:"schema_version"`
	WalletID      int64  `json:"wallet_id"`
	Value         string `json:"value"`
}

func CredentialEncryptionKeyFromPassphrase(passphrase string) ([]byte, error) {
	if len([]byte(passphrase)) < minimumCredentialEncryptionPassphraseBytes {
		return nil, errors.New("worm trading credential encryption key must contain at least 32 bytes")
	}
	if strings.TrimSpace(passphrase) != passphrase {
		return nil, errors.New("worm trading credential encryption key must not contain leading or trailing whitespace")
	}
	key, err := utilcrypto.KeyFromPassphrase(passphrase)
	if err != nil {
		return nil, errors.New("derive worm trading credential encryption key")
	}
	return key, nil
}

func newCredentialCipher(key []byte) (*credentialCipher, error) {
	if len(key) != 32 {
		return nil, errors.New("worm trading credential encryption key must be 32 bytes")
	}
	return &credentialCipher{key: append([]byte(nil), key...)}, nil
}

func (c *credentialCipher) encrypt(walletID int64, apiKey, apiSecret string) ([]byte, []byte, error) {
	if c == nil || len(c.key) != 32 {
		return nil, nil, errors.New("worm trading credential encryption is not configured")
	}
	if walletID <= 0 {
		return nil, nil, errors.New("worm credential wallet ID must be positive")
	}
	if strings.TrimSpace(apiKey) == "" || strings.TrimSpace(apiSecret) == "" {
		return nil, nil, errors.New("worm API credential is incomplete")
	}
	apiKeyPlaintext, err := json.Marshal(credentialPlaintext{SchemaVersion: 1, WalletID: walletID, Value: apiKey})
	if err != nil {
		return nil, nil, errors.New("encode Worm API key envelope")
	}
	apiSecretPlaintext, err := json.Marshal(credentialPlaintext{SchemaVersion: 1, WalletID: walletID, Value: apiSecret})
	if err != nil {
		return nil, nil, errors.New("encode Worm API secret envelope")
	}
	apiKeyCiphertext, err := utilcrypto.Encrypt(apiKeyPlaintext, c.key)
	if err != nil {
		return nil, nil, errors.New("encrypt worm API key")
	}
	apiSecretCiphertext, err := utilcrypto.Encrypt(apiSecretPlaintext, c.key)
	if err != nil {
		return nil, nil, errors.New("encrypt worm API secret")
	}
	return apiKeyCiphertext, apiSecretCiphertext, nil
}

func (c *credentialCipher) decrypt(walletID int64, apiKeyCiphertext, apiSecretCiphertext []byte) (string, string, error) {
	if c == nil || len(c.key) != 32 {
		return "", "", errors.New("worm trading credential encryption is not configured")
	}
	if walletID <= 0 {
		return "", "", errors.New("worm credential wallet ID must be positive")
	}
	if len(apiKeyCiphertext) == 0 || len(apiSecretCiphertext) == 0 {
		return "", "", errors.New("stored Worm API credential is incomplete")
	}
	apiKeyPlaintext, err := utilcrypto.Decrypt(apiKeyCiphertext, c.key)
	if err != nil {
		return "", "", errors.New("decrypt worm API key")
	}
	apiSecretPlaintext, err := utilcrypto.Decrypt(apiSecretCiphertext, c.key)
	if err != nil {
		return "", "", errors.New("decrypt worm API secret")
	}
	apiKey, err := decodeCredentialPlaintext(apiKeyPlaintext, walletID)
	if err != nil {
		return "", "", errors.New("decode Worm API key envelope")
	}
	apiSecret, err := decodeCredentialPlaintext(apiSecretPlaintext, walletID)
	if err != nil {
		return "", "", errors.New("decode Worm API secret envelope")
	}
	if strings.TrimSpace(apiKey) == "" || strings.TrimSpace(apiSecret) == "" {
		return "", "", errors.New("stored Worm API credential plaintext is invalid")
	}
	return apiKey, apiSecret, nil
}

func decodeCredentialPlaintext(raw []byte, walletID int64) (string, error) {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	var envelope credentialPlaintext
	if err := decoder.Decode(&envelope); err != nil {
		return "", err
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return "", errors.New("credential envelope contains trailing data")
	}
	if envelope.SchemaVersion != 1 || envelope.WalletID != walletID || envelope.Value == "" {
		return "", errors.New("credential envelope metadata is invalid")
	}
	return envelope.Value, nil
}
