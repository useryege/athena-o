package telegram

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	tgbot "github.com/go-telegram/bot"
)

const (
	DefaultBaseURL = "https://api.telegram.org"
	DefaultTimeout = 10 * time.Second
)

type Client interface {
	SendMessage(ctx context.Context, request SendMessageRequest) (*SendMessageResponse, error)
}

type Config struct {
	BotToken string
	ChatID   string
	BaseURL  string
	Timeout  time.Duration
}

type SendMessageRequest struct {
	Text string
}

type SendMessageResponse struct {
	MessageID int
}

type clientImpl struct {
	config Config
	bot    *tgbot.Bot
}

var _ Client = (*clientImpl)(nil)

func NewClient(config Config) (Client, error) {
	config = config.withDefaults()
	if strings.TrimSpace(config.BotToken) == "" {
		return nil, errors.New("telegram bot token is required")
	}
	if strings.TrimSpace(config.ChatID) == "" {
		return nil, errors.New("telegram chat id is required")
	}
	if _, err := url.ParseRequestURI(config.BaseURL); err != nil {
		return nil, fmt.Errorf("invalid telegram base url: %w", err)
	}

	botClient, err := tgbot.New(
		config.BotToken,
		tgbot.WithServerURL(config.BaseURL),
		tgbot.WithHTTPClient(config.Timeout, &http.Client{Timeout: config.Timeout}),
		tgbot.WithSkipGetMe(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create telegram bot client: %w", err)
	}

	return &clientImpl{
		config: config,
		bot:    botClient,
	}, nil
}

func (c Config) WithDefaults() Config {
	return c.withDefaults()
}

func (c Config) withDefaults() Config {
	c.BotToken = strings.TrimSpace(c.BotToken)
	c.ChatID = strings.TrimSpace(c.ChatID)
	c.BaseURL = strings.TrimRight(strings.TrimSpace(c.BaseURL), "/")
	if c.BaseURL == "" {
		c.BaseURL = DefaultBaseURL
	}
	if c.Timeout <= 0 {
		c.Timeout = DefaultTimeout
	}
	return c
}

func (c *clientImpl) SendMessage(ctx context.Context, request SendMessageRequest) (*SendMessageResponse, error) {
	text := strings.TrimSpace(request.Text)
	if text == "" {
		return nil, errors.New("telegram message text is required")
	}

	message, err := c.bot.SendMessage(ctx, &tgbot.SendMessageParams{
		ChatID: c.config.ChatID,
		Text:   text,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to send telegram message: %w", err)
	}
	return &SendMessageResponse{MessageID: message.ID}, nil
}
