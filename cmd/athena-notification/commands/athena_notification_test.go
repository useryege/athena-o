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

func TestTelegramInitAvatarsFromEnvDefaultsToFalse(t *testing.T) {
	t.Setenv(telegramInitAvatarsEnv, "")
	if telegramInitAvatarsFromEnv() {
		t.Fatalf("telegramInitAvatarsFromEnv = true, want false by default")
	}

	t.Setenv(telegramInitAvatarsEnv, "true")
	if !telegramInitAvatarsFromEnv() {
		t.Fatalf("telegramInitAvatarsFromEnv = false, want true from env")
	}
}

func TestDefaultTelegramProfileConfigsSkipEmbeddedAssetsByDefault(t *testing.T) {
	t.Setenv("ATHENA_NOTIFICATION_TELEGRAM_BOT_NAME", "ATHENA DEV")
	t.Setenv("ATHENA_NOTIFICATION_TELEGRAM_CHAT_TITLE", "ATHENA Dev Notifications")

	botConfig, err := defaultTelegramBotProfileConfig(false)
	if err != nil {
		t.Fatalf("defaultTelegramBotProfileConfig: %v", err)
	}
	if botConfig.Name != "ATHENA DEV" || botConfig.ProfilePhoto.Filename != "" || len(botConfig.ProfilePhoto.Data) != 0 {
		t.Fatalf("bot config = %#v, want override and no embedded avatar", botConfig)
	}

	chatConfig, err := defaultTelegramChatProfileConfig(false)
	if err != nil {
		t.Fatalf("defaultTelegramChatProfileConfig: %v", err)
	}
	if chatConfig.Title != "ATHENA Dev Notifications" || chatConfig.Photo.Filename != "" || len(chatConfig.Photo.Data) != 0 {
		t.Fatalf("chat config = %#v, want override and no embedded avatar", chatConfig)
	}
}

func TestDefaultTelegramProfileConfigsLoadEmbeddedAssetsWhenAvatarInitEnabled(t *testing.T) {
	botConfig, err := defaultTelegramBotProfileConfig(true)
	if err != nil {
		t.Fatalf("defaultTelegramBotProfileConfig: %v", err)
	}
	if botConfig.ProfilePhoto.Filename != defaultTelegramBotProfilePhotoPath || len(botConfig.ProfilePhoto.Data) == 0 {
		t.Fatalf("bot config = %#v, want embedded avatar", botConfig)
	}

	chatConfig, err := defaultTelegramChatProfileConfig(true)
	if err != nil {
		t.Fatalf("defaultTelegramChatProfileConfig: %v", err)
	}
	if chatConfig.Photo.Filename != defaultTelegramChatProfilePhotoPath || len(chatConfig.Photo.Data) == 0 {
		t.Fatalf("chat config = %#v, want embedded avatar", chatConfig)
	}
}
