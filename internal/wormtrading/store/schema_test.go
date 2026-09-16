package store

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTradingSourceRequiresExplicitDSN(t *testing.T) {
	t.Setenv(DSNEnv, " ")
	store, err := NewSQLStoreSource()(context.Background())
	require.Nil(t, store)
	require.ErrorContains(t, err, DSNEnv)
}
