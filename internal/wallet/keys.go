package wallet

import (
	"bytes"
	"crypto/ed25519"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/anyproto/go-slip10"
	"github.com/ethereum/go-ethereum/common/hexutil"
	ethcrypto "github.com/ethereum/go-ethereum/crypto"
	"github.com/mr-tron/base58/base58"
	"github.com/tyler-smith/go-bip32"
	"github.com/tyler-smith/go-bip39"
)

const (
	walletChainETH    = "ETH"
	walletChainBSC    = "BSC"
	walletChainBase   = "BASE"
	walletChainSolana = "SOLANA"

	walletSourceCreated    = "created"
	walletSourcePrivateKey = "private_key"
	walletSourceMnemonic   = "mnemonic"

	evmDerivationPath    = "m/44'/60'/0'/0/0"
	solanaDerivationPath = "m/44'/501'/0'/0'"
)

type walletKeyMaterial struct {
	chain          string
	address        string
	addressKey     string
	privateKey     string
	mnemonic       string
	source         string
	derivationPath string
}

func normalizeWalletChain(input string) (string, error) {
	chain := strings.ToUpper(strings.TrimSpace(input))
	switch chain {
	case walletChainETH, walletChainBSC, walletChainBase, walletChainSolana:
		return chain, nil
	case "":
		return "", errors.New("chain is required")
	default:
		return "", errors.New("chain must be one of ETH, BSC, BASE, SOLANA")
	}
}

func createWalletKeyMaterial(chain string) (walletKeyMaterial, error) {
	entropy, err := bip39.NewEntropy(128)
	if err != nil {
		return walletKeyMaterial{}, fmt.Errorf("generate mnemonic entropy: %w", err)
	}
	mnemonic, err := bip39.NewMnemonic(entropy)
	if err != nil {
		return walletKeyMaterial{}, fmt.Errorf("generate mnemonic: %w", err)
	}
	return mnemonicWalletKeyMaterial(chain, mnemonic, walletSourceCreated)
}

func privateKeyWalletKeyMaterial(chain string, privateKey string) (walletKeyMaterial, error) {
	if isEVMChain(chain) {
		return evmPrivateKeyMaterial(chain, privateKey)
	}
	return solanaPrivateKeyMaterial(chain, privateKey)
}

func mnemonicWalletKeyMaterial(chain string, mnemonic string, source string) (walletKeyMaterial, error) {
	mnemonic = normalizeMnemonic(mnemonic)
	if !bip39.IsMnemonicValid(mnemonic) {
		return walletKeyMaterial{}, errors.New("mnemonic is invalid")
	}
	if isEVMChain(chain) {
		return evmMnemonicMaterial(chain, mnemonic, source)
	}
	return solanaMnemonicMaterial(chain, mnemonic, source)
}

func evmPrivateKeyMaterial(chain string, input string) (walletKeyMaterial, error) {
	privateKey, err := ethcrypto.HexToECDSA(strings.TrimPrefix(strings.TrimSpace(input), "0x"))
	if err != nil {
		return walletKeyMaterial{}, fmt.Errorf("invalid EVM private key: %w", err)
	}
	address := ethcrypto.PubkeyToAddress(privateKey.PublicKey).Hex()
	return walletKeyMaterial{
		chain:      chain,
		address:    address,
		addressKey: strings.ToLower(address),
		privateKey: hexutil.Encode(ethcrypto.FromECDSA(privateKey)),
		source:     walletSourcePrivateKey,
	}, nil
}

func evmMnemonicMaterial(chain string, mnemonic string, source string) (walletKeyMaterial, error) {
	seed := bip39.NewSeed(mnemonic, "")
	key, err := bip32.NewMasterKey(seed)
	if err != nil {
		return walletKeyMaterial{}, fmt.Errorf("derive EVM master key: %w", err)
	}
	for _, child := range []uint32{
		44 + bip32.FirstHardenedChild,
		60 + bip32.FirstHardenedChild,
		0 + bip32.FirstHardenedChild,
		0,
		0,
	} {
		key, err = key.NewChildKey(child)
		if err != nil {
			return walletKeyMaterial{}, fmt.Errorf("derive EVM child key: %w", err)
		}
	}
	privateKey, err := ethcrypto.ToECDSA(key.Key)
	if err != nil {
		return walletKeyMaterial{}, fmt.Errorf("derive EVM private key: %w", err)
	}
	address := ethcrypto.PubkeyToAddress(privateKey.PublicKey).Hex()
	return walletKeyMaterial{
		chain:          chain,
		address:        address,
		addressKey:     strings.ToLower(address),
		privateKey:     hexutil.Encode(ethcrypto.FromECDSA(privateKey)),
		mnemonic:       mnemonic,
		source:         source,
		derivationPath: evmDerivationPath,
	}, nil
}

func solanaPrivateKeyMaterial(chain string, input string) (walletKeyMaterial, error) {
	privateKey, err := parseSolanaPrivateKey(input)
	if err != nil {
		return walletKeyMaterial{}, err
	}
	address := solanaWalletAddress(privateKey)
	return walletKeyMaterial{
		chain:      chain,
		address:    address,
		addressKey: address,
		privateKey: base58.Encode(privateKey),
		source:     walletSourcePrivateKey,
	}, nil
}

func solanaMnemonicMaterial(chain string, mnemonic string, source string) (walletKeyMaterial, error) {
	node, err := slip10.DeriveForPath(solanaDerivationPath, bip39.NewSeed(mnemonic, ""))
	if err != nil {
		return walletKeyMaterial{}, fmt.Errorf("derive Solana private key: %w", err)
	}
	publicKey, privateKey := node.Keypair()
	address := base58.Encode(publicKey)
	return walletKeyMaterial{
		chain:          chain,
		address:        address,
		addressKey:     address,
		privateKey:     base58.Encode(privateKey),
		mnemonic:       mnemonic,
		source:         source,
		derivationPath: solanaDerivationPath,
	}, nil
}

func parseSolanaPrivateKey(input string) (ed25519.PrivateKey, error) {
	trimmed := strings.TrimSpace(input)
	if trimmed == "" {
		return nil, errors.New("solana private key is required")
	}

	var raw []byte
	var err error
	if strings.HasPrefix(trimmed, "[") {
		if err := json.Unmarshal([]byte(trimmed), &raw); err != nil {
			return nil, fmt.Errorf("invalid solana private key json: %w", err)
		}
	} else if isHexEncodedSolanaKey(trimmed) {
		raw, err = hex.DecodeString(trimmed)
		if err != nil {
			return nil, fmt.Errorf("invalid solana private key hex: %w", err)
		}
	} else {
		raw, err = base58.Decode(trimmed)
		if err != nil {
			return nil, fmt.Errorf("invalid solana private key base58: %w", err)
		}
	}

	switch len(raw) {
	case ed25519.SeedSize:
		return ed25519.NewKeyFromSeed(raw), nil
	case ed25519.PrivateKeySize:
		key := ed25519.PrivateKey(raw)
		derived := ed25519.NewKeyFromSeed(key.Seed())
		if !bytes.Equal(derived, key) {
			return nil, errors.New("invalid solana private key: public key does not match seed")
		}
		return key, nil
	default:
		return nil, fmt.Errorf("invalid solana private key length: got %d bytes, want 32-byte seed or 64-byte keypair", len(raw))
	}
}

func isHexEncodedSolanaKey(input string) bool {
	if len(input) != ed25519.SeedSize*2 && len(input) != ed25519.PrivateKeySize*2 {
		return false
	}
	for _, r := range input {
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

func normalizeMnemonic(mnemonic string) string {
	return strings.ToLower(strings.Join(strings.Fields(mnemonic), " "))
}

func isEVMChain(chain string) bool {
	switch chain {
	case walletChainETH, walletChainBSC, walletChainBase:
		return true
	default:
		return false
	}
}
