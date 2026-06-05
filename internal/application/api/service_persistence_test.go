package api

import (
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/ethereum/go-ethereum/common"
	"github.com/redis/go-redis/v9"
	appstore "github.com/useryege/athena/internal/application/store"
	"github.com/useryege/athena/util/ave"
	"github.com/useryege/athena/util/redisport"
)

type servicePersistenceStore struct {
	appstore.Store
}

func TestNewServiceRequiresStore(t *testing.T) {
	_, err := NewService(ServiceOpts{
		V2FactoryContract: common.Address{},
		WethContract:      common.Address{},
		UsdtContract:      common.Address{},
		AthenaContract:    common.Address{},
		AveConfig:         ave.Config{},
	})
	if err == nil {
		t.Fatal("expected nil store error")
	}
}

func TestNewServiceUsesDirectStore(t *testing.T) {
	mini := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mini.Addr()})
	t.Cleanup(func() { _ = client.Close() })

	service, err := NewService(ServiceOpts{
		V2FactoryContract: common.Address{},
		WethContract:      common.Address{},
		UsdtContract:      common.Address{},
		AthenaContract:    common.Address{},
		AveConfig:         ave.Config{},
		Store:             &servicePersistenceStore{},
		RedisClient:       redisport.NewGoRedisAdapter(client),
	})
	if err != nil {
		t.Fatalf("new service: %v", err)
	}
	if service.store == nil {
		t.Fatal("service store is nil")
	}
	if _, ok := service.store.(*servicePersistenceStore); !ok {
		t.Fatalf("service store type = %T, want direct servicePersistenceStore", service.store)
	}
}
