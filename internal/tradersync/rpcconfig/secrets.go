package rpcconfig

import (
	"fmt"
	"os"
	"strings"
	"unicode"
)

// ResolveSecret loads one explicitly chosen source. Only one trailing LF is
// removed from file content; credential validation decides other whitespace.
func ResolveSecret(lookup func(string) (string, bool), name string) (string, error) {
	value, direct := lookup(name)
	path, file := lookup(name + "_FILE")
	if direct && file {
		return "", fmt.Errorf("%s and %s_FILE are mutually exclusive", name, name)
	}
	if !file {
		return value, nil
	}
	if path == "" {
		return "", fmt.Errorf("%s_FILE requires a path", name)
	}
	content, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("cannot read %s_FILE", name)
	}
	return strings.TrimSuffix(string(content), "\n"), nil
}

func validateToken(token string) error {
	if len(token) < 32 || strings.ContainsFunc(token, func(r rune) bool { return unicode.IsSpace(r) || unicode.IsControl(r) }) {
		return fmt.Errorf("Trader Sync internal token requires at least 32 bytes and no whitespace or control characters")
	}
	return nil
}
