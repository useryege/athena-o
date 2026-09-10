package types

import (
	"time"

	"github.com/ethereum/go-ethereum/common"
)

type Trade struct {
	Wallet                                                               common.Address
	Side, PositionID, CollateralRaw, SharesRaw, FeeRaw, CollateralSymbol string
	CollateralDecimals, SharesDecimals                                   uint8
	Exchange                                                             common.Address
	SourceVersion, PriceNumerator, PriceDenominator                      string
}

type CanonicalEvidence struct {
	Status               string
	BlockHash            common.Hash
	SettledAt, CheckedAt time.Time
	Reason               string
}
