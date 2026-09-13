package commands

import (
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCommandRequiresExplicitSharedDSN(t *testing.T) {
	t.Setenv("ATHENA_ACCOUNT_STATE_POSTGRES_DSN", "")
	t.Setenv("ATHENA_SERVER_POSTGRES_DSN", "postgres://legacy/db")
	for _, action := range []string{"up", "verify"} {
		cmd := NewCommand()
		cmd.SetArgs([]string{action})
		cmd.SetErr(&bytes.Buffer{})
		require.ErrorContains(t, cmd.Execute(), "ATHENA_ACCOUNT_STATE_POSTGRES_DSN")
	}
}
func TestCommandRejectsInvalidTimeoutAndArguments(t *testing.T) {
	for _, args := range [][]string{{"up", "--timeout=0s"}, {"verify", "--timeout=-1s"}, {"verify", "unexpected"}, {"status"}} {
		cmd := NewCommand()
		cmd.SetArgs(args)
		cmd.SetErr(&bytes.Buffer{})
		require.Error(t, cmd.Execute(), strings.Join(args, " "))
	}
}
