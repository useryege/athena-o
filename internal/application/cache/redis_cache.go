package cache

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/redis/go-redis/v9"
)

const SourceCodeBlacklistRedisKey = "source_code:blacklist:fields"

var _ SourceCodeBlacklistRemoteCache = &RedisBlacklistCache{}

type RedisBlacklistCache struct {
	client *redis.Client
}

func NewSourceCodeBlacklistRedisCache(client *redis.Client) SourceCodeBlacklistRemoteCache {
	if client == nil {
		return NoopRedisBlacklistCache{}
	}
	return &RedisBlacklistCache{client: client}
}

func (c *RedisBlacklistCache) Get(ctx context.Context) ([]string, bool, error) {
	if c == nil || c.client == nil {
		return nil, false, nil
	}

	value, err := c.client.Get(ctx, SourceCodeBlacklistRedisKey).Result()
	if errors.Is(err, redis.Nil) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}

	var fields []string
	if err := json.Unmarshal([]byte(value), &fields); err != nil {
		return nil, false, err
	}
	return fields, true, nil
}

func (c *RedisBlacklistCache) Set(ctx context.Context, fields []string) error {
	if c == nil || c.client == nil {
		return nil
	}

	payload, err := json.Marshal(fields)
	if err != nil {
		return err
	}
	return c.client.Set(ctx, SourceCodeBlacklistRedisKey, payload, 0).Err()
}

func (c *RedisBlacklistCache) Del(ctx context.Context) error {
	if c == nil || c.client == nil {
		return nil
	}
	return c.client.Del(ctx, SourceCodeBlacklistRedisKey).Err()
}

type NoopRedisBlacklistCache struct{}

func (NoopRedisBlacklistCache) Get(context.Context) ([]string, bool, error) {
	return nil, false, nil
}

func (NoopRedisBlacklistCache) Set(context.Context, []string) error {
	return nil
}

func (NoopRedisBlacklistCache) Del(context.Context) error {
	return nil
}
