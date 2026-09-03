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
	"github.com/useryege/athena/internal/token/adapters/normaltransactions"
	tokenpostgres "github.com/useryege/athena/internal/token/adapters/postgres"
	"github.com/useryege/athena/internal/token/adapters/sourcecode"
	"github.com/useryege/athena/internal/token/collection"
	collectionapp "github.com/useryege/athena/internal/token/collection/application"
	"github.com/useryege/athena/internal/token/telemetry"
	"github.com/useryege/athena/internal/token/workerhost"
	"github.com/useryege/athena/util/ave"
	"github.com/useryege/athena/util/cli"
	"github.com/useryege/athena/util/env"
)

const cliName = "athena-token-collector"

var collectorHealthAddresses = map[collection.DataType]string{
	collection.DataTypeAve: "127.0.0.1:8113", collection.DataTypeChainState: "127.0.0.1:8114",
	collection.DataTypeWalletAssetState: "127.0.0.1:8115", collection.DataTypeSimulationResult: "127.0.0.1:8116",
	collection.DataTypeContractCodeSource:       "127.0.0.1:8117",
	collection.DataTypeWalletNormalTransactions: "127.0.0.1:8120",
}

func NewCommand() *cobra.Command {
	var flags tokenworker.CommonFlags
	var dataTypeValue, aveAPIKey, aveAPIBaseURL, etherscanManagerAddress string
	command := &cobra.Command{Use: cliName, Short: "Collect one token project data type", DisableAutoGenTag: true, RunE: func(cmd *cobra.Command, _ []string) error {
		dataType, ok := collection.ParseDataType(strings.TrimSpace(dataTypeValue))
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
		var processor collectionapp.TaskProcessor
		switch dataType {
		case collection.DataTypeAve:
			provider, err := aveadapter.New(aveadapter.Config{APIKey: aveAPIKey, BaseURL: aveAPIBaseURL})
			if err != nil {
				return err
			}
			processor = collectionapp.AveProcessor{Provider: provider}
		case collection.DataTypeContractCodeSource:
			provider, err := sourcecode.New(etherscanManagerAddress)
			if err != nil {
				return err
			}
			host.AddClose(provider.Close)
			processor = collectionapp.ContractSourceProcessor{Codes: repository, Provider: provider}
		case collection.DataTypeWalletNormalTransactions:
			provider, err := normaltransactions.New(etherscanManagerAddress)
			if err != nil {
				return err
			}
			host.AddClose(provider.Close)
			processor = collectionapp.WalletNormalTransactionsProcessor{Provider: provider}
		case collection.DataTypeChainState, collection.DataTypeWalletAssetState, collection.DataTypeSimulationResult:
			clients := evm.NewChainClientRegistry(registry, flags.NodeWSProxyURL)
			host.AddClose(clients.Close)
			reader := evm.NewProjectStateReader(registry, clients)
			switch dataType {
			case collection.DataTypeChainState:
				processor = collectionapp.ChainStateProcessor{Reader: reader}
			case collection.DataTypeWalletAssetState:
				processor = collectionapp.WalletAssetStateProcessor{Reader: reader}
			case collection.DataTypeSimulationResult:
				processor = collectionapp.SimulationResultProcessor{Reader: reader}
			}
		}
		chainIDs := make([]int64, 0, len(registry.EnabledChains()))
		for _, chain := range registry.EnabledChains() {
			chainIDs = append(chainIDs, chain.ID)
		}
		application := collectionapp.NewCollector(repository, processor, collectionapp.CollectorOptions{ChainIDs: chainIDs, RetryInterval: collectionRetryInterval(dataType)})
		job := workerhost.PeriodicJob{Name: "data-collector-" + string(dataType), Interval: time.Second, Scope: telemetry.Scope{Component: "data_collector", DataType: string(dataType)}, RunOnce: func(ctx context.Context) (workerhost.JobResult, error) {
			count, err := application.RunOnce(ctx)
			return workerhost.JobResult{Processed: count}, err
		}}
		host.SetWorker(workerhost.NewPeriodicWorker([]workerhost.PeriodicJob{job}, host.Reporter()))
		common.GetVersion().LogStartupInfo("Athena Token Collector", map[string]any{"data_type": dataType})
		return host.Run(cmd.Context())
	}}
	flags.Bind(command, "")
	command.Flags().StringVar(&dataTypeValue, "data-type", env.StringFromEnv("ATHENA_TOKEN_DATA_TYPE", ""), "Collection type: ave|chain_state|wallet_asset_state|simulation_result|contract_code_source|wallet_normal_transactions")
	command.Flags().StringVar(&aveAPIKey, "ave-api-key", env.StringFromEnv("ATHENA_TOKEN_AVE_API_KEY", ""), "Ave API key")
	command.Flags().StringVar(&aveAPIBaseURL, "ave-api-base-url", env.StringFromEnv("ATHENA_TOKEN_AVE_API_BASE_URL", ave.DefaultBaseURL), "Ave API base URL")
	command.Flags().StringVar(&etherscanManagerAddress, "etherscan-manager-server-address", env.StringFromEnv("ATHENA_TOKEN_ETHERSCAN_MANAGER_SERVER_ADDRESS", fmt.Sprintf("%s:%d", common.DefaultLocalGRPCHost, common.DefaultPortEtherscanManager)), "Etherscan Manager gRPC address")
	command.AddCommand(cli.NewVersionCmd(cliName))
	return command
}

func collectionRetryInterval(dataType collection.DataType) time.Duration {
	switch dataType {
	case collection.DataTypeChainState:
		return env.ParseDurationFromEnv("ATHENA_TOKEN_CHAIN_STATE_RETRY_INTERVAL", 15*time.Second, time.Second, time.Hour)
	case collection.DataTypeWalletAssetState:
		return env.ParseDurationFromEnv("ATHENA_TOKEN_WALLET_ASSET_RETRY_INTERVAL", time.Minute, time.Second, time.Hour)
	case collection.DataTypeSimulationResult:
		return env.ParseDurationFromEnv("ATHENA_TOKEN_SIMULATION_RETRY_INTERVAL", time.Minute, time.Second, time.Hour)
	case collection.DataTypeAve:
		return env.ParseDurationFromEnv("ATHENA_TOKEN_AVE_RETRY_INTERVAL", 5*time.Minute, time.Second, 24*time.Hour)
	case collection.DataTypeContractCodeSource:
		return env.ParseDurationFromEnv("ATHENA_TOKEN_CONTRACT_SOURCE_RETRY_INTERVAL", 10*time.Minute, time.Second, 24*time.Hour)
	case collection.DataTypeWalletNormalTransactions:
		return env.ParseDurationFromEnv("ATHENA_TOKEN_WALLET_NORMAL_TRANSACTIONS_RETRY_INTERVAL", 10*time.Minute, time.Second, 24*time.Hour)
	default:
		return time.Minute
	}
}
