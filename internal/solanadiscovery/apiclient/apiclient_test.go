package apiclient

import (
	"context"
	"testing"
)

func TestInternalClientRequiresStrongToken(t *testing.T) {
	for _, token := range []string{"", "short", "0123456789abcdef0123456789abc de"} {
		if _, err := NewSolanaClientset("127.0.0.1:8112", token); err == nil {
			t.Fatalf("accepted token %q", token)
		}
	}
}

func TestBearerCredentialsCarryOnlyServiceToken(t *testing.T) {
	credentials := internalBearerCredentials{authorization: "Bearer 0123456789abcdef0123456789abcdef"}
	got, err := credentials.GetRequestMetadata(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got["authorization"] != "Bearer 0123456789abcdef0123456789abcdef" {
		t.Fatalf("metadata = %v", got)
	}
}
