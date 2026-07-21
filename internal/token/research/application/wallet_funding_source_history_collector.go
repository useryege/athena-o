package application

import (
	"context"
	"fmt"
	"time"

	"github.com/useryege/athena/internal/token/research"
	"github.com/useryege/athena/internal/token/shared"
)

const (
	walletFundingSourceChainID            = int64(56)
	walletFundingSourceHistoryLimit       = int32(100)
	walletFundingSourceTaskLease          = 90 * time.Second
	walletFundingSourceLeaseRenewInterval = 30 * time.Second
)

type WalletFundingSourceHistoryProvider interface {
	ListFundingSourcesBefore(context.Context, shared.Address, uint64, uint64, int32) (research.WalletFundingSourceHistory, error)
}

type WalletFundingSourceHistoryRepository interface {
	ClaimCollectionTasks(context.Context, research.DataCollectionType, []int64, int32) ([]research.ProjectDataCollectionTaskWithProject, error)
	ListPendingWalletFundingSourceHistoryWallets(context.Context, int64) ([]shared.Address, error)
	SaveWalletFundingSourceHistory(context.Context, SaveWalletFundingSourceHistoryCommand) (bool, error)
	RenewCollectionTaskLease(context.Context, int64, time.Duration) error
	CompleteWalletFundingSourceHistory(context.Context, CompleteWalletFundingSourceHistoryCommand) (bool, error)
	RetryCollectionTask(context.Context, RetryCollectionTaskCommand) error
	FailCollectionTask(context.Context, FailCollectionTaskCommand) error
}

type SaveWalletFundingSourceHistoryCommand struct {
	Project        research.ProjectCollectionContext
	Wallet         shared.Address
	RequestedCount int32
	History        research.WalletFundingSourceHistory
	FetchedAt      time.Time
}

type CompleteWalletFundingSourceHistoryCommand struct {
	Task        research.ProjectDataCollectionTask
	CompletedAt time.Time
}

type WalletFundingSourceHistoryCollector struct {
	repository WalletFundingSourceHistoryRepository
	provider   WalletFundingSourceHistoryProvider
	options    CollectorOptions
}

func NewWalletFundingSourceHistoryCollector(repository WalletFundingSourceHistoryRepository, provider WalletFundingSourceHistoryProvider, options CollectorOptions) *WalletFundingSourceHistoryCollector {
	if options.Limit <= 0 {
		options.Limit = 20
	}
	if options.Now == nil {
		options.Now = time.Now
	}
	return &WalletFundingSourceHistoryCollector{repository: repository, provider: provider, options: options}
}

func (collector *WalletFundingSourceHistoryCollector) RunOnce(ctx context.Context) (int, error) {
	if collector == nil || collector.repository == nil || collector.provider == nil {
		return 0, fmt.Errorf("wallet funding source history collector is not configured")
	}
	tasks, err := collector.repository.ClaimCollectionTasks(ctx, research.DataCollectionTypeWalletFundingSourceHistory, collector.options.ChainIDs, collector.options.Limit)
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

func (collector *WalletFundingSourceHistoryCollector) process(ctx context.Context, item research.ProjectDataCollectionTaskWithProject) error {
	stopRenewal := collector.renewLease(ctx, item.Task.ID)
	defer stopRenewal()
	if item.Project.ChainID != walletFundingSourceChainID {
		return collector.fail(ctx, item, fmt.Errorf("wallet funding source history only supports BSC mainnet chain_id=%d", item.Project.ChainID))
	}

	for {
		wallets, err := collector.repository.ListPendingWalletFundingSourceHistoryWallets(ctx, item.Project.ID)
		if err != nil {
			return collector.fail(ctx, item, err)
		}
		if len(wallets) == 0 {
			break
		}
		for _, wallet := range wallets {
			history, err := collector.provider.ListFundingSourcesBefore(ctx, wallet, item.Project.CreationBlockNumber, item.Project.CreationTransactionIndex, walletFundingSourceHistoryLimit)
			if err != nil {
				return collector.fail(ctx, item, err)
			}
			_, err = collector.repository.SaveWalletFundingSourceHistory(ctx, SaveWalletFundingSourceHistoryCommand{
				Project: item.Project, Wallet: wallet, RequestedCount: walletFundingSourceHistoryLimit,
				History: history, FetchedAt: collector.options.Now().UTC(),
			})
			if err != nil {
				return collector.fail(ctx, item, err)
			}
		}
	}

	_, err := collector.repository.CompleteWalletFundingSourceHistory(ctx, CompleteWalletFundingSourceHistoryCommand{Task: item.Task, CompletedAt: collector.options.Now().UTC()})
	if err != nil {
		return collector.fail(ctx, item, err)
	}
	return nil
}

func (collector *WalletFundingSourceHistoryCollector) renewLease(ctx context.Context, taskID int64) func() {
	renewCtx, cancel := context.WithCancel(ctx)
	done := make(chan struct{})
	go func() {
		defer close(done)
		ticker := time.NewTicker(walletFundingSourceLeaseRenewInterval)
		defer ticker.Stop()
		for {
			select {
			case <-renewCtx.Done():
				return
			case <-ticker.C:
				_ = collector.repository.RenewCollectionTaskLease(renewCtx, taskID, walletFundingSourceTaskLease)
			}
		}
	}()
	return func() { cancel(); <-done }
}

func (collector *WalletFundingSourceHistoryCollector) fail(ctx context.Context, item research.ProjectDataCollectionTaskWithProject, failure error) error {
	failedAt := collector.options.Now().UTC()
	backoff := time.Duration(1<<min(int(item.Task.Attempts), 4)) * time.Second
	if item.Task.Attempts < 4 {
		return collector.repository.RetryCollectionTask(ctx, RetryCollectionTaskCommand{Task: item.Task, LastError: failure.Error(), AvailableAt: failedAt.Add(backoff)})
	}
	return collector.repository.FailCollectionTask(ctx, FailCollectionTaskCommand{Task: item.Task, LastError: failure.Error(), FailedAt: failedAt, NextRunAt: failedAt.Add(item.Project.RefreshInterval)})
}
