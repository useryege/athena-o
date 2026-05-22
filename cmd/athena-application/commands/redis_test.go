package commands

import (
	"context"
	"strings"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

func TestRequireApplicationRedisNilClient(t *testing.T) {
	err := requireApplicationRedis(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "redis client is required") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRequireApplicationRedisPingFailure(t *testing.T) {
	mini := miniredis.RunT(t)
	addr := mini.Addr()
	mini.Close()

	client := redis.NewClient(&redis.Options{Addr: addr})
	t.Cleanup(func() { _ = client.Close() })

	err := requireApplicationRedis(context.Background(), client)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "failed to ping redis") {
		t.Fatalf("unexpected error: %v", err)
	}
}
