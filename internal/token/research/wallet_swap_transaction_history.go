package research

import "github.com/useryege/athena/internal/token/shared"

type WalletSwapTransaction struct {
	TransactionHash shared.Hash
}

type WalletSwapTransactionHistory struct {
	Transactions            []WalletSwapTransaction
	IndexedThroughBlock     uint64
	IndexedThroughTimestamp uint64
}
