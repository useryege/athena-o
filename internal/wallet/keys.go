package wallet

import (
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/ethereum/go-ethereum/common/hexutil"
	ethcrypto "github.com/ethereum/go-ethereum/crypto"
	"github.com/mr-tron/base58/base58"
)

const (
	walletTypeEVM    = "EVM"
	walletTypeSolana = "SOLANA"

	walletSourceCreated  = "created"
	walletSourceImported = "imported"
)

type walletKeyMaterial struct {
	walletType string
	address    string
	addressKey string
	privateKey string
	source     string
}

func normalizeWalletType(input string) (string, error) {
	walletType := strings.ToUpper(strings.TrimSpace(input))
	switch walletType {
	case walletTypeEVM, walletTypeSolana:
		return walletType, nil
	case "":
		return "", errors.New("wallet_type is required")
	default:
		return "", errors.New("wallet_type must be one of EVM, SOLANA")
	}
}

func createWalletKeyMaterial(walletType string) (walletKeyMaterial, error) {
	switch walletType {
	case walletTypeEVM:
		privateKey, err := ethcrypto.GenerateKey()
		if err != nil {
			return walletKeyMaterial{}, fmt.Errorf("generate EVM private key: %w", err)
		}
		address := ethcrypto.PubkeyToAddress(privateKey.PublicKey).Hex()
		return walletKeyMaterial{
			walletType: walletTypeEVM,
			address:    address,
			addressKey: strings.ToLower(address),
			privateKey: hexutil.Encode(ethcrypto.FromECDSA(privateKey)),
			source:     walletSourceCreated,
		}, nil
	case walletTypeSolana:
		publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
		if err != nil {
			return walletKeyMaterial{}, fmt.Errorf("generate Solana private key: %w", err)
		}
		address := base58.Encode(publicKey)
		return walletKeyMaterial{
			walletType: walletTypeSolana,
			address:    address,
			addressKey: address,
			privateKey: base58.Encode(privateKey),
			source:     walletSourceCreated,
		}, nil
	default:
		return walletKeyMaterial{}, errors.New("unsupported wallet type")
	}
}

func importedWalletKeyMaterial(walletType, privateKey string) (walletKeyMaterial, error) {
	switch walletType {
	case walletTypeEVM:
		return evmPrivateKeyMaterial(privateKey)
	case walletTypeSolana:
		return solanaPrivateKeyMaterial(privateKey)
	default:
		return walletKeyMaterial{}, errors.New("unsupported wallet type")
	}
}

func evmPrivateKeyMaterial(input string) (walletKeyMaterial, error) {
	encoded := strings.TrimSpace(input)
	if strings.HasPrefix(encoded, "0x") || strings.HasPrefix(encoded, "0X") {
		encoded = encoded[2:]
	}
	if len(encoded) != 64 {
		return walletKeyMaterial{}, errors.New("invalid EVM private key: must be exactly 32 bytes of hexadecimal")
	}
	privateKey, err := ethcrypto.HexToECDSA(encoded)
	if err != nil {
		return walletKeyMaterial{}, errors.New("invalid EVM private key: must be a valid secp256k1 scalar")
	}
	address := ethcrypto.PubkeyToAddress(privateKey.PublicKey).Hex()
	return walletKeyMaterial{
		walletType: walletTypeEVM,
		address:    address,
		addressKey: strings.ToLower(address),
		privateKey: hexutil.Encode(ethcrypto.FromECDSA(privateKey)),
		source:     walletSourceImported,
	}, nil
}

func solanaPrivateKeyMaterial(input string) (walletKeyMaterial, error) {
	privateKey, err := parseSolanaPrivateKey(input)
	if err != nil {
		return walletKeyMaterial{}, err
	}
	address := solanaWalletAddress(privateKey)
	return walletKeyMaterial{
		walletType: walletTypeSolana,
		address:    address,
		addressKey: address,
		privateKey: base58.Encode(privateKey),
		source:     walletSourceImported,
	}, nil
}

func parseSolanaPrivateKey(input string) (ed25519.PrivateKey, error) {
	trimmed := strings.TrimSpace(input)
	if trimmed == "" {
		return nil, errors.New("Solana private key is required")
	}

	var raw []byte
	var err error
	switch {
	case strings.HasPrefix(trimmed, "["):
		if err := json.Unmarshal([]byte(trimmed), &raw); err != nil {
			return nil, errors.New("invalid Solana private key JSON encoding")
		}
	case isHexEncodedSolanaKey(trimmed):
		encoded := strings.TrimPrefix(strings.TrimPrefix(trimmed, "0x"), "0X")
		raw, err = hex.DecodeString(encoded)
		if err != nil {
			return nil, errors.New("invalid Solana private key hexadecimal encoding")
		}
	default:
		raw, err = base58.Decode(trimmed)
		if err != nil {
			return nil, errors.New("invalid Solana private key Base58 encoding")
		}
	}

	switch len(raw) {
	case ed25519.SeedSize:
		return ed25519.NewKeyFromSeed(raw), nil
	case ed25519.PrivateKeySize:
		key := ed25519.PrivateKey(raw)
		derived := ed25519.NewKeyFromSeed(key.Seed())
		if !bytes.Equal(derived, key) {
			return nil, errors.New("invalid Solana private key: public key does not match seed")
		}
		return key, nil
	default:
		return nil, fmt.Errorf("invalid Solana private key length: got %d bytes, want 32-byte seed or 64-byte keypair", len(raw))
	}
}

func isHexEncodedSolanaKey(input string) bool {
	encoded := strings.TrimPrefix(strings.TrimPrefix(input, "0x"), "0X")
	if len(encoded) != ed25519.SeedSize*2 && len(encoded) != ed25519.PrivateKeySize*2 {
		return false
	}
	for _, r := range encoded {
		if (r < '0' || r > '9') && (r < 'a' || r > 'f') && (r < 'A' || r > 'F') {
			return false
		}
	}
	return true
}

func solanaWalletAddress(privateKey ed25519.PrivateKey) string {
	publicKey := privateKey.Public().(ed25519.PublicKey)
	return base58.Encode(publicKey)
}
