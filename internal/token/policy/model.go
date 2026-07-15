package policy

import (
	"time"

	"github.com/useryege/athena/internal/token/shared"
)

type ContractCodeBlocklistEntry struct {
	CodeHash       shared.Hash
	Note           string
	SourceChainID  int64
	SourceContract shared.Address
	CreatedAt      time.Time
}

type WalletBlocklistEntry struct {
	Wallet    shared.Address
	Note      string
	CreatedAt time.Time
}
