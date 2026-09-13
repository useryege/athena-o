package main

import (
	"fmt"
	"github.com/useryege/athena/cmd/athena-notification/commands"
	"os"
)

func main() {
	if err := commands.NewCommand().Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
