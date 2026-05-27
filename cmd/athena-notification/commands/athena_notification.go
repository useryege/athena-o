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
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health/grpc_health_v1"

	cmdutil "github.com/useryege/athena/cmd/util"
	"github.com/useryege/athena/common"
	"github.com/useryege/athena/internal/notification"
	"github.com/useryege/athena/internal/notification/apiclient"
	"github.com/useryege/athena/internal/notification/metrics"
	notificationstore "github.com/useryege/athena/internal/notification/store"
	"github.com/useryege/athena/util/cli"
	"github.com/useryege/athena/util/env"
	"github.com/useryege/athena/util/errors"
	"github.com/useryege/athena/util/healthz"
	utilio "github.com/useryege/athena/util/io"
	utiltelegram "github.com/useryege/athena/util/telegram"
	"github.com/useryege/athena/util/templates"
)

const cliName = "athena-notification"

func NewCommand() *cobra.Command {
	var (
		listenHost  string
		listenPort  int
		metricsHost string
		metricsPort int

		storeSrc func(context.Context) (*notificationstore.SQLStore, error)
	)

	command := &cobra.Command{
		Use:               cliName,
		Short:             "Run the Athena Notification service",
		Long:              "The Notification service manages notification-level workloads. This command runs the service in the foreground.",
		DisableAutoGenTag: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			vers := common.GetVersion()
			vers.LogStartupInfo(
				"Athena Notification",
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

			telegramClient, err := utiltelegram.NewClient(utiltelegram.Config{
				BotToken: env.StringFromEnv("ATHENA_NOTIFICATION_TELEGRAM_BOT_TOKEN", ""),
				ChatID:   env.StringFromEnv("ATHENA_NOTIFICATION_TELEGRAM_CHAT_ID", ""),
				BaseURL:  env.StringFromEnv("ATHENA_NOTIFICATION_TELEGRAM_API_URL", utiltelegram.DefaultBaseURL),
				Timeout:  time.Duration(env.ParseNumFromEnv("ATHENA_NOTIFICATION_TELEGRAM_TIMEOUT_SECONDS", int(utiltelegram.DefaultTimeout/time.Second), 1, 300)) * time.Second,
			})
			if err != nil {
				return err
			}

			metricsServer := metrics.NewMetricsServer()
			metricsMux := http.NewServeMux()
			metricsMux.Handle("/", metricsServer.GetHandler())
			go func() {
				errors.CheckError(http.ListenAndServe(fmt.Sprintf("%s:%d", metricsHost, metricsPort), metricsMux))
			}()

			server, err := notification.NewServer(notification.ServerOpts{Store: store, Sender: notification.NewTelegramSender(telegramClient)})
			if err != nil {
				return err
			}
			notificationGRPC := server.CreateGRPC()

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
				notificationGRPC.GracefulStop()
				if err := server.Stop(); err != nil {
					log.Printf("failed to stop notification server cleanly: %v", err)
				}
				wg.Done()
			}()

			log.Println("starting notification grpc server")
			err = notificationGRPC.Serve(listener)
			if err != nil && !stderrors.Is(err, grpc.ErrServerStopped) {
				errors.CheckError(err)
			}

			wg.Wait()
			log.Println("clean shutdown")
			return nil
		},
		Example: templates.Examples(`
			# Start the Athena Notification service
			$ athena-notification
		`),
	}

	command.Flags().StringVar(&cmdutil.LogFormat, "logformat", env.StringFromEnv("ATHENA_NOTIFICATION_LOGFORMAT", "json"), "Set the logging format. One of: json|text")
	command.Flags().StringVar(&cmdutil.LogLevel, "loglevel", env.StringFromEnv("ATHENA_NOTIFICATION_LOGLEVEL", "info"), "Set the logging level. One of: debug|info|warn|error")
	command.Flags().StringVar(&listenHost, "address", env.StringFromEnv("ATHENA_NOTIFICATION_LISTEN_ADDRESS", common.DefaultAddressNotification), "Listen on given address for incoming connections")
	command.Flags().IntVar(&listenPort, "port", common.DefaultPortNotification, "Listen on given port for incoming connections")
	command.Flags().StringVar(&metricsHost, "metrics-address", env.StringFromEnv("ATHENA_NOTIFICATION_METRICS_LISTEN_ADDRESS", common.DefaultAddressNotificationMetrics), "Listen on given address for metrics and health checks")
	command.Flags().IntVar(&metricsPort, "metrics-port", common.DefaultPortNotificationMetrics, "Start metrics server on given port")

	storeSrc = notificationstore.NewSQLStoreSource()

	command.AddCommand(cli.NewVersionCmd(cliName))
	return command
}
