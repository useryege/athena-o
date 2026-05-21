package cache

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/useryege/athena/internal/application/redisport"
	"github.com/useryege/athena/internal/application/store"
)

const (
	SourceCodeBlacklistRedisKey = "source_code:blacklist:fields"
	BytecodeBlacklistRedisKey   = "bytecode:blacklist:contracts"
	WalletBlacklistRedisKey     = "wallet:blacklist:contracts"
)

var _ SourceCodeBlacklistRemoteCache = &RedisBlacklistCache{}
var _ BytecodeBlacklistRemoteCache = &RedisBytecodeBlacklistCache{}
var _ WalletBlacklistRemoteCache = &RedisWalletBlacklistCache{}

type RedisBlacklistCache struct {
	client redisport.KVReaderWriter
}

func NewSourceCodeBlacklistRedisCache(client redisport.KVReaderWriter) SourceCodeBlacklistRemoteCache {
	if client == nil {
		return NoopRedisBlacklistCache{}
	}
	return &RedisBlacklistCache{client: client}
}

func (c *RedisBlacklistCache) Get(ctx context.Context) ([]string, bool, error) {
	if c == nil || c.client == nil {
		return nil, false, nil
	}

	value, err := c.client.Get(ctx, SourceCodeBlacklistRedisKey)
	if errors.Is(err, redisport.ErrNotFound) {
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
	return c.client.Set(ctx, SourceCodeBlacklistRedisKey, payload, 0)
}

func (c *RedisBlacklistCache) Del(ctx context.Context) error {
	if c == nil || c.client == nil {
		return nil
	}
	return c.client.Del(ctx, SourceCodeBlacklistRedisKey)
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

type RedisBytecodeBlacklistCache struct {
	client redisport.KVReaderWriter
}

func NewBytecodeBlacklistRedisCache(client redisport.KVReaderWriter) BytecodeBlacklistRemoteCache {
	if client == nil {
		return NoopRedisBytecodeBlacklistCache{}
	}
	return &RedisBytecodeBlacklistCache{client: client}
}

func (c *RedisBytecodeBlacklistCache) Get(ctx context.Context) ([]store.BytecodeBlacklistContract, bool, error) {
	if c == nil || c.client == nil {
		return nil, false, nil
	}
	value, err := c.client.Get(ctx, BytecodeBlacklistRedisKey)
	if errors.Is(err, redisport.ErrNotFound) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	var items []store.BytecodeBlacklistContract
	if err := json.Unmarshal([]byte(value), &items); err != nil {
		return nil, false, err
	}
	return items, true, nil
}

func (c *RedisBytecodeBlacklistCache) Set(ctx context.Context, items []store.BytecodeBlacklistContract) error {
	if c == nil || c.client == nil {
		return nil
	}
	payload, err := json.Marshal(items)
	if err != nil {
		return err
	}
	return c.client.Set(ctx, BytecodeBlacklistRedisKey, payload, 0)
}

func (c *RedisBytecodeBlacklistCache) Del(ctx context.Context) error {
	if c == nil || c.client == nil {
		return nil
	}
	return c.client.Del(ctx, BytecodeBlacklistRedisKey)
}

type NoopRedisBytecodeBlacklistCache struct{}

func (NoopRedisBytecodeBlacklistCache) Get(context.Context) ([]store.BytecodeBlacklistContract, bool, error) {
	return nil, false, nil
}

func (NoopRedisBytecodeBlacklistCache) Set(context.Context, []store.BytecodeBlacklistContract) error {
	return nil
}

func (NoopRedisBytecodeBlacklistCache) Del(context.Context) error {
	return nil
}

type RedisWalletBlacklistCache struct {
	client redisport.KVReaderWriter
}

func NewWalletBlacklistRedisCache(client redisport.KVReaderWriter) WalletBlacklistRemoteCache {
	if client == nil {
		return NoopRedisWalletBlacklistCache{}
	}
	return &RedisWalletBlacklistCache{client: client}
}

func (c *RedisWalletBlacklistCache) Get(ctx context.Context) ([]store.WalletBlacklistContract, bool, error) {
	if c == nil || c.client == nil {
		return nil, false, nil
	}
	value, err := c.client.Get(ctx, WalletBlacklistRedisKey)
	if errors.Is(err, redisport.ErrNotFound) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	var items []store.WalletBlacklistContract
	if err := json.Unmarshal([]byte(value), &items); err != nil {
		return nil, false, err
	}
	return items, true, nil
}

func (c *RedisWalletBlacklistCache) Set(ctx context.Context, items []store.WalletBlacklistContract) error {
	if c == nil || c.client == nil {
		return nil
	}
	payload, err := json.Marshal(items)
	if err != nil {
		return err
	}
	return c.client.Set(ctx, WalletBlacklistRedisKey, payload, 0)
}

func (c *RedisWalletBlacklistCache) Del(ctx context.Context) error {
	if c == nil || c.client == nil {
		return nil
	}
	return c.client.Del(ctx, WalletBlacklistRedisKey)
}

type NoopRedisWalletBlacklistCache struct{}

func (NoopRedisWalletBlacklistCache) Get(context.Context) ([]store.WalletBlacklistContract, bool, error) {
	return nil, false, nil
}

func (NoopRedisWalletBlacklistCache) Set(context.Context, []store.WalletBlacklistContract) error {
	return nil
}

func (NoopRedisWalletBlacklistCache) Del(context.Context) error {
	return nil
}
