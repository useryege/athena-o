package commands

import (
	"strings"
	"testing"

	"github.com/useryege/athena/internal/notification"
)

func TestTelegramTopicThreadsFromEnv(t *testing.T) {
	t.Setenv("ATHENA_NOTIFICATION_TELEGRAM_TOKEN_MESSAGE_THREAD_ID", "111")
	t.Setenv("ATHENA_NOTIFICATION_TELEGRAM_POLY_MESSAGE_THREAD_ID", "222")

	threads, err := telegramTopicThreadsFromEnv()
	if err != nil {
		t.Fatalf("telegramTopicThreadsFromEnv: %v", err)
	}
	if threads[notification.NotificationTopicToken] != 111 || threads[notification.NotificationTopicPoly] != 222 {
		t.Fatalf("threads = %#v, want token/poly ids", threads)
	}
}

func TestTelegramTopicThreadsFromEnvRequiresPositiveIntegers(t *testing.T) {
	t.Setenv("ATHENA_NOTIFICATION_TELEGRAM_TOKEN_MESSAGE_THREAD_ID", "111")
	t.Setenv("ATHENA_NOTIFICATION_TELEGRAM_POLY_MESSAGE_THREAD_ID", "0")

	_, err := telegramTopicThreadsFromEnv()
	if err == nil || !strings.Contains(err.Error(), "ATHENA_NOTIFICATION_TELEGRAM_POLY_MESSAGE_THREAD_ID") {
		t.Fatalf("telegramTopicThreadsFromEnv error = %v, want poly env error", err)
	}
}

func TestDefaultTelegramProfileConfigsLoadEmbeddedAssetsAndEnvOverrides(t *testing.T) {
	t.Setenv("ATHENA_NOTIFICATION_TELEGRAM_BOT_NAME", "ATHENA DEV")
	t.Setenv("ATHENA_NOTIFICATION_TELEGRAM_CHAT_TITLE", "ATHENA Dev Notifications")

	botConfig, err := defaultTelegramBotProfileConfig()
	if err != nil {
		t.Fatalf("defaultTelegramBotProfileConfig: %v", err)
	}
	if botConfig.Name != "ATHENA DEV" || botConfig.ProfilePhoto.Filename != defaultTelegramBotProfilePhotoPath || len(botConfig.ProfilePhoto.Data) == 0 {
		t.Fatalf("bot config = %#v, want override and embedded avatar", botConfig)
	}

	chatConfig, err := defaultTelegramChatProfileConfig()
	if err != nil {
		t.Fatalf("defaultTelegramChatProfileConfig: %v", err)
	}
	if chatConfig.Title != "ATHENA Dev Notifications" || chatConfig.Photo.Filename != defaultTelegramChatProfilePhotoPath || len(chatConfig.Photo.Data) == 0 {
		t.Fatalf("chat config = %#v, want override and embedded avatar", chatConfig)
	}
}
