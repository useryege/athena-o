package application

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/ethereum/go-ethereum/common"
	"github.com/redis/go-redis/v9"
)

func TestSetProject_WritesProjectWithoutUpdatingMaxBlock(t *testing.T) {
	ctx := context.Background()
	cache, _, cleanup := newTestSnapshotCache(t)
	defer cleanup()

	contract := common.HexToAddress("0x00000000000000000000000000000000000000A1")
	project := &Project{
		Meta: ProjectMeta{
			Contract:    contract,
			BlockNumber: 42,
			TxIndex:     3,
			IsArchived:  false,
		},
	}

	if err := cache.SetProject(ctx, project); err != nil {
		t.Fatalf("set project: %v", err)
	}

	got, ok, err := cache.GetProject(ctx, contract)
	if err != nil {
		t.Fatalf("get project: %v", err)
	}
	if !ok || got == nil {
		t.Fatalf("project not found in cache")
	}
	if got.Meta.Contract != contract {
		t.Fatalf("project contract = %s, want %s", got.Meta.Contract, contract)
	}

	maxBlock, ok, err := cache.GetMaxProjectBlockNumber(ctx)
	if err != nil {
		t.Fatalf("get max project block number: %v", err)
	}
	if ok {
		t.Fatalf("max project block number = %d, want not found", maxBlock)
	}
}

func TestReplaceAll_SetsMaxProjectBlockNumber(t *testing.T) {
	ctx := context.Background()
	cache, _, cleanup := newTestSnapshotCache(t)
	defer cleanup()

	projects := []*Project{
		{Meta: ProjectMeta{Contract: common.HexToAddress("0x00000000000000000000000000000000000000C1"), BlockNumber: 30}},
		{Meta: ProjectMeta{Contract: common.HexToAddress("0x00000000000000000000000000000000000000C2"), BlockNumber: 80}},
		{Meta: ProjectMeta{Contract: common.HexToAddress("0x00000000000000000000000000000000000000C3"), BlockNumber: 10}},
	}
	if err := cache.ReplaceAll(ctx, projects); err != nil {
		t.Fatalf("replace all: %v", err)
	}

	maxBlock, ok, err := cache.GetMaxProjectBlockNumber(ctx)
	if err != nil {
		t.Fatalf("get max project block number: %v", err)
	}
	if !ok {
		t.Fatalf("max project block number not found")
	}
	if maxBlock != 80 {
		t.Fatalf("max project block number = %d, want 80", maxBlock)
	}
}

func TestSetProject_UnarchiveFlowMovesIndexes(t *testing.T) {
	ctx := context.Background()
	cache, redisServer, cleanup := newTestSnapshotCache(t)
	defer cleanup()

	contract := common.HexToAddress("0x00000000000000000000000000000000000000B2")
	archived := &Project{
		Meta: ProjectMeta{
			Contract:    contract,
			BlockNumber: 8,
			TxIndex:     1,
			IsArchived:  true,
			ArchivedAt:  time.Unix(1710000000, 0).UTC(),
		},
	}
	if err := cache.SetProject(ctx, archived); err != nil {
		t.Fatalf("set archived project: %v", err)
	}

	unarchived := &Project{
		Meta: ProjectMeta{
			Contract:    contract,
			BlockNumber: 9,
			TxIndex:     2,
			IsArchived:  false,
		},
	}
	if err := cache.SetProject(ctx, unarchived); err != nil {
		t.Fatalf("set unarchived project: %v", err)
	}

	activeIDs, err := redisServer.ZMembers(projectIndexActive)
	if err != nil {
		t.Fatalf("read active index: %v", err)
	}
	if len(activeIDs) != 1 || activeIDs[0] != contract.Hex() {
		t.Fatalf("active index = %v, want [%s]", activeIDs, contract.Hex())
	}

	archivedIDs, err := redisServer.ZMembers(projectIndexArchived)
	if err != nil && err.Error() != "ERR no such key" {
		t.Fatalf("read archived index: %v", err)
	}
	if len(archivedIDs) != 0 {
		t.Fatalf("archived index = %v, want []", archivedIDs)
	}
}

func newTestSnapshotCache(t *testing.T) (*RedisProjectSnapshotCache, *miniredis.Miniredis, func()) {
	t.Helper()

	redisServer, err := miniredis.Run()
	if err != nil {
		t.Fatalf("start miniredis: %v", err)
	}
	client := redis.NewClient(&redis.Options{Addr: redisServer.Addr()})
	cache := &RedisProjectSnapshotCache{client: client}

	cleanup := func() {
		_ = client.Close()
		redisServer.Close()
	}
	return cache, redisServer, cleanup
}
