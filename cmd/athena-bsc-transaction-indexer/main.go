package main

import (
	"os"

	athenaBSCTransactionIndexerCommands "github.com/useryege/athena/cmd/athena-bsc-transaction-indexer/commands"
)

func main() {
	if err := athenaBSCTransactionIndexerCommands.NewCommand().Execute(); err != nil {
		os.Exit(1)
	}
}
