package e2etest

import (
	"os"
	"strings"
	"testing"
	"time"
)

func StringFromEnv(name string, fallback string) string {
	value := strings.TrimSpace(os.Getenv(name))
	if value == "" {
		return fallback
	}
	return value
}

func DurationFromEnv(t testing.TB, name string, fallback time.Duration) time.Duration {
	t.Helper()

	raw := strings.TrimSpace(os.Getenv(name))
	if raw == "" {
		return fallback
	}
	value, err := time.ParseDuration(raw)
	if err != nil {
		t.Fatalf("%s must be a Go duration such as 90s: %v", name, err)
	}
	if value <= 0 {
		t.Fatalf("%s must be positive, got %s", name, raw)
	}
	return value
}

func RequireEnvValue(t testing.TB, name string, expected string) {
	t.Helper()

	if value := strings.TrimSpace(os.Getenv(name)); value != expected {
		t.Fatalf("%s must be %q to run this E2E test", name, expected)
	}
}
