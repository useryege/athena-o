package runtime

import (
	"strings"
	"testing"

	ethcommon "github.com/ethereum/go-ethereum/common"
)

func TestNormalizeModeDefaultsEmptyToAPI(t *testing.T) {
	if got := NormalizeMode(""); got != ModeAPI {
		t.Fatalf("NormalizeMode(\"\") = %q, want %q", got, ModeAPI)
	}
	if got := NormalizeMode("  "); got != ModeAPI {
		t.Fatalf("NormalizeMode(spaces) = %q, want %q", got, ModeAPI)
	}
}

func TestParseLiquidityLockerAddresses(t *testing.T) {
	values := []string{
		"0x0000000000000000000000000000000000000001",
		" 0x0000000000000000000000000000000000000002 ",
	}
	got, err := parseLiquidityLockerAddresses(values)
	if err != nil {
		t.Fatalf("parseLiquidityLockerAddresses() error = %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("parseLiquidityLockerAddresses() len = %d, want 2", len(got))
	}
	if got[0] != ethcommon.HexToAddress(values[0]) {
		t.Fatalf("first address = %s, want %s", got[0], values[0])
	}
	if got[1] != ethcommon.HexToAddress(strings.TrimSpace(values[1])) {
		t.Fatalf("second address = %s, want %s", got[1], strings.TrimSpace(values[1]))
	}
}

func TestParseLiquidityLockerAddressesRejectsEmptyAndInvalid(t *testing.T) {
	tests := []struct {
		name   string
		values []string
	}{
		{name: "empty", values: []string{""}},
		{name: "invalid", values: []string{"not-an-address"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := parseLiquidityLockerAddresses(tt.values); err == nil {
				t.Fatal("parseLiquidityLockerAddresses() error = nil, want error")
			}
		})
	}
}

func TestParseRequiredAddress(t *testing.T) {
	value := "0x0000000000000000000000000000000000000001"
	got, err := parseRequiredAddress("test address", value, "--test-address", "TEST_ADDRESS")
	if err != nil {
		t.Fatalf("parseRequiredAddress() error = %v", err)
	}
	if got != ethcommon.HexToAddress(value) {
		t.Fatalf("parseRequiredAddress() = %s, want %s", got, value)
	}
}

func TestParseRequiredAddressRejectsMissingInvalidAndZero(t *testing.T) {
	tests := []struct {
		name  string
		value string
	}{
		{name: "missing", value: ""},
		{name: "invalid", value: "not-an-address"},
		{name: "zero", value: "0x0000000000000000000000000000000000000000"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := parseRequiredAddress("test address", tt.value, "--test-address", "TEST_ADDRESS"); err == nil {
				t.Fatal("parseRequiredAddress() error = nil, want error")
			}
		})
	}
}
