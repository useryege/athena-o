package commands

import (
	"context"
	stderrors "errors"
	"fmt"
	"net"
	"os"
	"os/signal"
	"sync"
	"syscall"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"

	cmdutil "github.com/useryege/athena/cmd/util"
	"github.com/useryege/athena/common"
	"github.com/useryege/athena/internal/token"
	tokenstore "github.com/useryege/athena/internal/token/store"
	"github.com/useryege/athena/util/ave"
	"github.com/useryege/athena/util/cli"
	"github.com/useryege/athena/util/env"
	"github.com/useryege/athena/util/errors"
	"github.com/useryege/athena/util/templates"
)

const cliName = "athena-token"

func NewCommand() *cobra.Command {
	var (
		listenHost        string
		listenPort        int
		mode              string
		ethNodeWSURL      string
		bscNodeWSURL      string
		ethAthenaContract string
		bscAthenaContract string
		ethEnabled        bool
		bscEnabled        bool
		nodeWSUseProxy    bool
		aveAPIKey         string
		aveAPIBaseURL     string
	)

	command := &cobra.Command{
		Use:               cliName,
		Short:             "Run the Athena Token service",
		Long:              "The Token service manages token-level workloads. This command runs the service in the foreground.",
		DisableAutoGenTag: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			vers := common.GetVersion()
			vers.LogStartupInfo(
				"Athena Token",
				map[string]any{
					"mode": token.NormalizeMode(mode),
					"port": listenPort,
				},
			)

			cli.SetLogFormat(cmdutil.LogFormat)
			cli.SetLogLevel(cmdutil.LogLevel)

			ctx := cmd.Context()

			server, err := token.NewServer(token.ServerOpts{
				Mode:              mode,
				StoreSrc:          tokenstore.NewSQLStoreSource(),
				EthNodeWSURL:      ethNodeWSURL,
				BSCNodeWSURL:      bscNodeWSURL,
				EthAthenaContract: ethAthenaContract,
				BSCAthenaContract: bscAthenaContract,
				EthEnabled:        ethEnabled,
				BSCEnabled:        bscEnabled,
				NodeWSUseProxy:    nodeWSUseProxy,
				AveAPIKey:         aveAPIKey,
				AveAPIBaseURL:     aveAPIBaseURL,
			})
			if err != nil {
				return err
			}

			if err := server.Start(ctx); err != nil {
				return err
			}

			switch token.NormalizeMode(mode) {
			case token.ModeGRPC:
				return runGRPCMode(ctx, server, listenHost, listenPort)
			case token.ModeChainIngestor, token.ModeProjectQualifier, token.ModeProjectDataCollector:
				return runWorkerMode(ctx, server)
			default:
				if err := server.Stop(); err != nil {
					log.Printf("failed to stop token server cleanly: %v", err)
				}
				return fmt.Errorf("unsupported athena-token mode %q", mode)
			}
		},
		Example: templates.Examples(`
			# Start the Athena Token gRPC service
			$ athena-token --mode grpc

			# Start the Athena Token chain ingestor worker
			$ athena-token --mode chain-ingestor

			# Start the Athena Token project qualifier worker
			$ athena-token --mode project-qualifier

			# Start the Athena Token project data collector worker
			$ athena-token --mode project-data-collector
		`),
	}

	command.Flags().StringVar(&cmdutil.LogFormat, "logformat", env.StringFromEnv("ATHENA_TOKEN_LOGFORMAT", "json"), "Set the logging format. One of: json|text")
	command.Flags().StringVar(&cmdutil.LogLevel, "loglevel", env.StringFromEnv("ATHENA_TOKEN_LOGLEVEL", "info"), "Set the logging level. One of: debug|info|warn|error")
	command.Flags().StringVar(&listenHost, "address", env.StringFromEnv("ATHENA_TOKEN_LISTEN_ADDRESS", common.DefaultAddressToken), "Listen on given address for incoming connections")
	command.Flags().IntVar(&listenPort, "port", common.DefaultPortToken, "Listen on given port for incoming connections")
	command.Flags().StringVar(&mode, "mode", env.StringFromEnv("ATHENA_TOKEN_MODE", token.ModeGRPC), "Run mode: grpc|chain-ingestor|project-qualifier|project-data-collector")
	command.Flags().StringVar(&ethNodeWSURL, "eth-node-ws-url", env.StringFromEnv("ATHENA_TOKEN_ETH_NODE_WS_URL", ""), "Ethereum Mainnet node WebSocket address for worker modes")
	command.Flags().StringVar(&bscNodeWSURL, "bsc-node-ws-url", env.StringFromEnv("ATHENA_TOKEN_BSC_NODE_WS_URL", ""), "BSC Mainnet node WebSocket address for worker modes")
	command.Flags().StringVar(&ethAthenaContract, "eth-athena-contract", env.StringFromEnv("ATHENA_TOKEN_ETH_ATHENA_CONTRACT", ""), "Ethereum Mainnet ATHENA contract address for project-qualifier mode")
	command.Flags().StringVar(&bscAthenaContract, "bsc-athena-contract", env.StringFromEnv("ATHENA_TOKEN_BSC_ATHENA_CONTRACT", ""), "BSC Mainnet ATHENA contract address for project-qualifier mode")
	command.Flags().BoolVar(&ethEnabled, "eth-enabled", env.ParseBoolFromEnv("ATHENA_TOKEN_ETH_ENABLED", true), "Whether to enable Ethereum Mainnet token worker logic")
	command.Flags().BoolVar(&bscEnabled, "bsc-enabled", env.ParseBoolFromEnv("ATHENA_TOKEN_BSC_ENABLED", true), "Whether to enable BSC Mainnet token worker logic")
	command.Flags().BoolVar(&nodeWSUseProxy, "node-ws-use-proxy", env.ParseBoolFromEnv("ATHENA_TOKEN_NODE_WS_USE_PROXY", false), "Whether to use proxy environment variables for node WebSocket connections")
	command.Flags().StringVar(&aveAPIKey, "ave-api-key", env.StringFromEnv("ATHENA_TOKEN_AVE_API_KEY", ""), "Ave API key for project data collector mode")
	command.Flags().StringVar(&aveAPIBaseURL, "ave-api-base-url", env.StringFromEnv("ATHENA_TOKEN_AVE_API_BASE_URL", ave.DefaultBaseURL), "Ave API base URL for project data collector mode")

	command.AddCommand(cli.NewVersionCmd(cliName))
	return command
}

func runGRPCMode(ctx context.Context, server *token.Server, listenHost string, listenPort int) error {
	tokenGRPC := server.CreateGRPC()

	lc := &net.ListenConfig{}
	listener, err := lc.Listen(ctx, "tcp", fmt.Sprintf("%s:%d", listenHost, listenPort))
	errors.CheckError(err)

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(sigCh)
	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		s := <-sigCh
		log.Printf("got signal %v, attempting graceful shutdown", s)
		tokenGRPC.GracefulStop()
		if err := server.Stop(); err != nil {
			log.Printf("failed to stop token server cleanly: %v", err)
		}
		wg.Done()
	}()

	log.Println("starting token grpc server")
	err = tokenGRPC.Serve(listener)
	if err != nil && !stderrors.Is(err, grpc.ErrServerStopped) {
		errors.CheckError(err)
	}

	wg.Wait()
	log.Println("clean shutdown")
	return nil
}

func runWorkerMode(ctx context.Context, server *token.Server) error {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(sigCh)
	select {
	case <-ctx.Done():
	case s := <-sigCh:
		log.Printf("got signal %v, attempting graceful shutdown", s)
	}
	if err := server.Stop(); err != nil {
		return err
	}
	log.Println("clean shutdown")
	return nil
}
