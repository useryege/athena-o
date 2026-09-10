package tradersync

import (
	"bytes"
	"fmt"
	"math/big"

	ethabi "github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	sourceabi "github.com/useryege/athena/internal/tradersync/abi"
	tsmodel "github.com/useryege/athena/internal/tradersync/types"
)

var orderFilledEvents = func() map[string]ethabi.Event {
	result := map[string]ethabi.Event{}
	for version, raw := range map[string][]byte{CoreExchangeVersion: sourceabi.CoreExchange, NegRiskExchangeVersion: sourceabi.CoreExchange, ComboExchangeVersion: sourceabi.CombosExchange} {
		parsed, err := ethabi.JSON(bytes.NewReader(raw))
		if err != nil {
			panic(err)
		}
		result[version] = parsed.Events["OrderFilled"]
	}
	return result
}()

// DecodeOwnTrade returns the order's actual funds wallet (topics[2]), regardless
// of its maker/taker trading role. The collector must match that wallet against
// its target set; a counterparty-only match does not belong to the target.
// Callers in the live pipeline must first verify canonical/version evidence.
func DecodeOwnTrade(log types.Log, version string) (tsmodel.Trade, error) {
	registered, ok := sourceVersions[version]
	if !ok || log.Address != registered.deployment.address || log.Removed {
		return tsmodel.Trade{}, fmt.Errorf("unrecognized execution source")
	}
	event := orderFilledEvents[version]
	if len(log.Topics) != 4 || log.Topics[0] != event.ID || len(log.Data) != 7*32 {
		return tsmodel.Trade{}, fmt.Errorf("not a canonical OrderFilled payload")
	}
	for _, topic := range log.Topics[2:] {
		if !bytes.Equal(topic[:12], make([]byte, 12)) {
			return tsmodel.Trade{}, fmt.Errorf("noncanonical indexed address")
		}
	}
	if !bytes.Equal(log.Data[:31], make([]byte, 31)) || log.Data[31] > 1 {
		return tsmodel.Trade{}, fmt.Errorf("unsupported OrderFilled side")
	}
	fields, err := event.Inputs.NonIndexed().Unpack(log.Data)
	if err != nil {
		return tsmodel.Trade{}, fmt.Errorf("decode OrderFilled: %w", err)
	}
	side := "BUY"
	collateral, shares := fields[2].(*big.Int), fields[3].(*big.Int)
	if fields[0].(uint8) == 1 {
		side = "SELL"
		collateral, shares = shares, collateral
	}
	trade := tsmodel.Trade{Wallet: common.BytesToAddress(log.Topics[2][12:]), Exchange: log.Address, Side: side, PositionID: fields[1].(*big.Int).String(), CollateralRaw: collateral.String(), SharesRaw: shares.String(), FeeRaw: fields[4].(*big.Int).String(), CollateralSymbol: registered.collateralSymbol, CollateralDecimals: registered.collateralDecimals, SharesDecimals: registered.sharesDecimals, SourceVersion: version}
	// Empty ratio fields mean unavailable. Preserve zero quantities as real facts.
	if shares.Sign() != 0 {
		trade.PriceNumerator = trade.CollateralRaw
		trade.PriceDenominator = trade.SharesRaw
	}
	return trade, nil
}
