package tradersync

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	tm "github.com/useryege/athena/internal/tradersync/types"
	pm "github.com/useryege/athena/util/polymarket"
	"math/big"
	"os"
	"strings"
	"testing"
	"time"
)

func TestComboNoMeansComplementOfConjunction(t *testing.T) {
	if got := ComboRelationship("NO"); got != "NOT(AND(legs))" {
		t.Fatal(got)
	}
	if ComboRelationship("YES") != "AND(legs)" || ComboRelationship("maybe") != "" || ComboRelationship("yes") != "" {
		t.Fatal("unverified relationship")
	}
}

type moduleNode struct {
	header                        *types.Header
	legs                          []*big.Int
	failLegs, failBinary, upgrade bool
	calls                         int
}

func (n *moduleNode) ChainID(context.Context) (*big.Int, error) { return big.NewInt(137), nil }
func (n *moduleNode) HeaderByHash(_ context.Context, h common.Hash) (*types.Header, error) {
	if n.header.Hash() != h {
		return nil, errors.New("wrong hash")
	}
	return n.header, nil
}
func (n *moduleNode) CodeAtHash(_ context.Context, a common.Address, _ common.Hash) ([]byte, error) {
	name := "combo-proxy"
	if a == combinatorialDeployment.implementation {
		name = "combinatorial-module"
	}
	if a == binaryDeployment.implementation {
		name = "binary-module"
	}
	if n.failBinary && (a == binaryDeployment.address || a == binaryDeployment.implementation) {
		return nil, errors.New("missing binary version")
	}
	b, e := os.ReadFile("testdata/runtime/" + name + ".hex")
	if e != nil {
		return nil, e
	}
	return common.FromHex(strings.TrimSpace(string(b))), nil
}
func (n *moduleNode) StorageAtHash(_ context.Context, a common.Address, slot, h common.Hash) ([]byte, error) {
	if slot != implementationSlot {
		return nil, errors.New("wrong slot")
	}
	impl := combinatorialDeployment.implementation
	if a == binaryDeployment.address {
		impl = binaryDeployment.implementation
	}
	return common.LeftPadBytes(impl.Bytes(), 32), nil
}
func (n *moduleNode) UpgradeLogs(context.Context, common.Hash, common.Address) ([]types.Log, error) {
	if n.upgrade {
		return []types.Log{{}}, nil
	}
	return []types.Log{}, nil
}
func (n *moduleNode) CallContractAtHash(_ context.Context, c ethereum.CallMsg, h common.Hash) ([]byte, error) {
	if h != n.header.Hash() {
		return nil, errors.New("not candidate hash")
	}
	n.calls++
	if *c.To == combinatorialDeployment.address {
		if n.failLegs {
			return nil, errors.New("getLegs unavailable")
		}
		return combinatorialABI.Methods["getLegs"].Outputs.Pack(n.legs)
	}
	if *c.To != binaryDeployment.address {
		return nil, errors.New("unexpected module")
	}
	if bytes.Equal(c.Data[:4], binaryABI.Methods["legacyConditionId"].ID) {
		return binaryABI.Methods["legacyConditionId"].Outputs.Pack([32]byte{})
	}
	return nil, errors.New("unexpected migration call")
}
func newModuleNode() *moduleNode {
	return &moduleNode{header: &types.Header{Number: big.NewInt(10), ParentHash: common.HexToHash("0xabc"), Difficulty: big.NewInt(0)}}
}
func TestComboGetLegsFailureAndVersionFailureRemainUnavailable(t *testing.T) {
	for _, versionFail := range []bool{false, true} {
		n := newModuleNode()
		n.failLegs = true
		n.upgrade = versionFail
		got := NewMetadataResolver(metadataGamma{}, n, &metadataMemory{}).Resolve(context.Background(), tm.Trade{SourceVersion: ComboExchangeVersion, PositionID: "1677415912060011761505225895152832125576889787899132551452768910625300545536"}, n.header.Hash())
		if got.LegsEvidence.Availability != "unavailable" || got.LegsEvidence.ReasonCode == "" || got.Legs != nil {
			t.Fatal(got)
		}
		if versionFail && n.calls != 0 {
			t.Fatal("called unverified module")
		}
	}
}
func TestComboPartialLegsAndModuleOneTwoDirectory(t *testing.T) {
	n := newModuleNode()
	a := new(big.Int).Lsh(big.NewInt(1), 248)
	b := new(big.Int).Lsh(big.NewInt(2), 248)
	b.Add(b, big.NewInt(1))
	unknown := new(big.Int).Lsh(big.NewInt(4), 248)
	n.legs = []*big.Int{a, b, unknown}
	index := map[string][]pm.ComboMarket{}
	for i, p := range []string{a.String(), b.String()} {
		index[p] = []pm.ComboMarket{{ID: fmt.Sprint(i + 1), PositionIDs: []string{p}, ConditionID: "condition"}}
	}
	g := metadataGamma{byID: func(id int64) (*pm.Market, error) {
		p := a.String()
		if id == 2 {
			p = b.String()
		}
		ids := []string{p}
		return &pm.Market{ID: fmt.Sprint(id), ConditionID: textptr("condition"), PositionIDs: &ids, Outcomes: textptr(`["Actual label"]`)}, nil
	}}
	got := NewMetadataResolver(g, n, &metadataMemory{index: index}).Resolve(context.Background(), tm.Trade{SourceVersion: ComboExchangeVersion, PositionID: "1677415912060011761505225895152832125576889787899132551452768910625300545537"}, n.header.Hash())
	if got.LegsEvidence.Availability != "available" || len(got.Legs) != 3 || got.Relationship != "NOT(AND(legs))" {
		t.Fatal(got)
	}
	for _, leg := range got.Legs[:2] {
		if leg.Market.Availability != "available" || leg.Market.Outcome != "Actual label" || leg.Market.Source != "combo_directory+gamma" {
			t.Fatal(leg)
		}
	}
	if got.Legs[2].PositionID != unknown.String() || got.Legs[2].Market.Availability != "unavailable" {
		t.Fatal(got)
	}
}

type recordedMigrationNode struct {
	*moduleNode
	responses map[string][]byte
	seen      map[string]bool
}

func (n *recordedMigrationNode) CallContractAtHash(ctx context.Context, c ethereum.CallMsg, h common.Hash) ([]byte, error) {
	if h != n.header.Hash() {
		return nil, errors.New("hash changed")
	}
	if *c.To == combinatorialDeployment.address {
		return n.moduleNode.CallContractAtHash(ctx, c, h)
	}
	if *c.To != binaryDeployment.address {
		return nil, errors.New("wrong module")
	}
	key := hexutil.Encode(c.Data)
	v, ok := n.responses[key]
	if !ok {
		return nil, fmt.Errorf("calldata differs from recorded migration: %s", key)
	}
	n.seen[key] = true
	return v, nil
}
func TestComboRecordedTwoMigratedBinaryLegs(t *testing.T) {
	data, e := os.ReadFile("testdata/metadata.json")
	if e != nil {
		t.Fatal(e)
	}
	var f struct {
		Position string   `json:"position_id"`
		Legs     []string `json:"decoded_legs"`
		Mapping  struct {
			Requests []struct {
				Params []json.RawMessage `json:"params"`
			} `json:"requests"`
			Responses []struct {
				Result string `json:"result"`
			} `json:"responses"`
		} `json:"binary_leg_mapping"`
		Markets []struct {
			Response pm.Market `json:"response"`
		} `json:"gamma_by_market_id"`
	}
	if e = json.Unmarshal(data, &f); e != nil {
		t.Fatal(e)
	}
	n := &recordedMigrationNode{moduleNode: newModuleNode(), responses: map[string][]byte{}, seen: map[string]bool{}}
	for _, id := range f.Legs {
		v, _ := new(big.Int).SetString(id, 10)
		n.legs = append(n.legs, v)
	}
	for i, req := range f.Mapping.Requests {
		var call struct {
			Data string `json:"data"`
		}
		if e = json.Unmarshal(req.Params[0], &call); e != nil {
			t.Fatal(e)
		}
		n.responses[call.Data] = common.FromHex(f.Mapping.Responses[i].Result)
	}
	g := metadataGamma{list: func(o pm.ListMarketsOptions) ([]pm.Market, error) {
		if o.Closed == nil {
			t.Fatal("missing closed selector")
		}
		if !*o.Closed {
			return []pm.Market{}, nil
		}
		var matches []pm.Market
		for _, record := range f.Markets {
			ids, _ := decodeMarketStrings(record.Response.ClobTokenIDs)
			for _, id := range ids {
				if id == o.ClobTokenIDs[0] {
					matches = append(matches, record.Response)
				}
			}
		}
		return matches, nil
	}}
	got := NewMetadataResolver(g, n, &metadataMemory{}).Resolve(context.Background(), tm.Trade{SourceVersion: ComboExchangeVersion, PositionID: f.Position, Side: "BUY"}, n.header.Hash())
	if len(got.Legs) != 2 || got.LegsEvidence.Availability != "available" || got.Relationship != "AND(legs)" {
		t.Fatal(got)
	}
	for i, want := range []struct{ id, outcome string }{{"3978602", "Yes"}, {"3978659", "VfB Stuttgart"}} {
		leg := got.Legs[i]
		if leg.Market.ID != want.id || leg.Market.Outcome != want.outcome || leg.Market.Availability != "available" || leg.PositionID != f.Legs[i] || !strings.HasPrefix(leg.Market.Source, "binary_migration:") {
			t.Fatal(leg)
		}
	}
	if len(n.seen) != 4 {
		t.Fatal("both migrations not verified", n.seen)
	}
	conflictGamma := metadataGamma{list: func(o pm.ListMarketsOptions) ([]pm.Market, error) {
		matches, err := g.list(o)
		for i := range matches {
			matches[i].ConditionID = textptr("0xconflicting")
		}
		return matches, err
	}, byID: func(int64) (*pm.Market, error) {
		t.Error("conflicting legacy evidence hidden by directory fallback")
		return nil, errors.New("must not fallback")
	}}
	conflictStore := &metadataMemory{index: map[string][]pm.ComboMarket{f.Legs[0]: {{ID: "1", ConditionID: "other", PositionIDs: []string{f.Legs[0]}}}}}
	conflicting := NewMetadataResolver(conflictGamma, n, conflictStore).Resolve(context.Background(), tm.Trade{SourceVersion: ComboExchangeVersion, PositionID: f.Position}, n.header.Hash())
	if len(conflicting.Legs) != 2 || conflicting.Legs[0].Market.ReasonCode != "condition_conflict" {
		t.Fatal(conflicting)
	}
}
func TestComboUnknownOutcomeAndDirectoryConflictsRemainVisible(t *testing.T) {
	n := newModuleNode()
	position := new(big.Int).Lsh(big.NewInt(2), 248)
	n.legs = []*big.Int{position}
	p := position.String()
	cases := []struct {
		name, id, condition string
		positions           *[]string
		outcomes            string
		reason              string
	}{
		{"missing positions", "1", "c", nil, `["Yes"]`, "directory_gamma_conflict"},
		{"condition conflict", "1", "wrong", &[]string{p}, `["Yes"]`, "directory_gamma_conflict"},
		{"wrong position", "1", "c", &[]string{"1"}, `["Yes"]`, "directory_gamma_conflict"},
		{"unknown outcome", "1", "c", &[]string{p}, `[""]`, "outcome_mapping_unavailable"},
		{"overflow market ID", "9223372036854775808", "c", &[]string{p}, `["Yes"]`, "market_id_out_of_range"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			g := metadataGamma{byID: func(int64) (*pm.Market, error) {
				return &pm.Market{ID: tc.id, ConditionID: &tc.condition, PositionIDs: tc.positions, Outcomes: &tc.outcomes}, nil
			}}
			store := &metadataMemory{index: map[string][]pm.ComboMarket{p: {{ID: tc.id, ConditionID: "c", PositionIDs: []string{p}}}}}
			got := NewMetadataResolver(g, n, store).Resolve(context.Background(), tm.Trade{SourceVersion: ComboExchangeVersion, PositionID: "1677415912060011761505225895152832125576889787899132551452768910625300545538"}, n.header.Hash())
			if got.Relationship != "" || got.Market.ReasonCode != "unknown_combo_outcome" || len(got.Legs) != 1 || got.Legs[0].Market.ReasonCode != tc.reason {
				t.Fatal(got)
			}
		})
	}
}

type cancellingDirectory struct{ cancel context.CancelFunc }

func (s cancellingDirectory) RefreshComboPage(ctx context.Context, fetch func(context.Context, string, int) (pm.ComboMarketPage, error)) (time.Time, error) {
	_, err := fetch(ctx, "resume", 100)
	s.cancel()
	return time.Now().Add(time.Hour), err
}

type failingDirectoryClient struct{}

func (failingDirectoryClient) ListComboMarkets(context.Context, string, int) (pm.ComboMarketPage, error) {
	return pm.ComboMarketPage{}, errors.New("directory offline")
}
func TestComboDirectoryRunReportsFailureAndCancels(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var reported error
	r := NewDirectoryRefresher(cancellingDirectory{cancel}, failingDirectoryClient{}, func(err error) { reported = err })
	if err := r.Run(ctx); !errors.Is(err, context.Canceled) || reported == nil {
		t.Fatal(err, reported)
	}
}

func TestComboProgressPublishesAllPositionsBeforeLegIO(t *testing.T) {
	n := newModuleNode()
	a := new(big.Int).Lsh(big.NewInt(2), 248)
	b := new(big.Int).Add(a, big.NewInt(1))
	n.legs = []*big.Int{a, b}
	entered := make(chan struct{})
	release := make(chan struct{})
	done := make(chan tm.TradeMetadata, 1)
	progress := make(chan tm.TradeMetadata, 10)
	memory := &metadataMemory{index: map[string][]pm.ComboMarket{a.String(): {{ID: "1", ConditionID: "condition", PositionIDs: []string{a.String(), b.String()}}}, b.String(): {{ID: "1", ConditionID: "condition", PositionIDs: []string{a.String(), b.String()}}}}}
	first := true
	g := metadataGamma{byID: func(int64) (*pm.Market, error) {
		if first {
			first = false
			close(entered)
			<-release
		}
		return &pm.Market{ID: "1", ConditionID: textptr("condition"), PositionIDs: &[]string{a.String(), b.String()}, Outcomes: textptr(`["YES","NO"]`)}, nil
	}}
	resolver := NewMetadataResolver(g, n, memory)
	go func() {
		done <- resolver.ResolveProgress(context.Background(), tm.Trade{SourceVersion: ComboExchangeVersion, PositionID: "1677415912060011761505225895152832125576889787899132551452768910625300545536"}, n.header.Hash(), func(v tm.TradeMetadata) { progress <- v })
	}()
	select {
	case <-entered:
	case <-time.After(time.Second):
		t.Fatal("leg lookup not started")
	}
	defer func() {
		close(release)
		result := <-done
		if len(result.Legs) != 2 || result.Legs[0].PositionID != a.String() {
			t.Error("snapshot mutation corrupted resolver", result)
		}
	}()
	var snapshot tm.TradeMetadata
	for {
		select {
		case snapshot = <-progress:
		default:
			goto drained
		}
	}
drained:
	if len(snapshot.Legs) != 2 || snapshot.Legs[0].PositionID != a.String() || snapshot.Legs[1].PositionID != b.String() {
		t.Fatalf("no complete positions at in-flight boundary: %+v", snapshot)
	}
	if snapshot.Legs[0].Market.Availability != "unavailable" {
		t.Fatal("unverified leg published available")
	}
	snapshot.Legs[0].PositionID = "caller mutation"
}
