package application

import (
	"context"
	"errors"
	"math/big"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	log "github.com/sirupsen/logrus"
	"github.com/useryege/athena/internal/application/evm"
	"github.com/useryege/athena/util/ethereumapi"
	"github.com/useryege/athena/util/evmtool"
)

type ProjectFilter struct {
	wg          sync.WaitGroup
	registry    ProjectRegistry
	inputCh     <-chan *Project
	evmFetcher  evm.EVMFetcher
	wethToken   common.Address
	projectSync *ProjectSync
}

func NewProjectFilter(
	registry ProjectRegistry,
	inputCh <-chan *Project,
	evmFetcher evm.EVMFetcher,
	apiFetcher ethereumapi.EthereumAPI,
	delayedFetchSem chan struct{},
	wethToken common.Address,
) *ProjectFilter {
	return &ProjectFilter{
		registry:    registry,
		inputCh:     inputCh,
		evmFetcher:  evmFetcher,
		wethToken:   wethToken,
		projectSync: NewProjectSync(evmFetcher, apiFetcher, delayedFetchSem, wethToken),
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

var ErrTotalSupplyIsNil = errors.New("totalSupply is nil")
var ErrTotalSupplyIsZero = errors.New("totalSupply is 0")
var ErrNameIsEmpty = errors.New("name is empty")
var ErrSymbolIsEmpty = errors.New("symbol is empty")
var ErrDecimalsIsZero = errors.New("decimals is 0")

func (f *ProjectFilter) initProject(ctx context.Context, event *Project) error {
	// try to call totalSupply
	totalSupply, err := f.evmFetcher.FetchTokenTotalSupply(ctx, event.Meta.Contract)
	if err != nil {
		return err
	}
	if totalSupply == nil {
		return ErrTotalSupplyIsNil
	}
	if totalSupply.Cmp(big.NewInt(0)) == 0 {
		return ErrTotalSupplyIsZero
	}

	// try to call balanceOf
	_, err = f.evmFetcher.FetchTokenBalanceOf(ctx, event.Meta.Contract, common.HexToAddress("0x0000000000000000000000000000000000000000"))
	if err != nil {
		return err
	}

	// try to call decimals
	decimals, err := f.evmFetcher.FetchTokenDecimals(ctx, event.Meta.Contract)
	if err != nil {
		return err
	}
	if decimals == 0 {
		return ErrDecimalsIsZero
	}

	// try to call name
	name, err := f.evmFetcher.FetchTokenName(ctx, event.Meta.Contract)
	if err != nil {
		return err
	}
	if name == "" {
		return ErrNameIsEmpty
	}

	// try to call symbol
	symbol, err := f.evmFetcher.FetchTokenSymbol(ctx, event.Meta.Contract)
	if err != nil {
		return err
	}
	if symbol == "" {
		return ErrSymbolIsEmpty
	}

	token0 := evmtool.GetToken0(event.Meta.Contract, f.wethToken)

	token1 := evmtool.GetToken1(event.Meta.Contract, f.wethToken)

	// set the values to the project state
	now := time.Now()
	event.Token.Name.MarkReady(name, now)
	event.Token.Symbol.MarkReady(symbol, now)
	event.Token.Decimals.MarkReady(decimals, now)
	event.Token.TotalSupply.MarkReady(totalSupply, now)
	event.WethV2Pool.Token0.MarkReady(token0, now)
	event.WethV2Pool.Token1.MarkReady(token1, now)

	event.PerfTrace.FilterCompletedAt = now
	return nil

}

func isExecutionRevertedError(err error) bool {
	if err == nil {
		return false
	}

	// Internal validation failures should be treated as ignorable init errors.
	if errors.Is(err, ErrTotalSupplyIsNil) ||
		errors.Is(err, ErrTotalSupplyIsZero) ||
		errors.Is(err, ErrNameIsEmpty) ||
		errors.Is(err, ErrSymbolIsEmpty) ||
		errors.Is(err, ErrDecimalsIsZero) {
		return true
	}

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
		f.projectSync.ResolveDelayedFields(ctx, event)
	}()
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
