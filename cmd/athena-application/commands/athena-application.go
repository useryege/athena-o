package commands

import (
	"context"
	stderrors "errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/redis/go-redis/v9"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health/grpc_health_v1"

	ethcommon "github.com/ethereum/go-ethereum/common"
	cmdutil "github.com/useryege/athena/cmd/util"
	"github.com/useryege/athena/common"
	"github.com/useryege/athena/internal/application"
	"github.com/useryege/athena/internal/application/apiclient"
	"github.com/useryege/athena/internal/application/metrics"
	"github.com/useryege/athena/internal/application/redisport"
	appstore "github.com/useryege/athena/internal/application/store"
	cacheutil "github.com/useryege/athena/util/cache"
	"github.com/useryege/athena/util/cli"
	"github.com/useryege/athena/util/deepseek"
	"github.com/useryege/athena/util/env"
	"github.com/useryege/athena/util/errors"
	"github.com/useryege/athena/util/healthz"
	utilio "github.com/useryege/athena/util/io"

	athenacontract "github.com/useryege/athena/pkg/abi/ATHENA"
)

const cliName = "athena-application"

func NewCommand() *cobra.Command {
	var (
		listenHost          string
		listenPort          int
		metricsHost         string
		metricsPort         int
		nodewsurl           string
		athenaContract      string
		etherscanAPIBaseURL string
		etherscanAPIKey     string
		deepseekAPIKey      string
		deepseekAPIBaseURL  string
		deepseekModel       string
		liquidityLockers    []string
		storeSrc            func(context.Context) (*appstore.SQLStore, error)
		redisClient         *redis.Client
		cacheSrc            func() (*cacheutil.Cache, error)
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
				},
			)

			cli.SetLogFormat(cmdutil.LogFormat)
			cli.SetLogLevel(cmdutil.LogLevel)

			ctx := cmd.Context()

			store, err := storeSrc(ctx)
			errors.CheckError(err)
			defer utilio.Close(store)

			_, err = cacheSrc()
			errors.CheckError(err)
			if err := requireApplicationRedis(ctx, redisClient); err != nil {
				return err
			}

			metricsServer := metrics.NewMetricsServer()
			http.Handle("/metrics", metricsServer.GetHandler())
			go func() { errors.CheckError(http.ListenAndServe(fmt.Sprintf("%s:%d", metricsHost, metricsPort), nil)) }()

			// create a new node client
			nodeClient, err := ethclient.Dial(nodewsurl)
			if err != nil {
				log.Fatalf("failed to connect to node websocket: %v", err)
			}

			athenaContractAddress, err := parseRequiredAddress("ATHENA contract address", athenaContract, "--athena-contract", "ATHENA_APPLICATION_ATHENA_CONTRACT")
			if err != nil {
				return err
			}

			athenaClient, err := athenacontract.NewATHENA(athenaContractAddress, nodeClient)
			if err != nil {
				return err
			}

			wethContractAddress, err := athenaClient.WethContract(&bind.CallOpts{Context: ctx})
			if err != nil {
				return err
			}
			usdtContractAddress, err := athenaClient.UsdtContract(&bind.CallOpts{Context: ctx})
			if err != nil {
				return err
			}

			wethDecimals, err := athenaClient.WethDecimals(&bind.CallOpts{Context: ctx})
			if err != nil {
				return err
			}
			usdtDecimals, err := athenaClient.UsdtDecimals(&bind.CallOpts{Context: ctx})
			if err != nil {
				return err
			}

			v2FactoryContractAddress, err := athenaClient.FactoryContract(&bind.CallOpts{Context: ctx})
			if err != nil {
				return err
			}

			if err := validateRequiredString("Etherscan API base URL", etherscanAPIBaseURL, "--etherscan-api-base-url", "ATHENA_APPLICATION_ETHERSCAN_API_BASE_URL"); err != nil {
				return err
			}

			if err := validateRequiredString("Etherscan API key", etherscanAPIKey, "--etherscan-api-key", "ATHENA_APPLICATION_ETHERSCAN_API_KEY"); err != nil {
				return err
			}

			liquidityLockerAddresses, err := parseLiquidityLockerAddresses(liquidityLockers)
			if err != nil {
				return err
			}

			server, err := application.NewServer(application.ApplicationServerOpts{
				NodeClient:          nodeClient,
				AthenaContract:      athenaContractAddress,
				EtherscanAPIBaseURL: etherscanAPIBaseURL,
				EtherscanAPIKey:     etherscanAPIKey,
				DeepSeekConfig: deepseek.Config{
					BaseURL: deepseekAPIBaseURL,
					APIKey:  deepseekAPIKey,
					Model:   deepseekModel,
				},
				Store:           store,
				LiquidityLocker: liquidityLockerAddresses,
				RedisClient:     redisport.NewGoRedisAdapter(redisClient),

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

			healthz.ServeHealthCheck(http.DefaultServeMux, func(r *http.Request) error {
				if val, ok := r.URL.Query()["full"]; ok && len(val) > 0 && val[0] == "true" {
					// connect to itself to make sure project controller is able to serve connection
					// used by liveness probe to auto restart project controller
					conn, err := apiclient.NewConnection(fmt.Sprintf("localhost:%d", listenPort))
					if err != nil {
						return err
					}
					defer utilio.Close(conn)
					client := grpc_health_v1.NewHealthClient(conn)
					res, err := client.Check(r.Context(), &grpc_health_v1.HealthCheckRequest{})
					if err != nil {
						return err
					}
					if res.Status != grpc_health_v1.HealthCheckResponse_SERVING {
						return fmt.Errorf("grpc health check status is '%v'", res.Status)
					}
					return nil
				}
				return nil
			})

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
	command.Flags().StringVar(&listenHost, "address", env.StringFromEnv("ATHENA_APPLICATION_LISTEN_ADDRESS", common.DefaultAddressApplication), "Listen on given address for incoming connections")
	command.Flags().IntVar(&listenPort, "port", common.DefaultPortApplication, "Listen on given port for incoming connections")
	command.Flags().StringVar(&metricsHost, "metrics-address", env.StringFromEnv("ATHENA_APPLICATION_METRICS_LISTEN_ADDRESS", common.DefaultAddressApplicationMetrics), "Listen on given address for metrics and health checks")
	command.Flags().IntVar(&metricsPort, "metrics-port", common.DefaultPortApplicationMetrics, "Start metrics server on given port")
	command.Flags().StringVar(&nodewsurl, "node-ws-url", env.StringFromEnv("ATHENA_APPLICATION_NODE_WS_URL", "ws://localhost:8546"), "Node WebSocket address")
	command.Flags().StringVar(&athenaContract, "athena-contract", env.StringFromEnv("ATHENA_APPLICATION_ATHENA_CONTRACT", ""), "ATHENA aggregation contract address")
	command.Flags().StringVar(&etherscanAPIBaseURL, "etherscan-api-base-url", env.StringFromEnv("ATHENA_APPLICATION_ETHERSCAN_API_BASE_URL", "https://api.etherscan.io/v2/api"), "Etherscan API base URL")
	command.Flags().StringVar(&etherscanAPIKey, "etherscan-api-key", env.StringFromEnv("ATHENA_APPLICATION_ETHERSCAN_API_KEY", ""), "Etherscan API key")
	command.Flags().StringVar(&deepseekAPIKey, "deepseek-api-key", env.StringFromEnv("ATHENA_APPLICATION_DEEPSEEK_API_KEY", ""), "DeepSeek API key")
	command.Flags().StringVar(&deepseekAPIBaseURL, "deepseek-api-base-url", env.StringFromEnv("ATHENA_APPLICATION_DEEPSEEK_BASE_URL", deepseek.DefaultBaseURL), "DeepSeek API base URL")
	command.Flags().StringVar(&deepseekModel, "deepseek-model", env.StringFromEnv("ATHENA_APPLICATION_DEEPSEEK_MODEL", deepseek.DefaultModel), "DeepSeek model for contract source quality analysis")
	command.Flags().StringSliceVar(&liquidityLockers, "liquidity-locker-addresses", env.StringsFromEnv("ATHENA_APPLICATION_LIQUIDITY_LOCKER_ADDRESSES", nil, ","), "Comma-separated liquidity locker wallet addresses")
	storeSrc = appstore.NewSQLStoreSource()
	cacheSrc = cacheutil.AddCacheFlagsToCmd(command, cacheutil.Options{
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

func validateRequiredString(name string, value string, flag string, envVar string) error {
	if strings.TrimSpace(value) == "" {
		return fmt.Errorf("%s is required.Set it by %s flag or %s environment variable", name, flag, envVar)
	}
	return nil
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

func requireApplicationRedis(ctx context.Context, client *redis.Client) error {
	if client == nil {
		return fmt.Errorf("redis client is required for athena-application")
	}
	pingCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	if err := client.Ping(pingCtx).Err(); err != nil {
		return fmt.Errorf("failed to ping redis: %w", err)
	}
	version, err := getRedisVersion(pingCtx, client)
	if err != nil {
		return fmt.Errorf("connected to redis, but failed to get redis version: %w", err)
	}
	log.Infof("connected to redis, version=%s", version)
	return nil
}

func getRedisVersion(ctx context.Context, client *redis.Client) (string, error) {
	info, err := client.Info(ctx, "server").Result()
	if err != nil {
		return "", err
	}
	for _, line := range strings.Split(info, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "redis_version:") {
			return strings.TrimSpace(strings.TrimPrefix(line, "redis_version:")), nil
		}
	}
	return "", fmt.Errorf("redis_version not found in INFO server response")
}
