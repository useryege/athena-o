package application

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/useryege/athena/internal/token/research"
	"github.com/useryege/athena/internal/token/shared"
)

type CollectionTaskRepository interface {
	ClaimCollectionTasks(context.Context, research.DataCollectionType, []int64, int32) ([]research.ProjectDataCollectionTaskWithProject, error)
	CommitCollection(context.Context, CommitCollectionCommand) (CommitCollectionResult, error)
	RetryCollectionTask(context.Context, RetryCollectionTaskCommand) error
	FailCollectionTask(context.Context, FailCollectionTaskCommand) error
}

type CollectionOutput struct {
	Observation        any
	BlockNumber        *uint64
	RecordObservation  bool
	CodeSource         *CodeSourceUpdate
	NormalTransactions []research.WalletNormalTransaction
	CompleteSchedule   bool
}

type CodeSourceUpdate struct {
	CodeHash   shared.Hash
	SourceCode string
}

type TaskProcessor interface {
	DataType() research.DataCollectionType
	Process(context.Context, research.ProjectCollectionContext) (CollectionOutput, error)
}

type CommitCollectionCommand struct {
	Task               research.ProjectDataCollectionTask
	CheckedAt          time.Time
	SchemaVersion      int32
	Payload            json.RawMessage
	ContentHash        shared.Hash
	BlockNumber        *uint64
	RecordObservation  bool
	CodeSource         *CodeSourceUpdate
	NormalTransactions []research.WalletNormalTransaction
	NextRunAt          time.Time
	CompleteSchedule   bool
}

type CommitCollectionResult struct {
	ObservationID int64
	Changed       bool
	Applied       bool
}

type RetryCollectionTaskCommand struct {
	Task        research.ProjectDataCollectionTask
	LastError   string
	AvailableAt time.Time
}

type FailCollectionTaskCommand struct {
	Task      research.ProjectDataCollectionTask
	LastError string
	FailedAt  time.Time
	NextRunAt time.Time
}

type CollectorOptions struct {
	ChainIDs []int64
	Limit    int32
	Now      func() time.Time
}

type Collector struct {
	repository CollectionTaskRepository
	processor  TaskProcessor
	options    CollectorOptions
}

func NewCollector(repository CollectionTaskRepository, processor TaskProcessor, options CollectorOptions) *Collector {
	if options.Limit <= 0 {
		options.Limit = 20
	}
	if options.Now == nil {
		options.Now = time.Now
	}
	return &Collector{repository: repository, processor: processor, options: options}
}

func (collector *Collector) RunOnce(ctx context.Context) (int, error) {
	if collector == nil || collector.repository == nil || collector.processor == nil {
		return 0, fmt.Errorf("token collector application is not configured")
	}
	tasks, err := collector.repository.ClaimCollectionTasks(ctx, collector.processor.DataType(), collector.options.ChainIDs, collector.options.Limit)
	if err != nil {
		return 0, err
	}
	processed := 0
	for _, item := range tasks {
		if err := collector.process(ctx, item); err != nil {
			return processed, err
		}
		processed++
	}
	return processed, nil
}

func (collector *Collector) process(ctx context.Context, item research.ProjectDataCollectionTaskWithProject) error {
	output, err := collector.processor.Process(ctx, item.Project)
	if err != nil {
		return collector.fail(ctx, item, err)
	}
	checkedAt := collector.options.Now().UTC()
	command := CommitCollectionCommand{
		Task: item.Task, CheckedAt: checkedAt, SchemaVersion: research.ObservationSchemaVersionV1,
		BlockNumber: output.BlockNumber, RecordObservation: output.RecordObservation, CodeSource: output.CodeSource,
		NormalTransactions: output.NormalTransactions, NextRunAt: checkedAt.Add(item.Project.RefreshInterval),
		CompleteSchedule: output.CompleteSchedule,
	}
	if output.RecordObservation {
		payload, err := json.Marshal(output.Observation)
		if err != nil {
			return collector.fail(ctx, item, err)
		}
		command.Payload, command.ContentHash, err = research.NormalizeObservation(command.SchemaVersion, payload)
		if err != nil {
			return collector.fail(ctx, item, err)
		}
	}
	if _, err := collector.repository.CommitCollection(ctx, command); err != nil {
		return collector.fail(ctx, item, err)
	}
	return nil
}

func (collector *Collector) fail(ctx context.Context, item research.ProjectDataCollectionTaskWithProject, failure error) error {
	failedAt := collector.options.Now().UTC()
	backoff := time.Duration(1<<min(int(item.Task.Attempts), 4)) * time.Second
	if item.Task.Attempts < 4 {
		return collector.repository.RetryCollectionTask(ctx, RetryCollectionTaskCommand{Task: item.Task, LastError: failure.Error(), AvailableAt: failedAt.Add(backoff)})
	}
	return collector.repository.FailCollectionTask(ctx, FailCollectionTaskCommand{Task: item.Task, LastError: failure.Error(), FailedAt: failedAt, NextRunAt: failedAt.Add(item.Project.RefreshInterval)})
}

type MarketDataProvider interface {
	GetMarketData(context.Context, int64, shared.Address) (research.AveObservationV1, error)
}

type AveProcessor struct{ Provider MarketDataProvider }

func (processor AveProcessor) DataType() research.DataCollectionType {
	return research.DataCollectionTypeAve
}
func (processor AveProcessor) Process(ctx context.Context, project research.ProjectCollectionContext) (CollectionOutput, error) {
	value, err := processor.Provider.GetMarketData(ctx, project.ChainID, project.Contract)
	return CollectionOutput{Observation: value, RecordObservation: err == nil}, err
}

type ProjectStateReader interface {
	ReadChainState(context.Context, research.ProjectCollectionContext) (research.ChainStateObservationV1, uint64, error)
	ReadWalletAssetState(context.Context, research.ProjectCollectionContext) (research.WalletAssetObservationV1, uint64, error)
	ReadSimulationResult(context.Context, research.ProjectCollectionContext) (research.SimulationObservationV1, uint64, error)
}

type ChainStateProcessor struct{ Reader ProjectStateReader }

func (processor ChainStateProcessor) DataType() research.DataCollectionType {
	return research.DataCollectionTypeChainState
}
func (processor ChainStateProcessor) Process(ctx context.Context, project research.ProjectCollectionContext) (CollectionOutput, error) {
	value, block, err := processor.Reader.ReadChainState(ctx, project)
	return CollectionOutput{Observation: value, BlockNumber: &block, RecordObservation: err == nil}, err
}

type WalletAssetStateProcessor struct{ Reader ProjectStateReader }

func (processor WalletAssetStateProcessor) DataType() research.DataCollectionType {
	return research.DataCollectionTypeWalletAssetState
}
func (processor WalletAssetStateProcessor) Process(ctx context.Context, project research.ProjectCollectionContext) (CollectionOutput, error) {
	value, block, err := processor.Reader.ReadWalletAssetState(ctx, project)
	return CollectionOutput{Observation: value, BlockNumber: &block, RecordObservation: err == nil}, err
}

type SimulationResultProcessor struct{ Reader ProjectStateReader }

func (processor SimulationResultProcessor) DataType() research.DataCollectionType {
	return research.DataCollectionTypeSimulationResult
}
func (processor SimulationResultProcessor) Process(ctx context.Context, project research.ProjectCollectionContext) (CollectionOutput, error) {
	value, block, err := processor.Reader.ReadSimulationResult(ctx, project)
	return CollectionOutput{Observation: value, BlockNumber: &block, RecordObservation: err == nil}, err
}

type ContractCodeSnapshot struct {
	SourceCode string
	Fetched    bool
}
type ContractCodeReader interface {
	GetContractCodeSnapshot(context.Context, shared.Hash) (*ContractCodeSnapshot, error)
}
type SourceCodeProvider interface {
	GetSourceCode(context.Context, int64, shared.Address) (string, error)
}

type ContractSourceProcessor struct {
	Codes    ContractCodeReader
	Provider SourceCodeProvider
}

func (processor ContractSourceProcessor) DataType() research.DataCollectionType {
	return research.DataCollectionTypeContractCodeSource
}
func (processor ContractSourceProcessor) Process(ctx context.Context, project research.ProjectCollectionContext) (CollectionOutput, error) {
	code, err := processor.Codes.GetContractCodeSnapshot(ctx, project.CodeHash)
	if err != nil {
		return CollectionOutput{}, err
	}
	if code != nil && code.Fetched {
		if code.SourceCode == "" {
			return CollectionOutput{RecordObservation: false}, nil
		}
		observation := research.ContractSourceObservationV1{CodeHash: project.CodeHash, SourceAvailable: true}
		return CollectionOutput{Observation: observation, RecordObservation: true, CodeSource: &CodeSourceUpdate{CodeHash: project.CodeHash, SourceCode: code.SourceCode}, CompleteSchedule: true}, nil
	}
	source, err := processor.Provider.GetSourceCode(ctx, project.ChainID, project.Contract)
	if err != nil {
		return CollectionOutput{}, err
	}
	if source == "" {
		return CollectionOutput{RecordObservation: false}, nil
	}
	observation := research.ContractSourceObservationV1{CodeHash: project.CodeHash, SourceAvailable: true}
	return CollectionOutput{Observation: observation, RecordObservation: true, CodeSource: &CodeSourceUpdate{CodeHash: project.CodeHash, SourceCode: source}, CompleteSchedule: true}, nil
}

type WalletNormalTransactionProvider interface {
	ListNormalTransactions(context.Context, int64, shared.Address, uint64) ([]research.WalletNormalTransaction, error)
}

type WalletNormalTransactionsProcessor struct {
	Provider WalletNormalTransactionProvider
}

func (processor WalletNormalTransactionsProcessor) DataType() research.DataCollectionType {
	return research.DataCollectionTypeWalletNormalTransactions
}

func (processor WalletNormalTransactionsProcessor) Process(ctx context.Context, project research.ProjectCollectionContext) (CollectionOutput, error) {
	if project.DeploymentBlockNumber == 0 {
		return CollectionOutput{CompleteSchedule: true}, nil
	}
	transactions := make([]research.WalletNormalTransaction, 0)
	endBlock := project.DeploymentBlockNumber - 1
	for _, wallet := range project.RelatedWallets {
		items, err := processor.Provider.ListNormalTransactions(ctx, project.ChainID, wallet, endBlock)
		if err != nil {
			return CollectionOutput{}, err
		}
		transactions = append(transactions, items...)
	}
	return CollectionOutput{NormalTransactions: transactions, CompleteSchedule: true}, nil
}
