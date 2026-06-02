package notification

import (
	"context"
	"errors"
	"fmt"

	log "github.com/sirupsen/logrus"
	utiltelegram "github.com/useryege/athena/util/telegram"
)

type TelegramProfileSyncer struct {
	client     utiltelegram.Client
	botConfig  utiltelegram.BotProfileConfig
	chatConfig utiltelegram.ChatProfileConfig
}

func NewTelegramProfileSyncer(client utiltelegram.Client, botConfig utiltelegram.BotProfileConfig, chatConfig utiltelegram.ChatProfileConfig) *TelegramProfileSyncer {
	return &TelegramProfileSyncer{client: client, botConfig: botConfig, chatConfig: chatConfig}
}

func (s *TelegramProfileSyncer) SyncProfile(ctx context.Context) error {
	if s.client == nil {
		return errors.New("telegram client is required")
	}
	botResult, err := s.client.EnsureBotProfile(ctx, s.botConfig)
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

	chatResult, err := s.client.EnsureChatProfile(ctx, s.chatConfig)
	if err != nil {
		return fmt.Errorf("failed to ensure telegram chat profile: %w", err)
	}
	log.WithFields(log.Fields{
		"chat_id":             chatResult.ChatID,
		"chat_type":           chatResult.ChatType,
		"is_forum":            chatResult.IsForum,
		"title_updated":       chatResult.TitleUpdated,
		"description_updated": chatResult.DescriptionUpdated,
		"photo_updated":       chatResult.PhotoUpdated,
	}).Info("telegram chat profile synchronized")
	return nil
}
