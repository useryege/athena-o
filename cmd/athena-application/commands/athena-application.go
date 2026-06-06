package commands

import (
	"context"
	stderrors "errors"
	"fmt"
	"net"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/redis/go-redis/v9"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"

	ethcommon "github.com/ethereum/go-ethereum/common"
	cmdutil "github.com/useryege/athena/cmd/util"
	"github.com/useryege/athena/common"
	"github.com/useryege/athena/internal/application"
	appoutbox "github.com/useryege/athena/internal/application/outbox"
	appstore "github.com/useryege/athena/internal/application/store"
	appworkflows "github.com/useryege/athena/internal/application/workflows"
	"github.com/useryege/athena/util/ave"
	cacheutil "github.com/useryege/athena/util/cache"
	"github.com/useryege/athena/util/cli"
	"github.com/useryege/athena/util/env"
	"github.com/useryege/athena/util/errors"
	"github.com/useryege/athena/util/ethereumapi"
	"github.com/useryege/athena/util/ethws"
	utilio "github.com/useryege/athena/util/io"
)

const cliName = "athena-application"

func NewCommand() *cobra.Command {
	var (
		applicationMode     string
		listenHost          string
		listenPort          int
		chainID             int64
		nodewsurl           string
		nodeWSUseProxy      bool
		athenaContract      string
		aveAPIKey           string
		aveAPIBaseURL       string
		etherscanAPIBaseURL string
		etherscanAPIKey     string
		temporalAddress     string
		temporalNamespace   string
		temporalIdentity    string
		confirmationDepth   uint64
		startBlock          uint64
		ingestPollInterval  time.Duration
		outboxPollInterval  time.Duration
		liquidityLockers    []string
		storeSrc            func(context.Context) (*appstore.SQLStore, error)
		redisClient         *redis.Client
	)

	command := &cobra.Command{
		Use:               cliName,
		Short:             "Run the Athena Application",
		Long:              "The Application manages application-level workloads and state transitions. This command runs the service in the foreground.",
		DisableAutoGenTag: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			vers := common.GetVersion()
			vers.LogStartupInfo(
				"Athena Application",
				map[string]any{
					"port": listenPort,
					"mode": normalizeApplicationMode(applicationMode),
				},
			)

			cli.SetLogFormat(cmdutil.LogFormat)
			cli.SetLogLevel(cmdutil.LogLevel)

			ctx := cmd.Context()

			store, err := storeSrc(ctx)
			errors.CheckError(err)
			defer utilio.Close(store)

			if normalizeApplicationMode(applicationMode) != applicationModeAPI {
				return runNonAPIMode(ctx, runtimeOptions{
					Mode:                applicationMode,
					ChainID:             chainID,
					NodeWSURL:           nodewsurl,
					NodeWSUseProxy:      nodeWSUseProxy,
					AthenaContract:      athenaContract,
					LiquidityLockers:    liquidityLockers,
					AveAPIKey:           aveAPIKey,
					AveAPIBaseURL:       aveAPIBaseURL,
					EtherscanAPIBaseURL: etherscanAPIBaseURL,
					EtherscanAPIKey:     etherscanAPIKey,
					TemporalAddress:     temporalAddress,
					TemporalNamespace:   temporalNamespace,
					TemporalIdentity:    temporalIdentity,
					ConfirmationDepth:   confirmationDepth,
					StartBlock:          startBlock,
					IngestPollInterval:  ingestPollInterval,
					OutboxPollInterval:  outboxPollInterval,
					Store:               store,
					RedisClient:         redisClient,
				})
			}

			// create a new node client
			nodeClient, err := ethws.DialContext(ctx, nodewsurl, nodeWSUseProxy)
			if err != nil {
				log.Fatalf("failed to connect to node websocket: %v", err)
			}
			defer nodeClient.Close()

			// chain id
			nodeChainID, err := nodeClient.ChainID(ctx)
			if err != nil {
				return fmt.Errorf("failed to fetch node chain id: %w", err)
			}
			log.Infof("node chain id: %d", nodeChainID.Int64())

			var apiFetcher ethereumapi.EthereumAPI
			if etherscanAPIBaseURL != "" && etherscanAPIKey != "" {
				apiFetcher = ethereumapi.NewEthereumAPI(etherscanAPIBaseURL, etherscanAPIKey, nodeChainID.Int64())
			}

			athenaContractAddress, v2FactoryContractAddress, wethContractAddress, usdtContractAddress, wethDecimals, usdtDecimals, err := loadAthenaContractOptions(ctx, nodeClient, athenaContract)
			if err != nil {
				return err
			}

			liquidityLockerAddresses, err := parseLiquidityLockerAddresses(liquidityLockers)
			if err != nil {
				return err
			}

			server, err := application.NewServer(application.ApplicationServerOpts{
				NodeClient:     nodeClient,
				AthenaContract: athenaContractAddress,
				ChainID:        nodeChainID.Int64(),
				AveConfig: ave.Config{
					BaseURL: aveAPIBaseURL,
					APIKey:  aveAPIKey,
				},
				Store:           store,
				LiquidityLocker: liquidityLockerAddresses,
				APIFetcher:      apiFetcher,

				// Fetch from Athena contract
				V2FactoryContract: v2FactoryContractAddress,
				WethContract:      wethContractAddress,
				UsdtContract:      usdtContractAddress,
				WethDecimals:      wethDecimals,
				UsdtDecimals:      usdtDecimals,
			})
			if err != nil {
				return err
			}

			applicationGrpc := server.CreateGRPC()

			lc := &net.ListenConfig{}
			listener, err := lc.Listen(ctx, "tcp", fmt.Sprintf("%s:%d", listenHost, listenPort))
			errors.CheckError(err)

			// start the background services
			if err := server.Start(); err != nil {
				return err
			}

			// Graceful shutdown code adapted from here: https://gist.github.com/embano1/e0bf49d24f1cdd07cffad93097c04f0a
			sigCh := make(chan os.Signal, 1)
			signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
			wg := sync.WaitGroup{}
			wg.Add(1)
			go func() {
				s := <-sigCh
				log.Printf("got signal %v, attempting graceful shutdown", s)
				applicationGrpc.GracefulStop()
				if err := server.Stop(); err != nil {
					log.Printf("failed to stop application server cleanly: %v", err)
				}
				wg.Done()
			}()

			log.Println("starting grpc server")
			err = applicationGrpc.Serve(listener)
			if err != nil && !stderrors.Is(err, grpc.ErrServerStopped) {
				errors.CheckError(err)
			}

			wg.Wait()
			log.Println("clean shutdown")
			return nil
		},
	}

	command.Flags().StringVar(&cmdutil.LogFormat, "logformat", env.StringFromEnv("ATHENA_APPLICATION_LOGFORMAT", "json"), "Set the logging format. One of: json|text")
	command.Flags().StringVar(&cmdutil.LogLevel, "loglevel", env.StringFromEnv("ATHENA_APPLICATION_LOGLEVEL", "info"), "Set the logging level. One of: debug|info|warn|error")
	command.Flags().StringVar(&applicationMode, "mode", env.StringFromEnv("ATHENA_APPLICATION_MODE", applicationModeAPI), "Run mode: api|chain-ingestor|outbox-worker|temporal-worker-control|temporal-worker-chain|temporal-worker-external")
	command.Flags().StringVar(&listenHost, "address", env.StringFromEnv("ATHENA_APPLICATION_LISTEN_ADDRESS", common.DefaultAddressApplication), "Listen on given address for incoming connections")
	command.Flags().IntVar(&listenPort, "port", common.DefaultPortApplication, "Listen on given port for incoming connections")
	command.Flags().Int64Var(&chainID, "chain-id", env.ParseInt64FromEnv("ATHENA_APPLICATION_CHAIN_ID", 0, 1, 9223372036854775807), "EVM chain ID for chain-scoped application modes")
	command.Flags().StringVar(&nodewsurl, "node-ws-url", env.StringFromEnv("ATHENA_APPLICATION_NODE_WS_URL", "ws://localhost:8546"), "Node WebSocket address")
	command.Flags().BoolVar(&nodeWSUseProxy, "node-ws-use-proxy", env.ParseBoolFromEnv("ATHENA_APPLICATION_NODE_WS_USE_PROXY", false), "Whether to use proxy environment variables for node WebSocket connections")
	command.Flags().StringVar(&athenaContract, "athena-contract", env.StringFromEnv("ATHENA_APPLICATION_ATHENA_CONTRACT", ""), "ATHENA aggregation contract address")
	command.Flags().StringVar(&aveAPIKey, "ave-api-key", env.StringFromEnv("ATHENA_APPLICATION_AVE_API_KEY", ""), "Ave API key for project logo fetching")
	command.Flags().StringVar(&aveAPIBaseURL, "ave-api-base-url", env.StringFromEnv("ATHENA_APPLICATION_AVE_API_BASE_URL", ave.DefaultBaseURL), "Ave API base URL")
	command.Flags().StringVar(&etherscanAPIBaseURL, "etherscan-api-base-url", env.StringFromEnv("ATHENA_APPLICATION_ETHERSCAN_API_BASE_URL", "https://api.etherscan.io/v2/api"), "Etherscan API base URL")
	command.Flags().StringVar(&etherscanAPIKey, "etherscan-api-key", env.StringFromEnv("ATHENA_APPLICATION_ETHERSCAN_API_KEY", ""), "Etherscan API key")
	command.Flags().StringVar(&temporalAddress, "temporal-address", env.StringFromEnv("ATHENA_APPLICATION_TEMPORAL_ADDRESS", appworkflows.DefaultTemporalAddress), "Temporal frontend host:port")
	command.Flags().StringVar(&temporalNamespace, "temporal-namespace", env.StringFromEnv("ATHENA_APPLICATION_TEMPORAL_NAMESPACE", appworkflows.DefaultTemporalNamespace), "Temporal namespace")
	command.Flags().StringVar(&temporalIdentity, "temporal-identity", env.StringFromEnv("ATHENA_APPLICATION_TEMPORAL_IDENTITY", ""), "Temporal worker identity")
	command.Flags().Uint64Var(&confirmationDepth, "confirmation-depth", uint64(env.ParseInt64FromEnv("ATHENA_APPLICATION_CONFIRMATION_DEPTH", 0, 0, 9223372036854775807)), "Chain ingestor confirmation depth; defaults by chain when zero")
	command.Flags().Uint64Var(&startBlock, "start-block", uint64(env.ParseInt64FromEnv("ATHENA_APPLICATION_START_BLOCK", 0, 0, 9223372036854775807)), "Chain ingestor start block when no checkpoint exists")
	command.Flags().DurationVar(&ingestPollInterval, "ingest-poll-interval", env.ParseDurationFromEnv("ATHENA_APPLICATION_INGEST_POLL_INTERVAL", 2*time.Second, time.Second, 24*time.Hour), "Chain ingestor poll interval")
	command.Flags().DurationVar(&outboxPollInterval, "outbox-poll-interval", env.ParseDurationFromEnv("ATHENA_APPLICATION_OUTBOX_POLL_INTERVAL", appoutbox.DefaultPollInterval, time.Second, 24*time.Hour), "Outbox worker poll interval")
	command.Flags().StringSliceVar(&liquidityLockers, "liquidity-locker-addresses", env.StringsFromEnv("ATHENA_APPLICATION_LIQUIDITY_LOCKER_ADDRESSES", nil, ","), "Comma-separated liquidity locker wallet addresses")
	storeSrc = appstore.NewSQLStoreSource()
	cacheutil.AddCacheFlagsToCmd(command, cacheutil.Options{
		OnClientCreated: func(client *redis.Client) {
			redisClient = client
		},
	})

	command.AddCommand(cli.NewVersionCmd(cliName))
	return command
}

func parseRequiredAddress(name string, value string, flag string, envVar string) (ethcommon.Address, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return ethcommon.Address{}, fmt.Errorf("%s is required.Set it by %s flag or %s environment variable", name, flag, envVar)
	}
	if !ethcommon.IsHexAddress(value) {
		return ethcommon.Address{}, fmt.Errorf("invalid %s %q", name, value)
	}

	address := ethcommon.HexToAddress(value)
	if address == (ethcommon.Address{}) {
		return ethcommon.Address{}, fmt.Errorf("%s cannot be zero address", name)
	}
	return address, nil
}

func parseLiquidityLockerAddresses(values []string) ([]ethcommon.Address, error) {
	if len(values) == 0 {
		return nil, nil
	}

	addresses := make([]ethcommon.Address, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			return nil, fmt.Errorf("liquidity locker address cannot be empty")
		}
		if !ethcommon.IsHexAddress(value) {
			return nil, fmt.Errorf("invalid liquidity locker address %q", value)
		}
		addresses = append(addresses, ethcommon.HexToAddress(value))
	}
	return addresses, nil
}
