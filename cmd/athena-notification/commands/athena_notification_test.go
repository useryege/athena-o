package commands

import "testing"

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
