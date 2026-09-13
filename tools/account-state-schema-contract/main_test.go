package main

import (
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"
)

func TestDatabaseDSNTargetsOnlyRandomDatabase(t *testing.T) {
	for _, source := range []string{"postgres://user:secret@localhost:5432/admin?sslmode=disable", "postgres://user:secret@localhost:5432/admin?dbname=admin&sslmode=disable", "host=localhost port=5432 user=user password=secret dbname=admin sslmode=disable"} {
		dsn, err := databaseDSN(source, "athena_contract_unique")
		require.NoError(t, err)
		config, err := pgx.ParseConfig(dsn)
		require.NoError(t, err)
		require.Equal(t, "athena_contract_unique", config.Database)
		require.Equal(t, "secret", config.Password)
	}
}
