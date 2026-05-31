package cache

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/useryege/athena/util/redisport"
)

type KVClient interface {
	Get(ctx context.Context, key string) (string, error)
	Set(ctx context.Context, key string, value any, expiration time.Duration) error
}

func SetJSON(ctx context.Context, client KVClient, key string, value any, ttl time.Duration) error {
	if client == nil {
		return nil
	}
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return client.Set(ctx, key, string(data), ttl)
}

func GetJSON[T any](ctx context.Context, client KVClient, key string) (T, bool, error) {
	var item T
	if client == nil {
		return item, false, nil
	}
	raw, err := client.Get(ctx, key)
	if err != nil {
		if errors.Is(err, redisport.ErrNotFound) {
			return item, false, nil
		}
		return item, false, err
	}
	if strings.TrimSpace(raw) == "" {
		return item, false, nil
	}
	if err := json.Unmarshal([]byte(raw), &item); err != nil {
		return item, false, err
	}
	return item, true, nil
}

func GetJSONInto(ctx context.Context, client KVClient, key string, target any) (bool, error) {
	if client == nil {
		return false, nil
	}
	raw, err := client.Get(ctx, key)
	if err != nil {
		if errors.Is(err, redisport.ErrNotFound) {
			return false, nil
		}
		return false, err
	}
	return DecodeJSON(raw, target)
}

func DecodeJSON(raw string, target any) (bool, error) {
	if strings.TrimSpace(raw) == "" {
		return false, nil
	}
	if err := json.Unmarshal([]byte(raw), target); err != nil {
		return false, err
	}
	return true, nil
}
