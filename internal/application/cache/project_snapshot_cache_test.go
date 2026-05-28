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
)

func TestRedisProjectSnapshotCacheListProjectsPage(t *testing.T) {
	ctx := context.Background()
	mini := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mini.Addr()})
	t.Cleanup(func() { _ = client.Close() })
	cache := NewProjectSnapshotCache(redisport.NewGoRedisAdapter(client))

	for i := 1; i <= 5; i++ {
		contract := common.BigToAddress(big.NewInt(int64(i)))
		if err := cache.SetProject(ctx, &Project{Meta: ProjectMeta{
			BlockNumber: uint64(i),
			Contract:    contract,
			TxIndex:     uint64(i),
		}}); err != nil {
			t.Fatalf("set project %d: %v", i, err)
		}
	}

	projects, total, page, pageSize, err := cache.ListProjectsPage(ctx, 2, 2)
	if err != nil {
		t.Fatalf("list projects page: %v", err)
	}
	if total != 5 || page != 2 || pageSize != 2 {
		t.Fatalf("pagination = total %d page %d pageSize %d, want 5/2/2", total, page, pageSize)
	}
	if len(projects) != 2 {
		t.Fatalf("len(projects) = %d, want 2", len(projects))
	}
	if got, want := projects[0].Meta.Contract, common.BigToAddress(big.NewInt(3)); got != want {
		t.Fatalf("projects[0].contract = %s, want %s", got, want)
	}
	if got, want := projects[1].Meta.Contract, common.BigToAddress(big.NewInt(4)); got != want {
		t.Fatalf("projects[1].contract = %s, want %s", got, want)
	}
}

func TestRedisProjectSnapshotCacheSetsTTL(t *testing.T) {
	ctx := context.Background()
	mini := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mini.Addr()})
	t.Cleanup(func() { _ = client.Close() })
	cache := NewProjectSnapshotCache(redisport.NewGoRedisAdapter(client))
	contract := common.BigToAddress(big.NewInt(10))

	if err := cache.SetProject(ctx, &Project{Meta: ProjectMeta{
		BlockNumber: 100,
		Contract:    contract,
		TxIndex:     1,
	}}); err != nil {
		t.Fatalf("set project: %v", err)
	}

	assertRedisTTLNear(t, client, projectDataV2Key(contract), projectSnapshotCacheTTL)
	assertRedisTTLNear(t, client, projectIndexAll, projectSnapshotCacheTTL)
}

func TestRedisProjectSnapshotCacheUpdateRefreshesTTL(t *testing.T) {
	ctx := context.Background()
	mini := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mini.Addr()})
	t.Cleanup(func() { _ = client.Close() })
	cache := NewProjectSnapshotCache(redisport.NewGoRedisAdapter(client))
	contract := common.BigToAddress(big.NewInt(11))

	if err := cache.SetProject(ctx, &Project{Meta: ProjectMeta{
		BlockNumber: 100,
		Contract:    contract,
		TxIndex:     1,
	}}); err != nil {
		t.Fatalf("set project: %v", err)
	}
	mini.FastForward(time.Hour)

	changed, err := cache.UpdateProject(ctx, contract, func(current *Project, exists bool) (*Project, bool, error) {
		if !exists || current == nil {
			t.Fatal("project missing during update")
		}
		current.Meta.IsBytecodeBlacklisted = true
		return current, true, nil
	})
	if err != nil {
		t.Fatalf("update project: %v", err)
	}
	if !changed {
		t.Fatal("changed = false, want true")
	}

	assertRedisTTLNear(t, client, projectDataV2Key(contract), projectSnapshotCacheTTL)
	assertRedisTTLNear(t, client, projectIndexAll, projectSnapshotCacheTTL)
}

func TestRedisProjectSnapshotCacheListProjectsByPairAddresses(t *testing.T) {
	ctx := context.Background()
	mini := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mini.Addr()})
	t.Cleanup(func() { _ = client.Close() })
	cache := NewProjectSnapshotCache(redisport.NewGoRedisAdapter(client))
	wethPair := common.HexToAddress("0x00000000000000000000000000000000000000a1")
	usdtPair := common.HexToAddress("0x00000000000000000000000000000000000000a2")
	first := common.HexToAddress("0x00000000000000000000000000000000000000c1")
	second := common.HexToAddress("0x00000000000000000000000000000000000000c2")

	if err := cache.SetProject(ctx, &Project{Meta: ProjectMeta{
		BlockNumber: 1,
		Contract:    first,
		WethPair:    wethPair,
	}}); err != nil {
		t.Fatalf("set first project: %v", err)
	}
	if err := cache.SetProject(ctx, &Project{Meta: ProjectMeta{
		BlockNumber: 2,
		Contract:    second,
		UsdtPair:    usdtPair,
	}}); err != nil {
		t.Fatalf("set second project: %v", err)
	}

	projects, err := cache.ListProjectsByPairAddresses(ctx, []common.Address{wethPair, usdtPair, wethPair, common.Address{}})
	if err != nil {
		t.Fatalf("list projects by pair addresses: %v", err)
	}
	if len(projects) != 2 {
		t.Fatalf("project count = %d, want 2", len(projects))
	}
	if projects[0].Meta.Contract != first || projects[1].Meta.Contract != second {
		t.Fatalf("projects = %s/%s, want %s/%s", projects[0].Meta.Contract.Hex(), projects[1].Meta.Contract.Hex(), first.Hex(), second.Hex())
	}
}

func TestRedisProjectSnapshotCacheUpdatesAndDeletesPairIndexes(t *testing.T) {
	ctx := context.Background()
	mini := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mini.Addr()})
	t.Cleanup(func() { _ = client.Close() })
	cache := NewProjectSnapshotCache(redisport.NewGoRedisAdapter(client))
	oldPair := common.HexToAddress("0x00000000000000000000000000000000000000a1")
	newPair := common.HexToAddress("0x00000000000000000000000000000000000000a2")
	contract := common.HexToAddress("0x00000000000000000000000000000000000000c1")

	if err := cache.SetProject(ctx, &Project{Meta: ProjectMeta{
		Contract: contract,
		WethPair: oldPair,
	}}); err != nil {
		t.Fatalf("set project: %v", err)
	}
	changed, err := cache.UpdateProject(ctx, contract, func(current *Project, exists bool) (*Project, bool, error) {
		if !exists || current == nil {
			t.Fatal("project missing during update")
		}
		current.Meta.WethPair = newPair
		return current, true, nil
	})
	if err != nil {
		t.Fatalf("update project: %v", err)
	}
	if !changed {
		t.Fatal("changed = false, want true")
	}
	oldProjects, err := cache.ListProjectsByPairAddresses(ctx, []common.Address{oldPair})
	if err != nil {
		t.Fatalf("list old pair projects: %v", err)
	}
	if len(oldProjects) != 0 {
		t.Fatalf("old pair project count = %d, want 0", len(oldProjects))
	}
	newProjects, err := cache.ListProjectsByPairAddresses(ctx, []common.Address{newPair})
	if err != nil {
		t.Fatalf("list new pair projects: %v", err)
	}
	if len(newProjects) != 1 || newProjects[0].Meta.Contract != contract {
		t.Fatalf("new pair projects = %+v, want contract %s", newProjects, contract.Hex())
	}

	if err := cache.DeleteProject(ctx, contract); err != nil {
		t.Fatalf("delete project: %v", err)
	}
	deletedProjects, err := cache.ListProjectsByPairAddresses(ctx, []common.Address{newPair})
	if err != nil {
		t.Fatalf("list deleted pair projects: %v", err)
	}
	if len(deletedProjects) != 0 {
		t.Fatalf("deleted pair project count = %d, want 0", len(deletedProjects))
	}
}

func assertRedisTTLNear(t *testing.T, client *redis.Client, key string, want time.Duration) {
	t.Helper()
	ttl, err := client.TTL(context.Background(), key).Result()
	if err != nil {
		t.Fatalf("ttl %s: %v", key, err)
	}
	if ttl <= 0 {
		t.Fatalf("ttl %s = %s, want positive", key, ttl)
	}
	if ttl < want-time.Minute || ttl > want {
		t.Fatalf("ttl %s = %s, want near %s", key, ttl, want)
	}
}

func TestRedisProjectSnapshotCachePersistsProjectReport(t *testing.T) {
	ctx := context.Background()
	mini := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mini.Addr()})
	t.Cleanup(func() { _ = client.Close() })
	cache := NewProjectSnapshotCache(redisport.NewGoRedisAdapter(client))
	contract := common.BigToAddress(big.NewInt(100))
	want := ProjectReport{
		IsPolicyEvaluated:          true,
		IsBlacklistedCreatorWallet: true,
		IsBlacklistedGenesisWallet: true,
		IsBlacklistedBytecode:      true,
		HasMintRisk:                true,
	}

	if err := cache.SetProject(ctx, &Project{
		Meta:   ProjectMeta{Contract: contract},
		Report: want,
	}); err != nil {
		t.Fatalf("set project: %v", err)
	}

	project, exists, err := cache.GetProject(ctx, contract)
	if err != nil {
		t.Fatalf("get project: %v", err)
	}
	if !exists || project == nil {
		t.Fatal("project missing after set")
	}
	if got := project.Report; got != want {
		t.Fatalf("project report = %+v, want %+v", got, want)
	}
}

func TestRedisProjectSnapshotCachePersistsCreatorHistoricalProjects(t *testing.T) {
	ctx := context.Background()
	mini := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mini.Addr()})
	t.Cleanup(func() { _ = client.Close() })
	cache := NewProjectSnapshotCache(redisport.NewGoRedisAdapter(client))
	contract := common.BigToAddress(big.NewInt(101))
	historicalProjects := []common.Address{
		common.HexToAddress("0x00000000000000000000000000000000000000a1"),
		common.HexToAddress("0x00000000000000000000000000000000000000a2"),
	}
	wethPair := common.HexToAddress("0x00000000000000000000000000000000000000b1")
	usdtPair := common.HexToAddress("0x00000000000000000000000000000000000000b2")
	fetchAt := mustParseTimeForTest(t, "2026-05-22T00:30:00Z")
	aveDetailFetchedAt := mustParseTimeForTest(t, "2026-05-22T00:31:00Z")

	if err := cache.SetProject(ctx, &Project{
		Meta: ProjectMeta{
			Contract:                  contract,
			WethPair:                  wethPair,
			UsdtPair:                  usdtPair,
			FetchAt:                   fetchAt,
			IsBytecodeBlacklisted:     true,
			CreatorHistoricalProjects: historicalProjects,
			CreatorResult: SimulateResult{
				CanMintViaTransferToWethPair: true,
			},
		},
		AveDetail: &ProjectAveDetail{
			Status:    1,
			Msg:       "SUCCESS",
			DataType:  1,
			IsAudited: true,
			FetchedAt: aveDetailFetchedAt,
			Token:     ProjectAveTokenDetail{LogoURL: "https://example.com/logo.png", Token: "token", Chain: "bsc"},
			Pairs:     []ProjectAvePair{{Pair: "pair-1", Chain: "bsc"}},
		},
	}); err != nil {
		t.Fatalf("set project: %v", err)
	}

	project, exists, err := cache.GetProject(ctx, contract)
	if err != nil {
		t.Fatalf("get project: %v", err)
	}
	if !exists || project == nil {
		t.Fatal("project missing after set")
	}
	if got := project.Meta.CreatorHistoricalProjects; len(got) != len(historicalProjects) || got[0] != historicalProjects[0] || got[1] != historicalProjects[1] {
		t.Fatalf("creator historical projects = %v, want %v", addressHexes(got), addressHexes(historicalProjects))
	}
	if got := project.Meta.WethPair; got != wethPair {
		t.Fatalf("weth pair = %s, want %s", got.Hex(), wethPair.Hex())
	}
	if got := project.Meta.UsdtPair; got != usdtPair {
		t.Fatalf("usdt pair = %s, want %s", got.Hex(), usdtPair.Hex())
	}
	if got := project.Meta.FetchAt; !got.Equal(fetchAt) {
		t.Fatalf("fetch at = %s, want %s", got, fetchAt)
	}
	if project.AveDetail == nil || project.AveDetail.Token.LogoURL != "https://example.com/logo.png" {
		t.Fatalf("ave detail = %+v, want logo", project.AveDetail)
	}
	if !project.Meta.IsBytecodeBlacklisted {
		t.Fatal("is bytecode blacklisted = false, want true")
	}
	if got := project.Meta.CreatorResult; !got.CanMintViaTransferToWethPair {
		t.Fatalf("creator result = %+v, want weth transfer mint flag", got)
	}

	values, err := client.HGetAll(ctx, projectDataV2Key(contract)).Result()
	if err != nil {
		t.Fatalf("hgetall project: %v", err)
	}
	if got := values[projectFieldCreatorHistoricalProjects]; got == "" {
		t.Fatal("raw creator historical projects is empty")
	}
	if got := values[projectFieldWethPair]; got != wethPair.Hex() {
		t.Fatalf("raw weth pair = %q, want %q", got, wethPair.Hex())
	}
	if got := values[projectFieldUsdtPair]; got != usdtPair.Hex() {
		t.Fatalf("raw usdt pair = %q, want %q", got, usdtPair.Hex())
	}
	if got := values[projectFieldFetchAt]; got != fetchAt.Format(time.RFC3339Nano) {
		t.Fatalf("raw fetch at = %q, want %q", got, fetchAt.Format(time.RFC3339Nano))
	}
	if got := values[projectFieldAveDetail]; got == "" || got == "null" {
		t.Fatalf("raw ave detail = %q, want detail", got)
	}
	if got := values[projectFieldBytecodeBlacklisted]; got != "1" {
		t.Fatalf("raw bytecode blacklist = %q, want 1", got)
	}
	if _, ok := values["creator_other_projects_resolved"]; ok {
		t.Fatal("old creator_other_projects_resolved field still present")
	}
	if _, ok := values["creator_other_projects_resolved_at"]; ok {
		t.Fatal("old creator_other_projects_resolved_at field still present")
	}
	if _, ok := values["creator_other_project_contracts"]; ok {
		t.Fatal("old creator_other_project_contracts field still present")
	}
	if _, ok := values["runtime_code_hash"]; ok {
		t.Fatal("old runtime_code_hash field still present")
	}
}

func mustParseTimeForTest(t *testing.T, value string) time.Time {
	t.Helper()
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		t.Fatalf("parse time: %v", err)
	}
	return parsed
}
