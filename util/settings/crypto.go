package settings

import (
	"github.com/useryege/athena/util/crypto"
)

// GetServerEncryptionKey generates a new server encryption key using the server signature as a passphrase
func (a *AthenaSettings) GetServerEncryptionKey() ([]byte, error) {
	return crypto.KeyFromPassphrase(string(a.ServerSignature))
}
