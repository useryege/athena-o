package accountcredentials

import (
	"fmt"
	"os"
	"strings"

	log "github.com/sirupsen/logrus"

	"github.com/useryege/athena/util"
)

// LoadJWTSigningKey reads ATHENA_JWT_SECRET or its _FILE form. Development
// without either setting receives a process-lifetime key, preserving the
// existing local behavior while making account identity fully database-owned.
func LoadJWTSigningKey() ([]byte, error) {
	value, err := envOrFile("ATHENA_JWT_SECRET")
	if err != nil {
		return nil, err
	}
	key := []byte(value)
	if len(key) == 0 {
		key, err = util.MakeSignature(32)
		if err != nil {
			return nil, fmt.Errorf("generate JWT signing key: %w", err)
		}
		log.Warn("Generated transient JWT secret because ATHENA_JWT_SECRET is not set; existing sessions and API Keys will be invalid after restart")
	}
	if len(key) < minimumJWTSigningKeyBytes {
		return nil, fmt.Errorf("ATHENA_JWT_SECRET must contain at least %d bytes", minimumJWTSigningKeyBytes)
	}
	return append([]byte(nil), key...), nil
}

func envOrFile(name string) (string, error) {
	if value := os.Getenv(name); value != "" {
		return value, nil
	}
	if path := os.Getenv(name + "_FILE"); path != "" {
		data, err := os.ReadFile(path)
		if err != nil {
			return "", fmt.Errorf("failed reading %s_FILE: %w", name, err)
		}
		return strings.TrimSpace(string(data)), nil
	}
	return "", nil
}
