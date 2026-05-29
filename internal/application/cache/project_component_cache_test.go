package cache

import (
	"context"
	"math/big"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/ethereum/go-ethereum/common"
	"github.com/redis/go-redis/v9"
	"github.com/useryege/athena/internal/application/redisport"
	appstore "github.com/useryege/athena/internal/application/store"
)

func TestRedisProjectComponentCacheListBasePage(t *testing.T) {
	ctx := context.Background()
	mini := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mini.Addr()})
	t.Cleanup(func() { _ = client.Close() })
	cache := NewProjectComponentCache(redisport.NewGoRedisAdapter(client))

	for i := 1; i <= 3; i++ {
		if err := cache.SetBase(ctx, appstore.ProjectBase{
			BlockNumber: uint64(i),
			Contract:    common.BigToAddress(big.NewInt(int64(i))),
			TxIndex:     uint64(i),
		}); err != nil {
			t.Fatalf("set base %d: %v", i, err)
		}
	}

	items, total, page, pageSize, err := cache.ListBasePage(ctx, 2, 2)
	if err != nil {
		t.Fatalf("list base page: %v", err)
	}
	if total != 3 || page != 2 || pageSize != 2 {
		t.Fatalf("pagination = total %d page %d pageSize %d, want 3/2/2", total, page, pageSize)
	}
	if len(items) != 1 {
		t.Fatalf("items len = %d, want 1", len(items))
	}
	if got, want := items[0].Contract, common.BigToAddress(big.NewInt(3)); got != want {
		t.Fatalf("contract = %s, want %s", got.Hex(), want.Hex())
	}

	assertRedisTTLNear(t, client, projectBaseKey(common.BigToAddress(big.NewInt(1))), projectComponentCacheTTL)
	assertRedisTTLNear(t, client, projectComponentIndexAll, projectComponentCacheTTL)
}

func TestRedisProjectComponentCacheListChainStatesByPairAddresses(t *testing.T) {
	ctx := context.Background()
	mini := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mini.Addr()})
	t.Cleanup(func() { _ = client.Close() })
	cache := NewProjectComponentCache(redisport.NewGoRedisAdapter(client))
	pair := common.HexToAddress("0x00000000000000000000000000000000000000a1")
	first := common.HexToAddress("0x00000000000000000000000000000000000000c1")
	second := common.HexToAddress("0x00000000000000000000000000000000000000c2")

	if err := cache.SetChainState(ctx, appstore.ProjectChainState{
		ProjectContract: first,
		WethPair:        pair,
		FetchedAt:       time.Unix(100, 0).UTC(),
	}); err != nil {
		t.Fatalf("set first chain state: %v", err)
	}
	if err := cache.SetChainState(ctx, appstore.ProjectChainState{
		ProjectContract: second,
		UsdtPair:        pair,
		FetchedAt:       time.Unix(200, 0).UTC(),
	}); err != nil {
		t.Fatalf("set second chain state: %v", err)
	}

	items, err := cache.ListChainStatesByPairAddresses(ctx, []common.Address{common.Address{}, pair, pair})
	if err != nil {
		t.Fatalf("list chain states by pair: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("items len = %d, want 2", len(items))
	}
	if items[0].ProjectContract != first || items[1].ProjectContract != second {
		t.Fatalf("contracts = %s/%s, want %s/%s", items[0].ProjectContract.Hex(), items[1].ProjectContract.Hex(), first.Hex(), second.Hex())
	}

	assertRedisTTLNear(t, client, projectChainPairKey(pair), projectComponentCacheTTL)
}
