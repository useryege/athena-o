package commands

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	cmdutil "github.com/useryege/athena/cmd/util"
	"github.com/useryege/athena/common"
	notificationapiclient "github.com/useryege/athena/internal/notification/apiclient"
	"github.com/useryege/athena/internal/sportslive"
	sportslivestore "github.com/useryege/athena/internal/sportslive/store"
	"github.com/useryege/athena/util/cli"
	"github.com/useryege/athena/util/env"
	utilio "github.com/useryege/athena/util/io"
)

const cliName = "athena-sports-live"

func NewCommand() *cobra.Command {
	var listenHost string
	var listenPort int
	var notificationEnabled bool
	var notificationServerAddress string
	var notificationInviteCode string
	var priceAlertCooldown time.Duration
	var storeSource func(context.Context) (*sportslivestore.SQLStore, error)

	command := &cobra.Command{
		Use:               cliName,
		Short:             "Run the Athena Sports Live service",
		Long:              "Sports Live owns live Polymarket sports synchronization, price history, and alerts.",
		DisableAutoGenTag: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			common.GetVersion().LogStartupInfo("Athena Sports Live", map[string]any{"port": listenPort})
			cli.SetLogFormat(cmdutil.LogFormat)
			cli.SetLogLevel(cmdutil.LogLevel)
			store, err := storeSource(cmd.Context())
			if err != nil {
				return err
			}
			defer utilio.Close(store)
			var notificationClientset notificationapiclient.Clientset
			if notificationEnabled {
				notificationClientset = notificationapiclient.NewNotificationClientset(notificationServerAddress)
			}
			server, err := sportslive.NewServer(sportslive.ServerOpts{
				Store:                  store,
				NotificationClientset:  notificationClientset,
				NotificationInviteCode: notificationInviteCode,
				PriceAlertsConfig: sportslive.SportsLivePriceAlertsConfig{
					Enabled:  notificationEnabled,
					Cooldown: priceAlertCooldown,
				},
				ScoreAlertsConfig: sportslive.SportsLiveScoreAlertsConfig{Enabled: notificationEnabled},
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
		Example: "Start the Athena Sports Live service:\n  athena-sports-live",
	}
	command.Flags().StringVar(&cmdutil.LogFormat, "logformat", env.StringFromEnv(common.EnvLogFormat, "json"), "Set logging format")
	command.Flags().StringVar(&cmdutil.LogLevel, "loglevel", env.StringFromEnv(common.EnvLogLevel, "info"), "Set logging level")
	command.Flags().StringVar(&listenHost, "address", env.StringFromEnv("ATHENA_SPORTS_LIVE_LISTEN_ADDRESS", common.DefaultAddressSportsLive), "Listen address")
	command.Flags().IntVar(&listenPort, "port", env.ParseNumFromEnv("ATHENA_SPORTS_LIVE_LISTEN_PORT", common.DefaultPortSportsLive, 1, 65535), "Listen port")
	command.Flags().BoolVar(&notificationEnabled, "notification-enabled", env.ParseBoolFromEnv("ATHENA_SPORTS_LIVE_NOTIFICATION_ENABLED", true), "Enable Sports Live notifications")
	command.Flags().StringVar(&notificationServerAddress, "notification-server-address", env.StringFromEnv("ATHENA_SPORTS_LIVE_NOTIFICATION_SERVER_ADDRESS", fmt.Sprintf("localhost:%d", common.DefaultPortNotification)), "Notification service address")
	command.Flags().StringVar(&notificationInviteCode, "notification-invite-code", env.StringFromEnv("ATHENA_SPORTS_LIVE_NOTIFICATION_INVITE_CODE", ""), "Polymarket invite code")
	command.Flags().DurationVar(&priceAlertCooldown, "price-alert-cooldown", env.ParseDurationFromEnv("ATHENA_SPORTS_LIVE_PRICE_ALERT_COOLDOWN", 15*time.Minute, time.Second, 24*time.Hour), "Price alert cooldown")
	storeSource = sportslivestore.NewSQLStoreSource()
	command.AddCommand(cli.NewVersionCmd(cliName))
	return command
}
