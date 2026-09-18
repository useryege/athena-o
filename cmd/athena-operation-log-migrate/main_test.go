package main

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMigrationRequiresExplicitAccountStateDSN(t *testing.T) {
	t.Setenv(dsnEnv, "")
	for _, action := range []string{"up", "verify"} {
		cmd := newCommand()
		cmd.SetArgs([]string{action})
		cmd.SetErr(&bytes.Buffer{})
		require.ErrorContains(t, cmd.Execute(), dsnEnv)
	}
}

func TestMigrationRejectsInvalidTimeoutAndArguments(t *testing.T) {
	for _, args := range [][]string{{"up", "--timeout=0s"}, {"verify", "--timeout=-1s"}, {"verify", "unexpected"}, {"status"}} {
		cmd := newCommand()
		cmd.SetArgs(args)
		cmd.SetErr(&bytes.Buffer{})
		require.Error(t, cmd.Execute())
	}
}
