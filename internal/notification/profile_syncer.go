package notification

import (
	"context"
	"fmt"

	log "github.com/sirupsen/logrus"
	utiltelegram "github.com/useryege/athena/util/telegram"
)

type TelegramProfileSyncer struct {
	client    utiltelegram.Client
	botConfig utiltelegram.BotProfileConfig
}

func NewTelegramProfileSyncer(client utiltelegram.Client, botConfig utiltelegram.BotProfileConfig) *TelegramProfileSyncer {
	return &TelegramProfileSyncer{client: client, botConfig: botConfig}
}

func (s *TelegramProfileSyncer) SyncProfile(ctx context.Context) (*utiltelegram.BotIdentity, error) {
	if s.client == nil {
		return nil, fmt.Errorf("telegram client is required")
	}
	botResult, err := s.client.EnsureBotProfile(ctx, s.botConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to ensure telegram bot profile: %w", err)
	}
	log.WithFields(log.Fields{
		"bot_id":                    botResult.Bot.ID,
		"bot_username":              botResult.Bot.Username,
		"name_updated":              botResult.NameUpdated,
		"description_updated":       botResult.DescriptionUpdated,
		"short_description_updated": botResult.ShortDescriptionUpdated,
		"profile_photo_updated":     botResult.ProfilePhotoUpdated,
	}).Info("telegram bot profile synchronized")
	return &botResult.Bot, nil
}
