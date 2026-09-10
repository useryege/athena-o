package tradersync

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/useryege/athena/internal/tradersync/activity"
	"math/big"
	"net/url"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	tm "github.com/useryege/athena/internal/tradersync/types"
	pm "github.com/useryege/athena/util/polymarket"
)

type MetadataGamma interface {
	ListMarkets(context.Context, pm.ListMarketsOptions) ([]pm.Market, error)
	GetMarketByID(context.Context, int64, pm.GetMarketOptions) (*pm.Market, error)
}
type MetadataRPC interface {
	VersionRPC
	CallContractAtHash(context.Context, ethereum.CallMsg, common.Hash) ([]byte, error)
}
type MetadataStore interface {
	LoadMetadata(context.Context, string) (tm.TradeMetadata, bool, error)
	SaveMetadata(context.Context, string, tm.TradeMetadata) error
	LookupComboPosition(context.Context, string) ([]pm.ComboMarket, error)
}
type MetadataResolver struct {
	gamma MetadataGamma
	node  MetadataRPC
	store MetadataStore
	slots chan struct{}
}

func NewMetadataResolver(gamma MetadataGamma, node MetadataRPC, store MetadataStore) *MetadataResolver {
	return &MetadataResolver{gamma: gamma, node: node, store: store, slots: make(chan struct{}, 4)}
}
func metadataEvidence(availability, reason, source string) tm.Evidence {
	return tm.Evidence{Availability: availability, ReasonCode: reason, Source: source, QueriedAt: time.Now().UTC()}
}
func missingMarket(position, reason, source string) tm.MarketRef {
	return tm.MarketRef{Evidence: metadataEvidence("unavailable", reason, source), PositionID: position}
}

// Resolve only enriches display data. The activity caller owns its post-finality
// wait budget; background enrichment may keep working with a longer context.
func (r *MetadataResolver) Resolve(ctx context.Context, trade tm.Trade, blockHash common.Hash) tm.TradeMetadata {
	return r.ResolveProgress(ctx, trade, blockHash, nil)
}

func (r *MetadataResolver) ResolveProgress(ctx context.Context, trade tm.Trade, blockHash common.Hash, publish func(tm.TradeMetadata)) tm.TradeMetadata {
	unavailable := tm.TradeMetadata{Market: missingMarket(trade.PositionID, "metadata_unavailable", "gamma"), LegsEvidence: metadataEvidence("unavailable", "not_combo", "gamma")}
	if trade.SourceVersion == ComboExchangeVersion {
		unavailable.LegsEvidence.ReasonCode = "metadata_unavailable"
	}
	select {
	case r.slots <- struct{}{}:
		defer func() { <-r.slots }()
	case <-ctx.Done():
		unavailable.Market.ReasonCode = "cancelled"
		return unavailable
	}
	if _, ok := decimalID(trade.PositionID); !ok {
		unavailable.Market.ReasonCode = "invalid_position_id"
		return unavailable
	}
	if trade.SourceVersion != CoreExchangeVersion && trade.SourceVersion != NegRiskExchangeVersion && trade.SourceVersion != ComboExchangeVersion {
		unavailable.Market.ReasonCode = "unknown_source_version"
		return unavailable
	}
	key := activity.MetadataKey(trade, blockHash.Hex())
	latest := unavailable
	emit := func(v tm.TradeMetadata) {
		latest = activity.MergeMetadata(latest, v)
		if publish != nil {
			publish(activity.CloneMetadata(latest))
		}
	}
	if r.store != nil {
		cached, found, err := r.store.LoadMetadata(ctx, key)
		if err == nil && found {
			emit(cached)
			if completeMetadata(cached, trade.SourceVersion == ComboExchangeVersion) {
				return activity.CloneMetadata(cached)
			}
		}
	}
	result := unavailable
	if trade.SourceVersion == ComboExchangeVersion {
		result = r.resolveCombo(ctx, trade.PositionID, blockHash, emit)
	} else if r.gamma != nil {
		result.Market = r.resolveToken(ctx, trade.PositionID, "")
	}
	emit(result)
	result = activity.CloneMetadata(latest)
	if r.store != nil {
		if err := r.store.SaveMetadata(ctx, key, result); err != nil && result.Market.Availability != "available" && !strings.Contains(result.Market.ReasonCode, "conflict") {
			result.Market.ReasonCode = "metadata_cache_unavailable"
		}
	}
	return result
}
func completeMetadata(m tm.TradeMetadata, combo bool) bool {
	return activity.CompleteMetadata(m, combo)
}
func decimalID(s string) (*big.Int, bool) {
	if s == "" || (len(s) > 1 && s[0] == '0') {
		return nil, false
	}
	for _, c := range s {
		if c < '0' || c > '9' {
			return nil, false
		}
	}
	n, ok := new(big.Int).SetString(s, 10)
	return n, ok && n.Sign() >= 0 && n.BitLen() <= 256
}
func decodeMarketStrings(s *string) ([]string, error) {
	if s == nil {
		return nil, fmt.Errorf("missing field")
	}
	var v []string
	if err := json.Unmarshal([]byte(*s), &v); err != nil || v == nil {
		return nil, fmt.Errorf("invalid array")
	}
	return v, nil
}
func exactMarket(m pm.Market, position string, ids []string) (tm.MarketRef, error) {
	outcomes, err := decodeMarketStrings(m.Outcomes)
	if err != nil || len(ids) != len(outcomes) {
		return tm.MarketRef{}, fmt.Errorf("outcome_mapping_unavailable")
	}
	index := -1
	for i, id := range ids {
		if id == position {
			if index >= 0 {
				return tm.MarketRef{}, fmt.Errorf("ambiguous_position")
			}
			index = i
		}
	}
	if index < 0 || strings.TrimSpace(outcomes[index]) == "" {
		return tm.MarketRef{}, fmt.Errorf("outcome_mapping_unavailable")
	}
	if m.ID == "" || m.ConditionID == nil || *m.ConditionID == "" {
		return tm.MarketRef{}, fmt.Errorf("market_identity_unavailable")
	}
	ref := tm.MarketRef{Evidence: metadataEvidence("available", "", "gamma"), ID: m.ID, ConditionID: *m.ConditionID, PositionID: position, Outcome: outcomes[index]}
	if m.Question != nil {
		ref.Title = *m.Question
	}
	if m.Slug != nil && *m.Slug != "" {
		ref.URL = "https://polymarket.com/event/" + url.PathEscape(*m.Slug)
	}
	return ref, nil
}
func (r *MetadataResolver) resolveToken(ctx context.Context, token, condition string) tm.MarketRef {
	missing := missingMarket(token, "market_not_found", "gamma")
	var match *tm.MarketRef
	for _, closed := range []bool{false, true} {
		callCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		markets, err := r.gamma.ListMarkets(callCtx, pm.ListMarketsOptions{ClobTokenIDs: []string{token}, Closed: &closed})
		cancel()
		if err != nil {
			missing.ReasonCode = "gamma_unavailable"
			return missing
		}
		for _, market := range markets {
			ids, err := decodeMarketStrings(market.ClobTokenIDs)
			if err != nil {
				continue
			}
			hit := false
			for _, id := range ids {
				if id == token {
					hit = true
				}
			}
			if !hit {
				continue
			}
			if condition != "" && (market.ConditionID == nil || !strings.EqualFold(*market.ConditionID, condition)) {
				missing.ReasonCode = "condition_conflict"
				return missing
			}
			ref, err := exactMarket(market, token, ids)
			if err != nil {
				missing.ReasonCode = err.Error()
				return missing
			}
			if match != nil && (match.ID != ref.ID || match.ConditionID != ref.ConditionID || match.Outcome != ref.Outcome) {
				missing.ReasonCode = "ambiguous_market"
				return missing
			}
			match = &ref
		}
	}
	if match != nil {
		return *match
	}
	return missing
}
