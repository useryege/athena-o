package notification

import (
	"context"
	"fmt"

	log "github.com/sirupsen/logrus"
	utiltelegram "github.com/useryege/athena/util/telegram"
)

type TelegramProfileSyncer struct {
	clients    map[string]utiltelegram.Client
	botConfig  utiltelegram.BotProfileConfig
	chatConfig utiltelegram.ChatProfileConfig
}

func NewTelegramProfileSyncer(clients map[string]utiltelegram.Client, botConfig utiltelegram.BotProfileConfig, chatConfig utiltelegram.ChatProfileConfig) *TelegramProfileSyncer {
	return &TelegramProfileSyncer{clients: copyTelegramClients(clients), botConfig: botConfig, chatConfig: chatConfig}
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

	for _, telegramChat := range supportedTelegramChats {
		client, err := telegramClientForChat(s.clients, telegramChat)
		if err != nil {
			return err
		}
		chatResult, err := client.EnsureChatProfile(ctx, s.chatConfig)
		if err != nil {
			return fmt.Errorf("failed to ensure %s telegram chat profile: %w", telegramChat, err)
		}
		log.WithFields(log.Fields{
			"telegram_chat":       telegramChat,
			"chat_id":             chatResult.ChatID,
			"chat_type":           chatResult.ChatType,
			"is_forum":            chatResult.IsForum,
			"title_updated":       chatResult.TitleUpdated,
			"description_updated": chatResult.DescriptionUpdated,
			"photo_updated":       chatResult.PhotoUpdated,
		}).Info("telegram chat profile synchronized")
	}
	return nil
}
