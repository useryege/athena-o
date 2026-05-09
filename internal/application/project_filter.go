package application

import (
	"context"
	"errors"
	"math/rand"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	log "github.com/sirupsen/logrus"
)

type ProjectFilter struct {
	wg              sync.WaitGroup
	registry        ProjectRegistry
	inputCh         <-chan *Project
	evmFetcher      EVMFetcher
	apiFetcher      APIFetcher
	delayedFetchSem chan struct{}
}

const defaultDelayedFetchConcurrency = 10

func NewProjectFilter(
	registry ProjectRegistry,
	inputCh <-chan *Project,
	evmFetcher EVMFetcher,
	apiFetcher APIFetcher,
	delayedFetchSem chan struct{},
) *ProjectFilter {
	if delayedFetchSem == nil {
		delayedFetchSem = make(chan struct{}, defaultDelayedFetchConcurrency)
	}

	return &ProjectFilter{
		registry:        registry,
		inputCh:         inputCh,
		evmFetcher:      evmFetcher,
		apiFetcher:      apiFetcher,
		delayedFetchSem: delayedFetchSem,
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

	// try to call totalSupply
	totalSupply, err := f.evmFetcher.FetchTotalSupply(ctx, event.Meta.Contract)
	if err != nil {
		return err
	}

	// try to call balanceOf
	_, err = f.evmFetcher.BalanceOf(ctx, event.Meta.Contract, common.HexToAddress("0x0000000000000000000000000000000000000000"))
	if err != nil {
		return err
	}

	// try to call decimals
	decimals, err := f.evmFetcher.FetchDecimals(ctx, event.Meta.Contract)
	if err != nil {
		return err
	}

	// try to call name
	name, err := f.evmFetcher.FetchName(ctx, event.Meta.Contract)
	if err != nil {
		return err
	}

	// try to call symbol
	symbol, err := f.evmFetcher.FetchSymbol(ctx, event.Meta.Contract)
	if err != nil {
		return err
	}

	// set the values to the project state
	now := time.Now()
	event.InitState.Name.MarkReady(name, now)
	event.InitState.Symbol.MarkReady(symbol, now)
	event.InitState.Decimals.MarkReady(decimals, now)
	event.InitState.TotalSupply.MarkReady(totalSupply, now)

	event.PerfTrace.FilterCompletedAt = now
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

func (f *ProjectFilter) startDelayedFieldResolve(ctx context.Context, event *Project) {
	f.wg.Add(1)
	go func() {
		defer f.wg.Done()
		f.resolveDelayedFields(ctx, event)
	}()
}

func (f *ProjectFilter) resolveDelayedFields(ctx context.Context, event *Project) {
	delay := 10 * time.Second
	const maxDelay = 2 * time.Minute

	for {
		if event.DelayedState.SourceCode.IsReady() && event.DelayedState.SourceCodeABI.IsReady() {
			return
		}

		if !event.DelayedState.SourceCode.IsReady() {
			sourceCode, err := f.fetchSourceCode(ctx, event)
			now := time.Now()
			if err != nil {
				event.DelayedState.SourceCode.MarkFailed(err, now)
				log.WithFields(log.Fields{
					"projectID": event.Meta.ProjectID,
					"contract":  event.Meta.Contract,
					"error":     err,
				}).Debug("failed to resolve source code")
			} else {
				event.DelayedState.SourceCode.MarkReady(sourceCode, now)
				log.WithFields(log.Fields{
					"projectID": event.Meta.ProjectID,
					"contract":  event.Meta.Contract,
				}).Info("source code resolved")
			}
		}

		if !event.DelayedState.SourceCodeABI.IsReady() {
			sourceCodeABI, err := f.fetchSourceCodeABI(ctx, event)
			now := time.Now()
			if err != nil {
				event.DelayedState.SourceCodeABI.MarkFailed(err, now)
				log.WithFields(log.Fields{
					"projectID": event.Meta.ProjectID,
					"contract":  event.Meta.Contract,
					"error":     err,
				}).Debug("failed to resolve source code ABI")
			} else {
				event.DelayedState.SourceCodeABI.MarkReady(sourceCodeABI, now)
				log.WithFields(log.Fields{
					"projectID": event.Meta.ProjectID,
					"contract":  event.Meta.Contract,
				}).Info("source code ABI resolved")
			}
		}

		if event.DelayedState.SourceCode.IsReady() && event.DelayedState.SourceCodeABI.IsReady() {
			return
		}

		select {
		case <-ctx.Done():
			return
		case <-time.After(withJitter(delay)):
		}

		delay *= 2
		if delay > maxDelay {
			delay = maxDelay
		}
	}
}

func (f *ProjectFilter) fetchSourceCode(ctx context.Context, event *Project) (string, error) {
	if err := f.acquireDelayedFetch(ctx); err != nil {
		return "", err
	}
	defer f.releaseDelayedFetch()

	return f.apiFetcher.FetchSourceCode(ctx, event.Meta.Contract)
}

func (f *ProjectFilter) fetchSourceCodeABI(ctx context.Context, event *Project) (string, error) {
	if err := f.acquireDelayedFetch(ctx); err != nil {
		return "", err
	}
	defer f.releaseDelayedFetch()

	return f.apiFetcher.FetchSourceCodeABI(ctx, event.Meta.Contract)
}

func (f *ProjectFilter) acquireDelayedFetch(ctx context.Context) error {
	select {
	case f.delayedFetchSem <- struct{}{}:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (f *ProjectFilter) releaseDelayedFetch() {
	<-f.delayedFetchSem
}

func withJitter(delay time.Duration) time.Duration {
	if delay <= 0 {
		return 0
	}

	jitterRange := delay / 2
	if jitterRange <= 0 {
		return delay
	}

	return delay + time.Duration(rand.Int63n(int64(jitterRange)))
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

			// resolve delayed fields in the background after the project becomes visible.
			f.startDelayedFieldResolve(ctx, event)

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
