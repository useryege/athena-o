package application

import (
	"context"
	"math/big"
	"sync"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/ethereum/go-ethereum/common"
	"github.com/redis/go-redis/v9"
	athenacontract "github.com/useryege/athena/pkg/abi/ATHENA"
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
			SourceCode:  "contract A {}",
			CreatorResult: SimulateResult{
				CanMintViaTransferToUsdtPair: true,
			},
			GenesisWallets: []GenesisWalletMeta{
				{
					Wallet:    common.HexToAddress("0x00000000000000000000000000000000000000A2"),
					NetAmount: big.NewInt(200),
					RatioBPS:  2500,
					RankIndex: 0,
				},
			},
		},
		ChainState: athenacontract.AthenaProject{
			Token: athenacontract.AthenaToken{Symbol: "ATH"},
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
	if got.ChainState.Token.Symbol != "ATH" {
		t.Fatalf("chain state symbol = %q, want ATH", got.ChainState.Token.Symbol)
	}
	if got.Meta.SourceCode != "contract A {}" {
		t.Fatalf("source code = %q, want contract A {}", got.Meta.SourceCode)
	}
	if !got.Meta.CreatorResult.CanMintViaTransferToUsdtPair {
		t.Fatalf("creator result not persisted")
	}
	if len(got.Meta.GenesisWallets) != 1 {
		t.Fatalf("genesis wallets len = %d, want 1", len(got.Meta.GenesisWallets))
	}
	if got.Meta.GenesisWallets[0].Wallet != common.HexToAddress("0x00000000000000000000000000000000000000A2") {
		t.Fatalf("genesis wallet = %s, want 0x...A2", got.Meta.GenesisWallets[0].Wallet.Hex())
	}
	if got.Meta.GenesisWallets[0].NetAmount.String() != "200" {
		t.Fatalf("genesis net amount = %s, want 200", got.Meta.GenesisWallets[0].NetAmount.String())
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

func TestUpdateProject_ConcurrentFieldUpdatesDoNotLoseData(t *testing.T) {
	ctx := context.Background()
	cache, _, cleanup := newTestSnapshotCache(t)
	defer cleanup()

	contract := common.HexToAddress("0x00000000000000000000000000000000000000D1")
	if err := cache.SetProject(ctx, &Project{
		Meta: ProjectMeta{
			Contract: contract,
		},
		ChainState: athenacontract.AthenaProject{
			Token: athenacontract.AthenaToken{Symbol: "OLD"},
		},
	}); err != nil {
		t.Fatalf("seed project: %v", err)
	}

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		_, err := cache.UpdateProject(ctx, contract, func(current *Project, exists bool) (*Project, bool, error) {
			if !exists || current == nil {
				return nil, false, nil
			}
			current.ChainState.Token.Symbol = "NEW"
			return current, true, nil
		})
		if err != nil {
			t.Errorf("update chain state: %v", err)
		}
	}()

	go func() {
		defer wg.Done()
		_, err := cache.UpdateProject(ctx, contract, func(current *Project, exists bool) (*Project, bool, error) {
			if !exists || current == nil {
				return nil, false, nil
			}
			current.Meta.CreatorResult = SimulateResult{CanMintViaTransferToWethPair: true}
			return current, true, nil
		})
		if err != nil {
			t.Errorf("update creator result: %v", err)
		}
	}()

	wg.Wait()

	got, ok, err := cache.GetProject(ctx, contract)
	if err != nil {
		t.Fatalf("get project: %v", err)
	}
	if !ok || got == nil {
		t.Fatalf("project missing after updates")
	}
	if got.ChainState.Token.Symbol != "NEW" {
		t.Fatalf("chain state symbol = %q, want NEW", got.ChainState.Token.Symbol)
	}
	if !got.Meta.CreatorResult.CanMintViaTransferToWethPair {
		t.Fatalf("creator result lost after concurrent update")
	}
}

func newTestSnapshotCache(t *testing.T) (*RedisProjectSnapshotCache, *miniredis.Miniredis, func()) {
	t.Helper()

	redisServer, err := miniredis.Run()
	if err != nil {
		t.Fatalf("start miniredis: %v", err)
	}
	client := redis.NewClient(&redis.Options{Addr: redisServer.Addr()})
	cache := &RedisProjectSnapshotCache{
		client:        client,
		contractLocks: map[string]*sync.Mutex{},
	}

	cleanup := func() {
		_ = client.Close()
		redisServer.Close()
	}
	return cache, redisServer, cleanup
}
