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

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"

	cmdutil "github.com/useryege/athena/cmd/util"
	"github.com/useryege/athena/common"
	"github.com/useryege/athena/internal/etherscanmanager"
	"github.com/useryege/athena/util/cli"
	"github.com/useryege/athena/util/env"
	"github.com/useryege/athena/util/errors"
	"github.com/useryege/athena/util/templates"
)

const cliName = "athena-etherscan-manager"

func NewCommand() *cobra.Command {
	var (
		listenHost                string
		listenPort                int
		etherscanAPIKeys          string
		etherscanGatewayAddrs     string
		etherscanGatewayAuthToken string
	)

	command := &cobra.Command{
		Use:               cliName,
		Short:             "Run the Athena Etherscan Manager service",
		Long:              "The Etherscan Manager service schedules Etherscan API keys and gateway clients. This command runs the service in the foreground.",
		DisableAutoGenTag: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			vers := common.GetVersion()
			vers.LogStartupInfo(
				"Athena Etherscan Manager",
				map[string]any{
					"port": listenPort,
				},
			)

			cli.SetLogFormat(cmdutil.LogFormat)
			cli.SetLogLevel(cmdutil.LogLevel)

			ctx := cmd.Context()

			manager, err := etherscanmanager.NewManager(etherscanmanager.ManagerOpts{
				APIKeys:      parseListEnv(etherscanAPIKeys),
				GatewayAddrs: parseListEnv(etherscanGatewayAddrs),
				AuthToken:    etherscanGatewayAuthToken,
			})
			if err != nil {
				return err
			}
			defer func() {
				if err := manager.Close(); err != nil {
					log.Printf("failed to close etherscan manager cleanly: %v", err)
				}
			}()

			server, err := etherscanmanager.NewServer(etherscanmanager.ServerOpts{
				Manager: manager,
			})
			if err != nil {
				return err
			}
			managerGRPC := server.CreateGRPC()

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
				managerGRPC.GracefulStop()
				if err := server.Stop(); err != nil {
					log.Printf("failed to stop etherscan-manager server cleanly: %v", err)
				}
				wg.Done()
			}()

			log.Println("starting etherscan-manager grpc server")
			err = managerGRPC.Serve(listener)
			if err != nil && !stderrors.Is(err, grpc.ErrServerStopped) {
				errors.CheckError(err)
			}

			wg.Wait()
			log.Println("clean shutdown")
			return nil
		},
		Example: templates.Examples(`
			# Start the Athena Etherscan Manager service
			$ athena-etherscan-manager
		`),
	}

	command.Flags().StringVar(&cmdutil.LogFormat, "logformat", env.StringFromEnv(common.EnvLogFormat, "json"), "Set the logging format. One of: json|text")
	command.Flags().StringVar(&cmdutil.LogLevel, "loglevel", env.StringFromEnv(common.EnvLogLevel, "info"), "Set the logging level. One of: debug|info|warn|error")
	command.Flags().StringVar(&listenHost, "address", env.StringFromEnv("ATHENA_ETHERSCAN_MANAGER_LISTEN_ADDRESS", common.DefaultAddressEtherscanManager), "Listen on given address for incoming connections")
	command.Flags().IntVar(&listenPort, "port", env.ParseNumFromEnv("ATHENA_ETHERSCAN_MANAGER_PORT", common.DefaultPortEtherscanManager, 1, 65535), "Listen on given port for incoming connections")
	command.Flags().StringVar(&etherscanAPIKeys, "etherscan-api-keys", env.StringFromEnv("ATHENA_ETHERSCAN_MANAGER_API_KEYS", ""), "Comma, space, or newline-separated Etherscan API keys")
	command.Flags().StringVar(&etherscanGatewayAddrs, "etherscan-gateway-addrs", env.StringFromEnv("ATHENA_ETHERSCAN_MANAGER_GATEWAY_ADDRS", ""), "Comma, space, or newline-separated Etherscan Gateway gRPC addresses in host:port form")
	command.Flags().StringVar(&etherscanGatewayAuthToken, "etherscan-gateway-auth-token", env.StringFromEnv("ATHENA_ETHERSCAN_GATEWAY_AUTH_TOKEN", ""), "Bearer token for Etherscan Gateway gRPC calls")

	command.AddCommand(cli.NewVersionCmd(cliName))
	return command
}

func parseListEnv(raw string) []string {
	parts := strings.FieldsFunc(raw, func(r rune) bool {
		return r == ',' || r == '\n' || r == '\r' || r == '\t' || r == ' '
	})
	values := make([]string, 0, len(parts))
	for _, part := range parts {
		value := strings.Trim(part, `"'`)
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		values = append(values, value)
	}
	return values
}
