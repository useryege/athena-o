package main

import (
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"k8s.io/klog/v2"

	athenaDexCommands "github.com/useryege/athena/cmd/athena-dex/commands"
	athenaNotificationCommands "github.com/useryege/athena/cmd/athena-notification/commands"

	athenaProjectControllerCommands "github.com/useryege/athena/cmd/athena-project-controller/commands"
	athenaServerCommands "github.com/useryege/athena/cmd/athena-server/commands"
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
	case "athena-dex":
		command = athenaDexCommands.NewCommand()
	case "athena-notification":
		command = athenaNotificationCommands.NewCommand()
	case "athena-project-controller":
		command = athenaProjectControllerCommands.NewCommand()
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
