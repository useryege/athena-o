package commands

import "testing"

func TestDefaultTelegramBotProfileConfigUsesEmbeddedAvatarAndDefaults(t *testing.T) {
	t.Setenv("ATHENA_NOTIFICATION_TELEGRAM_BOT_NAME", "")
	t.Setenv("ATHENA_NOTIFICATION_TELEGRAM_BOT_SHORT_DESCRIPTION", "")
	t.Setenv("ATHENA_NOTIFICATION_TELEGRAM_BOT_DESCRIPTION", "")

	config, err := defaultTelegramBotProfileConfig()
	if err != nil {
		t.Fatalf("defaultTelegramBotProfileConfig: %v", err)
	}
	if config.Name != defaultTelegramBotProfileName {
		t.Fatalf("Name = %q, want %q", config.Name, defaultTelegramBotProfileName)
	}
	if config.ShortDescription != defaultTelegramBotProfileShortDescription {
		t.Fatalf("ShortDescription = %q, want %q", config.ShortDescription, defaultTelegramBotProfileShortDescription)
	}
	if config.Description != defaultTelegramBotProfileDescription {
		t.Fatalf("Description = %q, want %q", config.Description, defaultTelegramBotProfileDescription)
	}
	if config.ProfilePhoto.Filename != defaultTelegramBotProfilePhotoPath {
		t.Fatalf("ProfilePhoto filename = %q, want %q", config.ProfilePhoto.Filename, defaultTelegramBotProfilePhotoPath)
	}
	if len(config.ProfilePhoto.Data) == 0 {
		t.Fatal("ProfilePhoto data is empty")
	}
}

func TestDefaultTelegramBotProfileConfigUsesEnvOverrides(t *testing.T) {
	t.Setenv("ATHENA_NOTIFICATION_TELEGRAM_BOT_NAME", "ATHENA Dev")
	t.Setenv("ATHENA_NOTIFICATION_TELEGRAM_BOT_SHORT_DESCRIPTION", "Dev alerts")
	t.Setenv("ATHENA_NOTIFICATION_TELEGRAM_BOT_DESCRIPTION", "Dev notification bot")

	config, err := defaultTelegramBotProfileConfig()
	if err != nil {
		t.Fatalf("defaultTelegramBotProfileConfig: %v", err)
	}
	if config.Name != "ATHENA Dev" {
		t.Fatalf("Name = %q", config.Name)
	}
	if config.ShortDescription != "Dev alerts" {
		t.Fatalf("ShortDescription = %q", config.ShortDescription)
	}
	if config.Description != "Dev notification bot" {
		t.Fatalf("Description = %q", config.Description)
	}
}

func TestDefaultTelegramChatProfileConfigUsesEmbeddedAvatarAndDefaults(t *testing.T) {
	t.Setenv("ATHENA_NOTIFICATION_TELEGRAM_CHAT_TITLE", "")
	t.Setenv("ATHENA_NOTIFICATION_TELEGRAM_CHAT_DESCRIPTION", "")

	config, err := defaultTelegramChatProfileConfig()
	if err != nil {
		t.Fatalf("defaultTelegramChatProfileConfig: %v", err)
	}
	if config.Title != defaultTelegramChatProfileTitle {
		t.Fatalf("Title = %q, want %q", config.Title, defaultTelegramChatProfileTitle)
	}
	if config.Description != defaultTelegramChatProfileDescription {
		t.Fatalf("Description = %q, want %q", config.Description, defaultTelegramChatProfileDescription)
	}
	if config.Photo.Filename != defaultTelegramChatProfilePhotoPath {
		t.Fatalf("Photo filename = %q, want %q", config.Photo.Filename, defaultTelegramChatProfilePhotoPath)
	}
	if len(config.Photo.Data) == 0 {
		t.Fatal("Photo data is empty")
	}
}

func TestDefaultTelegramChatProfileConfigUsesEnvOverrides(t *testing.T) {
	t.Setenv("ATHENA_NOTIFICATION_TELEGRAM_CHAT_TITLE", "ATHENA Dev Notifications")
	t.Setenv("ATHENA_NOTIFICATION_TELEGRAM_CHAT_DESCRIPTION", "Dev notification group")

	config, err := defaultTelegramChatProfileConfig()
	if err != nil {
		t.Fatalf("defaultTelegramChatProfileConfig: %v", err)
	}
	if config.Title != "ATHENA Dev Notifications" {
		t.Fatalf("Title = %q", config.Title)
	}
	if config.Description != "Dev notification group" {
		t.Fatalf("Description = %q", config.Description)
	}
}
