package redisport

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

func newTestAdapter(t *testing.T) (*miniredis.Miniredis, *redis.Client, Client) {
	t.Helper()
	mini := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mini.Addr()})
	adapter := NewGoRedisAdapter(client)
	return mini, client, adapter
}

func TestGoRedisAdapterKV(t *testing.T) {
	_, client, adapter := newTestAdapter(t)
	t.Cleanup(func() { _ = client.Close() })

	ctx := context.Background()

	_, err := adapter.Get(ctx, "missing")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}

	if err := adapter.Set(ctx, "k", "v", 0); err != nil {
		t.Fatalf("set failed: %v", err)
	}
	value, err := adapter.Get(ctx, "k")
	if err != nil {
		t.Fatalf("get failed: %v", err)
	}
	if value != "v" {
		t.Fatalf("unexpected value: %q", value)
	}

	if err := adapter.Del(ctx, "k"); err != nil {
		t.Fatalf("del failed: %v", err)
	}
	_, err = adapter.Get(ctx, "k")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound after delete, got %v", err)
	}
}

func TestGoRedisAdapterPipeline(t *testing.T) {
	_, client, adapter := newTestAdapter(t)
	t.Cleanup(func() { _ = client.Close() })

	ctx := context.Background()
	pipe := adapter.TxPipeline()
	pipe.HSet(ctx, "h", map[string]any{"a": "1", "b": "2"})
	pipe.HDel(ctx, "h", "a")
	pipe.Set(ctx, "k", "v", 0)
	pipe.ZAdd(ctx, "z", ZMember{Score: 1, Member: "one"}, ZMember{Score: 2, Member: "two"})
	pipe.ZRem(ctx, "z", "one")
	pipe.Del(ctx, "obsolete")
	if err := pipe.Exec(ctx); err != nil {
		t.Fatalf("pipeline exec failed: %v", err)
	}

	hash, err := client.HGetAll(ctx, "h").Result()
	if err != nil {
		t.Fatalf("hgetall failed: %v", err)
	}
	if len(hash) != 1 || hash["b"] != "2" {
		t.Fatalf("unexpected hash state: %#v", hash)
	}

	got, err := client.Get(ctx, "k").Result()
	if err != nil {
		t.Fatalf("get failed: %v", err)
	}
	if got != "v" {
		t.Fatalf("unexpected key value: %q", got)
	}

	zItems, err := client.ZRange(ctx, "z", 0, -1).Result()
	if err != nil {
		t.Fatalf("zrange failed: %v", err)
	}
	if len(zItems) != 1 || zItems[0] != "two" {
		t.Fatalf("unexpected zset state: %#v", zItems)
	}
}

func TestGoRedisAdapterStreamRoundTrip(t *testing.T) {
	_, client, adapter := newTestAdapter(t)
	t.Cleanup(func() { _ = client.Close() })

	ctx := context.Background()
	const (
		stream = "stream:test"
		group  = "group:test"
	)

	if err := adapter.XGroupCreateMkStream(ctx, stream, group, "0"); err != nil {
		t.Fatalf("xgroup create failed: %v", err)
	}
	if err := adapter.XAdd(ctx, XAddInput{
		Stream: stream,
		Values: map[string]any{"event": "payload"},
	}); err != nil {
		t.Fatalf("xadd failed: %v", err)
	}

	streams, err := adapter.XReadGroup(ctx, XReadGroupInput{
		Group:    group,
		Consumer: "consumer-1",
		Streams:  []string{stream, ">"},
		Count:    1,
		Block:    100 * time.Millisecond,
	})
	if err != nil {
		t.Fatalf("xreadgroup failed: %v", err)
	}
	if len(streams) != 1 || len(streams[0].Messages) != 1 {
		t.Fatalf("unexpected stream result: %#v", streams)
	}
	message := streams[0].Messages[0]
	if value, ok := message.Values["event"]; !ok || value != "payload" {
		t.Fatalf("unexpected message payload: %#v", message.Values)
	}

	if err := adapter.XAck(ctx, stream, group, message.ID); err != nil {
		t.Fatalf("xack failed: %v", err)
	}
}
