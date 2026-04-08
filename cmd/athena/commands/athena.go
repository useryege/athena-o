package commands

import (
	"fmt"

	"github.com/spf13/cobra"
)

func NewCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "athena",
		Short: "Run the Athena",
		Long:  "The Athena is responsible for managing the Athena lifecycle.",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("Hello, World!")
		},
	}
}
