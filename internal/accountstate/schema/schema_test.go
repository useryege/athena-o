package schema

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLoadDSNRejectsLegacyOnly(t *testing.T) {
	_, err := LoadDSN(func(name string) (string, bool) {
		if name == "ATHENA_SERVER_POSTGRES_DSN" {
			return "postgres://legacy/db", true
		}
		return "", false
	})
	require.Error(t, err)
}
func TestLoadDSNRequiresExplicitValue(t *testing.T) {
	for _, value := range []string{"", "  "} {
		_, err := LoadDSN(func(string) (string, bool) { return value, true })
		require.Error(t, err)
	}
	dsn, err := LoadDSN(func(name string) (string, bool) {
		require.Equal(t, "ATHENA_ACCOUNT_STATE_POSTGRES_DSN", name)
		return "  postgres://shared/db  ", true
	})
	require.NoError(t, err)
	require.Equal(t, "postgres://shared/db", dsn)
}

func TestMalformedDSNDoesNotExposeCredentials(t *testing.T) {
	_, err := ConnectVerified(context.Background(), "postgres://user:private-password@host:invalid/db")
	require.ErrorIs(t, err, ErrConfiguration)
	require.NotContains(t, err.Error(), "private-password")
	require.NotContains(t, err.Error(), "postgres://")
}
