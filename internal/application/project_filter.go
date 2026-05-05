package application

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	log "github.com/sirupsen/logrus"
	"github.com/useryege/athena/pkg/abi/ERC20"
)

type ProjectFilter struct {
	wg         sync.WaitGroup
	nodeClient *ethclient.Client
	registry   ProjectRegistry
	pool       RetryUntilReadyPool
	queue      RetryUntilReadyQueue
	inputCh    <-chan *Project
}

func NewProjectFilter(nodeClient *ethclient.Client, registry ProjectRegistry, inputCh <-chan *Project, pool RetryUntilReadyPool, queue RetryUntilReadyQueue) *ProjectFilter {
	return &ProjectFilter{
		nodeClient: nodeClient,
		registry:   registry,
		pool:       pool,
		inputCh:    inputCh,
		queue:      queue,
	}
}

func (f *ProjectFilter) Start(ctx context.Context) error {
	f.wg.Add(1)
	go func() {
		defer f.wg.Done()
		err := f.run(ctx)
		if err != nil && !errors.Is(err, context.Canceled) {
			log.Errorf("failed to test chain watcher: %v", err)
		}
	}()
	return nil
}

func (f *ProjectFilter) initProject(ctx context.Context, event *Project) error {
	reader := &bind.CallOpts{Context: ctx}
	// create ERC20 caller instance
	tokenCaller, err := ERC20.NewERC20Caller(event.Meta.Contract, f.nodeClient)
	if err != nil {
		return err
	}

	// try to call totalSupply
	totalSupply, err := tokenCaller.TotalSupply(reader)
	if err != nil {
		return err
	}

	// try to call balanceOf
	_, err = tokenCaller.BalanceOf(reader, common.HexToAddress("0x0000000000000000000000000000000000000000"))
	if err != nil {
		return err
	}

	// try to call decimals
	decimals, err := tokenCaller.Decimals(reader)
	if err != nil {
		return err
	}

	// try to call name
	name, err := tokenCaller.Name(reader)
	if err != nil {
		return err
	}

	// try to call symbol
	symbol, err := tokenCaller.Symbol(reader)
	if err != nil {
		return err
	}

	// set the values to the project state
	event.InitState.Name.Set(name)
	event.InitState.Symbol.Set(symbol)
	event.InitState.Decimals.Set(decimals)
	event.InitState.TotalSupply.Set(totalSupply)

	event.PerfTrace.FilterCompletedAt = time.Now()
	return nil

}

func isExecutionRevertedError(err error) bool {
	switch err.Error() {
	case "execution reverted":
		return true
	case "no contract code at given address":
		return true
	case "execution reverted: ERC721: address zero is not a valid owner":
		return true
	case "abi: attempting to unmarshal an empty string while arguments are expected":
		return true
	case "execution reverted: division or modulo by zero":
		return true
	default:
		return false
	}

}

func (f *ProjectFilter) scheduleInitialRetryResolve(ctx context.Context, event *Project) error {
	type retryFieldConfig struct {
		field RetryUntilReadyField
		name  string
	}

	retryFields := []retryFieldConfig{
		{field: RetryUntilReadyFieldSourceCode, name: "source code"},
		{field: RetryUntilReadyFieldSourceCodeABI, name: "source code ABI"},
	}

	now := time.Now()
	deadline := now.Add(1 * time.Minute)

	for _, cfg := range retryFields {
		if err := f.pool.Add(ctx, RetryUntilReadyFieldKey{
			ProjectID: event.Meta.ProjectID,
			Field:     cfg.field,
		}, deadline); err != nil {
			log.WithFields(log.Fields{
				"projectID": event.Meta.ProjectID,
				"field":     cfg.field,
				"error":     err,
			}).Errorf("failed to add %s to retry pool", cfg.name)
			return err
		}

		if err := f.queue.Enqueue(ctx, RetryUntilReadyResolveRequest{
			ProjectID: event.Meta.ProjectID,
			Field:     cfg.field,
			Reason:    ResolveReasonProjectCreated,
			CreatedAt: now,
		}); err != nil {
			log.WithFields(log.Fields{
				"projectID": event.Meta.ProjectID,
				"field":     cfg.field,
				"error":     err,
			}).Errorf("failed to enqueue %s resolve request", cfg.name)
			return err
		}
	}

	return nil
}

func (f *ProjectFilter) run(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case event, ok := <-f.inputCh:
			if !ok {
				return nil
			}

			// init the project
			if err := f.initProject(ctx, event); err != nil {
				// execution reverted
				if isExecutionRevertedError(err) {
					continue
				}
				log.WithFields(log.Fields{
					"projectID": event.Meta.ProjectID,
					"contract":  event.Meta.Contract,
					"error":     err,
				}).Error("failed to init project")
				continue
			}

			// store the project in the registry
			if err := f.registry.SetProject(ctx, event.Meta.ProjectID, event); err != nil {
				log.WithFields(log.Fields{
					"projectID": event.Meta.ProjectID,
					"error":     err,
				}).Error("failed to store project")
				continue
			}

			// schedule the initial retry resolve
			if err := f.scheduleInitialRetryResolve(ctx, event); err != nil {
				continue
			}

			log.WithFields(log.Fields{
				"component":         "Project Filter",
				"blockNumber":       event.Meta.BlockNumber,
				"blockTime":         event.Meta.BlockTime,
				"transaction":       event.Meta.Tx.Hash(),
				"executionDuration": event.PerfTrace.FilterCompletedAt.Sub(event.PerfTrace.BlockDiscoveredAt).Milliseconds(),
			}).Info("project filter received contract creation transaction")
		}
	}
}

func (f *ProjectFilter) Stop() error {
	f.wg.Wait()
	return nil
}
