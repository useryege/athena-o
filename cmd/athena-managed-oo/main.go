package main

import (
	"fmt"
	"os"

	"github.com/useryege/athena/cmd/athena-managed-oo/commands"
)

func main() {
	if err := commands.NewCommand().Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
