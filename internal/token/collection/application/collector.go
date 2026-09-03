package application

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/useryege/athena/internal/token/collection"
	"github.com/useryege/athena/internal/token/shared"
)

const (
	MaxFailureCount          = int32(3)
	defaultLeaseDuration     = 90 * time.Second
	defaultHeartbeatInterval = 30 * time.Second
)

var (
	ErrTaskLeaseLost            = errors.New("token collection task lease lost")
	ErrCollectionInfrastructure = errors.New("token collection infrastructure failure")
)

type TaskRepository interface {
	ClaimCollectionTask(context.Context, collection.DataType, []int64, time.Duration) (*collection.TaskWithProject, error)
	RenewCollectionTaskLease(context.Context, int64, int64, time.Duration) (bool, error)
	CommitCollection(context.Context, CommitCollectionCommand) (CommitCollectionResult, error)
	RetryCollectionTask(context.Context, RetryCollectionTaskCommand) (bool, error)
	FailCollectionTask(context.Context, FailCollectionTaskCommand) (bool, error)
}

type CollectionOutput struct {
	Result             any
	BlockNumber        *uint64
	CodeSource         *CodeSourceUpdate
	NormalTransactions []collection.WalletNormalTransaction
}

type CodeSourceUpdate struct {
	CodeHash   shared.Hash
	SourceCode string
}

type TaskProcessor interface {
	DataType() collection.DataType
	Process(context.Context, collection.ProjectContext) (CollectionOutput, error)
}

type CommitCollectionCommand struct {
	Task               collection.Task
	CollectedAt        time.Time
	SchemaVersion      int32
	Payload            json.RawMessage
	ContentHash        shared.Hash
	BlockNumber        *uint64
	CodeSource         *CodeSourceUpdate
	NormalTransactions []collection.WalletNormalTransaction
}

type CommitCollectionResult struct {
	Applied                 bool
	ProfileBuildTaskCreated bool
}

type RetryCollectionTaskCommand struct {
	Task         collection.Task
	LastError    string
	FailureCount int32
	AvailableAt  time.Time
}

type FailCollectionTaskCommand struct {
	Task         collection.Task
	LastError    string
	FailureCount int32
	FailedAt     time.Time
}

type CollectorOptions struct {
	ChainIDs          []int64
	RetryInterval     time.Duration
	LeaseDuration     time.Duration
	HeartbeatInterval time.Duration
	Now               func() time.Time
}

type Collector struct {
	repository TaskRepository
	processor  TaskProcessor
	options    CollectorOptions
}

func NewCollector(repository TaskRepository, processor TaskProcessor, options CollectorOptions) *Collector {
	if options.LeaseDuration <= 0 {
		options.LeaseDuration = defaultLeaseDuration
	}
	if options.HeartbeatInterval <= 0 {
		options.HeartbeatInterval = defaultHeartbeatInterval
	}
	if options.Now == nil {
		options.Now = time.Now
	}
	return &Collector{repository: repository, processor: processor, options: options}
}

func (collector *Collector) RunOnce(ctx context.Context) (int, error) {
	if err := collector.validate(); err != nil {
		return 0, err
	}
	item, err := collector.repository.ClaimCollectionTask(
		ctx,
		collector.processor.DataType(),
		collector.options.ChainIDs,
		collector.options.LeaseDuration,
	)
	if err != nil {
		return 0, err
	}
	if item == nil {
		return 0, nil
	}
	if err := collector.process(ctx, *item); err != nil {
		return 0, err
	}
	return 1, nil
}

func (collector *Collector) validate() error {
	if collector == nil || collector.repository == nil || collector.processor == nil {
		return fmt.Errorf("token collection application is not configured")
	}
	if collector.options.RetryInterval <= 0 {
		return fmt.Errorf("token collection retry interval must be positive")
	}
	if collector.options.LeaseDuration <= 0 {
		return fmt.Errorf("token collection lease duration must be positive")
	}
	if collector.options.HeartbeatInterval <= 0 || collector.options.HeartbeatInterval >= collector.options.LeaseDuration {
		return fmt.Errorf("token collection heartbeat interval must be positive and shorter than its lease")
	}
	return nil
}

func (collector *Collector) process(ctx context.Context, item collection.TaskWithProject) error {
	processingContext, cancelProcessing := context.WithCancel(ctx)
	heartbeatResult := make(chan error, 1)
	go collector.maintainLease(processingContext, cancelProcessing, item.Task, heartbeatResult)

	output, processingError := collector.processor.Process(processingContext, item.Project)
	var payload json.RawMessage
	var contentHash shared.Hash
	if processingError == nil {
		encoded, err := json.Marshal(output.Result)
		if err != nil {
			processingError = fmt.Errorf("marshal %s collection result: %w", item.Task.DataType, err)
		} else {
			payload, contentHash, processingError = collection.NormalizeResult(collection.ResultSchemaVersionV1, encoded)
			if processingError != nil {
				processingError = fmt.Errorf("normalize %s collection result: %w", item.Task.DataType, processingError)
			}
		}
	}

	cancelProcessing()
	heartbeatError := <-heartbeatResult
	if heartbeatError != nil {
		return heartbeatError
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if errors.Is(processingError, ErrCollectionInfrastructure) {
		// Infrastructure errors are not evidence that the upstream data source
		// failed. Leave the running claim for lease recovery without consuming a
		// failure attempt.
		return processingError
	}
	// Re-establish a full lease window immediately before the fenced terminal
	// transition. This also detects a claim lost between the final heartbeat and
	// persistence, in which case the collected output is deliberately discarded.
	renewed, err := collector.repository.RenewCollectionTaskLease(
		ctx,
		item.Task.ID,
		item.Task.ClaimGeneration,
		collector.options.LeaseDuration,
	)
	if err != nil {
		return fmt.Errorf("renew token collection task %d lease before completion: %w", item.Task.ID, err)
	}
	if !renewed {
		return ErrTaskLeaseLost
	}
	if processingError != nil {
		return collector.recordFailure(ctx, item.Task, processingError)
	}

	result, err := collector.repository.CommitCollection(ctx, CommitCollectionCommand{
		Task:               item.Task,
		CollectedAt:        collector.options.Now().UTC(),
		SchemaVersion:      collection.ResultSchemaVersionV1,
		Payload:            payload,
		ContentHash:        contentHash,
		BlockNumber:        output.BlockNumber,
		CodeSource:         output.CodeSource,
		NormalTransactions: output.NormalTransactions,
	})
	if err != nil {
		// Persistence failures do not describe the collected data and therefore do
		// not consume one of the three data-source failure attempts. The lease will
		// make the same logical task claimable again.
		return err
	}
	if !result.Applied {
		return ErrTaskLeaseLost
	}
	return nil
}

func (collector *Collector) maintainLease(ctx context.Context, cancel context.CancelFunc, task collection.Task, result chan<- error) {
	ticker := time.NewTicker(collector.options.HeartbeatInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			result <- nil
			return
		case <-ticker.C:
			if ctx.Err() != nil {
				result <- nil
				return
			}
			applied, err := collector.repository.RenewCollectionTaskLease(
				ctx,
				task.ID,
				task.ClaimGeneration,
				collector.options.LeaseDuration,
			)
			if err != nil {
				if ctx.Err() != nil {
					result <- nil
					return
				}
				cancel()
				result <- fmt.Errorf("renew token collection task %d lease: %w", task.ID, err)
				return
			}
			if !applied {
				if ctx.Err() != nil {
					result <- nil
					return
				}
				cancel()
				result <- ErrTaskLeaseLost
				return
			}
		}
	}
}

func (collector *Collector) recordFailure(ctx context.Context, task collection.Task, failure error) error {
	failedAt := collector.options.Now().UTC()
	failureCount := task.FailureCount + 1
	if failureCount < MaxFailureCount {
		applied, err := collector.repository.RetryCollectionTask(ctx, RetryCollectionTaskCommand{
			Task:         task,
			LastError:    failure.Error(),
			FailureCount: failureCount,
			AvailableAt:  failedAt.Add(collector.options.RetryInterval),
		})
		if err != nil {
			return err
		}
		if !applied {
			return ErrTaskLeaseLost
		}
		return nil
	}
	applied, err := collector.repository.FailCollectionTask(ctx, FailCollectionTaskCommand{
		Task:         task,
		LastError:    failure.Error(),
		FailureCount: failureCount,
		FailedAt:     failedAt,
	})
	if err != nil {
		return err
	}
	if !applied {
		return ErrTaskLeaseLost
	}
	return nil
}
