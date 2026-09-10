package tradersync

import (
	"bytes"
	"context"
	"encoding/hex"
	"fmt"
	"github.com/ethereum/go-ethereum"
	ethabi "github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	sourceabi "github.com/useryege/athena/internal/tradersync/abi"
	tm "github.com/useryege/athena/internal/tradersync/types"
	pm "github.com/useryege/athena/util/polymarket"
	"math/big"
	"slices"
	"strconv"
	"strings"
	"time"
)

var combinatorialDeployment = contractDeployment{address: common.HexToAddress("0x30000034706C7d8e12009DAB006Be20000c031A8"), codeHash: common.HexToHash("0xaaa52c8cc8a0e3fd27ce756cc6b4e70c51423e9b597b11f32d3e49f8b1fc890d"), implementation: common.HexToAddress("0xf96968a44022b17240b42c557693e7c383d2d8a3"), implementationHash: common.HexToHash("0xddc204af9baf769695592bc704873010537c4e941018dd5eec7cd4b27763f9ca")}
var binaryDeployment = contractDeployment{address: common.HexToAddress("0x1000008dD9001B968442c1000017eaE6E0dA00Ba"), codeHash: combinatorialDeployment.codeHash, implementation: common.HexToAddress("0xf6428c0b5fa9361c0708cddb95468cf54c56e9a2"), implementationHash: common.HexToHash("0xe378ba6d5142d89ab97c857f762043f2a388b9d2049fe1b79b5694881a3d445a")}
var combinatorialABI = mustModuleABI(sourceabi.CombinatorialModule)
var binaryABI = mustModuleABI(sourceabi.BinaryModule)

func mustModuleABI(raw []byte) ethabi.ABI {
	a, e := ethabi.JSON(bytes.NewReader(raw))
	if e != nil {
		panic(e)
	}
	return a
}

func ComboRelationship(outcome string) string {
	switch outcome {
	case "YES":
		return "AND(legs)"
	case "NO":
		return "NOT(AND(legs))"
	}
	return ""
}
func (r *MetadataResolver) moduleVersion(ctx context.Context, d contractDeployment, hash common.Hash) error {
	if r.node == nil || hash == (common.Hash{}) {
		return fmt.Errorf("module_version_unavailable")
	}
	chain, e := r.node.ChainID(ctx)
	if e != nil || chain == nil || chain.Cmp(big.NewInt(137)) != 0 {
		return fmt.Errorf("module_chain_unavailable")
	}
	h, e := r.node.HeaderByHash(ctx, hash)
	if e != nil || h == nil || h.Number == nil || !h.Number.IsUint64() || h.Hash() != hash || h.ParentHash == (common.Hash{}) {
		return fmt.Errorf("module_header_unavailable")
	}
	return verifyDeploymentAtHashes(ctx, r.node, d, hash, h.ParentHash)
}
func (r *MetadataResolver) callModule(ctx context.Context, d contractDeployment, a ethabi.ABI, method string, hash common.Hash, args ...any) ([]any, error) {
	input, e := a.Pack(method, args...)
	if e != nil {
		return nil, e
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	output, e := r.node.CallContractAtHash(ctx, ethereum.CallMsg{To: &d.address, Data: input}, hash)
	if e != nil {
		return nil, e
	}
	values, e := a.Unpack(method, output)
	if e != nil {
		return nil, e
	}
	canonical, e := a.Methods[method].Outputs.Pack(values...)
	if e != nil || !bytes.Equal(canonical, output) {
		return nil, fmt.Errorf("noncanonical module response")
	}
	return values, nil
}
func conditionParts(position string) ([31]byte, [32]byte, byte, byte, error) {
	var cond [31]byte
	var padded [32]byte
	n, ok := decimalID(position)
	if !ok {
		return cond, padded, 0, 0, fmt.Errorf("invalid_position_id")
	}
	n.FillBytes(padded[:])
	outcome, module := padded[31], padded[0]
	copy(cond[:], padded[:31])
	padded[31] = 0
	return cond, padded, module, outcome, nil
}
func (r *MetadataResolver) resolveCombo(ctx context.Context, position string, hash common.Hash, publish func(tm.TradeMetadata)) tm.TradeMetadata {
	result := tm.TradeMetadata{Market: missingMarket(position, "combo_identity_unavailable", "combinatorial_module"), LegsEvidence: metadataEvidence("unavailable", "module_version_unavailable", "combinatorial_module")}
	cond, padded, module, outcome, e := conditionParts(position)
	if e != nil || module != 3 {
		result.LegsEvidence.ReasonCode = "unknown_combo_module"
		return result
	}
	if e = r.moduleVersion(ctx, combinatorialDeployment, hash); e != nil {
		return result
	}
	source := "combinatorial_module:" + combinatorialDeployment.implementation.Hex() + "@" + hash.Hex()
	result.Market.ConditionID = "0x" + hex.EncodeToString(padded[:31])
	result.Market.Source = source
	switch outcome {
	case 0:
		result.Market.Outcome = "YES"
	case 1:
		result.Market.Outcome = "NO"
	default:
		result.Market.ReasonCode = "unknown_combo_outcome"
	}
	if result.Market.Outcome != "" {
		result.Market.Evidence = metadataEvidence("available", "", source)
		result.Relationship = ComboRelationship(result.Market.Outcome)
	}
	values, e := r.callModule(ctx, combinatorialDeployment, combinatorialABI, "getLegs", hash, cond)
	if e != nil || len(values) != 1 {
		result.LegsEvidence = metadataEvidence("unavailable", "get_legs_unavailable", source)
		return result
	}
	legs, ok := values[0].([]*big.Int)
	if !ok || len(legs) == 0 {
		result.LegsEvidence = metadataEvidence("unavailable", "legs_not_registered", source)
		return result
	}
	result.LegsEvidence = metadataEvidence("available", "", source)
	result.Legs = make([]tm.ComboLeg, len(legs))
	for i, leg := range legs {
		id := leg.String()
		result.Legs[i] = tm.ComboLeg{PositionID: id, Market: missingMarket(id, "metadata_pending", source)}
	}
	publish(result)
	for i := range result.Legs {
		result.Legs[i].Market = r.resolveLeg(ctx, result.Legs[i].PositionID, hash)
		publish(result)
	}

	return result
}
func (r *MetadataResolver) resolveLeg(ctx context.Context, position string, hash common.Hash) tm.MarketRef {
	cond, padded, module, outcome, e := conditionParts(position)
	if e != nil || (module != 1 && module != 2) {
		return missingMarket(position, "unknown_leg_module", "combo_directory")
	}
	if module == 1 && outcome < 2 && r.moduleVersion(ctx, binaryDeployment, hash) == nil {
		values, e := r.callModule(ctx, binaryDeployment, binaryABI, "legacyConditionId", hash, cond)
		if e == nil && len(values) == 1 {
			legacy, ok := values[0].([32]byte)
			if ok && legacy != ([32]byte{}) {
				// getLegacyPositionId takes the structured canonical condition, NOT the
				// legacy condition returned by the getter; see the fixed migration source.
				tokens, e := r.callModule(ctx, binaryDeployment, binaryABI, "getLegacyPositionId", hash, padded, new(big.Int).SetUint64(uint64(outcome)))
				if e == nil && len(tokens) == 1 {
					token, ok := tokens[0].(*big.Int)
					if ok && token.Sign() > 0 && r.gamma != nil {
						ref := r.resolveToken(ctx, token.String(), common.BytesToHash(legacy[:]).Hex())
						ref.PositionID = position
						ref.Source = "binary_migration:" + binaryDeployment.implementation.Hex() + "@" + hash.Hex() + "+gamma"
						// A successful mapping with conflicting Gamma evidence must stay visible.
						if ref.Availability == "available" || ref.ReasonCode == "condition_conflict" || ref.ReasonCode == "ambiguous_market" || ref.ReasonCode == "ambiguous_position" {
							return ref
						}
					}
				}
			}
		}
	}
	return r.directoryMarket(ctx, position)
}
func (r *MetadataResolver) directoryMarket(ctx context.Context, position string) tm.MarketRef {
	missing := missingMarket(position, "directory_cache_miss", "combo_directory+gamma")
	if r.store == nil || r.gamma == nil {
		return missing
	}
	records, e := r.store.LookupComboPosition(ctx, position)
	if e != nil {
		missing.ReasonCode = "directory_unavailable"
		return missing
	}
	if len(records) == 0 {
		return missing
	}
	if len(records) != 1 {
		missing.ReasonCode = "directory_mapping_conflict"
		return missing
	}
	record := records[0]
	id, e := strconv.ParseInt(record.ID, 10, 64)
	if e != nil || id <= 0 {
		missing.ReasonCode = "market_id_out_of_range"
		return missing
	}
	callCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	market, e := r.gamma.GetMarketByID(callCtx, id, pm.GetMarketOptions{})
	if e != nil || market == nil {
		missing.ReasonCode = "gamma_unavailable"
		return missing
	}
	if market.ID != record.ID || market.PositionIDs == nil || !slices.Equal(*market.PositionIDs, record.PositionIDs) || market.ConditionID == nil || !strings.EqualFold(*market.ConditionID, record.ConditionID) {
		missing.ReasonCode = "directory_gamma_conflict"
		return missing
	}
	ref, e := exactMarket(*market, position, *market.PositionIDs)
	if e != nil {
		missing.ReasonCode = e.Error()
		return missing
	}
	ref.Source = "combo_directory+gamma"
	return ref
}

type ComboDirectoryClient interface {
	ListComboMarkets(context.Context, string, int) (pm.ComboMarketPage, error)
}
type ComboDirectoryStore interface {
	RefreshComboPage(context.Context, func(context.Context, string, int) (pm.ComboMarketPage, error)) (time.Time, error)
}
type DirectoryRefresher struct {
	store   ComboDirectoryStore
	client  ComboDirectoryClient
	onError func(error)
}

func NewDirectoryRefresher(store ComboDirectoryStore, client ComboDirectoryClient, onError func(error)) *DirectoryRefresher {
	return &DirectoryRefresher{store: store, client: client, onError: onError}
}
func (r *DirectoryRefresher) Run(ctx context.Context) error {
	if r.store == nil || r.client == nil {
		return fmt.Errorf("directory dependencies required")
	}
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		next, err := r.store.RefreshComboPage(ctx, r.client.ListComboMarkets)
		if err != nil && r.onError != nil {
			r.onError(err)
		}
		if ctx.Err() != nil {
			return ctx.Err()
		}
		wait := time.Until(next)
		if wait <= 0 {
			wait = time.Second
		}
		timer := time.NewTimer(wait)
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
		}
	}
}
