package collection

import (
	"encoding/json"
	"math/big"
	"time"

	"github.com/useryege/athena/internal/token/shared"
)

const ResultSchemaVersionV1 = int32(1)

type DataType string

const (
	DataTypeChainState               DataType = "chain_state"
	DataTypeWalletAssetState         DataType = "wallet_asset_state"
	DataTypeSimulationResult         DataType = "simulation_result"
	DataTypeAve                      DataType = "ave"
	DataTypeContractCodeSource       DataType = "contract_code_source"
	DataTypeWalletNormalTransactions DataType = "wallet_normal_transactions"
)

func AllDataTypes() []DataType {
	return []DataType{
		DataTypeChainState,
		DataTypeWalletAssetState,
		DataTypeSimulationResult,
		DataTypeAve,
		DataTypeContractCodeSource,
		DataTypeWalletNormalTransactions,
	}
}

func ParseDataType(value string) (DataType, bool) {
	dataType := DataType(value)
	for _, candidate := range AllDataTypes() {
		if dataType == candidate {
			return dataType, true
		}
	}
	return "", false
}

type TaskStatus string

const (
	TaskStatusPending   TaskStatus = "pending"
	TaskStatusRunning   TaskStatus = "running"
	TaskStatusSucceeded TaskStatus = "succeeded"
	TaskStatusFailed    TaskStatus = "failed"
)

func (status TaskStatus) IsTerminal() bool {
	return status == TaskStatusSucceeded || status == TaskStatusFailed
}

type Task struct {
	ID              int64
	ProjectID       int64
	DataType        DataType
	Status          TaskStatus
	FailureCount    int32
	AvailableAt     time.Time
	ClaimGeneration int64
	LockedAt        time.Time
	LeaseExpiresAt  time.Time
	LastError       string
	FinishedAt      time.Time
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

type ProjectContext struct {
	ID                    int64
	ChainID               int64
	Contract              shared.Address
	CodeHash              shared.Hash
	WethPair              shared.Address
	UsdtPair              shared.Address
	DeploymentBlockNumber uint64
	RelatedWallets        []shared.Address
}

type TaskWithProject struct {
	Task    Task
	Project ProjectContext
}

type Result struct {
	TaskID        int64
	ProjectID     int64
	DataType      DataType
	SchemaVersion int32
	Payload       json.RawMessage
	ContentHash   shared.Hash
	BlockNumber   *uint64
	CollectedAt   time.Time
	CreatedAt     time.Time
}

type TaskDetail struct {
	Task   Task
	Result *Result
}

type TaskPage struct {
	Items    []Task
	Total    int64
	Page     int32
	PageSize int32
}

type NormalTransactionReceiptStatus string

const (
	NormalTransactionReceiptStatusUnspecified NormalTransactionReceiptStatus = "unspecified"
	NormalTransactionReceiptStatusFailed      NormalTransactionReceiptStatus = "failed"
	NormalTransactionReceiptStatusSuccess     NormalTransactionReceiptStatus = "success"
)

type WalletNormalTransaction struct {
	Wallet           shared.Address
	TransactionHash  shared.Hash
	BlockNumber      uint64
	BlockTimestamp   uint64
	TransactionIndex uint64
	Nonce            uint64
	FromAddress      shared.Address
	ToAddress        shared.Address
	Value            *big.Int
	Gas              uint64
	GasPrice         *big.Int
	GasUsed          uint64
	Input            string
	MethodID         string
	FunctionName     string
	ReceiptStatus    NormalTransactionReceiptStatus
	IsError          bool
	CollectedAt      time.Time
}
