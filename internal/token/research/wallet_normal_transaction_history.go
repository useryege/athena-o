package research

import (
	"math/big"

	"github.com/useryege/athena/internal/token/shared"
)

type WalletNormalTransactionReceiptStatus string

const (
	WalletNormalTransactionReceiptStatusUnspecified WalletNormalTransactionReceiptStatus = "unspecified"
	WalletNormalTransactionReceiptStatusFailed      WalletNormalTransactionReceiptStatus = "failed"
	WalletNormalTransactionReceiptStatusSuccess     WalletNormalTransactionReceiptStatus = "success"
)

type WalletNormalTransaction struct {
	BlockNumber       uint64
	BlockHash         shared.Hash
	BlockTimestamp    uint64
	TransactionHash   shared.Hash
	Nonce             uint64
	TransactionIndex  uint64
	From              shared.Address
	To                shared.Address
	Value             *big.Int
	Gas               uint64
	GasPrice          *big.Int
	Input             string
	MethodID          []byte
	FunctionName      string
	ContractAddress   shared.Address
	CumulativeGasUsed uint64
	ReceiptStatus     WalletNormalTransactionReceiptStatus
	GasUsed           uint64
	Confirmations     uint64
	IsError           bool
}
