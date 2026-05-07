package settings

import (
	"crypto/sha256"
	"encoding/base64"

	"github.com/useryege/athena/util/crypto"
)

// GetServerEncryptionKey generates a new server encryption key using the server signature as a passphrase
func (a *AthenaSettings) GetServerEncryptionKey() ([]byte, error) {
	return crypto.KeyFromPassphrase(string(a.ServerSignature))
}

// DexOAuth2ClientSecret calculates an arbitrary, but predictable OAuth2 client secret string derived
// from the server secret. This is called by the dex startup wrapper (athena-dex rundex), as well
// as the API server, such that they both independently come to the same conclusion of what the
// OAuth2 shared client secret should be.
func (a *AthenaSettings) DexOAuth2ClientSecret() string {
	h := sha256.New()
	_, err := h.Write(a.ServerSignature)
	if err != nil {
		panic(err)
	}

	sha := h.Sum(nil)

	return base64.URLEncoding.EncodeToString(sha)[:40]
}
