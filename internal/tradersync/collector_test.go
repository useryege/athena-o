package tradersync

import (
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"testing"
)

func TestCollectorFiltersOnlyActualWalletTopicAndDeduplicates(t *testing.T) {
	wallets := make([]common.Address, 0, 202)
	for i := 1; i <= 201; i++ {
		wallets = append(wallets, common.HexToAddress(fmt.Sprintf("0x%040x", i)))
	}
	wallets = append(wallets, wallets[0])
	filters := liveFilters(wallets)
	if len(filters) != 6 {
		t.Fatal("expected three chunks per source group", len(filters))
	}
	for _, filter := range filters {
		if filter.FromBlock != nil || filter.ToBlock != nil || filter.BlockHash != nil {
			t.Fatal("range backfill filter")
		}
		if len(filter.Topics) != 3 || filter.Topics[1] != nil || len(filter.Topics[2]) > 100 {
			t.Fatal("wallet must only occupy topics[2]", filter.Topics)
		}
		if len(filter.Addresses) == 2 {
			if filter.Addresses[0] != sourceVersions[CoreExchangeVersion].deployment.address || filter.Addresses[1] != sourceVersions[NegRiskExchangeVersion].deployment.address {
				t.Fatal(filter.Addresses)
			}
		} else if len(filter.Addresses) != 1 || filter.Addresses[0] != sourceVersions[ComboExchangeVersion].deployment.address {
			t.Fatal(filter.Addresses)
		}
	}
}
