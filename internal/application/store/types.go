package store

import (
	"encoding/json"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/useryege/athena/internal/application/model"
	athenacontract "github.com/useryege/athena/pkg/abi/ATHENA"
)

// ProjectRecord is a single row from the project table (DB record).
// For the aggregated domain view, use model.Project.
type ProjectRecord struct {
	ChainID     int64
	BlockTime   uint64
	BlockNumber uint64
	Contract    common.Address
	Creator     common.Address
	Tx          *types.Transaction
	TxHash      common.Hash
	TxIndex     uint64
	CreatedAt   time.Time
}

type ProjectChainState struct {
	ChainID         int64
	ProjectContract common.Address
	ChainState      athenacontract.AthenaProject
	RawChainState   json.RawMessage
	WethPair        common.Address
	UsdtPair        common.Address
	TokenName       string
	TokenSymbol     string
	FetchedAt       time.Time
	UpdatedAt       time.Time
}

type ProjectSimulationResult struct {
	ChainID         int64
	ProjectContract common.Address
	Result          model.SimulateResult
	FetchedAt       time.Time
	UpdatedAt       time.Time
}

type ProjectComponentState struct {
	ChainID         int64
	ProjectContract common.Address
	Component       string
	Status          string
	LastAttemptAt   time.Time
	LastSuccessAt   time.Time
	NextRunAt       time.Time
	LastError       string
	UpdatedAt       time.Time
}

type ProjectGenesisWallet struct {
	ID                int64
	ChainID           int64
	ProjectContract   common.Address
	Wallet            common.Address
	NetAmount         *big.Int
	RatioBPS          int64
	RankIndex         int32
	TotalSupply       *big.Int
	SourceTxHash      common.Hash
	SourceBlockNumber uint64
	CreatedAt         time.Time
}

type ProjectCreatorHistoricalProject struct {
	ID                        int64
	ChainID                   int64
	ProjectContract           common.Address
	HistoricalProjectContract common.Address
	RankIndex                 int32
	CreatedAt                 time.Time
}

type ChainInfo struct {
	ID      int64
	Name    string
	Enabled bool
}

type ChainIngestCheckpoint struct {
	ChainID              int64
	ChainName            string
	Enabled              bool
	FinalizedBlockNumber uint64
	FinalizedBlockHash   common.Hash
	CursorBlockNumber    uint64
	CursorBlockHash      common.Hash
	Status               string
	LockedAt             time.Time
	LockedBy             string
	UpdatedAt            time.Time
}

type ProjectCollectionState struct {
	ChainID         int64
	ProjectContract common.Address
	Status          string
	WorkflowID      string
	LastRequestedAt time.Time
	LastStartedAt   time.Time
	LastCompletedAt time.Time
	NextRunAt       time.Time
	LastError       string
	UpdatedAt       time.Time
	ComponentStates []ProjectComponentState
}
