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
	notificationapiclient "github.com/useryege/athena/internal/notification/apiclient"
	"github.com/useryege/athena/internal/worm"
	wormstore "github.com/useryege/athena/internal/worm/store"
	"github.com/useryege/athena/util/cli"
	"github.com/useryege/athena/util/env"
	"github.com/useryege/athena/util/errors"
	utilio "github.com/useryege/athena/util/io"
	"github.com/useryege/athena/util/templates"
	utilworm "github.com/useryege/athena/util/worm"
)

const cliName = "athena-worm"

func NewCommand() *cobra.Command {
	var (
		listenHost                string
		listenPort                int
		wormAPIBaseURL            string
		notificationEnabled       bool
		notificationServerAddress string

		storeSrc func(context.Context) (*wormstore.SQLStore, error)
	)

	command := &cobra.Command{
		Use:               cliName,
		Short:             "Run the Athena Worm service",
		Long:              "The Worm service manages worm-level workloads. This command runs the service in the foreground.",
		DisableAutoGenTag: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			vers := common.GetVersion()
			vers.LogStartupInfo(
				"Athena Worm",
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

			wormClient, err := utilworm.NewClient(utilworm.Config{BaseURL: wormAPIBaseURL})
			if err != nil {
				return err
			}

			var notificationClientset notificationapiclient.Clientset
			if notificationEnabled {
				notificationClientset = notificationapiclient.NewNotificationClientset(notificationServerAddress)
			}

			server, err := worm.NewServer(worm.ServerOpts{
				Store:                 store,
				WormClient:            wormClient,
				WormAPIBaseURL:        wormAPIBaseURL,
				NotificationClientset: notificationClientset,
			})
			if err != nil {
				return err
			}

			wormGRPC := server.CreateGRPC()

			lc := &net.ListenConfig{}
			listener, err := lc.Listen(ctx, "tcp", fmt.Sprintf("%s:%d", listenHost, listenPort))
			errors.CheckError(err)

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
				wormGRPC.GracefulStop()
				if err := server.Stop(); err != nil {
					log.Printf("failed to stop worm server cleanly: %v", err)
				}
				wg.Done()
			}()

			log.Println("starting worm grpc server")
			err = wormGRPC.Serve(listener)
			if err != nil && !stderrors.Is(err, grpc.ErrServerStopped) {
				errors.CheckError(err)
			}

			wg.Wait()
			log.Println("clean shutdown")
			return nil
		},
		Example: templates.Examples(`
			# Start the Athena Worm service
			$ athena-worm
		`),
	}

	command.Flags().StringVar(&cmdutil.LogFormat, "logformat", env.StringFromEnv(common.EnvLogFormat, "json"), "Set the logging format. One of: json|text")
	command.Flags().StringVar(&cmdutil.LogLevel, "loglevel", env.StringFromEnv(common.EnvLogLevel, "info"), "Set the logging level. One of: debug|info|warn|error")
	command.Flags().StringVar(&listenHost, "address", env.StringFromEnv("ATHENA_WORM_LISTEN_ADDRESS", common.DefaultAddressWorm), "Listen on given address for incoming connections")
	command.Flags().IntVar(&listenPort, "port", common.DefaultPortWorm, "Listen on given port for incoming connections")
	command.Flags().StringVar(&wormAPIBaseURL, "worm-api-base-url", env.StringFromEnv("ATHENA_WORM_API_BASE_URL", utilworm.DefaultBaseURL), "Worm API base URL")
	command.Flags().BoolVar(&notificationEnabled, "notification-enabled", env.ParseBoolFromEnv("ATHENA_WORM_NOTIFICATION_ENABLED", true), "Enable Worm notifications through Athena Notification")
	command.Flags().StringVar(&notificationServerAddress, "notification-server-address", env.StringFromEnv("ATHENA_WORM_NOTIFICATION_SERVER_ADDRESS", fmt.Sprintf("localhost:%d", common.DefaultPortNotification)), "Athena notification gRPC server address for Worm alerts")

	storeSrc = wormstore.NewSQLStoreSource()

	command.AddCommand(cli.NewVersionCmd(cliName))
	return command
}
