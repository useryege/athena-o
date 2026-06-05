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
	Tx          *types.Transaction
	Source      ProjectDiscoverySource
}

type ProjectDiscoverySource string

const (
	ProjectDiscoverySourceCatchUp     ProjectDiscoverySource = "catch_up"
	ProjectDiscoverySourceFollowHeads ProjectDiscoverySource = "follow_heads"
	ProjectDiscoverySourcePairSwap    ProjectDiscoverySource = "pair_swap"
)
