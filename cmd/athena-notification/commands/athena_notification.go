package commands

import (
	"fmt"

	"github.com/spf13/cobra"
)

func NewCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "athena-notification",
		Short: "Run the Athena Notification",
		Long:  "The Athena Notification is responsible for managing the Notification lifecycle.",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("Hello, World!")
		},
	}
}
