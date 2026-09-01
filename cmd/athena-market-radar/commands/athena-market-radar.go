package commands

import (
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"

	cmdutil "github.com/useryege/athena/cmd/util"
	"github.com/useryege/athena/common"
	"github.com/useryege/athena/internal/marketradar"
	notificationapiclient "github.com/useryege/athena/internal/notification/apiclient"
	"github.com/useryege/athena/util/cli"
	"github.com/useryege/athena/util/env"
	utilio "github.com/useryege/athena/util/io"
)

const cliName = "athena-market-radar"

func NewCommand() *cobra.Command {
	var listenHost string
	var listenPort int
	var notificationEnabled bool
	var notificationServerAddress string
	var notificationInviteCode string

	command := &cobra.Command{
		Use:               cliName,
		Short:             "Run the Athena Market Radar service",
		Long:              "Market Radar owns Polymarket hot-market discovery, realtime windows, movers, and mover alerts.",
		DisableAutoGenTag: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			common.GetVersion().LogStartupInfo("Athena Market Radar", map[string]any{"port": listenPort})
			cli.SetLogFormat(cmdutil.LogFormat)
			cli.SetLogLevel(cmdutil.LogLevel)

			var notificationClientset notificationapiclient.Clientset
			if notificationEnabled {
				var err error
				notificationClientset, err = notificationapiclient.NewNotificationClientset(
					notificationServerAddress,
					env.StringFromEnv(notificationapiclient.InternalAuthTokenEnv, ""),
				)
				if err != nil {
					return fmt.Errorf("create notification clientset: %w", err)
				}
				defer utilio.Close(notificationClientset)
			}
			server, err := marketradar.NewServer(marketradar.ServerOpts{
				NotificationClientset:  notificationClientset,
				NotificationInviteCode: notificationInviteCode,
				MoverAlertsConfig:      marketradar.MoverAlertsConfig{Enabled: notificationEnabled},
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
		Example: "Start the Athena Market Radar service:\n  athena-market-radar",
	}
	command.Flags().StringVar(&cmdutil.LogFormat, "logformat", env.StringFromEnv(common.EnvLogFormat, "json"), "Set logging format")
	command.Flags().StringVar(&cmdutil.LogLevel, "loglevel", env.StringFromEnv(common.EnvLogLevel, "info"), "Set logging level")
	command.Flags().StringVar(&listenHost, "address", env.StringFromEnv("ATHENA_MARKET_RADAR_LISTEN_ADDRESS", common.DefaultAddressMarketRadar), "Listen address")
	command.Flags().IntVar(&listenPort, "port", env.ParseNumFromEnv("ATHENA_MARKET_RADAR_LISTEN_PORT", common.DefaultPortMarketRadar, 1, 65535), "Listen port")
	command.Flags().BoolVar(&notificationEnabled, "notification-enabled", env.ParseBoolFromEnv("ATHENA_MARKET_RADAR_NOTIFICATION_ENABLED", true), "Enable mover notifications")
	command.Flags().StringVar(&notificationServerAddress, "notification-server-address", env.StringFromEnv("ATHENA_MARKET_RADAR_NOTIFICATION_SERVER_ADDRESS", fmt.Sprintf("%s:%d", common.DefaultLocalGRPCHost, common.DefaultPortNotification)), "Notification service address")
	command.Flags().StringVar(&notificationInviteCode, "notification-invite-code", env.StringFromEnv("ATHENA_MARKET_RADAR_NOTIFICATION_INVITE_CODE", ""), "Polymarket invite code")
	command.AddCommand(cli.NewVersionCmd(cliName))
	return command
}
