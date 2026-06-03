package main

import (
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"k8s.io/klog/v2"

	athenaApplicationCommands "github.com/useryege/athena/cmd/athena-application/commands"
	athenaMigrateCommands "github.com/useryege/athena/cmd/athena-migrate/commands"
	athenaNotificationCommands "github.com/useryege/athena/cmd/athena-notification/commands"
	athenaPolymarketCommands "github.com/useryege/athena/cmd/athena-polymarket/commands"
	athenaServerCommands "github.com/useryege/athena/cmd/athena-server/commands"
	athenaSolidityCommands "github.com/useryege/athena/cmd/athena-solidity/commands"
	athenaWalletCommands "github.com/useryege/athena/cmd/athena-wallet/commands"
	athenaWormCommands "github.com/useryege/athena/cmd/athena-worm/commands"
	"github.com/useryege/athena/util/log"
)

const (
	binaryNameEnv = "ATHENA_BINARY_NAME"
)

func init() {
	// Make sure klog uses the configured log level and format.
	klog.SetLogger(log.NewLogrusLogger(log.NewWithCurrentConfig()))
}

func main() {
	var command *cobra.Command

	binaryName := filepath.Base(os.Args[0])
	if val := os.Getenv(binaryNameEnv); val != "" {
		binaryName = val
	}

	// var isAthenaCLI bool

	switch binaryName {
	case "athena-server":
		command = athenaServerCommands.NewCommand()
	case "athena-notification":
		command = athenaNotificationCommands.NewCommand()
	case "athena-polymarket":
		command = athenaPolymarketCommands.NewCommand()
	case "athena-solidity":
		command = athenaSolidityCommands.NewCommand()
	case "athena-wallet":
		command = athenaWalletCommands.NewCommand()
	case "athena-application":
		command = athenaApplicationCommands.NewCommand()
	case "athena-worm":
		command = athenaWormCommands.NewCommand()
	case "athena-migrate":
		command = athenaMigrateCommands.NewCommand()
	default:
		os.Exit(1)
	}

	err := command.Execute()
	// if an error is present, try to look for various scenarios
	// such as if the error is from the execution of a normal athena command,
	// unknown command error or any other.
	if err != nil {
		os.Exit(1)
	}
}
