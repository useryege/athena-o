# `client.go` Template

Use this file for client state, constructor logic, and shared request helpers.

Read this template when the package needs a central `Client` type.

```go
package <third-party-name>

import (
	"context"
	"errors"
	"net/http"
	"strings"
)

var (
	ErrBaseURLRequired = errors.New("base url is required")
	ErrNotImplemented  = errors.New("not implemented")
)

type Client interface {
	GetUserInfo(ctx context.Context, userID string) (*UserInfo, error)
	DeleteOrder(ctx context.Context, orderID string) error
}

type client struct {
	httpClient *http.Client
	config Config
}

func NewClient(opts ...Option) (Client, error) {
	cfg := NewConfig(opts...)
	if cfg.baseURL == "" {
		return nil, ErrBaseURLRequired
	}

	return &client{
		config:     cfg,
		httpClient: &http.Client{Timeout: cfg.timeout},
	}, nil
}

func (c *client) doRequest(ctx context.Context, method string, url string, body any) (*http.Response, error) {
	_ = ctx
	_ = method
	_ = body

	req, err := http.NewRequestWithContext(ctx, method, strings.TrimRight(c.config.baseURL, "/")+url, nil)
	if err != nil {
		return nil, err
	}

	for key, value := range c.config.headers {
		req.Header.Set(key, value)
	}

	return c.httpClient.Do(req)
}

type UserInfo struct {
	ID    string `json:"id"`
	Email string `json:"email"`
	Name  string `json:"name"`
}

func (c *client) GetUserInfo(ctx context.Context, userID string) (*UserInfo, error) {
	_ = ctx
	_ = userID
	return nil, ErrNotImplemented
}

func (c *client) DeleteOrder(ctx context.Context, orderID string) error {
	_ = ctx
	_ = orderID
	return ErrNotImplemented
}

```

Prefer returning the exported interface from `NewClient(...)` and keep the concrete implementation private.

Keep shared request setup here. Define request or response structs immediately above the first method that uses them.

