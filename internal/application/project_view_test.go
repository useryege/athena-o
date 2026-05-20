package application

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
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

	list := projectToView(project, false)
	if list == nil {
		t.Fatalf("list view is nil")
	}
	if len(list.Meta.GenesisWallets) != 0 {
		t.Fatalf("list genesis wallets len = %d, want 0", len(list.Meta.GenesisWallets))
	}
}
