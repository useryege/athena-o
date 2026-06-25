package notification

import (
	"context"
	"fmt"

	log "github.com/sirupsen/logrus"
	utiltelegram "github.com/useryege/athena/util/telegram"
)

type TelegramProfileSyncer struct {
	clients   map[string]utiltelegram.Client
	botConfig utiltelegram.BotProfileConfig
}

func NewTelegramProfileSyncer(clients map[string]utiltelegram.Client, botConfig utiltelegram.BotProfileConfig) *TelegramProfileSyncer {
	return &TelegramProfileSyncer{clients: copyTelegramClients(clients), botConfig: botConfig}
}

func (s *TelegramProfileSyncer) SyncProfile(ctx context.Context) error {
	botClient, err := telegramClientForChat(s.clients, TelegramChatTest)
	if err != nil {
		return err
	}
	botResult, err := botClient.EnsureBotProfile(ctx, s.botConfig)
	if err != nil {
		return fmt.Errorf("failed to ensure telegram bot profile: %w", err)
	}
	log.WithFields(log.Fields{
		"bot_id":                    botResult.Bot.ID,
		"bot_username":              botResult.Bot.Username,
		"name_updated":              botResult.NameUpdated,
		"description_updated":       botResult.DescriptionUpdated,
		"short_description_updated": botResult.ShortDescriptionUpdated,
		"profile_photo_updated":     botResult.ProfilePhotoUpdated,
	}).Info("telegram bot profile synchronized")
	return nil
}
