package commands

import (
	"context"
	stderrors "errors"
	"fmt"
	"github.com/google/uuid"
	"net"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"

	cmdutil "github.com/useryege/athena/cmd/util"
	"github.com/useryege/athena/common"
	"github.com/useryege/athena/internal/notification"
	notificationapiclient "github.com/useryege/athena/internal/notification/apiclient"
	notificationstore "github.com/useryege/athena/internal/notification/store"
	"github.com/useryege/athena/util/cli"
	"github.com/useryege/athena/util/env"
	"github.com/useryege/athena/util/errors"
	utilio "github.com/useryege/athena/util/io"
	utiltelegram "github.com/useryege/athena/util/telegram"
	"github.com/useryege/athena/util/templates"
)

const cliName = "athena-notification"

const (
	defaultTelegramBotProfileName             = "ATHENA"
	defaultTelegramBotProfileShortDescription = "ATHENA notifications"
	defaultTelegramBotProfileDescription      = "ATHENA notification bot for account and system updates."
	telegramTestChatIDEnv                     = "ATHENA_NOTIFICATION_TEST_TELEGRAM_CHAT_ID"
	telegramProdChatIDEnv                     = "ATHENA_NOTIFICATION_PROD_TELEGRAM_CHAT_ID"
)

func NewCommand() *cobra.Command {
	var (
		listenHost           string
		listenPort           int
		workerConcurrency    int
		recoverStoppedSender string

		storeSrc func(context.Context) (*notificationstore.SQLStore, error)
	)

	command := &cobra.Command{
		Use:   cliName,
		Short: "Run the Athena Notification service",
		Long: "The Notification service manages system and account notification workloads. " +
			"ATHENA_NOTIFICATION_INTERNAL_AUTH_TOKEN must contain at least 32 bytes without whitespace and is required by every non-health RPC.",
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

			if recoverStoppedSender != "" {
				id, err := uuid.Parse(recoverStoppedSender)
				if err != nil {
					return fmt.Errorf("invalid stopped sender incarnation: %w", err)
				}
				log.WithField("incarnation", id).Warn("operator explicitly confirms the registered sender process has exited; recovering without Telegram calls")
				return store.ConfirmStoppedSender(ctx, id)
			}
			telegramClient, systemChatIDs, err := defaultTelegramClient()
			if err != nil {
				return err
			}

			profileConfig := defaultTelegramBotProfileConfig()
			sender := notification.NewTelegramSender(telegramClient, systemChatIDs)

			server, err := notification.NewServer(notification.ServerOpts{
				Store:             store,
				SiteURL:           env.StringFromEnv("ATHENA_URL", ""),
				Sender:            sender,
				ProfileSyncer:     notification.NewTelegramProfileSyncer(telegramClient, profileConfig),
				Poller:            notification.NewTelegramPoller(store, telegramClient),
				InternalAuthToken: env.StringFromEnv(notificationapiclient.InternalAuthTokenEnv, ""),
				WorkerConfig:      notification.WorkerConfig{Concurrency: workerConcurrency},
			})
			if err != nil {
				return err
			}
			notificationGRPC := server.CreateGRPC()

			lc := &net.ListenConfig{}
			listener, err := lc.Listen(ctx, "tcp", fmt.Sprintf("%s:%d", listenHost, listenPort))
			if err != nil {
				return err
			}

			defer listener.Close()
			if err := server.Start(ctx); err != nil {
				return err
			}

			sigCh := make(chan os.Signal, 1)
			signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
			defer signal.Stop(sigCh)
			served := make(chan error, 1)
			go func() { served <- notificationGRPC.Serve(listener) }()
			var runErr error
			select {
			case sig := <-sigCh:
				log.Printf("got signal %v, attempting graceful shutdown", sig)
			case runErr = <-server.Errors():
			case runErr = <-served:
				if stderrors.Is(runErr, grpc.ErrServerStopped) {
					runErr = nil
				}
			case <-ctx.Done():
				runErr = ctx.Err()
			}
			notificationGRPC.Stop()
			if err := server.Stop(); err != nil && runErr == nil {
				runErr = err
			}
			return runErr
		},
		Example: templates.Examples(`
			# Start the Athena Notification service
			$ athena-notification
		`),
	}

	command.Flags().StringVar(&cmdutil.LogFormat, "logformat", env.StringFromEnv(common.EnvLogFormat, "json"), "Set the logging format. One of: json|text")
	command.Flags().StringVar(&cmdutil.LogLevel, "loglevel", env.StringFromEnv(common.EnvLogLevel, "info"), "Set the logging level. One of: debug|info|warn|error")
	command.Flags().StringVar(&listenHost, "address", env.StringFromEnv("ATHENA_NOTIFICATION_LISTEN_ADDRESS", common.DefaultAddressNotification), "Listen on given address for incoming connections")
	command.Flags().IntVar(&listenPort, "port", common.DefaultPortNotification, "Listen on given port for incoming connections")
	command.Flags().IntVar(&workerConcurrency, "worker-concurrency", env.ParseNumFromEnv("ATHENA_NOTIFICATION_WORKER_CONCURRENCY", 12, 1, 12), "Maximum simultaneous Telegram chats")
	command.Flags().StringVar(&recoverStoppedSender, "recover-stopped-sender", "", "Confirm the registered process for this incarnation UUID has exited, recover its attempts without sending, then exit")

	storeSrc = notificationstore.NewSQLStoreSource()

	command.AddCommand(cli.NewVersionCmd(cliName))
	return command
}

func defaultTelegramClient() (utiltelegram.Client, map[string]string, error) {
	botToken := strings.TrimSpace(env.StringFromEnv("ATHENA_NOTIFICATION_TELEGRAM_BOT_TOKEN", ""))
	if botToken == "" {
		return nil, nil, fmt.Errorf("ATHENA_NOTIFICATION_TELEGRAM_BOT_TOKEN is required")
	}
	baseURL := env.StringFromEnv("ATHENA_NOTIFICATION_TELEGRAM_API_URL", utiltelegram.DefaultBaseURL)
	timeout := time.Duration(env.ParseNumFromEnv("ATHENA_NOTIFICATION_TELEGRAM_TIMEOUT_SECONDS", int(utiltelegram.DefaultTimeout/time.Second), 1, 300)) * time.Second
	chatIDs := make(map[string]string, 2)
	for _, item := range []struct {
		telegramChat string
		envName      string
	}{
		{telegramChat: notification.TelegramChatTest, envName: telegramTestChatIDEnv},
		{telegramChat: notification.TelegramChatProd, envName: telegramProdChatIDEnv},
	} {
		chatID := strings.TrimSpace(env.StringFromEnv(item.envName, ""))
		if chatID == "" {
			return nil, nil, fmt.Errorf("%s is required", item.envName)
		}
		chatIDs[item.telegramChat] = chatID
	}
	client, err := utiltelegram.NewClient(utiltelegram.Config{BotToken: botToken, BaseURL: baseURL, Timeout: timeout})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create telegram client: %w", err)
	}
	return client, chatIDs, nil
}

func defaultTelegramBotProfileConfig() utiltelegram.BotProfileConfig {
	return utiltelegram.BotProfileConfig{
		Name:             env.StringFromEnv("ATHENA_NOTIFICATION_TELEGRAM_BOT_NAME", defaultTelegramBotProfileName),
		ShortDescription: env.StringFromEnv("ATHENA_NOTIFICATION_TELEGRAM_BOT_SHORT_DESCRIPTION", defaultTelegramBotProfileShortDescription),
		Description:      env.StringFromEnv("ATHENA_NOTIFICATION_TELEGRAM_BOT_DESCRIPTION", defaultTelegramBotProfileDescription),
	}
}
