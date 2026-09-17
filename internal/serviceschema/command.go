package serviceschema

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/cobra"
)

// NewCommand reads only the schema DSN; it never invokes the parent's business
// startup path or requires upstream credentials.
func NewCommand(spec Spec) *cobra.Command {
	var timeout time.Duration
	root := &cobra.Command{Use: "schema", Short: "Prepare or verify this service's database schema", Args: cobra.NoArgs, RunE: func(*cobra.Command, []string) error { return fmt.Errorf("choose schema up or verify") }}
	root.PersistentFlags().DurationVar(&timeout, "timeout", DefaultTimeout, "Maximum schema operation duration")
	for _, action := range []string{"up", "verify"} {
		root.AddCommand(&cobra.Command{Use: action, Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, _ []string) error {
			if timeout <= 0 {
				return fmt.Errorf("schema timeout must be positive")
			}
			ctx, stop := signal.NotifyContext(cmd.Context(), os.Interrupt, syscall.SIGTERM)
			defer stop()
			ctx, cancel := context.WithTimeout(ctx, timeout)
			defer cancel()
			if action == "up" {
				return spec.Up(ctx)
			}
			pool, err := spec.ConnectVerified(ctx)
			if err != nil {
				return err
			}
			pool.Close()
			return nil
		}})
	}
	return root
}
