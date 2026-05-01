package application

import "github.com/ethereum/go-ethereum/common"

// CreationTxEvent carries minimal data for contract creation transactions.
type CreationTxEvent struct {
	BlockNumber uint64
	TxHash      common.Hash
}
