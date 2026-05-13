package cache

import "context"

const SourceCodeBlacklistRedisKey = "source_code:blacklist:fields"

type NoopRedisBlacklistCache struct{}

func NewNoopRedisBlacklistCache() NoopRedisBlacklistCache {
	return NoopRedisBlacklistCache{}
}

func (NoopRedisBlacklistCache) Get(ctx context.Context) ([]string, bool, error) {
	return nil, false, nil
}

func (NoopRedisBlacklistCache) Set(ctx context.Context, fields []string) error {
	return nil
}

func (NoopRedisBlacklistCache) Del(ctx context.Context) error {
	return nil
}
