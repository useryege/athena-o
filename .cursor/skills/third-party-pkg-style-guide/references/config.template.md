# `config.go` Template

Use this file when constructor input or defaulting logic becomes large enough to deserve its own file.

```go
package <third-party-name>

import (
	"time"
)

type Option func(*Config)

type Config struct {
	baseURL    string
	timeout    time.Duration
	headers    map[string]string
}

func WithBaseURL(baseURL string) Option {
	return func(c *Config) {
		c.baseURL = baseURL
	}
}

func WithHeaders(headers map[string]string) Option {
	return func(c *Config) {
		if c.headers == nil {
			c.headers = map[string]string{}
		}
		for key, value := range headers {
			c.headers[key] = value
		}
	}
}

func WithHeader(key, value string) Option {
	return func(c *Config) {
		if c.headers == nil {
			c.headers = map[string]string{}
		}
		c.headers[key] = value
	}
}

func WithTimeout(timeout time.Duration) Option {
	return func(c *Config) {
		c.timeout = timeout
	}
}

func NewConfig(opts ...Option) Config {
	// default values
	cfg := Config{
		timeout: 10 * time.Second,
		headers: map[string]string{},
		baseURL: "",
	}

	// apply options
	for _, opt := range opts {
		opt(&cfg)
	}

	// return the config
	return cfg
}
```