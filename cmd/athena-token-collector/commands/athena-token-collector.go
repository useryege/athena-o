package commands

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/useryege/athena/cmd/tokenworker"
	"github.com/useryege/athena/common"
	aveadapter "github.com/useryege/athena/internal/token/adapters/ave"
	"github.com/useryege/athena/internal/token/adapters/evm"
	tokenpostgres "github.com/useryege/athena/internal/token/adapters/postgres"
	"github.com/useryege/athena/internal/token/adapters/sourcecode"
	"github.com/useryege/athena/internal/token/research"
	researchapp "github.com/useryege/athena/internal/token/research/application"
	"github.com/useryege/athena/internal/token/telemetry"
	"github.com/useryege/athena/internal/token/workerhost"
	"github.com/useryege/athena/util/ave"
	"github.com/useryege/athena/util/cli"
	"github.com/useryege/athena/util/env"
)

const cliName = "athena-token-collector"

var collectorHealthAddresses = map[research.DataCollectionType]string{
	research.DataCollectionTypeAve: "127.0.0.1:8113", research.DataCollectionTypeChainState: "127.0.0.1:8114",
	research.DataCollectionTypeWalletAssetState: "127.0.0.1:8115", research.DataCollectionTypeSimulationResult: "127.0.0.1:8116",
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
		connection, registry, host, err := flags.Open(cmd.Context(), "collector-"+string(dataType))
		if err != nil {
			return err
		}
		repository := tokenpostgres.NewCollectionRepository(connection)
		var processor researchapp.TaskProcessor
		switch dataType {
		case research.DataCollectionTypeAve:
			provider, err := aveadapter.New(aveadapter.Config{APIKey: aveAPIKey, BaseURL: aveAPIBaseURL})
			if err != nil {
				return err
			}
			processor = researchapp.AveProcessor{Provider: provider}
		case research.DataCollectionTypeContractCodeSource:
			provider, err := sourcecode.New(ethereumAPIAddress)
			if err != nil {
				return err
			}
			host.AddClose(provider.Close)
			processor = researchapp.ContractSourceProcessor{Codes: repository, Provider: provider}
		case research.DataCollectionTypeChainState, research.DataCollectionTypeWalletAssetState, research.DataCollectionTypeSimulationResult:
			clients := evm.NewChainClientRegistry(registry)
			host.AddClose(clients.Close)
			reader := evm.NewProjectStateReader(registry, clients)
			switch dataType {
			case research.DataCollectionTypeChainState:
				processor = researchapp.ChainStateProcessor{Reader: reader}
			case research.DataCollectionTypeWalletAssetState:
				processor = researchapp.WalletAssetStateProcessor{Reader: reader}
			case research.DataCollectionTypeSimulationResult:
				processor = researchapp.SimulationResultProcessor{Reader: reader}
			}
		}
		chainIDs := make([]int64, 0, len(registry.EnabledChains()))
		for _, chain := range registry.EnabledChains() {
			chainIDs = append(chainIDs, chain.ID)
		}
		application := researchapp.NewCollector(repository, processor, researchapp.CollectorOptions{ChainIDs: chainIDs})
		job := workerhost.PeriodicJob{Name: "data-collector-" + string(dataType), Interval: time.Second, Scope: telemetry.Scope{Component: "data_collector", DataType: string(dataType)}, RunOnce: func(ctx context.Context) (workerhost.JobResult, error) {
			count, err := application.RunOnce(ctx)
			return workerhost.JobResult{Processed: count}, err
		}}
		host.SetWorker(workerhost.NewPeriodicWorker([]workerhost.PeriodicJob{job}, host.Reporter()))
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
