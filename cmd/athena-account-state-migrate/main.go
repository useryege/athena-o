package main

import (
	"fmt"
	"os"

	"github.com/useryege/athena/cmd/athena-account-state-migrate/commands"
)

func main() {
	if err := commands.NewCommand().Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
