package commands

import (
	"context"
	"io"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestCatalogBudgetRejectsInvalidSettingsBeforeDependencies(t *testing.T) {
	for _, tc := range []struct{ name, value string }{{"zero", "0s"}, {"negative", "-1s"}, {"malformed", "bad"}, {"below attempt", "4s"}} {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("ATHENA_WORM_TRADING_CATALOG_BUDGET", tc.value)
			cmd := NewCommand()
			ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
			defer cancel()
			cmd.SetContext(ctx)
			cmd.SetOut(io.Discard)
			cmd.SetErr(io.Discard)
			cmd.SetArgs([]string{})
			err := cmd.Execute()
			require.ErrorContains(t, err, "catalog budget")
		})
	}
}
func TestCatalogBudgetFlagOverridesEnvironment(t *testing.T) {
	t.Setenv("ATHENA_WORM_TRADING_CATALOG_BUDGET", "45s")
	cmd := NewCommand()
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	cmd.SetContext(ctx)
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)
	cmd.SetArgs([]string{"--worm-catalog-budget", "1s"})
	require.ErrorContains(t, cmd.Execute(), "catalog budget")
}
