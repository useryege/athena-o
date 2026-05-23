package redisport

import (
	"context"
	"errors"
	"time"
)

var ErrNotFound = errors.New("redis value not found")

type ZMember struct {
	Score  float64
	Member string
}

type XAddInput struct {
	Stream string
	Values map[string]any
}

type XReadGroupInput struct {
	Group    string
	Consumer string
	Streams  []string
	Count    int64
	Block    time.Duration
}

type XMessage struct {
	ID     string
	Values map[string]any
}

type XStream struct {
	Stream   string
	Messages []XMessage
}

type KVReaderWriter interface {
	Get(ctx context.Context, key string) (string, error)
	Set(ctx context.Context, key string, value any, expiration time.Duration) error
	Del(ctx context.Context, keys ...string) error
}

type HashReader interface {
	HGetAll(ctx context.Context, key string) (map[string]string, error)
}

type SortedSetReader interface {
	ZRange(ctx context.Context, key string, start, stop int64) ([]string, error)
	ZRevRange(ctx context.Context, key string, start, stop int64) ([]string, error)
	ZCard(ctx context.Context, key string) (int64, error)
}

type Scanner interface {
	Scan(ctx context.Context, cursor uint64, match string, count int64) ([]string, uint64, error)
}

type StreamClient interface {
	XAdd(ctx context.Context, input XAddInput) error
	XReadGroup(ctx context.Context, input XReadGroupInput) ([]XStream, error)
	XAck(ctx context.Context, stream, group string, ids ...string) error
	XGroupCreateMkStream(ctx context.Context, stream, group, start string) error
}

type Pipeline interface {
	Del(ctx context.Context, keys ...string)
	HSet(ctx context.Context, key string, values map[string]any)
	HDel(ctx context.Context, key string, fields ...string)
	ZRem(ctx context.Context, key string, members ...string)
	ZAdd(ctx context.Context, key string, members ...ZMember)
	Set(ctx context.Context, key string, value any, expiration time.Duration)
	Expire(ctx context.Context, key string, expiration time.Duration)
	Exec(ctx context.Context) error
}

type TxRunner interface {
	TxPipeline() Pipeline
}

type Client interface {
	KVReaderWriter
	HashReader
	SortedSetReader
	Scanner
	StreamClient
	TxRunner
}
