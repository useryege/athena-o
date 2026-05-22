package deepseek

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const (
	DefaultBaseURL     = "https://api.deepseek.com"
	DefaultModel       = "deepseek-v4-flash"
	DefaultTimeout     = 60 * time.Second
	DefaultMaxTokens   = 4096
	DefaultTemperature = 0.2

	errorBodyLimit = 4096
)

type Client interface {
	CreateChatCompletion(ctx context.Context, request ChatCompletionRequest) (*ChatCompletionResponse, error)
}

type Config struct {
	BaseURL     string
	APIKey      string
	Model       string
	Timeout     time.Duration
	MaxTokens   int
	Temperature float64
}

type ChatCompletionRequest struct {
	Model       string        `json:"model,omitempty"`
	Messages    []ChatMessage `json:"messages"`
	MaxTokens   int           `json:"max_tokens,omitempty"`
	Temperature *float64      `json:"temperature,omitempty"`
}

type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ChatCompletionResponse struct {
	ID      string `json:"id,omitempty"`
	Model   string `json:"model,omitempty"`
	Content string `json:"content"`
}

type clientImpl struct {
	config Config
	client *http.Client
}

func NewClient(config Config) (Client, error) {
	config = config.withDefaults()
	if strings.TrimSpace(config.APIKey) == "" {
		return nil, errors.New("deepseek api key is required")
	}
	return &clientImpl{
		config: config,
		client: &http.Client{Timeout: config.Timeout},
	}, nil
}

func (c Config) WithDefaults() Config {
	return c.withDefaults()
}

func (c Config) withDefaults() Config {
	c.BaseURL = strings.TrimSpace(c.BaseURL)
	if c.BaseURL == "" {
		c.BaseURL = DefaultBaseURL
	}
	c.Model = strings.TrimSpace(c.Model)
	if c.Model == "" {
		c.Model = DefaultModel
	}
	if c.Timeout <= 0 {
		c.Timeout = DefaultTimeout
	}
	if c.MaxTokens <= 0 {
		c.MaxTokens = DefaultMaxTokens
	}
	if c.Temperature == 0 {
		c.Temperature = DefaultTemperature
	}
	return c
}

func (c *clientImpl) CreateChatCompletion(ctx context.Context, request ChatCompletionRequest) (*ChatCompletionResponse, error) {
	request = c.applyRequestDefaults(request)
	if len(request.Messages) == 0 {
		return nil, errors.New("deepseek chat completion messages are required")
	}

	body, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to encode deepseek chat completion request: %w", err)
	}

	endpoint := strings.TrimRight(c.config.BaseURL, "/") + "/chat/completions"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create deepseek chat completion request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.config.APIKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send deepseek chat completion request: %w", err)
	}
	defer resp.Body.Close()

	if err := ensureHTTPSuccess(resp); err != nil {
		return nil, err
	}

	var raw chatCompletionResponseBody
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, fmt.Errorf("failed to decode deepseek chat completion response: %w", err)
	}
	if len(raw.Choices) == 0 {
		return nil, errors.New("deepseek chat completion response has no choices")
	}
	content := strings.TrimSpace(raw.Choices[0].Message.Content)
	if content == "" {
		return nil, errors.New("deepseek chat completion response content is empty")
	}
	return &ChatCompletionResponse{
		ID:      raw.ID,
		Model:   raw.Model,
		Content: content,
	}, nil
}

func (c *clientImpl) applyRequestDefaults(request ChatCompletionRequest) ChatCompletionRequest {
	if strings.TrimSpace(request.Model) == "" {
		request.Model = c.config.Model
	}
	if request.MaxTokens <= 0 {
		request.MaxTokens = c.config.MaxTokens
	}
	if request.Temperature == nil {
		temperature := c.config.Temperature
		request.Temperature = &temperature
	}
	return request
}

type chatCompletionResponseBody struct {
	ID      string `json:"id"`
	Model   string `json:"model"`
	Choices []struct {
		Message ChatMessage `json:"message"`
	} `json:"choices"`
}

func ensureHTTPSuccess(resp *http.Response) error {
	if resp.StatusCode >= http.StatusOK && resp.StatusCode < http.StatusMultipleChoices {
		return nil
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, errorBodyLimit))
	if err != nil {
		return fmt.Errorf("deepseek chat completion request failed with status %s and unreadable body: %w", resp.Status, err)
	}
	return fmt.Errorf("deepseek chat completion request failed with status %s: %s", resp.Status, string(body))
}
