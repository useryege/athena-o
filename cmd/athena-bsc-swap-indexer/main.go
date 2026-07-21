package main

import (
	"os"

	athenaBSCSwapIndexerCommands "github.com/useryege/athena/cmd/athena-bsc-swap-indexer/commands"
)

func main() {
	if err := athenaBSCSwapIndexerCommands.NewCommand().Execute(); err != nil {
		os.Exit(1)
	}
}
