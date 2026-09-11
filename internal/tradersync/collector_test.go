package tradersync

import (
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	liverpc "github.com/useryege/athena/internal/tradersync/rpc"
	"testing"
	"time"
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

func TestCheckpointCombinesOnlyFreshCoveredObservations(t *testing.T) {
	now := time.Now()
	session := &liverpc.Session{}
	pong := liverpc.PongObservation{Session: session, At: now.Add(-time.Second), Sequence: 1, Alive: true}
	latest := now.Add(-2 * time.Second)
	covered := now.Add(-3 * time.Second)
	wallets := []common.Address{common.HexToAddress("0x123")}
	got, e := combineCheckpoint(pong, latest, covered, now, 1, 2, 3, wallets)
	if e != nil || !got.At.Equal(latest) || got.Token != 1 || got.Epoch != 2 || got.FilterRevision != 3 {
		t.Fatalf("valid health combination missing: %+v %v", got, e)
	}
	for _, tc := range []struct {
		name string
		p    liverpc.PongObservation
		l, c time.Time
	}{{"old pong", liverpc.PongObservation{Session: session, At: now.Add(-21 * time.Second), Sequence: 1, Alive: true}, latest, covered}, {"old latest", pong, now.Add(-16 * time.Second), covered}, {"new coverage", pong, latest, now}, {"pong received before coverage but consumed after", pong, now, now.Add(-500 * time.Millisecond)}, {"closed", liverpc.PongObservation{Session: session, At: pong.At, Sequence: 1}, latest, covered}, {"no matched nonce", liverpc.PongObservation{Session: session, At: pong.At, Alive: true}, latest, covered}} {
		t.Run(tc.name, func(t *testing.T) {
			if _, e := combineCheckpoint(tc.p, tc.l, tc.c, now, 1, 2, 3, wallets); e == nil {
				t.Fatal("ineligible health combination accepted")
			}
		})
	}
}
