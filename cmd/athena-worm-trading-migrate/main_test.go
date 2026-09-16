package main

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMigrationRequiresExplicitDSNAndPositiveTimeout(t *testing.T) {
	t.Setenv("ATHENA_WORM_TRADING_POSTGRES_DSN", "")
	for _, action := range []string{"up", "verify"} {
		t.Run(action, func(t *testing.T) {
			cmd := newCommand()
			cmd.SetArgs([]string{action})
			require.ErrorContains(t, cmd.Execute(), "ATHENA_WORM_TRADING_POSTGRES_DSN")
			cmd = newCommand()
			cmd.SetArgs([]string{action, "--timeout=0s"})
			require.ErrorContains(t, cmd.Execute(), "timeout must be positive")
		})
	}
}
