package notification

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	tgbot "github.com/go-telegram/bot"
	utiltelegram "github.com/useryege/athena/util/telegram"
)

type Sender interface {
	CreateTopic(ctx context.Context, label string) (int, error)
	Send(ctx context.Context, request SendRequest) (string, error)
}

type SendRequest struct {
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
	client utiltelegram.Client
}

func NewTelegramSender(client utiltelegram.Client) *TelegramSender {
	return &TelegramSender{client: client}
}

func (s *TelegramSender) CreateTopic(ctx context.Context, label string) (int, error) {
	if s.client == nil {
		return 0, fmt.Errorf("telegram client is required")
	}
	topic, err := s.client.CreateForumTopic(ctx, utiltelegram.CreateForumTopicRequest{Name: label})
	if err != nil {
		return 0, fmt.Errorf("failed to create telegram topic %q: %w", label, err)
	}
	if topic == nil || topic.MessageThreadID <= 0 {
		return 0, fmt.Errorf("telegram topic %q returned invalid message thread id", label)
	}
	return topic.MessageThreadID, nil
}

func (s *TelegramSender) Send(ctx context.Context, request SendRequest) (string, error) {
	if s.client == nil {
		return "", fmt.Errorf("telegram client is required")
	}
	if request.MessageThreadID <= 0 {
		return "", fmt.Errorf("telegram message thread id is required")
	}
	resp, err := s.client.SendMessage(ctx, utiltelegram.SendMessageRequest{Text: request.Text, MessageThreadID: request.MessageThreadID})
	if err != nil {
		var rateLimitErr *tgbot.TooManyRequestsError
		if errors.As(err, &rateLimitErr) && rateLimitErr.RetryAfter > 0 {
			return "", &RateLimitError{RetryAfter: time.Duration(rateLimitErr.RetryAfter) * time.Second, Err: err}
		}
		return "", err
	}
	return strconv.Itoa(resp.MessageID), nil
}

func RetryAfterFromError(err error) (time.Duration, bool) {
	var rateLimitErr *RateLimitError
	if errors.As(err, &rateLimitErr) && rateLimitErr.RetryAfter > 0 {
		return rateLimitErr.RetryAfter, true
	}
	return 0, false
}
