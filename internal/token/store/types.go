package store

import (
	"encoding/json"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common"
)

const (
	ChainIngestStatusRunning = "running"
	ChainIngestStatusStopped = "stopped"
)

const (
	ProjectCandidateStatusPending   = "pending"
	ProjectCandidateStatusQualified = "qualified"
	ProjectCandidateStatusRejected  = "rejected"
)

const (
	ProjectDataCollectionTypeAve                = "ave"
	ProjectDataCollectionTypeChainState         = "chain_state"
	ProjectDataCollectionTypeWalletAssetState   = "wallet_asset_state"
	ProjectDataCollectionTypeSimulationResult   = "simulation_result"
	ProjectDataCollectionTypeContractCodeSource = "contract_code_source"
)

const (
	ProjectDataCollectionStatusPending   = "pending"
	ProjectDataCollectionStatusSucceeded = "succeeded"
	ProjectDataCollectionStatusFailed    = "failed"
)

const (
	ProjectRelatedWalletRoleCreator          = "creator"
	ProjectRelatedWalletRoleInitialRecipient = "initial_recipient"
)

type ChainIngestCheckpoint struct {
	ChainID           int64
	ChainName         string
	Enabled           bool
	CursorBlockNumber uint64
	Status            string
	CreatedAt         time.Time
}

type Chain struct {
	ID        int64
	Name      string
	Enabled   bool
	CreatedAt time.Time
}

type ProjectCandidate struct {
	ID          int64
	ChainID     int64
	Contract    common.Address
	TxSender    common.Address
	TxHash      common.Hash
	TxIndex     uint64
	BlockNumber uint64
	BlockTime   uint64
	Status      string
	CreatedAt   time.Time
}

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

type ProjectAveData struct {
	ProjectID   int64
	AveResponse json.RawMessage
	FetchedAt   time.Time
	CreatedAt   time.Time
}

type ProjectChainStateData struct {
	ProjectID  int64
	ChainState json.RawMessage
	FetchedAt  time.Time
	CreatedAt  time.Time
}

type ProjectRelatedWallet struct {
	ProjectID int64
	Wallet    common.Address
	Role      string
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

type WalletAssetState struct {
	ChainID       int64
	Wallet        common.Address
	WethBalance   *big.Int
	UsdtBalance   *big.Int
	NativeBalance *big.Int
	UsdtValue     *big.Int
	FetchedAt     time.Time
	CreatedAt     time.Time
}

type ProjectSimulationResult struct {
	ProjectID                          int64
	Wallet                             common.Address
	CanMintFromDeadViaTransferFrom     bool
	CanMintFromZeroViaTransferFrom     bool
	CanMintFromWethPairViaTransferFrom bool
	CanMintFromUsdtPairViaTransferFrom bool
	CanMintViaTransferToWethPair       bool
	CanMintViaTransferToUsdtPair       bool
	FetchedAt                          time.Time
	CreatedAt                          time.Time
}

type BytecodeBlacklistEntry struct {
	CodeHash       common.Hash
	Note           string
	SourceChainID  int64
	SourceContract common.Address
	CreatedAt      time.Time
}

type WalletBlacklistEntry struct {
	Wallet    common.Address
	Note      string
	CreatedAt time.Time
}

type ProjectDataCollectionTask struct {
	ProjectID     int64
	DataType      string
	Status        string
	Revision      int64
	Attempts      int32
	NextAttemptAt time.Time
	LastError     string
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

type ProjectDataCollectionTaskWithProject struct {
	Task    ProjectDataCollectionTask
	Project Project
}

type ProjectDataCollectionTaskPage struct {
	Items    []ProjectDataCollectionTask
	Total    int64
	Page     int32
	PageSize int32
}

type ProjectReport struct {
	ProjectID                 int64
	WethPairIsCreated         *bool
	WethPairIsRemoveLiquidity *bool
	WethPairIsMint            *bool
	WethPairQuoteUsdtValueInt *big.Int
	WethPairLastSwapTimestamp *uint64
	UsdtPairIsCreated         *bool
	UsdtPairIsRemoveLiquidity *bool
	UsdtPairIsMint            *bool
	UsdtPairQuoteUsdtValueInt *big.Int
	UsdtPairLastSwapTimestamp *uint64
	SourceUpdatedAt           time.Time
	EvaluatedAt               time.Time
	CreatedAt                 time.Time
}

type ProjectReportListItem struct {
	Report              ProjectReport
	ChainID             int64
	Name                string
	Symbol              string
	Contract            common.Address
	EvaluationStatus    string
	EvaluationAttempts  int32
	EvaluationLastError string
	EvaluationUpdatedAt time.Time
}

type ProjectReportPage struct {
	Items    []ProjectReportListItem
	Total    int64
	Page     int32
	PageSize int32
}

type ProjectReportEvaluationTask struct {
	ProjectID       int64
	Status          string
	Revision        int64
	Attempts        int32
	NextAttemptAt   time.Time
	LastError       string
	CreatedAt       time.Time
	UpdatedAt       time.Time
	SourceUpdatedAt time.Time
	HasChainState   bool
	ChainState      json.RawMessage
}

type ProjectCandidatePage struct {
	Items    []ProjectCandidate
	Total    int64
	Page     int32
	PageSize int32
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
