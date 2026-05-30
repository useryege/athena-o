package api

import (
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/ethereum/go-ethereum/common"
	"github.com/redis/go-redis/v9"
	"github.com/useryege/athena/internal/application/persistence"
	"github.com/useryege/athena/internal/application/redisport"
	appstore "github.com/useryege/athena/internal/application/store"
	"github.com/useryege/athena/util/ave"
)

type servicePersistenceStore struct {
	appstore.Store
}

func TestNewServiceRequiresRedisClient(t *testing.T) {
	_, err := NewService(nil, common.Address{}, common.Address{}, common.Address{}, 0, 0, common.Address{}, ave.Config{}, &servicePersistenceStore{}, nil, nil, nil, nil)
	if err == nil {
		t.Fatal("expected nil redis client error")
	}
}

func TestNewServiceUsesRedisBufferedStore(t *testing.T) {
	mini := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mini.Addr()})
	t.Cleanup(func() { _ = client.Close() })

	service, err := NewService(nil, common.Address{}, common.Address{}, common.Address{}, 0, 0, common.Address{}, ave.Config{}, &servicePersistenceStore{}, nil, redisport.NewGoRedisAdapter(client), nil, nil)
	if err != nil {
		t.Fatalf("new service: %v", err)
	}
	if service.persistenceFlush == nil {
		t.Fatal("persistence flusher is nil")
	}
	if _, ok := service.store.(*persistence.RedisBufferedStore); !ok {
		t.Fatalf("service store type = %T, want RedisBufferedStore", service.store)
	}
}
