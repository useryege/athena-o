package commands

import (
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"google.golang.org/grpc/health/grpc_health_v1"

	cmdutil "github.com/useryege/athena/cmd/util"
	"github.com/useryege/athena/common"
	"github.com/useryege/athena/internal/blocksniffer"
	"github.com/useryege/athena/internal/blocksniffer/apiclient"
	"github.com/useryege/athena/internal/blocksniffer/metrics"
	"github.com/useryege/athena/util/cli"
	"github.com/useryege/athena/util/env"
	"github.com/useryege/athena/util/errors"
	"github.com/useryege/athena/util/healthz"
	utilio "github.com/useryege/athena/util/io"
)

const cliName = "athena-block-sniffer"

// NewCommand returns a new instance of the athena-block-sniffer command.
func NewCommand() *cobra.Command {
	var (
		listenHost  string
		listenPort  int
		metricsHost string
		metricsPort int
	)
	command := &cobra.Command{
		Use:               cliName,
		Short:             "Run the Athena Block Sniffer",
		Long:              "The Block Sniffer monitors EVM blocks and on-chain events in real time. This command runs the service in the foreground.",
		DisableAutoGenTag: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			vers := common.GetVersion()
			vers.LogStartupInfo(
				"Athena Block Sniffer",
				map[string]any{
					"port": listenPort,
				},
			)

			cli.SetLogFormat(cmdutil.LogFormat)
			cli.SetLogLevel(cmdutil.LogLevel)

			metricsServer := metrics.NewMetricsServer()
			http.Handle("/metrics", metricsServer.GetHandler())
			go func() { errors.CheckError(http.ListenAndServe(fmt.Sprintf("%s:%d", metricsHost, metricsPort), nil)) }()

			server := blocksniffer.NewServer()
			grpc := server.CreateGRPC()
			ctx := cmd.Context()

			lc := &net.ListenConfig{}
			listener, err := lc.Listen(ctx, "tcp", fmt.Sprintf("%s:%d", listenHost, listenPort))
			errors.CheckError(err)

			healthz.ServeHealthCheck(http.DefaultServeMux, func(r *http.Request) error {
				if val, ok := r.URL.Query()["full"]; ok && len(val) > 0 && val[0] == "true" {
					// connect to itself to make sure commit server is able to serve connection
					// used by liveness probe to auto restart commit server
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

			// Graceful shutdown code adapted from here: https://gist.github.com/embano1/e0bf49d24f1cdd07cffad93097c04f0a
			sigCh := make(chan os.Signal, 1)
			signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
			wg := sync.WaitGroup{}
			wg.Add(1)
			go func() {
				s := <-sigCh
				log.Printf("got signal %v, attempting graceful shutdown", s)
				grpc.GracefulStop()
				wg.Done()
			}()

			log.Println("starting grpc server")
			err = grpc.Serve(listener)
			errors.CheckError(err)
			wg.Wait()
			log.Println("clean shutdown")
			return nil
		},
	}

	command.Flags().StringVar(&cmdutil.LogFormat, "logformat", env.StringFromEnv("ATHENA_BLOCK_SNIFFER_LOGFORMAT", "json"), "Set the logging format. One of: json|text")
	command.Flags().StringVar(&cmdutil.LogLevel, "loglevel", env.StringFromEnv("ATHENA_BLOCK_SNIFFER_LOGLEVEL", "info"), "Set the logging level. One of: debug|info|warn|error")
	command.Flags().StringVar(&listenHost, "address", env.StringFromEnv("ATHENA_BLOCK_SNIFFER_LISTEN_ADDRESS", common.DefaultAddressBlockSniffer), "Listen on given address for incoming connections")
	command.Flags().IntVar(&listenPort, "port", common.DefaultPortBlockSniffer, "Listen on given port for incoming connections")
	command.Flags().StringVar(&metricsHost, "metrics-address", env.StringFromEnv("ATHENA_BLOCK_SNIFFER_METRICS_LISTEN_ADDRESS", common.DefaultAddressBlockSnifferMetrics), "Listen on given address for metrics and health checks")
	command.Flags().IntVar(&metricsPort, "metrics-port", common.DefaultPortBlockSnifferMetrics, "Start metrics server on given port")

	command.AddCommand(cli.NewVersionCmd(cliName))
	return command
}
