package main

import (
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"k8s.io/klog/v2"

	athenaEtherscanGatewayCommands "github.com/useryege/athena/cmd/athena-etherscan-gateway/commands"
	athenaEtherscanManagerCommands "github.com/useryege/athena/cmd/athena-etherscan-manager/commands"
	athenaManagedOOCommands "github.com/useryege/athena/cmd/athena-managed-oo/commands"
	athenaMarketRadarCommands "github.com/useryege/athena/cmd/athena-market-radar/commands"
	athenaMigrateCommands "github.com/useryege/athena/cmd/athena-migrate/commands"
	athenaNotificationCommands "github.com/useryege/athena/cmd/athena-notification/commands"
	athenaProfitSharingCommands "github.com/useryege/athena/cmd/athena-profit-sharing/commands"
	athenaServerCommands "github.com/useryege/athena/cmd/athena-server/commands"
	athenaSportsHistoryCommands "github.com/useryege/athena/cmd/athena-sports-history/commands"
	athenaSportsLiveCommands "github.com/useryege/athena/cmd/athena-sports-live/commands"
	athenaTokenAPICommands "github.com/useryege/athena/cmd/athena-token-api/commands"
	athenaTokenChainProcessorCommands "github.com/useryege/athena/cmd/athena-token-chain-processor/commands"
	athenaTokenCollectorCommands "github.com/useryege/athena/cmd/athena-token-collector/commands"
	athenaTokenReportBuilderCommands "github.com/useryege/athena/cmd/athena-token-report-builder/commands"
	athenaTokenSchedulerCommands "github.com/useryege/athena/cmd/athena-token-scheduler/commands"
	athenaTokenSelectorCommands "github.com/useryege/athena/cmd/athena-token-selector/commands"
	athenaTokenSwapProcessorCommands "github.com/useryege/athena/cmd/athena-token-swap-processor/commands"
	athenaWalletCommands "github.com/useryege/athena/cmd/athena-wallet/commands"
	athenaWormMarketsCommands "github.com/useryege/athena/cmd/athena-worm-markets/commands"
	athenaWormTradingCommands "github.com/useryege/athena/cmd/athena-worm-trading/commands"
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
	case "athena-etherscan-gateway":
		command = athenaEtherscanGatewayCommands.NewCommand()
	case "athena-etherscan-manager":
		command = athenaEtherscanManagerCommands.NewCommand()
	case "athena-notification":
		command = athenaNotificationCommands.NewCommand()
	case "athena-profit-sharing":
		command = athenaProfitSharingCommands.NewCommand()
	case "athena-market-radar":
		command = athenaMarketRadarCommands.NewCommand()
	case "athena-sports-live":
		command = athenaSportsLiveCommands.NewCommand()
	case "athena-sports-history":
		command = athenaSportsHistoryCommands.NewCommand()
	case "athena-managed-oo":
		command = athenaManagedOOCommands.NewCommand()
	case "athena-token-api":
		command = athenaTokenAPICommands.NewCommand()
	case "athena-token-chain-processor":
		command = athenaTokenChainProcessorCommands.NewCommand()
	case "athena-token-swap-processor":
		command = athenaTokenSwapProcessorCommands.NewCommand()
	case "athena-token-scheduler":
		command = athenaTokenSchedulerCommands.NewCommand()
	case "athena-token-collector":
		command = athenaTokenCollectorCommands.NewCommand()
	case "athena-token-report-builder":
		command = athenaTokenReportBuilderCommands.NewCommand()
	case "athena-token-selector":
		command = athenaTokenSelectorCommands.NewCommand()
	case "athena-wallet":
		command = athenaWalletCommands.NewCommand()
	case "athena-worm-markets":
		command = athenaWormMarketsCommands.NewCommand()
	case "athena-worm-trading":
		command = athenaWormTradingCommands.NewCommand()
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
