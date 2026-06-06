package cache

import (
	"context"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/ethereum/go-ethereum/common"
	"github.com/redis/go-redis/v9"
	appstore "github.com/useryege/athena/internal/application/store"
	"github.com/useryege/athena/util/redisport"
)

func TestProjectComponentCacheSeparatesSameContractByChainID(t *testing.T) {
	ctx := context.Background()
	mini := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mini.Addr()})
	t.Cleanup(func() { _ = client.Close() })

	cache := NewProjectComponentCache(redisport.NewGoRedisAdapter(client))
	contract := common.HexToAddress("0x1000000000000000000000000000000000000001")
	ethCreator := common.HexToAddress("0x1000000000000000000000000000000000000002")
	bscCreator := common.HexToAddress("0x1000000000000000000000000000000000000003")

	if err := cache.SetProject(ctx, appstore.ProjectRecord{ChainID: 1, Contract: contract, Creator: ethCreator, BlockNumber: 10}); err != nil {
		t.Fatalf("set eth project: %v", err)
	}
	if err := cache.SetProject(ctx, appstore.ProjectRecord{ChainID: 56, Contract: contract, Creator: bscCreator, BlockNumber: 20}); err != nil {
		t.Fatalf("set bsc project: %v", err)
	}

	ethProject, ok, err := cache.GetProject(ctx, 1, contract)
	if err != nil || !ok {
		t.Fatalf("get eth project ok=%v err=%v", ok, err)
	}
	bscProject, ok, err := cache.GetProject(ctx, 56, contract)
	if err != nil || !ok {
		t.Fatalf("get bsc project ok=%v err=%v", ok, err)
	}
	if ethProject.Creator != ethCreator || ethProject.BlockNumber != 10 {
		t.Fatalf("eth project = %#v, want eth creator/block", ethProject)
	}
	if bscProject.Creator != bscCreator || bscProject.BlockNumber != 20 {
		t.Fatalf("bsc project = %#v, want bsc creator/block", bscProject)
	}
}
