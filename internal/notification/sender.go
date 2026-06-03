package notification

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"sync"
	"time"

	tgbot "github.com/go-telegram/bot"
	utiltelegram "github.com/useryege/athena/util/telegram"
)

type Sender interface {
	Send(ctx context.Context, request SendRequest) (string, error)
}

type TopicProvisioner interface {
	ProvisionTopics(ctx context.Context, configs []TopicConfig) error
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
	threadsMu    sync.RWMutex
	topicThreads map[string]int
}

func NewTelegramSender(client utiltelegram.Client, topicThreads map[string]int) (*TelegramSender, error) {
	if err := validateTopicThreads(topicThreads); err != nil {
		return nil, err
	}
	threads := make(map[string]int, len(topicThreads))
	for topic, threadID := range topicThreads {
		threads[NormalizeTopicKey(topic)] = threadID
	}
	return &TelegramSender{client: client, topicThreads: threads}, nil
}

func (s *TelegramSender) ProvisionTopics(ctx context.Context, configs []TopicConfig) error {
	if s.client == nil {
		return fmt.Errorf("telegram client is required")
	}
	configs, err := normalizeTopicConfigs(configs)
	if err != nil {
		return err
	}
	threads := make(map[string]int, len(configs))
	for _, config := range configs {
		topic, err := s.client.CreateForumTopic(ctx, utiltelegram.CreateForumTopicRequest{Name: config.Title})
		if err != nil {
			return fmt.Errorf("failed to create telegram topic %s: %w", config.Key, err)
		}
		if topic == nil || topic.MessageThreadID <= 0 {
			return fmt.Errorf("telegram topic %s returned invalid message thread id", config.Key)
		}
		threads[config.Key] = topic.MessageThreadID
	}
	s.threadsMu.Lock()
	s.topicThreads = threads
	s.threadsMu.Unlock()
	return nil
}

func (s *TelegramSender) Send(ctx context.Context, request SendRequest) (string, error) {
	if s.client == nil {
		return "", fmt.Errorf("telegram client is required")
	}
	topic := NormalizeTopicKey(request.Topic)
	s.threadsMu.RLock()
	threadID, ok := s.topicThreads[topic]
	s.threadsMu.RUnlock()
	if !ok || threadID <= 0 {
		return "", fmt.Errorf("telegram topic %s is not initialized", topic)
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
	for topic, threadID := range topicThreads {
		topic = NormalizeTopicKey(topic)
		if topic == "" {
			return fmt.Errorf("telegram topic key is required")
		}
		if threadID <= 0 {
			return fmt.Errorf("telegram message thread id is required for topic %s", topic)
		}
	}
	return nil
}
