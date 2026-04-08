package main

import (
	"os"
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

	switch binaryName {
	case "athena-server":
		command = athenaServerCommands.NewCommand()
	case "athena-application-controller":
		command = athenaApplicationControllerCommands.NewCommand()
	case "athena-dex":
		command = athenaDexCommands.NewCommand()
	case "athena-notification":
		command = athenaNotificationCommands.NewCommand()
	case "athena":
		command = athenaCommands.NewCommand()
	case "athena-k8s-auth":
		command = athenaK8sAuthCommands.NewCommand()
	default:
		os.Exit(1)
	}

	err := command.Execute()
	if err != nil {
		os.Exit(1)
	}
}
