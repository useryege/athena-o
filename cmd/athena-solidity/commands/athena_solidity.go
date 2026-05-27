package commands

import (
	"context"
	stderrors "errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health/grpc_health_v1"

	cmdutil "github.com/useryege/athena/cmd/util"
	"github.com/useryege/athena/common"
	"github.com/useryege/athena/internal/solidity"
	"github.com/useryege/athena/internal/solidity/apiclient"
	"github.com/useryege/athena/internal/solidity/metrics"
	soliditystore "github.com/useryege/athena/internal/solidity/store"
	"github.com/useryege/athena/util/cli"
	"github.com/useryege/athena/util/env"
	"github.com/useryege/athena/util/errors"
	"github.com/useryege/athena/util/healthz"
	utilio "github.com/useryege/athena/util/io"
	"github.com/useryege/athena/util/templates"
)

const cliName = "athena-solidity"

func NewCommand() *cobra.Command {
	var (
		listenHost  string
		listenPort  int
		metricsHost string
		metricsPort int

		storeSrc func(context.Context) (*soliditystore.SQLStore, error)
	)

	command := &cobra.Command{
		Use:               cliName,
		Short:             "Run the Athena Solidity service",
		Long:              "The Solidity service manages contract source workloads. This command runs the service in the foreground.",
		DisableAutoGenTag: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			vers := common.GetVersion()
			vers.LogStartupInfo(
				"Athena Solidity",
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

			metricsServer := metrics.NewMetricsServer()
			metricsMux := http.NewServeMux()
			metricsMux.Handle("/", metricsServer.GetHandler())
			go func() {
				errors.CheckError(http.ListenAndServe(fmt.Sprintf("%s:%d", metricsHost, metricsPort), metricsMux))
			}()

			server, err := solidity.NewServer(solidity.ServerOpts{Store: store})
			if err != nil {
				return err
			}
			solidityGRPC := server.CreateGRPC()

			lc := &net.ListenConfig{}
			listener, err := lc.Listen(ctx, "tcp", fmt.Sprintf("%s:%d", listenHost, listenPort))
			errors.CheckError(err)

			healthz.ServeHealthCheck(metricsMux, func(r *http.Request) error {
				if val, ok := r.URL.Query()["full"]; ok && len(val) > 0 && val[0] == "true" {
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
				}
				return nil
			})

			if err := server.Start(); err != nil {
				return err
			}

			sigCh := make(chan os.Signal, 1)
			signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
			wg := sync.WaitGroup{}
			wg.Add(1)
			go func() {
				s := <-sigCh
				log.Printf("got signal %v, attempting graceful shutdown", s)
				solidityGRPC.GracefulStop()
				if err := server.Stop(); err != nil {
					log.Printf("failed to stop solidity server cleanly: %v", err)
				}
				wg.Done()
			}()

			log.Println("starting solidity grpc server")
			err = solidityGRPC.Serve(listener)
			if err != nil && !stderrors.Is(err, grpc.ErrServerStopped) {
				errors.CheckError(err)
			}

			wg.Wait()
			log.Println("clean shutdown")
			return nil
		},
		Example: templates.Examples(`
			# Start the Athena Solidity service
			$ athena-solidity
		`),
	}

	command.Flags().StringVar(&cmdutil.LogFormat, "logformat", env.StringFromEnv("ATHENA_SOLIDITY_LOGFORMAT", "json"), "Set the logging format. One of: json|text")
	command.Flags().StringVar(&cmdutil.LogLevel, "loglevel", env.StringFromEnv("ATHENA_SOLIDITY_LOGLEVEL", "info"), "Set the logging level. One of: debug|info|warn|error")
	command.Flags().StringVar(&listenHost, "address", env.StringFromEnv("ATHENA_SOLIDITY_LISTEN_ADDRESS", common.DefaultAddressSolidity), "Listen on given address for incoming connections")
	command.Flags().IntVar(&listenPort, "port", common.DefaultPortSolidity, "Listen on given port for incoming connections")
	command.Flags().StringVar(&metricsHost, "metrics-address", env.StringFromEnv("ATHENA_SOLIDITY_METRICS_LISTEN_ADDRESS", common.DefaultAddressSolidityMetrics), "Listen on given address for metrics and health checks")
	command.Flags().IntVar(&metricsPort, "metrics-port", common.DefaultPortSolidityMetrics, "Start metrics server on given port")

	storeSrc = soliditystore.NewSQLStoreSource()

	command.AddCommand(cli.NewVersionCmd(cliName))
	return command
}
