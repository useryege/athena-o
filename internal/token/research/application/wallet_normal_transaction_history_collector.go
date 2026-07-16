package application

import (
	"context"
	"fmt"
	"time"

	"github.com/useryege/athena/internal/token/research"
	"github.com/useryege/athena/internal/token/shared"
)

const (
	walletNormalTransactionHistoryLimit = int32(300)
	walletHistoryTaskLease              = 90 * time.Second
	walletHistoryTaskLeaseRenewInterval = 30 * time.Second
)

type WalletNormalTransactionHistoryProvider interface {
	ListNormalTransactionsBefore(context.Context, int64, shared.Address, uint64, uint64, int32) ([]research.WalletNormalTransaction, error)
}

type WalletNormalTransactionHistoryRepository interface {
	ClaimCollectionTasks(context.Context, research.DataCollectionType, []int64, int32) ([]research.ProjectDataCollectionTaskWithProject, error)
	ListPendingWalletNormalTransactionHistoryWallets(context.Context, int64) ([]shared.Address, error)
	SaveWalletNormalTransactionHistory(context.Context, SaveWalletNormalTransactionHistoryCommand) (bool, error)
	RenewCollectionTaskLease(context.Context, int64, time.Duration) error
	CompleteWalletNormalTransactionHistory(context.Context, CompleteWalletNormalTransactionHistoryCommand) (bool, error)
	RetryCollectionTask(context.Context, RetryCollectionTaskCommand) error
	FailCollectionTask(context.Context, FailCollectionTaskCommand) error
}

type SaveWalletNormalTransactionHistoryCommand struct {
	Project        research.ProjectCollectionContext
	Wallet         shared.Address
	RequestedCount int32
	Transactions   []research.WalletNormalTransaction
	FetchedAt      time.Time
}

type CompleteWalletNormalTransactionHistoryCommand struct {
	Task        research.ProjectDataCollectionTask
	CompletedAt time.Time
}

type WalletNormalTransactionHistoryCollector struct {
	repository WalletNormalTransactionHistoryRepository
	provider   WalletNormalTransactionHistoryProvider
	options    CollectorOptions
}

func NewWalletNormalTransactionHistoryCollector(repository WalletNormalTransactionHistoryRepository, provider WalletNormalTransactionHistoryProvider, options CollectorOptions) *WalletNormalTransactionHistoryCollector {
	if options.Limit <= 0 {
		options.Limit = 20
	}
	if options.Now == nil {
		options.Now = time.Now
	}
	return &WalletNormalTransactionHistoryCollector{repository: repository, provider: provider, options: options}
}

func (collector *WalletNormalTransactionHistoryCollector) RunOnce(ctx context.Context) (int, error) {
	if collector == nil || collector.repository == nil || collector.provider == nil {
		return 0, fmt.Errorf("wallet normal transaction history collector is not configured")
	}
	tasks, err := collector.repository.ClaimCollectionTasks(ctx, research.DataCollectionTypeWalletNormalTransactionHistory, collector.options.ChainIDs, collector.options.Limit)
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

func (collector *WalletNormalTransactionHistoryCollector) process(ctx context.Context, item research.ProjectDataCollectionTaskWithProject) error {
	stopRenewal := collector.renewLease(ctx, item.Task.ID)
	defer stopRenewal()

	for {
		wallets, err := collector.repository.ListPendingWalletNormalTransactionHistoryWallets(ctx, item.Project.ID)
		if err != nil {
			return collector.fail(ctx, item, err)
		}
		if len(wallets) == 0 {
			break
		}
		for _, wallet := range wallets {
			transactions, err := collector.provider.ListNormalTransactionsBefore(ctx, item.Project.ChainID, wallet, item.Project.CreationBlockNumber, item.Project.CreationTransactionIndex, walletNormalTransactionHistoryLimit)
			if err != nil {
				return collector.fail(ctx, item, err)
			}
			_, err = collector.repository.SaveWalletNormalTransactionHistory(ctx, SaveWalletNormalTransactionHistoryCommand{
				Project: item.Project, Wallet: wallet, RequestedCount: walletNormalTransactionHistoryLimit,
				Transactions: transactions, FetchedAt: collector.options.Now().UTC(),
			})
			if err != nil {
				return collector.fail(ctx, item, err)
			}
		}
	}

	_, err := collector.repository.CompleteWalletNormalTransactionHistory(ctx, CompleteWalletNormalTransactionHistoryCommand{Task: item.Task, CompletedAt: collector.options.Now().UTC()})
	if err != nil {
		return collector.fail(ctx, item, err)
	}
	return nil
}

func (collector *WalletNormalTransactionHistoryCollector) renewLease(ctx context.Context, taskID int64) func() {
	renewCtx, cancel := context.WithCancel(ctx)
	done := make(chan struct{})
	go func() {
		defer close(done)
		ticker := time.NewTicker(walletHistoryTaskLeaseRenewInterval)
		defer ticker.Stop()
		for {
			select {
			case <-renewCtx.Done():
				return
			case <-ticker.C:
				_ = collector.repository.RenewCollectionTaskLease(renewCtx, taskID, walletHistoryTaskLease)
			}
		}
	}()
	return func() { cancel(); <-done }
}

func (collector *WalletNormalTransactionHistoryCollector) fail(ctx context.Context, item research.ProjectDataCollectionTaskWithProject, failure error) error {
	failedAt := collector.options.Now().UTC()
	backoff := time.Duration(1<<min(int(item.Task.Attempts), 4)) * time.Second
	if item.Task.Attempts < 4 {
		return collector.repository.RetryCollectionTask(ctx, RetryCollectionTaskCommand{Task: item.Task, LastError: failure.Error(), AvailableAt: failedAt.Add(backoff)})
	}
	return collector.repository.FailCollectionTask(ctx, FailCollectionTaskCommand{Task: item.Task, LastError: failure.Error(), FailedAt: failedAt, NextRunAt: failedAt.Add(item.Project.RefreshInterval)})
}
