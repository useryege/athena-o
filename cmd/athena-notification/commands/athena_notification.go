package commands

import (
	"context"
	stderrors "errors"
	"fmt"
	"net"
	"os"
	"os/signal"
	"strings"
	"sync"
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
		listenHost         string
		listenPort         int
		workerSendInterval time.Duration
		workerPollInterval time.Duration
		workerBatchSize    int
		workerMaxAttempts  int
		workerLockTimeout  time.Duration

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

			telegramClient, systemChatIDs, err := defaultTelegramClient()
			if err != nil {
				return err
			}

			profileConfig := defaultTelegramBotProfileConfig()
			sender := notification.NewTelegramSender(telegramClient, systemChatIDs)

			server, err := notification.NewServer(notification.ServerOpts{
				Store:             store,
				Sender:            sender,
				ProfileSyncer:     notification.NewTelegramProfileSyncer(telegramClient, profileConfig),
				Poller:            notification.NewTelegramPoller(store, telegramClient),
				InternalAuthToken: env.StringFromEnv(notificationapiclient.InternalAuthTokenEnv, ""),
				WorkerConfig: notification.WorkerConfig{
					SendInterval: workerSendInterval,
					PollInterval: workerPollInterval,
					BatchSize:    workerBatchSize,
					MaxAttempts:  workerMaxAttempts,
					LockTimeout:  workerLockTimeout,
				},
			})
			if err != nil {
				return err
			}
			notificationGRPC := server.CreateGRPC()

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

	command.Flags().StringVar(&cmdutil.LogFormat, "logformat", env.StringFromEnv(common.EnvLogFormat, "json"), "Set the logging format. One of: json|text")
	command.Flags().StringVar(&cmdutil.LogLevel, "loglevel", env.StringFromEnv(common.EnvLogLevel, "info"), "Set the logging level. One of: debug|info|warn|error")
	command.Flags().StringVar(&listenHost, "address", env.StringFromEnv("ATHENA_NOTIFICATION_LISTEN_ADDRESS", common.DefaultAddressNotification), "Listen on given address for incoming connections")
	command.Flags().IntVar(&listenPort, "port", common.DefaultPortNotification, "Listen on given port for incoming connections")
	command.Flags().DurationVar(&workerSendInterval, "worker-send-interval", env.ParseDurationFromEnv("ATHENA_NOTIFICATION_WORKER_SEND_INTERVAL", notification.DefaultWorkerConfig().SendInterval, time.Millisecond, time.Minute), "Delay between Telegram notification sends")
	command.Flags().DurationVar(&workerPollInterval, "worker-poll-interval", env.ParseDurationFromEnv("ATHENA_NOTIFICATION_WORKER_POLL_INTERVAL", notification.DefaultWorkerConfig().PollInterval, time.Millisecond, time.Minute), "Delay between empty notification queue polls")
	command.Flags().IntVar(&workerBatchSize, "worker-batch-size", env.ParseNumFromEnv("ATHENA_NOTIFICATION_WORKER_BATCH_SIZE", notification.DefaultWorkerConfig().BatchSize, 1, 100), "Maximum notification deliveries claimed per worker poll")
	command.Flags().IntVar(&workerMaxAttempts, "worker-max-attempts", env.ParseNumFromEnv("ATHENA_NOTIFICATION_WORKER_MAX_ATTEMPTS", notification.DefaultWorkerConfig().MaxAttempts, 1, 100), "Maximum Telegram send attempts before a notification delivery fails")
	command.Flags().DurationVar(&workerLockTimeout, "worker-lock-timeout", env.ParseDurationFromEnv("ATHENA_NOTIFICATION_WORKER_LOCK_TIMEOUT", notification.DefaultWorkerConfig().LockTimeout, time.Second, time.Hour), "Duration after which an in-flight notification delivery lock can be reclaimed")

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
