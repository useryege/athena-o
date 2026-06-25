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
	CreateTopic(ctx context.Context, telegramChat string, label string) (int, error)
	Send(ctx context.Context, request SendRequest) (string, error)
}

type SendRequest struct {
	TelegramChat    string
	MessageThreadID int
	Text            string
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
	clients map[string]utiltelegram.Client
}

func NewTelegramSender(clients map[string]utiltelegram.Client) *TelegramSender {
	return &TelegramSender{clients: copyTelegramClients(clients)}
}

func (s *TelegramSender) CreateTopic(ctx context.Context, telegramChat string, label string) (int, error) {
	client, err := telegramClientForChat(s.clients, telegramChat)
	if err != nil {
		return 0, err
	}
	topic, err := client.CreateForumTopic(ctx, utiltelegram.CreateForumTopicRequest{Name: label})
	if err != nil {
		return 0, fmt.Errorf("failed to create %s telegram topic %q: %w", telegramChat, label, err)
	}
	if topic == nil || topic.MessageThreadID <= 0 {
		return 0, fmt.Errorf("%s telegram topic %q returned invalid message thread id", telegramChat, label)
	}
	return topic.MessageThreadID, nil
}

func (s *TelegramSender) Send(ctx context.Context, request SendRequest) (string, error) {
	client, err := telegramClientForChat(s.clients, request.TelegramChat)
	if err != nil {
		return "", err
	}
	if request.MessageThreadID <= 0 {
		return "", fmt.Errorf("telegram message thread id is required")
	}
	resp, err := client.SendMessage(ctx, utiltelegram.SendMessageRequest{
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

func copyTelegramClients(clients map[string]utiltelegram.Client) map[string]utiltelegram.Client {
	out := make(map[string]utiltelegram.Client, len(clients))
	for key, client := range clients {
		key = strings.TrimSpace(strings.ToLower(key))
		if key != "" {
			out[key] = client
		}
	}
	return out
}

func telegramClientForChat(clients map[string]utiltelegram.Client, telegramChat string) (utiltelegram.Client, error) {
	telegramChat = strings.TrimSpace(strings.ToLower(telegramChat))
	if telegramChat == "" {
		return nil, fmt.Errorf("telegram_chat is required")
	}
	client := clients[telegramChat]
	if client == nil {
		return nil, fmt.Errorf("%s telegram client is required", telegramChat)
	}
	return client, nil
}

func RetryAfterFromError(err error) (time.Duration, bool) {
	var rateLimitErr *RateLimitError
	if errors.As(err, &rateLimitErr) && rateLimitErr.RetryAfter > 0 {
		return rateLimitErr.RetryAfter, true
	}
	return 0, false
}
