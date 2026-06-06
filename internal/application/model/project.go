package model

import (
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	athenacontract "github.com/useryege/athena/pkg/abi/ATHENA"
	utilave "github.com/useryege/athena/util/ave"
)

type Project struct {
	Meta      ProjectMeta
	AveDetail *ProjectAveDetail
}

type ProjectRef struct {
	ChainID  int64          `json:"chain_id"`
	Contract common.Address `json:"contract"`
}

type ProjectMeta struct {
	ChainID                            int64
	BlockTime                          uint64
	BlockNumber                        uint64
	Contract                           common.Address
	Creator                            common.Address
	WethPair                           common.Address
	UsdtPair                           common.Address
	FetchAt                            time.Time
	TxHash                             common.Hash
	TxIndex                            uint64
	GenesisTx                          *types.Transaction
	ChainState                         athenacontract.AthenaProject
	GenesisWallets                     []GenesisWalletMeta
	GenesisWalletsFetchedAt            time.Time
	CreatorResult                      SimulateResult
	CreatorHistoricalProjects          []common.Address
	CreatorHistoricalProjectsFetchedAt time.Time
}

type ProjectAveDetail struct {
	ChainID   int64
	FetchedAt time.Time
	Data      utilave.TokenDetailData
}

type GenesisWalletMeta struct {
	Wallet    common.Address
	NetAmount *big.Int
	RatioBPS  int64
	RankIndex int32
}

func FormatOptionalTime(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.UTC().Format(time.RFC3339Nano)
}
