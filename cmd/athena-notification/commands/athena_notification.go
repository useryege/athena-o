package commands

import (
	"context"
	stderrors "errors"
	"fmt"
	"net"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"

	"github.com/useryege/athena/assets"
	cmdutil "github.com/useryege/athena/cmd/util"
	"github.com/useryege/athena/common"
	"github.com/useryege/athena/internal/notification"
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
	defaultTelegramBotProfileShortDescription = "ATHENA operational alerts"
	defaultTelegramBotProfileDescription      = "ATHENA notification bot for operational alerts and system updates."
	defaultTelegramBotProfilePhotoPath        = "telegram-bot-avatar.jpg"
	defaultTelegramChatProfileTitle           = "ATHENA Notifications"
	defaultTelegramChatProfileDescription     = "ATHENA notification group for operational alerts and system updates."
	defaultTelegramChatProfilePhotoPath       = "telegram-group-avatar.jpg"
	telegramInitAvatarsEnv                    = "ATHENA_NOTIFICATION_TELEGRAM_INIT_AVATARS"
)

func NewCommand() *cobra.Command {
	var (
		listenHost string
		listenPort int

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

			initAvatars := telegramInitAvatarsFromEnv()
			profileConfig, err := defaultTelegramBotProfileConfig(initAvatars)
			if err != nil {
				return err
			}
			chatProfileConfig, err := defaultTelegramChatProfileConfig(initAvatars)
			if err != nil {
				return err
			}
			topicThreads, err := telegramTopicThreadsFromEnv()
			if err != nil {
				return err
			}
			sender, err := notification.NewTelegramSender(telegramClient, topicThreads)
			if err != nil {
				return err
			}

			server, err := notification.NewServer(notification.ServerOpts{
				Store:         store,
				Sender:        sender,
				ProfileSyncer: notification.NewTelegramProfileSyncer(telegramClient, profileConfig, chatProfileConfig),
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

	command.Flags().StringVar(&cmdutil.LogFormat, "logformat", env.StringFromEnv("ATHENA_NOTIFICATION_LOGFORMAT", "json"), "Set the logging format. One of: json|text")
	command.Flags().StringVar(&cmdutil.LogLevel, "loglevel", env.StringFromEnv("ATHENA_NOTIFICATION_LOGLEVEL", "info"), "Set the logging level. One of: debug|info|warn|error")
	command.Flags().StringVar(&listenHost, "address", env.StringFromEnv("ATHENA_NOTIFICATION_LISTEN_ADDRESS", common.DefaultAddressNotification), "Listen on given address for incoming connections")
	command.Flags().IntVar(&listenPort, "port", common.DefaultPortNotification, "Listen on given port for incoming connections")

	storeSrc = notificationstore.NewSQLStoreSource()

	command.AddCommand(cli.NewVersionCmd(cliName))
	return command
}

func defaultTelegramBotProfileConfig(initAvatar bool) (utiltelegram.BotProfileConfig, error) {
	config := utiltelegram.BotProfileConfig{
		Name:             env.StringFromEnv("ATHENA_NOTIFICATION_TELEGRAM_BOT_NAME", defaultTelegramBotProfileName),
		ShortDescription: env.StringFromEnv("ATHENA_NOTIFICATION_TELEGRAM_BOT_SHORT_DESCRIPTION", defaultTelegramBotProfileShortDescription),
		Description:      env.StringFromEnv("ATHENA_NOTIFICATION_TELEGRAM_BOT_DESCRIPTION", defaultTelegramBotProfileDescription),
	}
	if initAvatar {
		photo, err := assets.Embedded.ReadFile(defaultTelegramBotProfilePhotoPath)
		if err != nil {
			return utiltelegram.BotProfileConfig{}, fmt.Errorf("failed to read default telegram bot avatar: %w", err)
		}
		config.ProfilePhoto = utiltelegram.SetMyProfilePhotoRequest{
			Filename: defaultTelegramBotProfilePhotoPath,
			Data:     photo,
		}
	}
	return config, nil
}

func telegramInitAvatarsFromEnv() bool {
	return env.ParseBoolFromEnv(telegramInitAvatarsEnv, false)
}

func defaultTelegramChatProfileConfig(initAvatar bool) (utiltelegram.ChatProfileConfig, error) {
	config := utiltelegram.ChatProfileConfig{
		Title:       env.StringFromEnv("ATHENA_NOTIFICATION_TELEGRAM_CHAT_TITLE", defaultTelegramChatProfileTitle),
		Description: env.StringFromEnv("ATHENA_NOTIFICATION_TELEGRAM_CHAT_DESCRIPTION", defaultTelegramChatProfileDescription),
	}
	if initAvatar {
		photo, err := assets.Embedded.ReadFile(defaultTelegramChatProfilePhotoPath)
		if err != nil {
			return utiltelegram.ChatProfileConfig{}, fmt.Errorf("failed to read default telegram chat avatar: %w", err)
		}
		config.Photo = utiltelegram.SetChatPhotoRequest{
			Filename: defaultTelegramChatProfilePhotoPath,
			Data:     photo,
		}
	}
	return config, nil
}

func telegramTopicThreadsFromEnv() (map[string]int, error) {
	tokenThreadID, err := requiredPositiveIntEnv("ATHENA_NOTIFICATION_TELEGRAM_TOKEN_MESSAGE_THREAD_ID")
	if err != nil {
		return nil, err
	}
	polyThreadID, err := requiredPositiveIntEnv("ATHENA_NOTIFICATION_TELEGRAM_POLY_MESSAGE_THREAD_ID")
	if err != nil {
		return nil, err
	}
	return map[string]int{
		notification.NotificationTopicToken: tokenThreadID,
		notification.NotificationTopicPoly:  polyThreadID,
	}, nil
}

func requiredPositiveIntEnv(name string) (int, error) {
	raw, ok := os.LookupEnv(name)
	if !ok || strings.TrimSpace(raw) == "" {
		return 0, fmt.Errorf("%s is required", name)
	}
	value, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil || value <= 0 {
		return 0, fmt.Errorf("%s must be a positive integer", name)
	}
	return value, nil
}
