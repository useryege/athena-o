package main

import (
	"fmt"
	"github.com/useryege/athena/cmd/athena-server/commands"
	"os"
)

func main() {
	if err := commands.NewCommand().Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
