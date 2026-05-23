package application

import (
	"context"
	"math/big"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/ethereum/go-ethereum/common"
	"github.com/redis/go-redis/v9"
	"github.com/useryege/athena/internal/application/redisport"
	athenacontract "github.com/useryege/athena/pkg/abi/ATHENA"
)

func TestRedisProjectSnapshotCacheListActiveProjectsPage(t *testing.T) {
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
			SourceCode:  "contract Source {}",
		}}); err != nil {
			t.Fatalf("set project %d: %v", i, err)
		}
	}

	projects, total, page, pageSize, err := cache.ListActiveProjectsPage(ctx, 2, 2)
	if err != nil {
		t.Fatalf("list active projects page: %v", err)
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

func TestRedisProjectSnapshotCacheReplaceAllRequiresCache(t *testing.T) {
	var cache *RedisProjectSnapshotCache

	if err := cache.ReplaceAll(context.Background(), nil); err == nil {
		t.Fatal("ReplaceAll error = nil, want error")
	}
}

func TestRedisProjectSnapshotCacheReplaceAllRequiresClient(t *testing.T) {
	cache := NewProjectSnapshotCache(nil)

	if err := cache.ReplaceAll(context.Background(), nil); err == nil {
		t.Fatal("ReplaceAll error = nil, want error")
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
		IsBlacklistedSourceCode:    true,
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
	sourceCodeHash := common.HexToHash("0x0101010101010101010101010101010101010101010101010101010101010101")
	codeBinHash := common.HexToHash("0x1111111111111111111111111111111111111111111111111111111111111111")

	if err := cache.SetProject(ctx, &Project{
		Meta: ProjectMeta{
			Contract:       contract,
			SourceCodeHash: sourceCodeHash,
			CodeBinHash:    codeBinHash,
		},
		Runtime: ProjectRuntime{
			CreatorHistoricalProjects: historicalProjects,
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
	if got := project.Runtime.CreatorHistoricalProjects; len(got) != len(historicalProjects) || got[0] != historicalProjects[0] || got[1] != historicalProjects[1] {
		t.Fatalf("creator historical projects = %v, want %v", addressHexes(got), addressHexes(historicalProjects))
	}
	if got := project.Meta.SourceCodeHash; got != sourceCodeHash {
		t.Fatalf("source code hash = %s, want %s", got.Hex(), sourceCodeHash.Hex())
	}
	if got := project.Meta.CodeBinHash; got != codeBinHash {
		t.Fatalf("code bin hash = %s, want %s", got.Hex(), codeBinHash.Hex())
	}

	values, err := client.HGetAll(ctx, projectDataV2Key(contract)).Result()
	if err != nil {
		t.Fatalf("hgetall project: %v", err)
	}
	if got := values[projectFieldCreatorHistoricalProjects]; got == "" {
		t.Fatal("raw creator historical projects is empty")
	}
	if got := values[projectFieldSourceCodeHash]; got != sourceCodeHash.Hex() {
		t.Fatalf("raw source code hash = %q, want %q", got, sourceCodeHash.Hex())
	}
	if got := values[projectFieldCodeBinHash]; got != codeBinHash.Hex() {
		t.Fatalf("raw code bin hash = %q, want %q", got, codeBinHash.Hex())
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

func TestProjectListItemIncludesOnlyListFields(t *testing.T) {
	project := &Project{Meta: ProjectMeta{
		Contract:                common.BigToAddress(big.NewInt(1)),
		BlockTime:               100,
		BlockNumber:             200,
		TxIndex:                 3,
		SourceCode:              "contract Source {}",
		SourceCodeHash:          common.HexToHash("0x3333333333333333333333333333333333333333333333333333333333333333"),
		CodeBinHash:             common.HexToHash("0x4444444444444444444444444444444444444444444444444444444444444444"),
		SourceQualityReport:     "## Report",
		SourceQualityReportedAt: mustParseTimeForTest(t, "2026-05-22T00:00:00Z"),
	}, Runtime: ProjectRuntime{
		ChainState: athenacontract.AthenaProject{
			Token: athenacontract.AthenaToken{
				Name:   "Token",
				Symbol: "TKN",
			},
			WethPair: athenacontract.AthenaPair{
				QuoteUsdtValue:    big.NewInt(123),
				IsRemoveLiquidity: true,
			},
			UsdtPair: athenacontract.AthenaPair{
				QuoteUsdtValue: big.NewInt(456),
			},
			AssetState: athenacontract.AthenaAssetState{
				UsdtValue: big.NewInt(789),
			},
		},
		CreatorResult: SimulateResult{
			CanMintFromZeroViaTransferFrom: true,
		},
	}}

	listItem := projectToListItem(project)
	if listItem.Contract != project.Meta.Contract.String() {
		t.Fatalf("list contract = %q, want %q", listItem.Contract, project.Meta.Contract.String())
	}
	if listItem.Name != "Token" || listItem.Symbol != "TKN" {
		t.Fatalf("list token = %q/%q, want Token/TKN", listItem.Name, listItem.Symbol)
	}
	if !listItem.IsOpenSource || !listItem.HasMintRisk {
		t.Fatalf("list booleans = openSource %t mintRisk %t, want both true",
			listItem.IsOpenSource, listItem.HasMintRisk)
	}
	if listItem.WethPairQuoteUsdtValue != "123" || !listItem.WethPairRemoveLiquidity {
		t.Fatalf("list WETH pair = %q/%t, want 123/true", listItem.WethPairQuoteUsdtValue, listItem.WethPairRemoveLiquidity)
	}
	if listItem.UsdtPairQuoteUsdtValue != "456" || listItem.UsdtPairRemoveLiquidity {
		t.Fatalf("list USDT pair = %q/%t, want 456/false", listItem.UsdtPairQuoteUsdtValue, listItem.UsdtPairRemoveLiquidity)
	}
	if listItem.CreatorAssetUsdtValue != "789" {
		t.Fatalf("list creator asset = %q, want 789", listItem.CreatorAssetUsdtValue)
	}
	if listItem.BlockTime != 100 || listItem.BlockNumber != 200 || listItem.TxIndex != 3 {
		t.Fatalf("list chain position = %d/%d/%d, want 100/200/3", listItem.BlockTime, listItem.BlockNumber, listItem.TxIndex)
	}

	detailView := projectToView(project, true)
	if detailView.Meta.SourceCode == "" || detailView.Meta.SourceQualityReport == "" || detailView.Meta.SourceQualityReportedAt == "" {
		t.Fatalf("detail view missing source detail fields: %#v", detailView.Meta)
	}
	if !detailView.Meta.IsOpenSource {
		t.Fatal("detail isOpenSource = false, want true")
	}
	if detailView.Meta.SourceCodeHash != project.Meta.SourceCodeHash.Hex() || detailView.Meta.CodeBinHash != project.Meta.CodeBinHash.Hex() {
		t.Fatalf("detail hashes = %q/%q, want %q/%q", detailView.Meta.SourceCodeHash, detailView.Meta.CodeBinHash, project.Meta.SourceCodeHash.Hex(), project.Meta.CodeBinHash.Hex())
	}
}
