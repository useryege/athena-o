package main

import (
	"github.com/useryege/athena/cmd/athena-solana-discovery/commands"
	"os"
)

func main() {
	if err := commands.NewCommand().Execute(); err != nil {
		os.Exit(1)
	}
}
