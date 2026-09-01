package notification

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	tgbot "github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	utiltelegram "github.com/useryege/athena/util/telegram"
)

type Sender interface {
	CreateSystemTopic(ctx context.Context, telegramChat string, label string) (int, error)
	Send(ctx context.Context, request SendRequest) (string, error)
}

type SendRequest struct {
	SystemTelegramChat string
	TelegramChatID     int64
	MessageThreadID    int
	Text               string
}

type RateLimitError struct {
	RetryAfter time.Duration
	Err        error
}

func (e *RateLimitError) Error() string {
	if e.Err == nil {
		return fmt.Sprintf("telegram rate limited: retry_after %s", e.RetryAfter)
	}
	return e.Err.Error()
}

func (e *RateLimitError) Unwrap() error {
	return e.Err
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

func (s *TelegramSender) Send(ctx context.Context, request SendRequest) (string, error) {
	if s.client == nil {
		return "", errors.New("telegram client is required")
	}
	chatID := ""
	if request.TelegramChatID > 0 {
		chatID = strconv.FormatInt(request.TelegramChatID, 10)
	} else {
		var err error
		chatID, err = s.systemChatID(request.SystemTelegramChat)
		if err != nil {
			return "", err
		}
		if request.MessageThreadID <= 0 {
			return "", errors.New("system telegram message thread id is required")
		}
	}
	resp, err := s.client.SendMessage(ctx, utiltelegram.SendMessageRequest{
		ChatID:          chatID,
		Text:            request.Text,
		MessageThreadID: request.MessageThreadID,
		ParseMode:       models.ParseModeHTML,
	})
	if err != nil {
		var rateLimitErr *tgbot.TooManyRequestsError
		if errors.As(err, &rateLimitErr) && rateLimitErr.RetryAfter > 0 {
			return "", &RateLimitError{RetryAfter: time.Duration(rateLimitErr.RetryAfter) * time.Second, Err: err}
		}
		return "", err
	}
	return strconv.Itoa(resp.MessageID), nil
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

func RetryAfterFromError(err error) (time.Duration, bool) {
	var rateLimitErr *RateLimitError
	if errors.As(err, &rateLimitErr) && rateLimitErr.RetryAfter > 0 {
		return rateLimitErr.RetryAfter, true
	}
	return 0, false
}

func IsTelegramRecipientUnreachable(err error) bool {
	if errors.Is(err, utiltelegram.ErrRecipientUnreachable) ||
		errors.Is(err, tgbot.ErrorForbidden) || errors.Is(err, tgbot.ErrorNotFound) {
		return true
	}
	message := strings.ToLower(err.Error())
	return strings.Contains(message, "chat not found") ||
		strings.Contains(message, "bot was blocked by the user") ||
		strings.Contains(message, "user is deactivated")
}
