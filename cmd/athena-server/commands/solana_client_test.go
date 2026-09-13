package commands

import "testing"

func TestSolanaClientConfigurationDoesNotBlockAPI(t *testing.T) {
	for _, tc := range []struct{ address, token string }{
		{"127.0.0.1:8112", ""},
		{"", "0123456789abcdef0123456789abcdef"},
	} {
		client := newOptionalSolanaClientset(tc.address, tc.token)
		if client != nil {
			t.Fatalf("address=%q: unexpected client", tc.address)
		}
	}
}
