package commands

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"

	cmdutil "github.com/useryege/athena/cmd/util"
	"github.com/useryege/athena/common"
	"github.com/useryege/athena/internal/managedoo"
	managedoostore "github.com/useryege/athena/internal/managedoo/store"
	notificationapiclient "github.com/useryege/athena/internal/notification/apiclient"
	"github.com/useryege/athena/util/cli"
	"github.com/useryege/athena/util/env"
	utilio "github.com/useryege/athena/util/io"
)

const cliName = "athena-managed-oo"

func NewCommand() *cobra.Command {
	var listenHost string
	var listenPort int
	var notificationEnabled bool
	var notificationServerAddress string
	var notificationInviteCode string
	var polygonRPCURL string
	var storeSource func(context.Context) (*managedoostore.SQLStore, error)

	command := &cobra.Command{
		Use:               cliName,
		Short:             "Run the Athena Managed OO service",
		Long:              "Managed OO owns Polymarket oracle log ingestion, market enrichment, queries, and alerts.",
		DisableAutoGenTag: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			common.GetVersion().LogStartupInfo("Athena Managed OO", map[string]any{"port": listenPort, "polygonRPCURL": polygonRPCURL})
			cli.SetLogFormat(cmdutil.LogFormat)
			cli.SetLogLevel(cmdutil.LogLevel)
			store, err := storeSource(cmd.Context())
			if err != nil {
				return err
			}
			defer utilio.Close(store)
			var notificationClientset notificationapiclient.Clientset
			if notificationEnabled {
				notificationClientset, err = notificationapiclient.NewNotificationClientset(notificationServerAddress)
				if err != nil {
					return fmt.Errorf("create notification clientset: %w", err)
				}
				defer utilio.Close(notificationClientset)
			}
			server, err := managedoo.NewServer(managedoo.ServerOpts{
				Store:                  store,
				NotificationClientset:  notificationClientset,
				NotificationInviteCode: notificationInviteCode,
				PolygonRPCURL:          polygonRPCURL,
				ProposedAlertsConfig:   managedoo.ManagedOOProposedAlertsConfig{Enabled: notificationEnabled},
				DisputedAlertsConfig:   managedoo.ManagedOODisputedAlertsConfig{Enabled: notificationEnabled},
			})
			if err != nil {
				return err
			}
			listener, err := (&net.ListenConfig{}).Listen(cmd.Context(), "tcp", fmt.Sprintf("%s:%d", listenHost, listenPort))
			if err != nil {
				return err
			}
			if err := server.Start(); err != nil {
				return err
			}
			grpcServer := server.CreateGRPC()
			signalContext, stopSignals := signal.NotifyContext(cmd.Context(), os.Interrupt, syscall.SIGTERM)
			defer stopSignals()
			go func() {
				<-signalContext.Done()
				grpcServer.GracefulStop()
			}()
			if err := grpcServer.Serve(listener); err != nil {
				_ = server.Stop()
				return err
			}
			return server.Stop()
		},
		Example: "Start the Athena Managed OO service:\n  athena-managed-oo",
	}
	command.Flags().StringVar(&cmdutil.LogFormat, "logformat", env.StringFromEnv(common.EnvLogFormat, "json"), "Set logging format")
	command.Flags().StringVar(&cmdutil.LogLevel, "loglevel", env.StringFromEnv(common.EnvLogLevel, "info"), "Set logging level")
	command.Flags().StringVar(&listenHost, "address", env.StringFromEnv("ATHENA_MANAGED_OO_LISTEN_ADDRESS", common.DefaultAddressManagedOO), "Listen address")
	command.Flags().IntVar(&listenPort, "port", env.ParseNumFromEnv("ATHENA_MANAGED_OO_LISTEN_PORT", common.DefaultPortManagedOO, 1, 65535), "Listen port")
	command.Flags().BoolVar(&notificationEnabled, "notification-enabled", env.ParseBoolFromEnv("ATHENA_MANAGED_OO_NOTIFICATION_ENABLED", true), "Enable Managed OO notifications")
	command.Flags().StringVar(&notificationServerAddress, "notification-server-address", env.StringFromEnv("ATHENA_MANAGED_OO_NOTIFICATION_SERVER_ADDRESS", fmt.Sprintf("%s:%d", common.DefaultLocalGRPCHost, common.DefaultPortNotification)), "Notification service address")
	command.Flags().StringVar(&notificationInviteCode, "notification-invite-code", env.StringFromEnv("ATHENA_MANAGED_OO_NOTIFICATION_INVITE_CODE", ""), "Polymarket invite code")
	command.Flags().StringVar(&polygonRPCURL, "polygon-rpc-url", env.StringFromEnv("ATHENA_MANAGED_OO_POLYGON_RPC_URL", "https://polygon-rpc.com"), "Polygon JSON-RPC URL")
	storeSource = managedoostore.NewSQLStoreSource()
	command.AddCommand(cli.NewVersionCmd(cliName))
	return command
}
