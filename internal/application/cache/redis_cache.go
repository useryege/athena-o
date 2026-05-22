package cache

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/useryege/athena/internal/application/redisport"
	"github.com/useryege/athena/internal/application/store"
)

const (
	SourceCodeBlacklistRedisKey        = "source_code:blacklist:fields"
	SourceCodeBlacklistVersionRedisKey = "source_code:blacklist:fields:version"
	BytecodeBlacklistRedisKey          = "bytecode:blacklist:contracts"
	BytecodeBlacklistVersionRedisKey   = "bytecode:blacklist:contracts:version"
	SourcecodeBlacklistRedisKey        = "sourcecode:blacklist:contracts"
	SourcecodeBlacklistVersionRedisKey = "sourcecode:blacklist:contracts:version"
	WalletBlacklistRedisKey            = "wallet:blacklist:entries"
	WalletBlacklistVersionRedisKey     = "wallet:blacklist:entries:version"
)

var _ SourceCodeBlacklistRemoteCache = &RedisBlacklistCache{}
var _ BytecodeBlacklistRemoteCache = &RedisBytecodeBlacklistCache{}
var _ SourcecodeBlacklistContractRemoteCache = &RedisSourcecodeBlacklistContractCache{}
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

func (c *RedisBlacklistCache) Get(ctx context.Context) ([]string, string, bool, error) {
	if c == nil || c.client == nil {
		return nil, "", false, nil
	}

	value, err := c.client.Get(ctx, SourceCodeBlacklistRedisKey)
	if errors.Is(err, redisport.ErrNotFound) {
		return nil, "", false, nil
	}
	if err != nil {
		return nil, "", false, err
	}

	var fields []string
	if err := json.Unmarshal([]byte(value), &fields); err != nil {
		return nil, "", false, err
	}
	version, ok, err := getRedisVersion(ctx, c.client, SourceCodeBlacklistVersionRedisKey)
	if err != nil {
		return nil, "", false, err
	}
	if !ok {
		version, err = setRedisVersion(ctx, c.client, SourceCodeBlacklistVersionRedisKey)
		if err != nil {
			return nil, "", false, err
		}
	}
	return fields, version, true, nil
}

func (c *RedisBlacklistCache) Set(ctx context.Context, fields []string) (string, error) {
	if c == nil || c.client == nil {
		return "", nil
	}

	payload, err := json.Marshal(fields)
	if err != nil {
		return "", err
	}
	return setRedisPayloadWithVersion(ctx, c.client, SourceCodeBlacklistRedisKey, SourceCodeBlacklistVersionRedisKey, payload)
}

func (c *RedisBlacklistCache) Del(ctx context.Context) error {
	if c == nil || c.client == nil {
		return nil
	}
	return c.client.Del(ctx, SourceCodeBlacklistRedisKey, SourceCodeBlacklistVersionRedisKey)
}

func (c *RedisBlacklistCache) Version(ctx context.Context) (string, bool, error) {
	if c == nil || c.client == nil {
		return "", false, nil
	}
	return getRedisVersion(ctx, c.client, SourceCodeBlacklistVersionRedisKey)
}

type NoopRedisBlacklistCache struct{}

func (NoopRedisBlacklistCache) Get(context.Context) ([]string, string, bool, error) {
	return nil, "", false, nil
}

func (NoopRedisBlacklistCache) Set(context.Context, []string) (string, error) {
	return "", nil
}

func (NoopRedisBlacklistCache) Del(context.Context) error {
	return nil
}

func (NoopRedisBlacklistCache) Version(context.Context) (string, bool, error) {
	return "", false, nil
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

func (c *RedisBytecodeBlacklistCache) Get(ctx context.Context) ([]store.BytecodeBlacklistContract, string, bool, error) {
	if c == nil || c.client == nil {
		return nil, "", false, nil
	}
	value, err := c.client.Get(ctx, BytecodeBlacklistRedisKey)
	if errors.Is(err, redisport.ErrNotFound) {
		return nil, "", false, nil
	}
	if err != nil {
		return nil, "", false, err
	}
	var items []store.BytecodeBlacklistContract
	if err := json.Unmarshal([]byte(value), &items); err != nil {
		return nil, "", false, err
	}
	version, ok, err := getRedisVersion(ctx, c.client, BytecodeBlacklistVersionRedisKey)
	if err != nil {
		return nil, "", false, err
	}
	if !ok {
		version, err = setRedisVersion(ctx, c.client, BytecodeBlacklistVersionRedisKey)
		if err != nil {
			return nil, "", false, err
		}
	}
	return items, version, true, nil
}

func (c *RedisBytecodeBlacklistCache) Set(ctx context.Context, items []store.BytecodeBlacklistContract) (string, error) {
	if c == nil || c.client == nil {
		return "", nil
	}
	payload, err := json.Marshal(items)
	if err != nil {
		return "", err
	}
	return setRedisPayloadWithVersion(ctx, c.client, BytecodeBlacklistRedisKey, BytecodeBlacklistVersionRedisKey, payload)
}

func (c *RedisBytecodeBlacklistCache) Del(ctx context.Context) error {
	if c == nil || c.client == nil {
		return nil
	}
	return c.client.Del(ctx, BytecodeBlacklistRedisKey, BytecodeBlacklistVersionRedisKey)
}

func (c *RedisBytecodeBlacklistCache) Version(ctx context.Context) (string, bool, error) {
	if c == nil || c.client == nil {
		return "", false, nil
	}
	return getRedisVersion(ctx, c.client, BytecodeBlacklistVersionRedisKey)
}

type NoopRedisBytecodeBlacklistCache struct{}

func (NoopRedisBytecodeBlacklistCache) Get(context.Context) ([]store.BytecodeBlacklistContract, string, bool, error) {
	return nil, "", false, nil
}

func (NoopRedisBytecodeBlacklistCache) Set(context.Context, []store.BytecodeBlacklistContract) (string, error) {
	return "", nil
}

func (NoopRedisBytecodeBlacklistCache) Del(context.Context) error {
	return nil
}

func (NoopRedisBytecodeBlacklistCache) Version(context.Context) (string, bool, error) {
	return "", false, nil
}

type RedisSourcecodeBlacklistContractCache struct {
	client redisport.KVReaderWriter
}

func NewSourcecodeBlacklistContractRedisCache(client redisport.KVReaderWriter) SourcecodeBlacklistContractRemoteCache {
	if client == nil {
		return NoopRedisSourcecodeBlacklistContractCache{}
	}
	return &RedisSourcecodeBlacklistContractCache{client: client}
}

func (c *RedisSourcecodeBlacklistContractCache) Get(ctx context.Context) ([]store.SourcecodeBlacklistContract, string, bool, error) {
	if c == nil || c.client == nil {
		return nil, "", false, nil
	}
	value, err := c.client.Get(ctx, SourcecodeBlacklistRedisKey)
	if errors.Is(err, redisport.ErrNotFound) {
		return nil, "", false, nil
	}
	if err != nil {
		return nil, "", false, err
	}
	var items []store.SourcecodeBlacklistContract
	if err := json.Unmarshal([]byte(value), &items); err != nil {
		return nil, "", false, err
	}
	version, ok, err := getRedisVersion(ctx, c.client, SourcecodeBlacklistVersionRedisKey)
	if err != nil {
		return nil, "", false, err
	}
	if !ok {
		version, err = setRedisVersion(ctx, c.client, SourcecodeBlacklistVersionRedisKey)
		if err != nil {
			return nil, "", false, err
		}
	}
	return items, version, true, nil
}

func (c *RedisSourcecodeBlacklistContractCache) Set(ctx context.Context, items []store.SourcecodeBlacklistContract) (string, error) {
	if c == nil || c.client == nil {
		return "", nil
	}
	payload, err := json.Marshal(items)
	if err != nil {
		return "", err
	}
	return setRedisPayloadWithVersion(ctx, c.client, SourcecodeBlacklistRedisKey, SourcecodeBlacklistVersionRedisKey, payload)
}

func (c *RedisSourcecodeBlacklistContractCache) Del(ctx context.Context) error {
	if c == nil || c.client == nil {
		return nil
	}
	return c.client.Del(ctx, SourcecodeBlacklistRedisKey, SourcecodeBlacklistVersionRedisKey)
}

func (c *RedisSourcecodeBlacklistContractCache) Version(ctx context.Context) (string, bool, error) {
	if c == nil || c.client == nil {
		return "", false, nil
	}
	return getRedisVersion(ctx, c.client, SourcecodeBlacklistVersionRedisKey)
}

type NoopRedisSourcecodeBlacklistContractCache struct{}

func (NoopRedisSourcecodeBlacklistContractCache) Get(context.Context) ([]store.SourcecodeBlacklistContract, string, bool, error) {
	return nil, "", false, nil
}

func (NoopRedisSourcecodeBlacklistContractCache) Set(context.Context, []store.SourcecodeBlacklistContract) (string, error) {
	return "", nil
}

func (NoopRedisSourcecodeBlacklistContractCache) Del(context.Context) error {
	return nil
}

func (NoopRedisSourcecodeBlacklistContractCache) Version(context.Context) (string, bool, error) {
	return "", false, nil
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

func (c *RedisWalletBlacklistCache) Get(ctx context.Context) ([]store.WalletBlacklistEntry, string, bool, error) {
	if c == nil || c.client == nil {
		return nil, "", false, nil
	}
	value, err := c.client.Get(ctx, WalletBlacklistRedisKey)
	if errors.Is(err, redisport.ErrNotFound) {
		return nil, "", false, nil
	}
	if err != nil {
		return nil, "", false, err
	}
	var items []store.WalletBlacklistEntry
	if err := json.Unmarshal([]byte(value), &items); err != nil {
		return nil, "", false, err
	}
	version, ok, err := getRedisVersion(ctx, c.client, WalletBlacklistVersionRedisKey)
	if err != nil {
		return nil, "", false, err
	}
	if !ok {
		version, err = setRedisVersion(ctx, c.client, WalletBlacklistVersionRedisKey)
		if err != nil {
			return nil, "", false, err
		}
	}
	return items, version, true, nil
}

func (c *RedisWalletBlacklistCache) Set(ctx context.Context, items []store.WalletBlacklistEntry) (string, error) {
	if c == nil || c.client == nil {
		return "", nil
	}
	payload, err := json.Marshal(items)
	if err != nil {
		return "", err
	}
	return setRedisPayloadWithVersion(ctx, c.client, WalletBlacklistRedisKey, WalletBlacklistVersionRedisKey, payload)
}

func (c *RedisWalletBlacklistCache) Del(ctx context.Context) error {
	if c == nil || c.client == nil {
		return nil
	}
	return c.client.Del(ctx, WalletBlacklistRedisKey, WalletBlacklistVersionRedisKey)
}

func (c *RedisWalletBlacklistCache) Version(ctx context.Context) (string, bool, error) {
	if c == nil || c.client == nil {
		return "", false, nil
	}
	return getRedisVersion(ctx, c.client, WalletBlacklistVersionRedisKey)
}

type NoopRedisWalletBlacklistCache struct{}

func (NoopRedisWalletBlacklistCache) Get(context.Context) ([]store.WalletBlacklistEntry, string, bool, error) {
	return nil, "", false, nil
}

func (NoopRedisWalletBlacklistCache) Set(context.Context, []store.WalletBlacklistEntry) (string, error) {
	return "", nil
}

func (NoopRedisWalletBlacklistCache) Del(context.Context) error {
	return nil
}

func (NoopRedisWalletBlacklistCache) Version(context.Context) (string, bool, error) {
	return "", false, nil
}

func getRedisVersion(ctx context.Context, client redisport.KVReaderWriter, key string) (string, bool, error) {
	version, err := client.Get(ctx, key)
	if errors.Is(err, redisport.ErrNotFound) {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	return version, true, nil
}

func setRedisPayloadWithVersion(ctx context.Context, client redisport.KVReaderWriter, payloadKey, versionKey string, payload []byte) (string, error) {
	version := newBlacklistVersion()
	if tx, ok := client.(redisport.TxRunner); ok {
		pipe := tx.TxPipeline()
		pipe.Set(ctx, payloadKey, payload, 0)
		pipe.Set(ctx, versionKey, version, 0)
		if err := pipe.Exec(ctx); err != nil {
			return "", err
		}
		return version, nil
	}
	if err := client.Set(ctx, payloadKey, payload, 0); err != nil {
		return "", err
	}
	if err := client.Set(ctx, versionKey, version, 0); err != nil {
		return "", err
	}
	return version, nil
}

func setRedisVersion(ctx context.Context, client redisport.KVReaderWriter, key string) (string, error) {
	version := newBlacklistVersion()
	if err := client.Set(ctx, key, version, 0); err != nil {
		return "", err
	}
	return version, nil
}
