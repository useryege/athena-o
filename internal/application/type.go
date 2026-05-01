package application

import "github.com/ethereum/go-ethereum/core/types"

// CreationTxEvent carries minimal data for contract creation transactions.
type Project struct {
	BlockNumber uint64
	Tx          *types.Transaction
}
