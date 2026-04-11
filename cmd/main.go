package main

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/spf13/cobra"
	"k8s.io/klog/v2"

	athenaApplicationControllerCommands "github.com/useryege/athena/cmd/athena-application-controller/commands"
	athenaDexCommands "github.com/useryege/athena/cmd/athena-dex/commands"
	athenaK8sAuthCommands "github.com/useryege/athena/cmd/athena-k8s-auth/commands"
	athenaNotificationCommands "github.com/useryege/athena/cmd/athena-notification/commands"
	athenaServerCommands "github.com/useryege/athena/cmd/athena-server/commands"
	athenaCommands "github.com/useryege/athena/cmd/athena/commands"
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

	isAthenaCLI := false

	switch binaryName {
	case "athena-server":
		command = athenaServerCommands.NewCommand()
	case "athena-application-controller":
		command = athenaApplicationControllerCommands.NewCommand()
	case "athena-dex":
		command = athenaDexCommands.NewCommand()
	case "athena-notification":
		command = athenaNotificationCommands.NewCommand()
	case "athena", "athena-linux-amd64", "athena-darwin-amd64", "athena-windows-amd64.exe":
		command = athenaCommands.NewCommand()
		isAthenaCLI = true
	case "athena-k8s-auth":
		command = athenaK8sAuthCommands.NewCommand()
		isAthenaCLI = true
	default:
		command = athenaCommands.NewCommand()
		isAthenaCLI = true
	}

	if isAthenaCLI {
		// silence errors and usages since we'll be printing them manually.
		// This is because if we execute a plugin, the initial
		// errors and usage are always going to get printed that we don't want.
		command.SilenceErrors = true
		command.SilenceUsage = true
	}

	err := command.Execute()
	// if an error is present, try to look for various scenarios
	// such as if the error is from the execution of a normal athena command,
	// unknown command error or any other.
	if err != nil {
		pluginErr := athenaCommands.NewDefaultPluginHandler().HandleCommandExecutionError(err, isAthenaCLI, os.Args)
		if pluginErr != nil {
			var exitErr *exec.ExitError
			if errors.As(pluginErr, &exitErr) {
				// Return the actual plugin exit code
				os.Exit(exitErr.ExitCode())
			}
			// Fallback to exit code 1 if the error isn't an exec.ExitError
			os.Exit(1)
		}
	}
}
