package wallet

import (
	"crypto/ed25519"
	"strings"
	"testing"

	"github.com/mr-tron/base58/base58"
)

const testMnemonic = "test test test test test test test test test test test junk"

func TestMnemonicWalletKeyMaterialDerivesEVMWallet(t *testing.T) {
	material, err := mnemonicWalletKeyMaterial(walletChainETH, testMnemonic, walletSourceMnemonic)
	if err != nil {
		t.Fatalf("mnemonic wallet material: %v", err)
	}
	if material.chain != walletChainETH {
		t.Fatalf("chain = %q, want %q", material.chain, walletChainETH)
	}
	if !strings.HasPrefix(material.address, "0x") || len(material.address) != 42 {
		t.Fatalf("address = %q, want EVM address", material.address)
	}
	if material.addressKey != strings.ToLower(material.address) {
		t.Fatalf("address key = %q, want lower-case address", material.addressKey)
	}
	if !strings.HasPrefix(material.privateKey, "0x") || len(material.privateKey) != 66 {
		t.Fatalf("private key = %q, want 32-byte hex private key", material.privateKey)
	}
	if material.mnemonic != testMnemonic {
		t.Fatalf("mnemonic = %q, want normalized test mnemonic", material.mnemonic)
	}
	if material.derivationPath != evmDerivationPath {
		t.Fatalf("derivation path = %q, want %q", material.derivationPath, evmDerivationPath)
	}
}

func TestMnemonicWalletKeyMaterialDerivesSolanaWallet(t *testing.T) {
	material, err := mnemonicWalletKeyMaterial(walletChainSolana, testMnemonic, walletSourceMnemonic)
	if err != nil {
		t.Fatalf("mnemonic wallet material: %v", err)
	}
	if material.address == "" || material.addressKey != material.address {
		t.Fatalf("address/address key = %q/%q", material.address, material.addressKey)
	}
	raw, err := base58.Decode(material.privateKey)
	if err != nil {
		t.Fatalf("decode private key: %v", err)
	}
	if len(raw) != ed25519.PrivateKeySize {
		t.Fatalf("private key length = %d, want %d", len(raw), ed25519.PrivateKeySize)
	}
	if material.derivationPath != solanaDerivationPath {
		t.Fatalf("derivation path = %q, want %q", material.derivationPath, solanaDerivationPath)
	}
}

func TestPrivateKeyWalletKeyMaterialParsesSolanaSeed(t *testing.T) {
	seed := make([]byte, ed25519.SeedSize)
	for i := range seed {
		seed[i] = byte(i + 1)
	}
	material, err := privateKeyWalletKeyMaterial(walletChainSolana, base58.Encode(seed))
	if err != nil {
		t.Fatalf("private key wallet material: %v", err)
	}
	raw, err := base58.Decode(material.privateKey)
	if err != nil {
		t.Fatalf("decode private key: %v", err)
	}
	if len(raw) != ed25519.PrivateKeySize {
		t.Fatalf("private key length = %d, want %d", len(raw), ed25519.PrivateKeySize)
	}
	if material.source != walletSourcePrivateKey {
		t.Fatalf("source = %q, want %q", material.source, walletSourcePrivateKey)
	}
}
