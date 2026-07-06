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

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"

	cmdutil "github.com/useryege/athena/cmd/util"
	"github.com/useryege/athena/common"
	"github.com/useryege/athena/internal/ethereumapi"
	ethereumapistore "github.com/useryege/athena/internal/ethereumapi/store"
	"github.com/useryege/athena/util/cli"
	"github.com/useryege/athena/util/env"
	"github.com/useryege/athena/util/errors"
	utilethereumapi "github.com/useryege/athena/util/ethereumapi"
	utilio "github.com/useryege/athena/util/io"
	"github.com/useryege/athena/util/ratelimit"
	"github.com/useryege/athena/util/templates"
)

const cliName = "athena-ethereum-api"

const (
	defaultEtherscanRateLimitRequests = 3
	defaultEtherscanRateLimitPeriod   = time.Second
	defaultEtherscanRateLimitBurst    = 3
)

func NewCommand() *cobra.Command {
	var (
		listenHost          string
		listenPort          int
		etherscanAPIKey     string
		etherscanAPIBaseURL string

		storeSrc func(context.Context) (*ethereumapistore.SQLStore, error)
	)

	command := &cobra.Command{
		Use:               cliName,
		Short:             "Run the Athena Ethereum API service",
		Long:              "The Ethereum API service provides access to persisted Ethereum API data. This command runs the service in the foreground.",
		DisableAutoGenTag: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			vers := common.GetVersion()
			vers.LogStartupInfo(
				"Athena Ethereum API",
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

			if strings.TrimSpace(etherscanAPIKey) == "" {
				return fmt.Errorf("ATHENA_ETHEREUM_API_ETHERSCAN_API_KEY is required")
			}
			etherscanRateLimiter, err := ratelimit.New(ratelimit.Config{
				Requests: defaultEtherscanRateLimitRequests,
				Per:      defaultEtherscanRateLimitPeriod,
				Burst:    defaultEtherscanRateLimitBurst,
			})
			if err != nil {
				return fmt.Errorf("create etherscan rate limiter: %w", err)
			}
			etherscanClient := utilethereumapi.NewEthereumAPIWithConfig(utilethereumapi.Config{
				BaseURL:     etherscanAPIBaseURL,
				APIKey:      etherscanAPIKey,
				RateLimiter: etherscanRateLimiter,
				Timeout:     utilethereumapi.DefaultTimeout,
			})

			server, err := ethereumapi.NewServer(ethereumapi.ServerOpts{
				Store:       store,
				EthereumAPI: etherscanClient,
			})
			if err != nil {
				return err
			}
			ethereumAPIGRPC := server.CreateGRPC()

			lc := &net.ListenConfig{}
			listener, err := lc.Listen(ctx, "tcp", fmt.Sprintf("%s:%d", listenHost, listenPort))
			errors.CheckError(err)

			if err := server.Start(ctx); err != nil {
				return err
			}

			sigCh := make(chan os.Signal, 1)
			signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
			wg := sync.WaitGroup{}
			wg.Add(1)
			go func() {
				s := <-sigCh
				log.Printf("got signal %v, attempting graceful shutdown", s)
				ethereumAPIGRPC.GracefulStop()
				if err := server.Stop(); err != nil {
					log.Printf("failed to stop ethereum-api server cleanly: %v", err)
				}
				wg.Done()
			}()

			log.Println("starting ethereum-api grpc server")
			err = ethereumAPIGRPC.Serve(listener)
			if err != nil && !stderrors.Is(err, grpc.ErrServerStopped) {
				errors.CheckError(err)
			}

			wg.Wait()
			log.Println("clean shutdown")
			return nil
		},
		Example: templates.Examples(`
			# Start the Athena Ethereum API service
			$ athena-ethereum-api
		`),
	}

	command.Flags().StringVar(&cmdutil.LogFormat, "logformat", env.StringFromEnv(common.EnvLogFormat, "json"), "Set the logging format. One of: json|text")
	command.Flags().StringVar(&cmdutil.LogLevel, "loglevel", env.StringFromEnv(common.EnvLogLevel, "info"), "Set the logging level. One of: debug|info|warn|error")
	command.Flags().StringVar(&listenHost, "address", env.StringFromEnv("ATHENA_ETHEREUM_API_LISTEN_ADDRESS", common.DefaultAddressEthereumAPI), "Listen on given address for incoming connections")
	command.Flags().IntVar(&listenPort, "port", env.ParseNumFromEnv("ATHENA_ETHEREUM_API_PORT", common.DefaultPortEthereumAPI, 1, 65535), "Listen on given port for incoming connections")
	command.Flags().StringVar(&etherscanAPIKey, "etherscan-api-key", env.StringFromEnv("ATHENA_ETHEREUM_API_ETHERSCAN_API_KEY", ""), "Etherscan API key")
	command.Flags().StringVar(&etherscanAPIBaseURL, "etherscan-api-base-url", env.StringFromEnv("ATHENA_ETHEREUM_API_ETHERSCAN_API_BASE_URL", utilethereumapi.DefaultBaseURL), "Etherscan API base URL")

	storeSrc = ethereumapistore.NewSQLStoreSource()

	command.AddCommand(cli.NewVersionCmd(cliName))
	return command
}
