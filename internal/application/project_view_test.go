package application

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	athenacontract "github.com/useryege/athena/pkg/abi/ATHENA"
)

func TestProjectToViewGenesisWalletsVisibility(t *testing.T) {
	project := &Project{
		Meta: ProjectMeta{
			Contract: common.HexToAddress("0x1111111111111111111111111111111111111111"),
			GenesisWallets: []GenesisWalletMeta{
				{
					Wallet:    common.HexToAddress("0x2222222222222222222222222222222222222222"),
					NetAmount: big.NewInt(123),
					RatioBPS:  456,
					RankIndex: 0,
				},
			},
		},
		ChainState: athenacontract.AthenaProject{
			AssetState: athenacontract.AthenaAssetState{
				TokenBalance:  big.NewInt(1000),
				WethBalance:   big.NewInt(2000),
				UsdtBalance:   big.NewInt(3000),
				NativeBalance: big.NewInt(4000),
				UsdtValue:     big.NewInt(5000),
			},
			GenesisWalletAssetStates: []athenacontract.AthenaGenesisWalletAssetState{
				{
					Wallet: common.HexToAddress("0x2222222222222222222222222222222222222222"),
					AssetState: athenacontract.AthenaAssetState{
						TokenBalance:  big.NewInt(11),
						WethBalance:   big.NewInt(22),
						UsdtBalance:   big.NewInt(33),
						NativeBalance: big.NewInt(44),
						UsdtValue:     big.NewInt(55),
					},
				},
			},
		},
	}

	detail := projectToView(project, true)
	if detail == nil {
		t.Fatalf("detail view is nil")
	}
	if len(detail.Meta.GenesisWallets) != 1 {
		t.Fatalf("detail genesis wallets len = %d, want 1", len(detail.Meta.GenesisWallets))
	}
	if detail.Meta.GenesisWallets[0].Wallet != "0x2222222222222222222222222222222222222222" {
		t.Fatalf("detail wallet = %s, want 0x222...2222", detail.Meta.GenesisWallets[0].Wallet)
	}
	if detail.Meta.GenesisWallets[0].NetAmount != "123" {
		t.Fatalf("detail netAmount = %s, want 123", detail.Meta.GenesisWallets[0].NetAmount)
	}
	if detail.Meta.GenesisWallets[0].RatioBps != 456 {
		t.Fatalf("detail ratioBps = %d, want 456", detail.Meta.GenesisWallets[0].RatioBps)
	}
	if detail.Meta.GenesisWallets[0].Rank != 0 {
		t.Fatalf("detail rank = %d, want 0", detail.Meta.GenesisWallets[0].Rank)
	}
	if detail.ChainState.AssetState.UsdtValue != "5000" {
		t.Fatalf("detail assetState usdtValue = %s, want 5000", detail.ChainState.AssetState.UsdtValue)
	}
	if len(detail.ChainState.GenesisWalletAssetStates) != 1 {
		t.Fatalf("detail genesis wallet asset states len = %d, want 1", len(detail.ChainState.GenesisWalletAssetStates))
	}
	if detail.ChainState.GenesisWalletAssetStates[0].Wallet != "0x2222222222222222222222222222222222222222" {
		t.Fatalf("detail asset wallet = %s, want 0x222...2222", detail.ChainState.GenesisWalletAssetStates[0].Wallet)
	}
	if detail.ChainState.GenesisWalletAssetStates[0].AssetState.UsdtValue != "55" {
		t.Fatalf("detail asset usdtValue = %s, want 55", detail.ChainState.GenesisWalletAssetStates[0].AssetState.UsdtValue)
	}

	list := projectToView(project, false)
	if list == nil {
		t.Fatalf("list view is nil")
	}
	if len(list.Meta.GenesisWallets) != 0 {
		t.Fatalf("list genesis wallets len = %d, want 0", len(list.Meta.GenesisWallets))
	}
	if len(list.ChainState.GenesisWalletAssetStates) != 0 {
		t.Fatalf("list genesis wallet asset states len = %d, want 0", len(list.ChainState.GenesisWalletAssetStates))
	}
}
