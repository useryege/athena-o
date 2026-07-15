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

	ProjectCandidateStatusPending   = "pending"
	ProjectCandidateStatusValidated = "validated"
	ProjectCandidateStatusRejected  = "rejected"

	ProjectResearchStatusResearching = "researching"
	ProjectResearchStatusSelected    = "selected"
	ProjectResearchStatusRejected    = "rejected"
	ProjectResearchStatusExpired     = "expired"

	ProjectDataCollectionTypeAve                = "ave"
	ProjectDataCollectionTypeChainState         = "chain_state"
	ProjectDataCollectionTypeWalletAssetState   = "wallet_asset_state"
	ProjectDataCollectionTypeSimulationResult   = "simulation_result"
	ProjectDataCollectionTypeContractCodeSource = "contract_code_source"

	ProjectDataCollectionScheduleStatusActive    = "active"
	ProjectDataCollectionScheduleStatusCompleted = "completed"
	ProjectDataCollectionScheduleStatusPaused    = "paused"

	ProjectTaskStatusPending   = "pending"
	ProjectTaskStatusRunning   = "running"
	ProjectTaskStatusSucceeded = "succeeded"
	ProjectTaskStatusFailed    = "failed"

	ProjectSelectionOutcomeSelected = "selected"
	ProjectSelectionOutcomeRejected = "rejected"
	ProjectSelectionOutcomeDeferred = "deferred"

	ProjectRelatedWalletRoleCreator          = "creator"
	ProjectRelatedWalletRoleInitialRecipient = "initial_recipient"
)

// Aliases used by generic task validation and existing collector call sites.
const (
	ProjectDataCollectionStatusPending   = ProjectTaskStatusPending
	ProjectDataCollectionStatusRunning   = ProjectTaskStatusRunning
	ProjectDataCollectionStatusSucceeded = ProjectTaskStatusSucceeded
	ProjectDataCollectionStatusFailed    = ProjectTaskStatusFailed
)

type ChainIngestCheckpoint struct {
	ChainID           int64
	ChainName         string
	Enabled           bool
	CursorBlockNumber uint64
	Status            string
	CreatedAt         time.Time
	UpdatedAt         time.Time
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

// Collection payload DTOs; persisted only inside project_observation JSON.
type WalletAssetState struct {
	ChainID       int64          `json:"chainId"`
	Wallet        common.Address `json:"wallet"`
	WethBalance   *big.Int       `json:"wethBalance"`
	UsdtBalance   *big.Int       `json:"usdtBalance"`
	NativeBalance *big.Int       `json:"nativeBalance"`
	UsdtValue     *big.Int       `json:"usdtValue"`
	FetchedAt     time.Time      `json:"fetchedAt"`
}
type ProjectSimulationResult struct {
	ProjectID                          int64          `json:"projectId"`
	Wallet                             common.Address `json:"wallet"`
	CanMintFromDeadViaTransferFrom     bool           `json:"canMintFromDeadViaTransferFrom"`
	CanMintFromZeroViaTransferFrom     bool           `json:"canMintFromZeroViaTransferFrom"`
	CanMintFromWethPairViaTransferFrom bool           `json:"canMintFromWethPairViaTransferFrom"`
	CanMintFromUsdtPairViaTransferFrom bool           `json:"canMintFromUsdtPairViaTransferFrom"`
	CanMintViaTransferToWethPair       bool           `json:"canMintViaTransferToWethPair"`
	CanMintViaTransferToUsdtPair       bool           `json:"canMintViaTransferToUsdtPair"`
	FetchedAt                          time.Time      `json:"fetchedAt"`
}

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

type ProjectDataCollectionSchedule struct {
	ProjectID           int64
	DataType            string
	Status              string
	RefreshInterval     time.Duration
	NextRunAt           time.Time
	LatestTaskRevision  int64
	ConsecutiveFailures int32
	LastError           string
	LastCheckedAt       time.Time
	CreatedAt           time.Time
	UpdatedAt           time.Time
}
type ProjectDataCollectionTask struct {
	ID             int64
	ProjectID      int64
	DataType       string
	Status         string
	Revision       int64
	Attempts       int32
	AvailableAt    time.Time
	LockedAt       time.Time
	LeaseExpiresAt time.Time
	LastError      string
	CreatedAt      time.Time
	UpdatedAt      time.Time
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

type ProjectObservation struct {
	ID            int64
	ProjectID     int64
	DataType      string
	ContentHash   common.Hash
	Payload       json.RawMessage
	BlockNumber   *uint64
	ObservedAt    time.Time
	LastCheckedAt time.Time
	CreatedAt     time.Time
}

type ProjectResearchState struct {
	ProjectID                   int64
	ChainID                     int64
	Contract                    common.Address
	Status                      string
	EvidenceRevision            int64
	CurrentReportRevision       int64
	CurrentSelectionID          int64
	CurrentSelectionOutcome     string
	LastEvaluatedReportRevision int64
	LastEvaluatedAt             time.Time
	ExpiresAt                   time.Time
	CreatedAt                   time.Time
	UpdatedAt                   time.Time
}
type ProjectResearchStatePage struct {
	Items    []ProjectResearchState
	Total    int64
	Page     int32
	PageSize int32
}

type ProjectReportRevision struct {
	ID                        int64
	ProjectID                 int64
	ChainID                   int64
	Contract                  common.Address
	Revision                  int64
	ContentHash               common.Hash
	CompletenessStatus        string
	Evidence                  json.RawMessage
	Report                    json.RawMessage
	ObservedBlockNumber       *uint64
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
	BuiltAt                   time.Time
	CreatedAt                 time.Time
}
type ProjectReportRevisionPage struct {
	Items    []ProjectReportRevision
	Total    int64
	Page     int32
	PageSize int32
}
type ProjectReportListItem struct {
	Report         ProjectReportRevision
	ChainID        int64
	Name           string
	Symbol         string
	Contract       common.Address
	BuildStatus    string
	BuildAttempts  int32
	BuildLastError string
	BuildUpdatedAt time.Time
}
type ProjectReportPage struct {
	Items    []ProjectReportListItem
	Total    int64
	Page     int32
	PageSize int32
}

type ProjectReportBuildTask struct {
	ID               int64
	ProjectID        int64
	EvidenceRevision int64
	Status           string
	Attempts         int32
	AvailableAt      time.Time
	LeaseExpiresAt   time.Time
	LastError        string
}
type ProjectSelectionEvaluationTask struct {
	ID             int64
	ProjectID      int64
	ReportRevision int64
	Status         string
	Attempts       int32
	AvailableAt    time.Time
	LeaseExpiresAt time.Time
	LastError      string
}
type ProjectSelection struct {
	ID              int64
	ProjectID       int64
	ChainID         int64
	Contract        common.Address
	Outcome         string
	StrategyKey     string
	StrategyVersion string
	ReportRevision  int64
	ReasonCodes     []string
	ReasonDetail    string
	DecidedAt       time.Time
	CreatedAt       time.Time
}
type ProjectSelectionPage struct {
	Items    []ProjectSelection
	Total    int64
	Page     int32
	PageSize int32
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
