package model

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

type DiscoveredProjectCandidate struct {
	ChainID     int64
	BlockTime   uint64
	BlockNumber uint64
	TxIndex     uint64
	Contract    common.Address
	Creator     common.Address
	TxHash      common.Hash
	WethPair    common.Address
	UsdtPair    common.Address
	Tx          *types.Transaction
	Source      ProjectDiscoverySource
}

type TokenValidation struct {
	IsValidERC20 bool
	WethPair     common.Address
	UsdtPair     common.Address
}

type ProjectDiscoverySource string

const (
	ProjectDiscoverySourceCatchUp     ProjectDiscoverySource = "catch_up"
	ProjectDiscoverySourceFollowHeads ProjectDiscoverySource = "follow_heads"
	ProjectDiscoverySourcePairSwap    ProjectDiscoverySource = "pair_swap"
)
