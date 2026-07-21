package research

import (
	"math/big"

	"github.com/useryege/athena/internal/token/shared"
)

type WalletFundingSourceTransaction struct {
	BlockNumber      uint64
	BlockHash        shared.Hash
	BlockTimestamp   uint64
	TransactionHash  shared.Hash
	TransactionIndex uint64
	From             shared.Address
	To               shared.Address
	ValueWei         *big.Int
}

type WalletFundingSourceHistory struct {
	Transactions            []WalletFundingSourceTransaction
	IndexedThroughBlock     uint64
	IndexedThroughTimestamp uint64
}
