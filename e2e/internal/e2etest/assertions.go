package e2etest

import (
	"strings"
	"testing"
	"time"

	ethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

func RequireAddress(t testing.TB, name string, value string) {
	t.Helper()

	if !ethcommon.IsHexAddress(value) {
		t.Fatalf("expected %s to be a valid EVM address, got %q", name, value)
	}
}

func RequireHexBytes(t testing.TB, name string, value string, wantLen int) {
	t.Helper()

	decoded, err := hexutil.Decode(strings.TrimSpace(value))
	if err != nil {
		t.Fatalf("expected %s to be hex bytes: %v", name, err)
	}
	if len(decoded) != wantLen {
		t.Fatalf("expected %s to be %d bytes, got %d", name, wantLen, len(decoded))
	}
}

func RequireRFC3339Nano(t testing.TB, name string, value string) {
	t.Helper()

	if strings.TrimSpace(value) == "" {
		t.Fatalf("expected %s", name)
	}
	if _, err := time.Parse(time.RFC3339Nano, value); err != nil {
		t.Fatalf("expected %s to be RFC3339Nano, got %q: %v", name, value, err)
	}
}
