package api

import (
	"math/big"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	athenacontract "github.com/useryege/athena/pkg/abi/ATHENA"
)

func TestProjectListItemIncludesOnlyListFields(t *testing.T) {
	project := &Project{Meta: ProjectMeta{
		Contract:                common.BigToAddress(big.NewInt(1)),
		BlockTime:               100,
		BlockNumber:             200,
		TxIndex:                 3,
		FetchAt:                 mustParseTimeForTest(t, "2026-05-21T23:58:00.123456789Z"),
		GenesisWalletsFetchedAt: mustParseTimeForTest(t, "2026-05-22T00:01:00Z"),
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
	}, AveDetail: &ProjectAveDetail{
		Status:    1,
		Msg:       "SUCCESS",
		DataType:  1,
		FetchedAt: mustParseTimeForTest(t, "2026-05-22T00:00:30Z"),
		Token: ProjectAveTokenDetail{
			LogoURL:       "https://example.com/logo.png",
			Token:         "token",
			Chain:         "bsc",
			HasMintMethod: true,
			IsMintable:    "1",
			Holders:       1234,
			MarketCap:     "5678.9",
			IsHoneypot:    true,
		},
	}}

	listItem := projectToListItem(project)
	if listItem.Contract != project.Meta.Contract.String() {
		t.Fatalf("list contract = %q, want %q", listItem.Contract, project.Meta.Contract.String())
	}
	if listItem.Name != "Token" || listItem.Symbol != "TKN" {
		t.Fatalf("list token = %q/%q, want Token/TKN", listItem.Name, listItem.Symbol)
	}
	if listItem.IsOpenSource || !listItem.HasMintRisk {
		t.Fatalf("list booleans = openSource %t mintRisk %t, want false/true",
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
	if listItem.AveLogo != project.AveDetail.Token.LogoURL {
		t.Fatalf("list ave logo = %q, want %q", listItem.AveLogo, project.AveDetail.Token.LogoURL)
	}
	if !listItem.AveDetailAvailable || !listItem.AveIsHoneypot || !listItem.AveHasMintMethod || listItem.AveIsMintable != "1" || listItem.AveHolders != 1234 || listItem.AveMarketCap != "5678.9" {
		t.Fatalf("list ave fields = available %t honeypot %t mintMethod %t mintable %q holders %d marketCap %q, want populated ave values",
			listItem.AveDetailAvailable, listItem.AveIsHoneypot, listItem.AveHasMintMethod, listItem.AveIsMintable, listItem.AveHolders, listItem.AveMarketCap)
	}
	projectWithoutAve := *project
	projectWithoutAve.AveDetail = nil
	listItemWithoutAve := projectToListItem(&projectWithoutAve)
	if listItemWithoutAve.AveDetailAvailable || listItemWithoutAve.AveIsHoneypot || listItemWithoutAve.AveHasMintMethod || listItemWithoutAve.AveIsMintable != "" || listItemWithoutAve.AveHolders != 0 || listItemWithoutAve.AveMarketCap != "" {
		t.Fatalf("list ave fields without detail = available %t honeypot %t mintMethod %t mintable %q holders %d marketCap %q, want zero values",
			listItemWithoutAve.AveDetailAvailable, listItemWithoutAve.AveIsHoneypot, listItemWithoutAve.AveHasMintMethod, listItemWithoutAve.AveIsMintable, listItemWithoutAve.AveHolders, listItemWithoutAve.AveMarketCap)
	}

	detailView := projectToView(project, true)
	if detailView.Meta.FetchAt != project.Meta.FetchAt.UTC().Format(time.RFC3339Nano) {
		t.Fatalf("detail fetch at = %q, want %q", detailView.Meta.FetchAt, project.Meta.FetchAt.UTC().Format(time.RFC3339Nano))
	}
	if detailView.Meta.GenesisWalletsFetchedAt == "" {
		t.Fatalf("detail view missing fetched-at fields: %#v", detailView.Meta)
	}
	if detailView.AveDetail.Token.LogoURL != project.AveDetail.Token.LogoURL || detailView.AveDetail.FetchedAt == "" {
		t.Fatalf("detail view missing ave detail fields: %#v", detailView.AveDetail)
	}
}

func mustParseTimeForTest(t *testing.T, value string) time.Time {
	t.Helper()
	parsed, err := time.Parse(time.RFC3339Nano, value)
	if err != nil {
		t.Fatalf("parse time %q: %v", value, err)
	}
	return parsed
}
