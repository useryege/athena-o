package commands

import (
	"fmt"

	"github.com/spf13/cobra"
)

func NewCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "athena-application-controller",
		Short: "Run the Athena Application Controller",
		Long:  "The Athena Application Controller is responsible for managing the application lifecycle.",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("Hello, World!")
		},
	}
}
