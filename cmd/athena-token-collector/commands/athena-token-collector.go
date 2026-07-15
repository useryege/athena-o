package commands

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/useryege/athena/cmd/tokenworker"
	"github.com/useryege/athena/common"
	aveadapter "github.com/useryege/athena/internal/token/adapters/ave"
	"github.com/useryege/athena/internal/token/adapters/evm"
	"github.com/useryege/athena/internal/token/adapters/sourcecode"
	"github.com/useryege/athena/internal/token/adapters/workers/collector"
	"github.com/useryege/athena/internal/token/research"
	"github.com/useryege/athena/util/ave"
	"github.com/useryege/athena/util/cli"
	"github.com/useryege/athena/util/env"
)

const cliName = "athena-token-collector"

var collectorHealthAddresses = map[research.DataCollectionType]string{
	research.DataCollectionTypeAve:                "127.0.0.1:8113",
	research.DataCollectionTypeChainState:         "127.0.0.1:8114",
	research.DataCollectionTypeWalletAssetState:   "127.0.0.1:8115",
	research.DataCollectionTypeSimulationResult:   "127.0.0.1:8116",
	research.DataCollectionTypeContractCodeSource: "127.0.0.1:8117",
}

func NewCommand() *cobra.Command {
	var flags tokenworker.CommonFlags
	var dataTypeValue, aveAPIKey, aveAPIBaseURL, ethereumAPIAddress string
	command := &cobra.Command{Use: cliName, Short: "Collect one token research data type", DisableAutoGenTag: true, RunE: func(cmd *cobra.Command, _ []string) error {
		dataType, ok := research.ParseDataCollectionType(strings.TrimSpace(dataTypeValue))
		if !ok {
			return fmt.Errorf("unsupported data type %q", dataTypeValue)
		}
		if flags.HealthListenAddress == "" {
			flags.HealthListenAddress = collectorHealthAddresses[dataType]
		}
		var marketData research.MarketDataProvider
		var sourceCodeProvider research.SourceCodeProvider
		if dataType == research.DataCollectionTypeAve {
			provider, err := aveadapter.New(aveadapter.Config{APIKey: aveAPIKey, BaseURL: aveAPIBaseURL})
			if err != nil {
				return err
			}
			marketData = provider
		}
		if dataType == research.DataCollectionTypeContractCodeSource {
			provider, err := sourcecode.New(ethereumAPIAddress)
			if err != nil {
				return err
			}
			defer func() { _ = provider.Close() }()
			sourceCodeProvider = provider
		}
		database, registry, host, err := flags.Open(cmd.Context(), "collector-"+string(dataType))
		if err != nil {
			return err
		}
		var clients *evm.ChainClientRegistry
		if dataType == research.DataCollectionTypeChainState || dataType == research.DataCollectionTypeWalletAssetState || dataType == research.DataCollectionTypeSimulationResult {
			clients = evm.NewChainClientRegistry(registry)
			host.AddClose(clients.Close)
		}
		if provider, ok := sourceCodeProvider.(*sourcecode.Provider); ok {
			host.AddClose(provider.Close)
		}
		host.SetWorker(collector.NewWorker(collector.Options{Store: database.Research(), Chains: registry.Chains(), Clients: clients, DataType: dataType, MarketData: marketData, SourceCode: sourceCodeProvider, Telemetry: host.Reporter()}))
		common.GetVersion().LogStartupInfo("Athena Token Collector", map[string]any{"data_type": dataType})
		return host.Run(cmd.Context())
	}}
	flags.Bind(command, "")
	command.Flags().StringVar(&dataTypeValue, "data-type", env.StringFromEnv("ATHENA_TOKEN_DATA_TYPE", ""), "Collection type: ave|chain_state|wallet_asset_state|simulation_result|contract_code_source")
	command.Flags().StringVar(&aveAPIKey, "ave-api-key", env.StringFromEnv("ATHENA_TOKEN_AVE_API_KEY", ""), "Ave API key")
	command.Flags().StringVar(&aveAPIBaseURL, "ave-api-base-url", env.StringFromEnv("ATHENA_TOKEN_AVE_API_BASE_URL", ave.DefaultBaseURL), "Ave API base URL")
	command.Flags().StringVar(&ethereumAPIAddress, "ethereum-api-server-address", env.StringFromEnv("ATHENA_TOKEN_ETHEREUM_API_SERVER_ADDRESS", fmt.Sprintf("localhost:%d", common.DefaultPortEthereumAPI)), "Ethereum API gRPC address")
	_ = command.MarkFlagRequired("data-type")
	command.AddCommand(cli.NewVersionCmd(cliName))
	return command
}
