package application

import (
	"context"
	"fmt"
	"time"

	"github.com/useryege/athena/internal/token/research"
	"github.com/useryege/athena/internal/token/shared"
)

const (
	walletSwapTransactionChainID            = int64(56)
	walletSwapTransactionHistoryLimit       = int32(100)
	walletSwapTransactionTaskLease          = 90 * time.Second
	walletSwapTransactionLeaseRenewInterval = 30 * time.Second
)

type WalletSwapTransactionHistoryProvider interface {
	ListSwapTransactionsBeforeBlock(context.Context, shared.Address, uint64, int32) (research.WalletSwapTransactionHistory, error)
}

type WalletSwapTransactionHistoryRepository interface {
	ClaimCollectionTasks(context.Context, research.DataCollectionType, []int64, int32) ([]research.ProjectDataCollectionTaskWithProject, error)
	ListPendingWalletSwapTransactionHistoryWallets(context.Context, int64) ([]shared.Address, error)
	SaveWalletSwapTransactionHistory(context.Context, SaveWalletSwapTransactionHistoryCommand) (bool, error)
	RenewCollectionTaskLease(context.Context, int64, time.Duration) error
	CompleteWalletSwapTransactionHistory(context.Context, CompleteWalletSwapTransactionHistoryCommand) (bool, error)
	RetryCollectionTask(context.Context, RetryCollectionTaskCommand) error
	FailCollectionTask(context.Context, FailCollectionTaskCommand) error
}

type SaveWalletSwapTransactionHistoryCommand struct {
	Project        research.ProjectCollectionContext
	Wallet         shared.Address
	RequestedCount int32
	History        research.WalletSwapTransactionHistory
	FetchedAt      time.Time
}

type CompleteWalletSwapTransactionHistoryCommand struct {
	Task        research.ProjectDataCollectionTask
	CompletedAt time.Time
}

type WalletSwapTransactionHistoryCollector struct {
	repository WalletSwapTransactionHistoryRepository
	provider   WalletSwapTransactionHistoryProvider
	options    CollectorOptions
}

func NewWalletSwapTransactionHistoryCollector(repository WalletSwapTransactionHistoryRepository, provider WalletSwapTransactionHistoryProvider, options CollectorOptions) *WalletSwapTransactionHistoryCollector {
	if options.Limit <= 0 {
		options.Limit = 20
	}
	if options.Now == nil {
		options.Now = time.Now
	}
	return &WalletSwapTransactionHistoryCollector{repository: repository, provider: provider, options: options}
}

func (collector *WalletSwapTransactionHistoryCollector) RunOnce(ctx context.Context) (int, error) {
	if collector == nil || collector.repository == nil || collector.provider == nil {
		return 0, fmt.Errorf("wallet swap transaction history collector is not configured")
	}
	tasks, err := collector.repository.ClaimCollectionTasks(ctx, research.DataCollectionTypeWalletSwapTransactionHistory, collector.options.ChainIDs, collector.options.Limit)
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

func (collector *WalletSwapTransactionHistoryCollector) process(ctx context.Context, item research.ProjectDataCollectionTaskWithProject) error {
	stopRenewal := collector.renewLease(ctx, item.Task.ID)
	defer stopRenewal()
	if item.Project.ChainID != walletSwapTransactionChainID {
		return collector.fail(ctx, item, fmt.Errorf("wallet swap transaction history only supports BSC mainnet chain_id=%d", item.Project.ChainID))
	}

	for {
		wallets, err := collector.repository.ListPendingWalletSwapTransactionHistoryWallets(ctx, item.Project.ID)
		if err != nil {
			return collector.fail(ctx, item, err)
		}
		if len(wallets) == 0 {
			break
		}
		for _, wallet := range wallets {
			history, err := collector.provider.ListSwapTransactionsBeforeBlock(ctx, wallet, item.Project.CreationBlockNumber, walletSwapTransactionHistoryLimit)
			if err != nil {
				return collector.fail(ctx, item, err)
			}
			_, err = collector.repository.SaveWalletSwapTransactionHistory(ctx, SaveWalletSwapTransactionHistoryCommand{
				Project: item.Project, Wallet: wallet, RequestedCount: walletSwapTransactionHistoryLimit,
				History: history, FetchedAt: collector.options.Now().UTC(),
			})
			if err != nil {
				return collector.fail(ctx, item, err)
			}
		}
	}

	_, err := collector.repository.CompleteWalletSwapTransactionHistory(ctx, CompleteWalletSwapTransactionHistoryCommand{Task: item.Task, CompletedAt: collector.options.Now().UTC()})
	if err != nil {
		return collector.fail(ctx, item, err)
	}
	return nil
}

func (collector *WalletSwapTransactionHistoryCollector) renewLease(ctx context.Context, taskID int64) func() {
	renewCtx, cancel := context.WithCancel(ctx)
	done := make(chan struct{})
	go func() {
		defer close(done)
		ticker := time.NewTicker(walletSwapTransactionLeaseRenewInterval)
		defer ticker.Stop()
		for {
			select {
			case <-renewCtx.Done():
				return
			case <-ticker.C:
				_ = collector.repository.RenewCollectionTaskLease(renewCtx, taskID, walletSwapTransactionTaskLease)
			}
		}
	}()
	return func() { cancel(); <-done }
}

func (collector *WalletSwapTransactionHistoryCollector) fail(ctx context.Context, item research.ProjectDataCollectionTaskWithProject, failure error) error {
	failedAt := collector.options.Now().UTC()
	backoff := time.Duration(1<<min(int(item.Task.Attempts), 4)) * time.Second
	if item.Task.Attempts < 4 {
		return collector.repository.RetryCollectionTask(ctx, RetryCollectionTaskCommand{Task: item.Task, LastError: failure.Error(), AvailableAt: failedAt.Add(backoff)})
	}
	return collector.repository.FailCollectionTask(ctx, FailCollectionTaskCommand{Task: item.Task, LastError: failure.Error(), FailedAt: failedAt, NextRunAt: failedAt.Add(item.Project.RefreshInterval)})
}
