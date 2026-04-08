package commands

import (
	"fmt"

	"github.com/spf13/cobra"
)

func NewCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "athena-dex",
		Short: "Run the Athena Dex",
		Long:  "The Athena Dex is responsible for managing the Dex lifecycle.",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("Hello, World!")
		},
	}
}
