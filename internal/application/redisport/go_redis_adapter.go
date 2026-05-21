package redisport

import (
	"context"
	"errors"
	"time"

	"github.com/redis/go-redis/v9"
)

var _ Client = (*GoRedisAdapter)(nil)

type GoRedisAdapter struct {
	client *redis.Client
}

func NewGoRedisAdapter(client *redis.Client) Client {
	if client == nil {
		return nil
	}
	return &GoRedisAdapter{client: client}
}

func (a *GoRedisAdapter) Get(ctx context.Context, key string) (string, error) {
	value, err := a.client.Get(ctx, key).Result()
	if err != nil {
		return "", normalizeErr(err)
	}
	return value, nil
}

func (a *GoRedisAdapter) Set(ctx context.Context, key string, value any, expiration time.Duration) error {
	return normalizeErr(a.client.Set(ctx, key, value, expiration).Err())
}

func (a *GoRedisAdapter) Del(ctx context.Context, keys ...string) error {
	if len(keys) == 0 {
		return nil
	}
	return normalizeErr(a.client.Del(ctx, keys...).Err())
}

func (a *GoRedisAdapter) HGetAll(ctx context.Context, key string) (map[string]string, error) {
	values, err := a.client.HGetAll(ctx, key).Result()
	if err != nil {
		return nil, normalizeErr(err)
	}
	return values, nil
}

func (a *GoRedisAdapter) ZRange(ctx context.Context, key string, start, stop int64) ([]string, error) {
	items, err := a.client.ZRange(ctx, key, start, stop).Result()
	if err != nil {
		return nil, normalizeErr(err)
	}
	return items, nil
}

func (a *GoRedisAdapter) ZRevRange(ctx context.Context, key string, start, stop int64) ([]string, error) {
	items, err := a.client.ZRevRange(ctx, key, start, stop).Result()
	if err != nil {
		return nil, normalizeErr(err)
	}
	return items, nil
}

func (a *GoRedisAdapter) ZCard(ctx context.Context, key string) (int64, error) {
	size, err := a.client.ZCard(ctx, key).Result()
	if err != nil {
		return 0, normalizeErr(err)
	}
	return size, nil
}

func (a *GoRedisAdapter) Scan(ctx context.Context, cursor uint64, match string, count int64) ([]string, uint64, error) {
	keys, nextCursor, err := a.client.Scan(ctx, cursor, match, count).Result()
	if err != nil {
		return nil, 0, normalizeErr(err)
	}
	return keys, nextCursor, nil
}

func (a *GoRedisAdapter) XAdd(ctx context.Context, input XAddInput) error {
	return normalizeErr(a.client.XAdd(ctx, &redis.XAddArgs{
		Stream: input.Stream,
		Values: input.Values,
	}).Err())
}

func (a *GoRedisAdapter) XReadGroup(ctx context.Context, input XReadGroupInput) ([]XStream, error) {
	streams, err := a.client.XReadGroup(ctx, &redis.XReadGroupArgs{
		Group:    input.Group,
		Consumer: input.Consumer,
		Streams:  input.Streams,
		Count:    input.Count,
		Block:    input.Block,
	}).Result()
	if err != nil {
		return nil, normalizeErr(err)
	}
	converted := make([]XStream, 0, len(streams))
	for _, stream := range streams {
		messages := make([]XMessage, 0, len(stream.Messages))
		for _, message := range stream.Messages {
			messages = append(messages, XMessage{
				ID:     message.ID,
				Values: message.Values,
			})
		}
		converted = append(converted, XStream{
			Stream:   stream.Stream,
			Messages: messages,
		})
	}
	return converted, nil
}

func (a *GoRedisAdapter) XAck(ctx context.Context, stream, group string, ids ...string) error {
	if len(ids) == 0 {
		return nil
	}
	return normalizeErr(a.client.XAck(ctx, stream, group, ids...).Err())
}

func (a *GoRedisAdapter) XGroupCreateMkStream(ctx context.Context, stream, group, start string) error {
	return normalizeErr(a.client.XGroupCreateMkStream(ctx, stream, group, start).Err())
}

func (a *GoRedisAdapter) TxPipeline() Pipeline {
	return &goRedisPipeline{inner: a.client.TxPipeline()}
}

type goRedisPipeline struct {
	inner redis.Pipeliner
}

func (p *goRedisPipeline) Del(ctx context.Context, keys ...string) {
	if len(keys) == 0 {
		return
	}
	p.inner.Del(ctx, keys...)
}

func (p *goRedisPipeline) HSet(ctx context.Context, key string, values map[string]any) {
	if len(values) == 0 {
		return
	}
	p.inner.HSet(ctx, key, values)
}

func (p *goRedisPipeline) HDel(ctx context.Context, key string, fields ...string) {
	if len(fields) == 0 {
		return
	}
	p.inner.HDel(ctx, key, fields...)
}

func (p *goRedisPipeline) ZRem(ctx context.Context, key string, members ...string) {
	if len(members) == 0 {
		return
	}
	membersAny := make([]any, 0, len(members))
	for _, member := range members {
		membersAny = append(membersAny, member)
	}
	p.inner.ZRem(ctx, key, membersAny...)
}

func (p *goRedisPipeline) ZAdd(ctx context.Context, key string, members ...ZMember) {
	if len(members) == 0 {
		return
	}
	zs := make([]redis.Z, 0, len(members))
	for _, member := range members {
		zs = append(zs, redis.Z{Score: member.Score, Member: member.Member})
	}
	p.inner.ZAdd(ctx, key, zs...)
}

func (p *goRedisPipeline) Set(ctx context.Context, key string, value any, expiration time.Duration) {
	p.inner.Set(ctx, key, value, expiration)
}

func (p *goRedisPipeline) Exec(ctx context.Context) error {
	_, err := p.inner.Exec(ctx)
	return normalizeErr(err)
}

func normalizeErr(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, redis.Nil) {
		return ErrNotFound
	}
	return err
}
