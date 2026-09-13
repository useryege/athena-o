package schema

import (
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

const DSNEnv = "ATHENA_ACCOUNT_STATE_POSTGRES_DSN"

var ErrConfiguration = errors.New("account-state configuration invalid")

// LoadDSN requires one explicit shared database location. No legacy lookup or
// local database fallback is permitted.
func LoadDSN(lookup func(string) (string, bool)) (string, error) {
	value, _ := lookup(DSNEnv)
	value = strings.TrimSpace(value)
	if value == "" {
		return "", fmt.Errorf("%w: %s is required", ErrConfiguration, DSNEnv)
	}
	return value, nil
}
func validateDSN(dsn string) error {
	if strings.TrimSpace(dsn) == "" {
		return fmt.Errorf("%w: %s is required", ErrConfiguration, DSNEnv)
	}
	if _, err := pgxpool.ParseConfig(dsn); err != nil {
		// pgx parse errors can include the original DSN, including its password.
		return fmt.Errorf("%w: %s is malformed", ErrConfiguration, DSNEnv)
	}
	return nil
}
