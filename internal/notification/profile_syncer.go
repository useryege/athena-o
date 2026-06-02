package notification

import (
	"context"
	"errors"
	"fmt"

	log "github.com/sirupsen/logrus"
	utiltelegram "github.com/useryege/athena/util/telegram"
)

type TelegramProfileSyncer struct {
	client utiltelegram.Client
	config utiltelegram.BotProfileConfig
}

func NewTelegramProfileSyncer(client utiltelegram.Client, config utiltelegram.BotProfileConfig) *TelegramProfileSyncer {
	return &TelegramProfileSyncer{client: client, config: config}
}

func (s *TelegramProfileSyncer) SyncProfile(ctx context.Context) error {
	if s.client == nil {
		return errors.New("telegram client is required")
	}
	result, err := s.client.EnsureBotProfile(ctx, s.config)
	if err != nil {
		return fmt.Errorf("failed to ensure telegram bot profile: %w", err)
	}
	log.WithFields(log.Fields{
		"bot_id":                    result.Bot.ID,
		"bot_username":              result.Bot.Username,
		"name_updated":              result.NameUpdated,
		"description_updated":       result.DescriptionUpdated,
		"short_description_updated": result.ShortDescriptionUpdated,
		"profile_photo_updated":     result.ProfilePhotoUpdated,
	}).Info("telegram bot profile synchronized")
	return nil
}
