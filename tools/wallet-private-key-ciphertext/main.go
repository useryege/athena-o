package main

import (
	"encoding/hex"
	"flag"
	"fmt"
	"os"
	"strings"

	"golang.org/x/term"

	utilcrypto "github.com/useryege/athena/util/crypto"
)

const walletEncryptionKeyEnv = "ATHENA_WALLET_ENCRYPTION_KEY"

func main() {
	privateKeyFlag := flag.String("private-key", "", "plaintext private key (non-interactive; may appear in shell history)")
	encryptionKeyFlag := flag.String("encryption-key", "", "wallet encryption passphrase (defaults to ATHENA_WALLET_ENCRYPTION_KEY; non-interactive; may appear in shell history)")
	outputFormat := flag.String("format", "sql", "output format: sql or hex")
	flag.Parse()

	privateKey := *privateKeyFlag
	if privateKey == "" {
		var err error
		privateKey, err = readSecret("Private key: ")
		if err != nil {
			exitWithError("reading private key: %v", err)
		}
	}
	if privateKey == "" {
		exitWithError("private key is required")
	}

	encryptionKeyPassphrase := *encryptionKeyFlag
	if encryptionKeyPassphrase == "" {
		encryptionKeyPassphrase = os.Getenv(walletEncryptionKeyEnv)
	}
	if encryptionKeyPassphrase == "" {
		var err error
		encryptionKeyPassphrase, err = readSecret("Wallet encryption key: ")
		if err != nil {
			exitWithError("reading wallet encryption key: %v", err)
		}
	}
	if encryptionKeyPassphrase == "" {
		exitWithError("wallet encryption key is required")
	}

	key, err := utilcrypto.KeyFromPassphrase(encryptionKeyPassphrase)
	if err != nil {
		exitWithError("deriving wallet encryption key: %v", err)
	}
	ciphertext, err := utilcrypto.Encrypt([]byte(privateKey), key)
	if err != nil {
		exitWithError("encrypting private key: %v", err)
	}

	ciphertextHex := hex.EncodeToString(ciphertext)
	switch strings.ToLower(strings.TrimSpace(*outputFormat)) {
	case "hex":
		fmt.Println(ciphertextHex)
	case "sql":
		fmt.Printf("decode('%s', 'hex')\n", ciphertextHex)
	default:
		exitWithError("unsupported -format %q; use sql or hex", *outputFormat)
	}
}

func readSecret(prompt string) (string, error) {
	fmt.Fprint(os.Stderr, prompt)
	secretBytes, err := term.ReadPassword(int(os.Stdin.Fd()))
	fmt.Fprintln(os.Stderr)
	if err != nil {
		return "", err
	}
	return string(secretBytes), nil
}

func exitWithError(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "error: "+format+"\n", args...)
	os.Exit(1)
}
