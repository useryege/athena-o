package catalog

import (
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common"
)

type ContractCode struct {
	CodeHash            common.Hash
	SourceCode          string
	SourceCodeFetchedAt time.Time
	DeploymentCount     int64
	CreatedAt           time.Time
}

type Project struct {
	ID          int64
	ChainID     int64
	Contract    common.Address
	TxSender    common.Address
	TxHash      common.Hash
	TxIndex     uint64
	BlockNumber uint64
	BlockTime   uint64
	CodeHash    common.Hash
	Name        string
	Symbol      string
	Decimals    uint8
	TotalSupply *big.Int
	WethPair    common.Address
	UsdtPair    common.Address
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
	Wallet    common.Address
	Role      RelatedWalletRole
	CreatedAt time.Time
}

type ProjectInitialRecipient struct {
	ID                int64
	ProjectID         int64
	Wallet            common.Address
	RatioBPS          int64
	RankIndex         int32
	SourceTxHash      common.Hash
	SourceBlockNumber uint64
	CreatedAt         time.Time
}

type ContractCodePage struct {
	Items    []ContractCode
	Total    int64
	Page     int32
	PageSize int32
}

type ProjectPage struct {
	Items    []Project
	Total    int64
	Page     int32
	PageSize int32
}
