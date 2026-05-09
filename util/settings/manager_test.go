package settings

import (
	"encoding/base64"
	"testing"

	"github.com/useryege/athena/util"
	"github.com/useryege/athena/util/password"
)

func TestManager_InitializeSettings(t *testing.T) {
	initialPasswordBytes, err := util.MakeSignature(initialPasswordLength)
	if err != nil {
		t.Fatalf("failed to make signature: %v", err)
	}

	initialPassword := base64.RawURLEncoding.EncodeToString(initialPasswordBytes)

	hashedPassword, err := password.HashPassword(initialPassword)
	if err != nil {
		t.Fatalf("failed to hash password: %v", err)
	}

	t.Logf("initial password: %s", initialPassword)
	t.Logf("hashed password: %s", hashedPassword)
}
