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
	"github.com/useryege/athena/util/cli"
	"github.com/useryege/athena/util/env"
	"github.com/useryege/athena/util/errors"
	"github.com/useryege/athena/util/templates"
)

const cliName = "athena-token"

func NewCommand() *cobra.Command {
	var (
		listenHost     string
		listenPort     int
		mode           string
		ethNodeWSURL   string
		bscNodeWSURL   string
		nodeWSUseProxy bool
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
				Mode:           mode,
				StoreSrc:       tokenstore.NewSQLStoreSource(),
				EthNodeWSURL:   ethNodeWSURL,
				BSCNodeWSURL:   bscNodeWSURL,
				NodeWSUseProxy: nodeWSUseProxy,
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
			case token.ModeChainIngestor:
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
		`),
	}

	command.Flags().StringVar(&cmdutil.LogFormat, "logformat", env.StringFromEnv("ATHENA_TOKEN_LOGFORMAT", "json"), "Set the logging format. One of: json|text")
	command.Flags().StringVar(&cmdutil.LogLevel, "loglevel", env.StringFromEnv("ATHENA_TOKEN_LOGLEVEL", "info"), "Set the logging level. One of: debug|info|warn|error")
	command.Flags().StringVar(&listenHost, "address", env.StringFromEnv("ATHENA_TOKEN_LISTEN_ADDRESS", common.DefaultAddressToken), "Listen on given address for incoming connections")
	command.Flags().IntVar(&listenPort, "port", common.DefaultPortToken, "Listen on given port for incoming connections")
	command.Flags().StringVar(&mode, "mode", env.StringFromEnv("ATHENA_TOKEN_MODE", token.ModeGRPC), "Run mode: grpc|chain-ingestor")
	command.Flags().StringVar(&ethNodeWSURL, "eth-node-ws-url", env.StringFromEnv("ATHENA_TOKEN_ETH_NODE_WS_URL", ""), "Ethereum Mainnet node WebSocket address for chain-ingestor mode")
	command.Flags().StringVar(&bscNodeWSURL, "bsc-node-ws-url", env.StringFromEnv("ATHENA_TOKEN_BSC_NODE_WS_URL", ""), "BSC Mainnet node WebSocket address for chain-ingestor mode")
	command.Flags().BoolVar(&nodeWSUseProxy, "node-ws-use-proxy", env.ParseBoolFromEnv("ATHENA_TOKEN_NODE_WS_USE_PROXY", false), "Whether to use proxy environment variables for node WebSocket connections")

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
