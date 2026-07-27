package commands

import (
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

	"github.com/useryege/athena/cmd/tokenchain"
	cmdutil "github.com/useryege/athena/cmd/util"
	"github.com/useryege/athena/common"
	"github.com/useryege/athena/internal/token/adapters/evm"
	tokenpostgres "github.com/useryege/athena/internal/token/adapters/postgres"
	catalogapp "github.com/useryege/athena/internal/token/catalog/application"
	"github.com/useryege/athena/internal/token/discovery"
	discoveryapp "github.com/useryege/athena/internal/token/discovery/application"
	policyapp "github.com/useryege/athena/internal/token/policy/application"
	projectviewapp "github.com/useryege/athena/internal/token/projectview/application"
	reportingapp "github.com/useryege/athena/internal/token/reporting/application"
	researchapp "github.com/useryege/athena/internal/token/research/application"
	selectionapp "github.com/useryege/athena/internal/token/selection/application"
	"github.com/useryege/athena/internal/tokenapi"
	"github.com/useryege/athena/util/cli"
	"github.com/useryege/athena/util/env"
	"github.com/useryege/athena/util/errors"
	"github.com/useryege/athena/util/ethws"
	"github.com/useryege/athena/util/templates"
)

const cliName = "athena-token-api"

func NewCommand() *cobra.Command {
	var (
		listenHost     string
		listenPort     int
		chains         tokenchain.Flags
		nodeWSProxyURL string
	)

	command := &cobra.Command{
		Use:               cliName,
		Short:             "Run the Athena Token API service",
		Long:              "The Token API service manages token-api-level workloads. This command runs the service in the foreground.",
		DisableAutoGenTag: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			vers := common.GetVersion()
			vers.LogStartupInfo(
				"Athena Token API",
				map[string]any{
					"port": listenPort,
				},
			)

			cli.SetLogFormat(cmdutil.LogFormat)
			cli.SetLogLevel(cmdutil.LogLevel)

			ctx := cmd.Context()

			if err := ethws.ValidateProxyURL(nodeWSProxyURL); err != nil {
				return err
			}
			registry, err := chains.Registry()
			if err != nil {
				return err
			}
			connection, err := tokenpostgres.NewConnectionSource()(ctx)
			if err != nil {
				return err
			}
			chains := make([]discovery.Chain, 0, len(registry.Chains()))
			for _, chain := range registry.Chains() {
				chains = append(chains, discovery.Chain{ID: chain.ID, Name: chain.Name, Enabled: chain.Enabled})
			}
			chainRepository := tokenpostgres.NewChainRepository(connection)
			if err := chainRepository.SyncChains(ctx, chains); err != nil {
				_ = connection.Close()
				return err
			}
			evmRegistry := evm.NewChainClientRegistry(registry, nodeWSProxyURL)
			server, err := tokenapi.NewServer(tokenapi.ServerOpts{
				Applications: tokenapi.Applications{
					Catalog:     catalogapp.NewQueries(tokenpostgres.NewCatalogRepository(connection)),
					Research:    researchapp.NewQueries(tokenpostgres.NewResearchReadRepository(connection)),
					Reporting:   reportingapp.NewQueries(tokenpostgres.NewReportingRepository(connection)),
					Selection:   selectionapp.NewQueries(tokenpostgres.NewSelectionRepository(connection)),
					ProjectView: projectviewapp.NewQueries(tokenpostgres.NewProjectViewRepository(connection)),
					Policy:      policyapp.NewService(tokenpostgres.NewPolicyRepository(connection), evmRegistry),
					Operations:  discoveryapp.NewOperations(chainRepository, evmRegistry),
				},
				Close: func() error {
					evmErr := evmRegistry.Close()
					databaseErr := connection.Close()
					if evmErr != nil {
						return evmErr
					}
					return databaseErr
				},
			})
			if err != nil {
				_ = evmRegistry.Close()
				_ = connection.Close()
				return err
			}
			tokenAPIGRPC := server.CreateGRPC()

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
				tokenAPIGRPC.GracefulStop()
				if err := server.Stop(); err != nil {
					log.Printf("failed to stop token API server cleanly: %v", err)
				}
				wg.Done()
			}()

			log.Println("starting token API grpc server")
			err = tokenAPIGRPC.Serve(listener)
			if err != nil && !stderrors.Is(err, grpc.ErrServerStopped) {
				errors.CheckError(err)
			}

			wg.Wait()
			log.Println("clean shutdown")
			return nil
		},
		Example: templates.Examples(`
			# Start the Athena Token API service
			$ athena-token-api
		`),
	}

	command.Flags().StringVar(&cmdutil.LogFormat, "logformat", env.StringFromEnv(common.EnvLogFormat, "json"), "Set the logging format. One of: json|text")
	command.Flags().StringVar(&cmdutil.LogLevel, "loglevel", env.StringFromEnv(common.EnvLogLevel, "info"), "Set the logging level. One of: debug|info|warn|error")
	command.Flags().StringVar(&listenHost, "address", env.StringFromEnv("ATHENA_TOKEN_API_LISTEN_ADDRESS", common.DefaultAddressTokenAPI), "Listen on given address for incoming connections")
	command.Flags().IntVar(&listenPort, "port", common.DefaultPortTokenAPI, "Listen on given port for incoming connections")
	chains.Bind(command)
	command.Flags().StringVar(&nodeWSProxyURL, "node-ws-proxy-url", env.StringFromEnv(ethws.ProxyURLEnvironmentVariable, ""), "Proxy URL used only for Token EVM WebSocket connections")

	command.AddCommand(cli.NewVersionCmd(cliName))
	return command
}
