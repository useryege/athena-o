package notification

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/go-telegram/bot/models"
	"github.com/useryege/athena/internal/notification/delivery"
	utiltelegram "github.com/useryege/athena/util/telegram"
)

type Sender interface {
	SystemChatID(string) (int64, error)
	CreateSystemTopic(ctx context.Context, telegramChat string, label string) (int, error)
	Send(ctx context.Context, request SendRequest, started func(time.Time)) delivery.Outcome
}

type SendRequest struct {
	Format          string
	TelegramChatID  int64
	MessageThreadID int
	Text            string
}

type TelegramSender struct {
	client        utiltelegram.Client
	systemChatIDs map[string]string
}

func NewTelegramSender(client utiltelegram.Client, systemChatIDs map[string]string) *TelegramSender {
	return &TelegramSender{client: client, systemChatIDs: copySystemChatIDs(systemChatIDs)}
}

func (s *TelegramSender) CreateSystemTopic(ctx context.Context, telegramChat string, label string) (int, error) {
	chatID, err := s.systemChatID(telegramChat)
	if err != nil {
		return 0, err
	}
	topic, err := s.client.CreateForumTopic(ctx, utiltelegram.CreateForumTopicRequest{ChatID: chatID, Name: label})
	if err != nil {
		return 0, fmt.Errorf("failed to create %s telegram topic %q: %w", telegramChat, label, err)
	}
	if topic == nil || topic.MessageThreadID <= 0 {
		return 0, fmt.Errorf("%s telegram topic %q returned invalid message thread id", telegramChat, label)
	}
	return topic.MessageThreadID, nil
}

func (s *TelegramSender) Send(ctx context.Context, request SendRequest, started func(time.Time)) delivery.Outcome {
	if s.client == nil {
		return delivery.Outcome{Kind: "failed", Code: "client_unavailable"}
	}
	if request.TelegramChatID == 0 {
		return delivery.Outcome{Kind: "failed", Code: "invalid_recipient"}
	}
	var mode models.ParseMode
	switch request.Format {
	case "plain":
	case "html":
		mode = models.ParseModeHTML
	default:
		return delivery.Outcome{Kind: "failed", Code: "invalid_format"}
	}
	chatID := strconv.FormatInt(request.TelegramChatID, 10)
	resp, err := s.client.SendMessage(utiltelegram.WithSendStarted(utiltelegram.WithSendTimeout(ctx, 5*time.Second), started), utiltelegram.SendMessageRequest{
		ChatID: chatID, Text: request.Text, MessageThreadID: request.MessageThreadID, ParseMode: mode,
	})
	if err != nil {
		var sendErr *utiltelegram.SendError
		if errors.As(err, &sendErr) {
			return delivery.Outcome{Kind: sendErr.Kind, Code: sendErr.Code, RetryAfter: sendErr.RetryAfter}
		}
		return delivery.Outcome{Kind: "unknown", Code: "unclassified_send_error"}
	}
	if resp == nil || resp.MessageID <= 0 {
		return delivery.Outcome{Kind: "unknown", Code: "invalid_receipt"}
	}
	return delivery.Outcome{Kind: "sent", MessageID: strconv.Itoa(resp.MessageID)}
}

func (s *TelegramSender) systemChatID(telegramChat string) (string, error) {
	telegramChat = strings.TrimSpace(strings.ToLower(telegramChat))
	if telegramChat == "" {
		return "", errors.New("telegram_chat is required")
	}
	chatID := strings.TrimSpace(s.systemChatIDs[telegramChat])
	if chatID == "" {
		return "", fmt.Errorf("%s telegram chat id is required", telegramChat)
	}
	return chatID, nil
}

func copySystemChatIDs(chatIDs map[string]string) map[string]string {
	out := make(map[string]string, len(chatIDs))
	for key, chatID := range chatIDs {
		key = strings.TrimSpace(strings.ToLower(key))
		chatID = strings.TrimSpace(chatID)
		if key != "" && chatID != "" {
			out[key] = chatID
		}
	}
	return out
}

func (s *TelegramSender) SystemChatID(name string) (int64, error) {
	raw, err := s.systemChatID(name)
	if err != nil {
		return 0, err
	}
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || id >= 0 {
		return 0, fmt.Errorf("system Telegram chat %q requires a numeric group ID", name)
	}
	return id, nil
}
