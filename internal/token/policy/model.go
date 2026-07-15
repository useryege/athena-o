package policy

import (
	"time"

	"github.com/ethereum/go-ethereum/common"
)

type ContractCodeBlocklistEntry struct {
	CodeHash       common.Hash
	Note           string
	SourceChainID  int64
	SourceContract common.Address
	CreatedAt      time.Time
}

type WalletBlocklistEntry struct {
	Wallet    common.Address
	Note      string
	CreatedAt time.Time
}
