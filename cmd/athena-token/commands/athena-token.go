package commands

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	cmdutil "github.com/useryege/athena/cmd/util"
	"github.com/useryege/athena/common"
	"github.com/useryege/athena/internal/token"
	tokenstore "github.com/useryege/athena/internal/token/store"
	"github.com/useryege/athena/util/ave"
	"github.com/useryege/athena/util/cli"
	"github.com/useryege/athena/util/env"
	"github.com/useryege/athena/util/templates"
)

const cliName = "athena-token"

func NewCommand() *cobra.Command {
	var (
		mode                                                                                                          string
		ethNodeWSURLs, bscNodeWSURLs                                                                                  []string
		ethAthenaContract, bscAthenaContract                                                                          string
		ethEnabled, bscEnabled, nodeWSUseProxy                                                                        bool
		aveAPIKey, aveAPIBaseURL, ethereumAPIAddress                                                                  string
		chainStateInterval, walletAssetInterval, simulationInterval, aveInterval, contractSourceInterval, researchTTL time.Duration
	)
	command := &cobra.Command{Use: cliName, Short: "Run Athena Token discovery or research", Long: "Runs one consolidated Token Intelligence worker process.", DisableAutoGenTag: true, RunE: func(cmd *cobra.Command, _ []string) error {
		cli.SetLogFormat(cmdutil.LogFormat)
		cli.SetLogLevel(cmdutil.LogLevel)
		chains := token.ChainRuntimeOptions{EthNodeWSURLs: ethNodeWSURLs, BSCNodeWSURLs: bscNodeWSURLs, EthAthenaContract: ethAthenaContract, BSCAthenaContract: bscAthenaContract, EthEnabled: ethEnabled, BSCEnabled: bscEnabled, NodeWSUseProxy: nodeWSUseProxy}
		runtime, err := token.NewRuntime(token.RuntimeOptions{
			Mode: mode, StoreSrc: tokenstore.NewSQLStoreSource(),
			Discovery: token.DiscoveryOptions{Chains: chains},
			Research:  token.ResearchOptions{Chains: chains, AveAPIKey: aveAPIKey, AveAPIBaseURL: aveAPIBaseURL, EthereumAPIAddress: ethereumAPIAddress, ChainStateInterval: chainStateInterval, WalletAssetInterval: walletAssetInterval, SimulationInterval: simulationInterval, AveInterval: aveInterval, ContractSourceInterval: contractSourceInterval, ResearchTTL: researchTTL},
		})
		if err != nil {
			return err
		}
		common.GetVersion().LogStartupInfo("Athena Token", map[string]any{"mode": token.NormalizeMode(mode)})
		if err = runtime.Start(cmd.Context()); err != nil {
			return err
		}
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
		defer signal.Stop(sigCh)
		select {
		case <-cmd.Context().Done():
		case sig := <-sigCh:
			log.WithField("signal", sig).Info("stopping token runtime")
		}
		return runtime.Stop()
	}, Example: templates.Examples(`
			# Scan chains and validate project candidates
			$ ATHENA_TOKEN_MODE=discovery athena-token

			# Schedule collection, build reports, and evaluate selections
			$ ATHENA_TOKEN_MODE=research athena-token
		`)}
	command.Flags().StringVar(&cmdutil.LogFormat, "logformat", env.StringFromEnv(common.EnvLogFormat, "json"), "Set log format: json|text")
	command.Flags().StringVar(&cmdutil.LogLevel, "loglevel", env.StringFromEnv(common.EnvLogLevel, "info"), "Set log level")
	command.Flags().StringVar(&mode, "mode", env.StringFromEnv("ATHENA_TOKEN_MODE", token.ModeDiscovery), "Run mode: discovery|research")
	command.Flags().StringSliceVar(&ethNodeWSURLs, "eth-node-ws-urls", env.StringsFromEnv("ATHENA_TOKEN_ETH_NODE_WS_URLS", nil, ","), "Ethereum node WebSocket addresses")
	command.Flags().StringSliceVar(&bscNodeWSURLs, "bsc-node-ws-urls", env.StringsFromEnv("ATHENA_TOKEN_BSC_NODE_WS_URLS", nil, ","), "BSC node WebSocket addresses")
	command.Flags().StringVar(&ethAthenaContract, "eth-athena-contract", env.StringFromEnv("ATHENA_TOKEN_ETH_ATHENA_CONTRACT", ""), "Ethereum ATHENA reader contract")
	command.Flags().StringVar(&bscAthenaContract, "bsc-athena-contract", env.StringFromEnv("ATHENA_TOKEN_BSC_ATHENA_CONTRACT", ""), "BSC ATHENA reader contract")
	command.Flags().BoolVar(&ethEnabled, "eth-enabled", env.ParseBoolFromEnv("ATHENA_TOKEN_ETH_ENABLED", true), "Enable Ethereum")
	command.Flags().BoolVar(&bscEnabled, "bsc-enabled", env.ParseBoolFromEnv("ATHENA_TOKEN_BSC_ENABLED", true), "Enable BSC")
	command.Flags().BoolVar(&nodeWSUseProxy, "node-ws-use-proxy", env.ParseBoolFromEnv("ATHENA_TOKEN_NODE_WS_USE_PROXY", false), "Use proxy for node WebSocket connections")
	command.Flags().StringVar(&aveAPIKey, "ave-api-key", env.StringFromEnv("ATHENA_TOKEN_AVE_API_KEY", ""), "Ave API key")
	command.Flags().StringVar(&aveAPIBaseURL, "ave-api-base-url", env.StringFromEnv("ATHENA_TOKEN_AVE_API_BASE_URL", ave.DefaultBaseURL), "Ave API base URL")
	command.Flags().StringVar(&ethereumAPIAddress, "ethereum-api-server-address", env.StringFromEnv("ATHENA_TOKEN_ETHEREUM_API_SERVER_ADDRESS", fmt.Sprintf("localhost:%d", common.DefaultPortEthereumAPI)), "Ethereum API gRPC address")
	command.Flags().DurationVar(&chainStateInterval, "chain-state-interval", env.ParseDurationFromEnv("ATHENA_TOKEN_CHAIN_STATE_INTERVAL", 15*time.Second, time.Second, time.Hour), "Chain state refresh interval")
	command.Flags().DurationVar(&walletAssetInterval, "wallet-asset-interval", env.ParseDurationFromEnv("ATHENA_TOKEN_WALLET_ASSET_INTERVAL", time.Minute, time.Second, time.Hour), "Wallet asset refresh interval")
	command.Flags().DurationVar(&simulationInterval, "simulation-interval", env.ParseDurationFromEnv("ATHENA_TOKEN_SIMULATION_INTERVAL", time.Minute, time.Second, time.Hour), "Simulation refresh interval")
	command.Flags().DurationVar(&aveInterval, "ave-interval", env.ParseDurationFromEnv("ATHENA_TOKEN_AVE_INTERVAL", 5*time.Minute, time.Second, 24*time.Hour), "Ave refresh interval")
	command.Flags().DurationVar(&contractSourceInterval, "contract-source-interval", env.ParseDurationFromEnv("ATHENA_TOKEN_CONTRACT_SOURCE_INTERVAL", 10*time.Minute, time.Second, 24*time.Hour), "Contract source retry interval")
	command.Flags().DurationVar(&researchTTL, "research-ttl", env.ParseDurationFromEnv("ATHENA_TOKEN_RESEARCH_TTL", 168*time.Hour, time.Hour, 30*24*time.Hour), "Research expiration duration")
	command.AddCommand(cli.NewVersionCmd(cliName))
	return command
}
