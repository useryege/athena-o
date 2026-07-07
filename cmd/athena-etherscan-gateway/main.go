package main

import (
	"os"

	athenaEtherscanGatewayCommands "github.com/useryege/athena/cmd/athena-etherscan-gateway/commands"
)

func main() {
	if err := athenaEtherscanGatewayCommands.NewCommand().Execute(); err != nil {
		os.Exit(1)
	}
}
