package commands

import (
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
	"github.com/useryege/athena/internal/etherscangateway"
	"github.com/useryege/athena/util/cli"
	"github.com/useryege/athena/util/env"
	"github.com/useryege/athena/util/errors"
	utilethereumapi "github.com/useryege/athena/util/ethereumapi"
	"github.com/useryege/athena/util/templates"
)

const cliName = "athena-etherscan-gateway"

func NewCommand() *cobra.Command {
	var (
		listenAddress string
		authToken     string
		timeout       time.Duration
	)
	defaultListenAddress := fmt.Sprintf("%s:%d", common.DefaultAddressEtherscanGateway, common.DefaultPortEtherscanGateway)

	command := &cobra.Command{
		Use:               cliName,
		Short:             "Run the Athena Etherscan Gateway service",
		Long:              "The Etherscan Gateway service exposes a narrow gRPC interface that sends Etherscan requests from the gateway host.",
		DisableAutoGenTag: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			vers := common.GetVersion()
			vers.LogStartupInfo(
				"Athena Etherscan Gateway",
				map[string]any{
					"listen_address": listenAddress,
				},
			)

			cli.SetLogFormat(cmdutil.LogFormat)
			cli.SetLogLevel(cmdutil.LogLevel)

			if strings.TrimSpace(authToken) == "" {
				return fmt.Errorf("ATHENA_ETHERSCAN_GATEWAY_AUTH_TOKEN is required")
			}

			server, err := etherscangateway.NewServer(etherscangateway.ServerOpts{
				AuthToken: authToken,
				Timeout:   timeout,
			})
			if err != nil {
				return err
			}
			gatewayGRPC := server.CreateGRPC()

			ctx := cmd.Context()
			lc := &net.ListenConfig{}
			listener, err := lc.Listen(ctx, "tcp", listenAddress)
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
				gatewayGRPC.GracefulStop()
				if err := server.Stop(); err != nil {
					log.Printf("failed to stop etherscan-gateway server cleanly: %v", err)
				}
				wg.Done()
			}()

			log.Println("starting etherscan-gateway grpc server")
			err = gatewayGRPC.Serve(listener)
			if err != nil && !stderrors.Is(err, grpc.ErrServerStopped) {
				errors.CheckError(err)
			}

			wg.Wait()
			log.Println("clean shutdown")
			return nil
		},
		Example: templates.Examples(`
			# Start the Athena Etherscan Gateway service
			$ athena-etherscan-gateway
		`),
	}

	command.Flags().StringVar(&cmdutil.LogFormat, "logformat", env.StringFromEnv(common.EnvLogFormat, "json"), "Set the logging format. One of: json|text")
	command.Flags().StringVar(&cmdutil.LogLevel, "loglevel", env.StringFromEnv(common.EnvLogLevel, "info"), "Set the logging level. One of: debug|info|warn|error")
	command.Flags().StringVar(&listenAddress, "listen-address", env.StringFromEnv("ATHENA_ETHERSCAN_GATEWAY_LISTEN_ADDRESS", defaultListenAddress), "Listen address for incoming gRPC connections")
	command.Flags().StringVar(&authToken, "auth-token", env.StringFromEnv("ATHENA_ETHERSCAN_GATEWAY_AUTH_TOKEN", ""), "Bearer token required in gRPC metadata authorization")
	command.Flags().DurationVar(&timeout, "timeout", env.ParseDurationFromEnv("ATHENA_ETHERSCAN_GATEWAY_TIMEOUT", utilethereumapi.DefaultTimeout, time.Second, time.Hour), "Timeout for outbound Etherscan requests")

	command.AddCommand(cli.NewVersionCmd(cliName))
	return command
}
