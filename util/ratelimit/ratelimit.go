package ratelimit

import (
	"context"
	"errors"
	"time"

	"golang.org/x/time/rate"
)

type Limiter interface {
	Wait(ctx context.Context) error
}

type Config struct {
	Requests int
	Per      time.Duration
	Burst    int
}

func New(config Config) (Limiter, error) {
	if config == (Config{}) {
		return Noop(), nil
	}
	if config.Requests <= 0 {
		return nil, errors.New("rate limit requests must be greater than 0")
	}
	if config.Per <= 0 {
		return nil, errors.New("rate limit period must be greater than 0")
	}
	if config.Burst < 0 {
		return nil, errors.New("rate limit burst must be greater than or equal to 0")
	}
	if config.Burst == 0 {
		config.Burst = config.Requests
	}
	limit := rate.Limit(float64(config.Requests) / config.Per.Seconds())
	return &limiter{limiter: rate.NewLimiter(limit, config.Burst)}, nil
}

func Noop() Limiter {
	return noopLimiter{}
}

type limiter struct {
	limiter *rate.Limiter
}

func (l *limiter) Wait(ctx context.Context) error {
	return l.limiter.Wait(ctx)
}

type noopLimiter struct{}

func (noopLimiter) Wait(ctx context.Context) error {
	return ctx.Err()
}
