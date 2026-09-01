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
	"github.com/useryege/athena/internal/wormmarkets"
	wormmarketsstore "github.com/useryege/athena/internal/wormmarkets/store"
	"github.com/useryege/athena/util/cli"
	"github.com/useryege/athena/util/env"
	"github.com/useryege/athena/util/errors"
	utilio "github.com/useryege/athena/util/io"
	"github.com/useryege/athena/util/templates"
	utilworm "github.com/useryege/athena/util/worm"
)

const cliName = "athena-worm-markets"

func NewCommand() *cobra.Command {
	var (
		listenHost                string
		listenPort                int
		wormAPIBaseURL            string
		notificationEnabled       bool
		notificationServerAddress string

		storeSrc func(context.Context) (*wormmarketsstore.SQLStore, error)
	)

	command := &cobra.Command{
		Use:               cliName,
		Short:             "Run the Athena Worm Markets service",
		Long:              "The Worm Markets service synchronizes and serves Worm sports markets. This command runs the service in the foreground.",
		DisableAutoGenTag: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			vers := common.GetVersion()
			vers.LogStartupInfo(
				"Athena Worm Markets",
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
				notificationClientset, err = notificationapiclient.NewNotificationClientset(
					notificationServerAddress,
					env.StringFromEnv(notificationapiclient.InternalAuthTokenEnv, ""),
				)
				if err != nil {
					return fmt.Errorf("create notification clientset: %w", err)
				}
				defer utilio.Close(notificationClientset)
			}

			server, err := wormmarkets.NewServer(wormmarkets.ServerOpts{
				Store:                 store,
				WormClient:            wormClient,
				WormAPIBaseURL:        wormAPIBaseURL,
				NotificationClientset: notificationClientset,
			})
			if err != nil {
				return err
			}

			wormMarketsGRPC := server.CreateGRPC()

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
				wormMarketsGRPC.GracefulStop()
				if err := server.Stop(); err != nil {
					log.Printf("failed to stop worm markets server cleanly: %v", err)
				}
				wg.Done()
			}()

			log.Println("starting worm markets grpc server")
			err = wormMarketsGRPC.Serve(listener)
			if err != nil && !stderrors.Is(err, grpc.ErrServerStopped) {
				errors.CheckError(err)
			}

			wg.Wait()
			log.Println("clean shutdown")
			return nil
		},
		Example: templates.Examples(`
			# Start the Athena Worm Markets service
			$ athena-worm-markets
		`),
	}

	command.Flags().StringVar(&cmdutil.LogFormat, "logformat", env.StringFromEnv(common.EnvLogFormat, "json"), "Set the logging format. One of: json|text")
	command.Flags().StringVar(&cmdutil.LogLevel, "loglevel", env.StringFromEnv(common.EnvLogLevel, "info"), "Set the logging level. One of: debug|info|warn|error")
	command.Flags().StringVar(&listenHost, "address", env.StringFromEnv("ATHENA_WORM_MARKETS_LISTEN_ADDRESS", common.DefaultAddressWormMarkets), "Listen on given address for incoming connections")
	command.Flags().IntVar(&listenPort, "port", common.DefaultPortWormMarkets, "Listen on given port for incoming connections")
	command.Flags().StringVar(&wormAPIBaseURL, "worm-api-base-url", env.StringFromEnv("ATHENA_WORM_MARKETS_API_BASE_URL", utilworm.DefaultBaseURL), "Worm API base URL")
	command.Flags().BoolVar(&notificationEnabled, "notification-enabled", env.ParseBoolFromEnv("ATHENA_WORM_MARKETS_NOTIFICATION_ENABLED", true), "Enable Worm Markets notifications through Athena Notification")
	command.Flags().StringVar(&notificationServerAddress, "notification-server-address", env.StringFromEnv("ATHENA_WORM_MARKETS_NOTIFICATION_SERVER_ADDRESS", fmt.Sprintf("%s:%d", common.DefaultLocalGRPCHost, common.DefaultPortNotification)), "Athena notification gRPC server address for Worm Markets alerts")

	storeSrc = wormmarketsstore.NewSQLStoreSource()

	command.AddCommand(cli.NewVersionCmd(cliName))
	return command
}
