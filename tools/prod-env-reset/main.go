package main

import (
	"crypto/rand"
	"encoding/base64"
	"flag"
	"fmt"
	"math/big"
	"os"
	"strings"
)

const servicePasswordAlphabet = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

var targetKeys = []string{
	"POSTGRES_PASSWORD",
	"REDIS_PASSWORD",
	"MINIO_ROOT_PASSWORD",
	"ATHENA_ACCOUNT_AVATAR_S3_ACCESS_KEY_ID",
	"ATHENA_ACCOUNT_AVATAR_S3_SECRET_ACCESS_KEY",
	"ATHENA_JWT_SECRET",
	"ATHENA_WALLET_INTERNAL_AUTH_TOKEN",
	"ATHENA_WALLET_WORM_EXECUTION_SIGNER_TOKEN",
	"ATHENA_WORM_TRADING_INTERNAL_AUTH_TOKEN",
	"ATHENA_WORM_TRADING_CREDENTIAL_ENCRYPTION_KEY",
}

func main() {
	envFile := flag.String("env-file", ".env.prod", "production env file to update")
	flag.Parse()

	replacements, err := newSecretValues()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error generating secrets: %v\n", err)
		os.Exit(1)
	}

	data, err := os.ReadFile(*envFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error reading %s: %v\n", *envFile, err)
		os.Exit(1)
	}

	updated := updateEnv(string(data), replacements)
	if err := os.WriteFile(*envFile, []byte(updated), 0o600); err != nil {
		fmt.Fprintf(os.Stderr, "error writing %s: %v\n", *envFile, err)
		os.Exit(1)
	}
	if err := os.Chmod(*envFile, 0o600); err != nil {
		fmt.Fprintf(os.Stderr, "error securing %s: %v\n", *envFile, err)
		os.Exit(1)
	}

	fmt.Fprintf(os.Stderr, "updated %s: %s\n", *envFile, strings.Join(targetKeys, ", "))
}

func newSecretValues() (map[string]string, error) {
	postgresPassword, err := randomString(32, servicePasswordAlphabet)
	if err != nil {
		return nil, err
	}
	redisPassword, err := randomString(32, servicePasswordAlphabet)
	if err != nil {
		return nil, err
	}
	minioRootPassword, err := randomString(40, servicePasswordAlphabet)
	if err != nil {
		return nil, err
	}
	minioAccessKey, err := randomString(20, servicePasswordAlphabet)
	if err != nil {
		return nil, err
	}
	minioSecretKey, err := randomString(40, servicePasswordAlphabet)
	if err != nil {
		return nil, err
	}
	jwtSecretBytes := make([]byte, 32)
	if _, err := rand.Read(jwtSecretBytes); err != nil {
		return nil, err
	}
	walletInternalAuthToken, err := randomString(40, servicePasswordAlphabet)
	if err != nil {
		return nil, err
	}
	walletWormExecutionSignerToken, err := randomStringDifferentFrom(
		40,
		servicePasswordAlphabet,
		walletInternalAuthToken,
	)
	if err != nil {
		return nil, err
	}
	wormTradingInternalAuthToken, err := randomStringDifferentFrom(
		40,
		servicePasswordAlphabet,
		walletInternalAuthToken,
		walletWormExecutionSignerToken,
	)
	if err != nil {
		return nil, err
	}
	wormTradingCredentialEncryptionKey, err := randomString(48, servicePasswordAlphabet)
	if err != nil {
		return nil, err
	}

	return map[string]string{
		"POSTGRES_PASSWORD":                             postgresPassword,
		"REDIS_PASSWORD":                                redisPassword,
		"MINIO_ROOT_PASSWORD":                           minioRootPassword,
		"ATHENA_ACCOUNT_AVATAR_S3_ACCESS_KEY_ID":        minioAccessKey,
		"ATHENA_ACCOUNT_AVATAR_S3_SECRET_ACCESS_KEY":    minioSecretKey,
		"ATHENA_JWT_SECRET":                             base64.StdEncoding.EncodeToString(jwtSecretBytes),
		"ATHENA_WALLET_INTERNAL_AUTH_TOKEN":             walletInternalAuthToken,
		"ATHENA_WALLET_WORM_EXECUTION_SIGNER_TOKEN":     walletWormExecutionSignerToken,
		"ATHENA_WORM_TRADING_INTERNAL_AUTH_TOKEN":       wormTradingInternalAuthToken,
		"ATHENA_WORM_TRADING_CREDENTIAL_ENCRYPTION_KEY": wormTradingCredentialEncryptionKey,
	}, nil
}

func randomStringDifferentFrom(length int, alphabet string, forbidden ...string) (string, error) {
	for {
		value, err := randomString(length, alphabet)
		if err != nil {
			return "", err
		}
		distinct := true
		for _, existing := range forbidden {
			if value == existing {
				distinct = false
				break
			}
		}
		if distinct {
			return value, nil
		}
	}
}

func randomString(length int, alphabet string) (string, error) {
	out := make([]byte, length)
	max := big.NewInt(int64(len(alphabet)))
	for i := range out {
		n, err := rand.Int(rand.Reader, max)
		if err != nil {
			return "", err
		}
		out[i] = alphabet[n.Int64()]
	}
	return string(out), nil
}

func updateEnv(input string, replacements map[string]string) string {
	hadTrailingNewline := strings.HasSuffix(input, "\n")
	lines := []string{}
	if input != "" {
		lines = strings.Split(strings.TrimSuffix(input, "\n"), "\n")
	}
	seen := map[string]bool{}

	for i, line := range lines {
		key, ok := envLineKey(line)
		if !ok {
			continue
		}
		value, shouldReplace := replacements[key]
		if !shouldReplace {
			continue
		}
		lines[i] = fmt.Sprintf("%s='%s'", key, value)
		seen[key] = true
	}

	for _, key := range targetKeys {
		if seen[key] {
			continue
		}
		lines = append(lines, fmt.Sprintf("%s='%s'", key, replacements[key]))
	}

	output := strings.Join(lines, "\n")
	if hadTrailingNewline || output != "" {
		output += "\n"
	}
	return output
}

func envLineKey(line string) (string, bool) {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" || strings.HasPrefix(trimmed, "#") || strings.HasPrefix(trimmed, "export ") {
		return "", false
	}

	key, _, ok := strings.Cut(trimmed, "=")
	if !ok {
		return "", false
	}
	key = strings.TrimSpace(key)
	if key == "" {
		return "", false
	}
	for _, r := range key {
		if (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' {
			continue
		}
		return "", false
	}
	return key, true
}
