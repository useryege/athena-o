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
	Send(ctx context.Context, request SendRequest) (string, error)
}

type SendRequest struct {
	Topic string
	Text  string
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
	client       utiltelegram.Client
	topicThreads map[string]int
}

func NewTelegramSender(client utiltelegram.Client, topicThreads map[string]int) (*TelegramSender, error) {
	if err := validateTopicThreads(topicThreads); err != nil {
		return nil, err
	}
	threads := make(map[string]int, len(topicThreads))
	for topic, threadID := range topicThreads {
		threads[topic] = threadID
	}
	return &TelegramSender{client: client, topicThreads: threads}, nil
}

func (s *TelegramSender) Send(ctx context.Context, request SendRequest) (string, error) {
	if s.client == nil {
		return "", fmt.Errorf("telegram client is required")
	}
	threadID, ok := s.topicThreads[request.Topic]
	if !ok || threadID <= 0 {
		return "", fmt.Errorf("telegram message thread id is not configured for topic %s", request.Topic)
	}
	resp, err := s.client.SendMessage(ctx, utiltelegram.SendMessageRequest{Text: request.Text, MessageThreadID: threadID})
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

func validateTopicThreads(topicThreads map[string]int) error {
	for _, topic := range []string{notificationTopicToken, notificationTopicPoly} {
		if topicThreads[topic] <= 0 {
			return fmt.Errorf("telegram message thread id is required for topic %s", topic)
		}
	}
	return nil
}
