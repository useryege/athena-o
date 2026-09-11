package tradersync

import (
	"context"
	"errors"
	"github.com/ethereum/go-ethereum/common"
	tm "github.com/useryege/athena/internal/tradersync/types"
	pm "github.com/useryege/athena/util/polymarket"
	"reflect"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

type metadataMemory struct {
	mu     sync.Mutex
	values map[string]tm.TradeMetadata
	index  map[string][]pm.ComboMarket
}

func (m *metadataMemory) LoadMetadata(_ context.Context, key string) (tm.TradeMetadata, bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	v, ok := m.values[key]
	return v, ok, nil
}
func (m *metadataMemory) SaveMetadata(_ context.Context, key string, v tm.TradeMetadata) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.values == nil {
		m.values = map[string]tm.TradeMetadata{}
	}
	m.values[key] = v
	return nil
}
func (m *metadataMemory) LookupComboPosition(_ context.Context, p string) ([]pm.ComboMarket, error) {
	return m.index[p], nil
}

type metadataGamma struct {
	list func(pm.ListMarketsOptions) ([]pm.Market, error)
	byID func(int64) (*pm.Market, error)
}

func (m metadataGamma) ListMarkets(_ context.Context, o pm.ListMarketsOptions) ([]pm.Market, error) {
	return m.list(o)
}
func (m metadataGamma) GetMarketByID(_ context.Context, id int64, _ pm.GetMarketOptions) (*pm.Market, error) {
	return m.byID(id)
}
func textptr(s string) *string { return &s }
func TestMarketExactClosedTokenIndexAndLateEnrichment(t *testing.T) {
	huge := "999999999999999999999999999999999999999999999999999999999"
	calls := 0
	missing := true
	g := metadataGamma{list: func(o pm.ListMarketsOptions) ([]pm.Market, error) {
		calls++
		if o.Closed == nil || len(o.ClobTokenIDs) != 1 || o.ClobTokenIDs[0] != huge {
			t.Fatal(o)
		}
		if !*o.Closed || missing {
			return []pm.Market{}, nil
		}
		return []pm.Market{{ID: "8", Question: textptr("closed match"), Slug: textptr("closed-match"), ConditionID: textptr("0x123"), ClobTokenIDs: textptr(`["123","` + huge + `"]`), Outcomes: textptr(`["wrong","VfB Stuttgart"]`)}}, nil
	}}
	r := NewMetadataResolver(g, nil, &metadataMemory{})
	trade := tm.Trade{SourceVersion: CoreExchangeVersion, PositionID: huge, Side: "SELL", CollateralRaw: "99", SharesRaw: "100", FeeRaw: "1"}
	original := trade
	if got := r.Resolve(context.Background(), trade, common.Hash{}); got.Market.Availability != "unavailable" {
		t.Fatal(got)
	}
	missing = false
	got := r.Resolve(context.Background(), trade, common.Hash{})
	if got.Market.Availability != "available" || got.Market.Outcome != "VfB Stuttgart" || got.Market.PositionID != huge || got.Market.ID != "8" {
		t.Fatal(got)
	}
	n := calls
	if cached := r.Resolve(context.Background(), trade, common.Hash{}); cached.Market.ID != "8" || calls != n {
		t.Fatal(cached, calls)
	}
	if !reflect.DeepEqual(trade, original) {
		t.Fatal("metadata rewrote trade")
	}
}
func TestMarketRejectsAmbiguousOrMalformedOutcome(t *testing.T) {
	for _, outcomes := range []string{`[]`, `["Yes",""]`, `null`, `[1,2]`} {
		g := metadataGamma{list: func(pm.ListMarketsOptions) ([]pm.Market, error) {
			return []pm.Market{{ID: "1", ConditionID: textptr("0x1"), ClobTokenIDs: textptr(`["1","2"]`), Outcomes: textptr(outcomes)}}, nil
		}}
		got := NewMetadataResolver(g, nil, &metadataMemory{}).Resolve(context.Background(), tm.Trade{SourceVersion: CoreExchangeVersion, PositionID: "2"}, common.Hash{})
		if got.Market.Availability != "unavailable" || got.Market.ReasonCode == "" {
			t.Fatal(got)
		}
	}
	g := metadataGamma{list: func(pm.ListMarketsOptions) ([]pm.Market, error) { return nil, errors.New("offline") }}
	if got := NewMetadataResolver(g, nil, &metadataMemory{}).Resolve(context.Background(), tm.Trade{SourceVersion: CoreExchangeVersion, PositionID: "2"}, common.Hash{}); got.Market.Availability != "unavailable" {
		t.Fatal(got)
	}
}

type waitingGamma struct {
	entered chan struct{}
	release chan struct{}
	active  atomic.Int32
	peak    atomic.Int32
}

func (g *waitingGamma) ListMarkets(ctx context.Context, o pm.ListMarketsOptions) ([]pm.Market, error) {
	if *o.Closed {
		return []pm.Market{}, nil
	}
	n := g.active.Add(1)
	defer g.active.Add(-1)
	for {
		old := g.peak.Load()
		if n <= old || g.peak.CompareAndSwap(old, n) {
			break
		}
	}
	g.entered <- struct{}{}
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-g.release:
	}
	return []pm.Market{{ID: "1", ConditionID: textptr("condition"), ClobTokenIDs: textptr(`["1"]`), Outcomes: textptr(`["Yes"]`)}}, nil
}
func (*waitingGamma) GetMarketByID(context.Context, int64, pm.GetMarketOptions) (*pm.Market, error) {
	return nil, errors.New("unused")
}
func TestMarketConcurrencyFourAndCallerOwnsBudget(t *testing.T) {
	g := &waitingGamma{entered: make(chan struct{}, 8), release: make(chan struct{})}
	r := NewMetadataResolver(g, nil, nil)
	done := make(chan tm.TradeMetadata, 8)
	for i := 0; i < 8; i++ {
		go func() {
			done <- r.Resolve(context.Background(), tm.Trade{SourceVersion: CoreExchangeVersion, PositionID: "1"}, common.Hash{})
		}()
	}
	for i := 0; i < 4; i++ {
		select {
		case <-g.entered:
		case <-time.After(time.Second):
			t.Fatal("four workers not entered")
		}
	}
	// Caller context intentionally permits a provider response after two seconds.
	select {
	case result := <-done:
		t.Fatal("resolver imposed premature budget", result)
	case <-time.After(2100 * time.Millisecond):
	}
	if g.peak.Load() != 4 {
		t.Fatal(g.peak.Load())
	}
	close(g.release)
	for i := 0; i < 8; i++ {
		select {
		case got := <-done:
			if got.Market.Availability != "available" {
				t.Fatal(got)
			}
		case <-time.After(time.Second):
			t.Fatal("stalled")
		}
	}
	if g.peak.Load() > 4 {
		t.Fatal("concurrency exceeded", g.peak.Load())
	}
}
