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
	"github.com/useryege/athena/internal/token/adapters/walletfunding"
	"github.com/useryege/athena/internal/token/adapters/walletswaps"
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
	research.DataCollectionTypeContractCodeSource:           "127.0.0.1:8117",
	research.DataCollectionTypeWalletFundingSourceHistory:   "127.0.0.1:8120",
	research.DataCollectionTypeWalletSwapTransactionHistory: "127.0.0.1:8121",
}

func NewCommand() *cobra.Command {
	var flags tokenworker.CommonFlags
	var dataTypeValue, aveAPIKey, aveAPIBaseURL, ethereumAPIAddress, bscInboundAddress, bscSwapAddress string
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
		var walletFundingProvider *walletfunding.Provider
		var walletSwapProvider *walletswaps.Provider
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
		case research.DataCollectionTypeWalletFundingSourceHistory:
			provider, err := walletfunding.New(bscInboundAddress)
			if err != nil {
				return err
			}
			host.AddClose(provider.Close)
			walletFundingProvider = provider
		case research.DataCollectionTypeWalletSwapTransactionHistory:
			provider, err := walletswaps.New(bscSwapAddress)
			if err != nil {
				return err
			}
			host.AddClose(provider.Close)
			walletSwapProvider = provider
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
			if (dataType == research.DataCollectionTypeWalletFundingSourceHistory || dataType == research.DataCollectionTypeWalletSwapTransactionHistory) && chain.ID != 56 {
				continue
			}
			chainIDs = append(chainIDs, chain.ID)
		}
		var application interface {
			RunOnce(context.Context) (int, error)
		}
		switch dataType {
		case research.DataCollectionTypeWalletFundingSourceHistory:
			application = researchapp.NewWalletFundingSourceHistoryCollector(repository, walletFundingProvider, researchapp.CollectorOptions{ChainIDs: chainIDs})
		case research.DataCollectionTypeWalletSwapTransactionHistory:
			application = researchapp.NewWalletSwapTransactionHistoryCollector(repository, walletSwapProvider, researchapp.CollectorOptions{ChainIDs: chainIDs})
		default:
			application = researchapp.NewCollector(repository, processor, researchapp.CollectorOptions{ChainIDs: chainIDs})
		}
		job := workerhost.PeriodicJob{Name: "data-collector-" + string(dataType), Interval: time.Second, Scope: telemetry.Scope{Component: "data_collector", DataType: string(dataType)}, RunOnce: func(ctx context.Context) (workerhost.JobResult, error) {
			count, err := application.RunOnce(ctx)
			return workerhost.JobResult{Processed: count}, err
		}}
		host.SetWorker(workerhost.NewPeriodicWorker([]workerhost.PeriodicJob{job}, host.Reporter()))
		common.GetVersion().LogStartupInfo("Athena Token Collector", map[string]any{"data_type": dataType})
		return host.Run(cmd.Context())
	}}
	flags.Bind(command, "")
	command.Flags().StringVar(&dataTypeValue, "data-type", env.StringFromEnv("ATHENA_TOKEN_DATA_TYPE", ""), "Collection type: ave|chain_state|wallet_asset_state|simulation_result|contract_code_source|wallet_funding_source_history|wallet_swap_transaction_history")
	command.Flags().StringVar(&aveAPIKey, "ave-api-key", env.StringFromEnv("ATHENA_TOKEN_AVE_API_KEY", ""), "Ave API key")
	command.Flags().StringVar(&aveAPIBaseURL, "ave-api-base-url", env.StringFromEnv("ATHENA_TOKEN_AVE_API_BASE_URL", ave.DefaultBaseURL), "Ave API base URL")
	command.Flags().StringVar(&ethereumAPIAddress, "ethereum-api-server-address", env.StringFromEnv("ATHENA_TOKEN_ETHEREUM_API_SERVER_ADDRESS", fmt.Sprintf("localhost:%d", common.DefaultPortEthereumAPI)), "Ethereum API gRPC address")
	command.Flags().StringVar(&bscInboundAddress, "bsc-inbound-server-address", env.StringFromEnv("ATHENA_TOKEN_BSC_INBOUND_SERVER_ADDRESS", "localhost:8130"), "BSC inbound transaction gRPC address")
	command.Flags().StringVar(&bscSwapAddress, "bsc-swap-server-address", env.StringFromEnv("ATHENA_TOKEN_BSC_SWAP_SERVER_ADDRESS", "localhost:8130"), "BSC swap transaction gRPC address")
	_ = command.MarkFlagRequired("data-type")
	command.AddCommand(cli.NewVersionCmd(cliName))
	return command
}
