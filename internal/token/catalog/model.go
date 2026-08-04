package catalog

import (
	"math/big"
	"time"

	"github.com/useryege/athena/internal/token/shared"
)

type ContractCode struct {
	CodeHash            shared.Hash
	SourceCode          string
	SourceCodeFetchedAt time.Time
	DeploymentCount     int64
	CreatedAt           time.Time
}

type Project struct {
	ID          int64
	ChainID     int64
	Contract    shared.Address
	TxSender    shared.Address
	TxHash      shared.Hash
	TxIndex     uint64
	BlockNumber uint64
	BlockTime   uint64
	CodeHash    shared.Hash
	Name        string
	Symbol      string
	Decimals    uint8
	TotalSupply *big.Int
	WethPair    shared.Address
	UsdtPair    shared.Address
	CreatedAt   time.Time
}

type ProjectTokenMetadata struct {
	Name        string
	Symbol      string
	Decimals    uint8
	TotalSupply *big.Int
}

type RelatedWalletRole string

const (
	RelatedWalletRoleCreator          RelatedWalletRole = "creator"
	RelatedWalletRoleInitialRecipient RelatedWalletRole = "initial_recipient"
)

type ProjectRelatedWallet struct {
	ProjectID int64
	Wallet    shared.Address
	Role      RelatedWalletRole
	CreatedAt time.Time
}

type ProjectInitialRecipient struct {
	ID                int64
	ProjectID         int64
	Wallet            shared.Address
	RatioBPS          int64
	RankIndex         int32
	SourceTxHash      shared.Hash
	SourceBlockNumber uint64
	CreatedAt         time.Time
}

type ContractCodePage struct {
	Items    []ContractCode
	Total    int64
	Page     int32
	PageSize int32
}
