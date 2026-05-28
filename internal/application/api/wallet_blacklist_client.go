package api

import (
	"context"

	"github.com/ethereum/go-ethereum/common"
	walletapiclient "github.com/useryege/athena/internal/wallet/apiclient"
)

type walletBlacklistLister interface {
	ListWalletBlacklist(ctx context.Context) ([]common.Address, error)
}

type walletBlacklistClientLister struct {
	clientSet walletapiclient.Clientset
}

func newWalletBlacklistClientLister(clientSet walletapiclient.Clientset) walletBlacklistLister {
	if clientSet == nil {
		return nil
	}
	return &walletBlacklistClientLister{clientSet: clientSet}
}

func (l *walletBlacklistClientLister) ListWalletBlacklist(ctx context.Context) ([]common.Address, error) {
	if l == nil || l.clientSet == nil {
		return nil, nil
	}
	closer, client, err := l.clientSet.NewWalletServiceClient()
	if err != nil {
		return nil, err
	}
	defer closer.Close()

	resp, err := client.ListWalletBlacklistEntries(ctx, &walletapiclient.ListWalletBlacklistEntriesRequest{})
	if err != nil {
		return nil, err
	}
	items := resp.GetItems()
	wallets := make([]common.Address, 0, len(items))
	for _, item := range items {
		if item == nil || !common.IsHexAddress(item.GetWallet()) {
			continue
		}
		wallets = append(wallets, common.HexToAddress(item.GetWallet()))
	}
	return wallets, nil
}
