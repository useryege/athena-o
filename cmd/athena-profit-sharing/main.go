package main

import (
	"fmt"
	"os"

	"github.com/useryege/athena/cmd/athena-profit-sharing/commands"
)

func main() {
	if err := commands.NewCommand().Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
