package simulation

import (
	"context"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/rpc"
	appcache "github.com/useryege/athena/internal/application/cache"
	appcomponents "github.com/useryege/athena/internal/application/components"
	"github.com/useryege/athena/internal/application/evm"
	appstore "github.com/useryege/athena/internal/application/store"
)

type SimulationNodeClient interface {
	BatchCallContext(ctx context.Context, b []rpc.BatchElem) error
}

type Options struct {
	ChainID    int64
	Store      appstore.Store
	Cache      appcache.ProjectComponentCache
	Fetcher    evm.AthenaFetcher
	NodeClient SimulationNodeClient
}

type Component struct {
	chainID    int64
	store      appstore.Store
	cache      appcache.ProjectComponentCache
	fetcher    evm.AthenaFetcher
	nodeClient SimulationNodeClient
}

func NewComponent(opts Options) *Component {
	if opts.Store == nil || opts.Fetcher == nil || opts.NodeClient == nil {
		return nil
	}
	c := &Component{
		store:      opts.Store,
		chainID:    opts.ChainID,
		cache:      opts.Cache,
		fetcher:    opts.Fetcher,
		nodeClient: opts.NodeClient,
	}
	return c
}

func (c *Component) Start(context.Context) error { return nil }

func (c *Component) Stop() error { return nil }

func (c *Component) Collect(ctx context.Context, chainID int64, contract common.Address) error {
	if c == nil || c.store == nil || c.fetcher == nil || c.nodeClient == nil {
		return nil
	}
	if contract == (common.Address{}) {
		return nil
	}
	if err := c.refresh(ctx, chainID, contract); err != nil {
		_ = appcomponents.MarkComponentFailed(ctx, c.store, chainID, contract, appstore.ProjectComponentSimulation, err, nowUTC())
		return err
	}
	return nil
}

func (c *Component) refresh(ctx context.Context, chainID int64, contract common.Address) error {
	base, err := appcomponents.LoadProjectBase(ctx, c.cache, c.store, chainID, contract)
	if err != nil || base == nil {
		return err
	}
	chainState, err := appcomponents.LoadProjectChainState(ctx, c.cache, c.store, chainID, contract)
	if err != nil || chainState == nil {
		return err
	}
	if err := appcomponents.MarkComponentRunning(ctx, c.store, chainID, contract, appstore.ProjectComponentSimulation, nowUTC()); err != nil {
		return err
	}
	simulationState, err := c.fetcher.FetchSimulationState(ctx, appcomponents.ProjectQuery(*base, nil))
	if err != nil {
		return err
	}
	wethPair := chainState.WethPair
	usdtPair := chainState.UsdtPair
	if wethPair == (common.Address{}) || usdtPair == (common.Address{}) {
		at := nowUTC()
		if err := appcomponents.MarkComponentSuccess(ctx, c.store, chainID, contract, appstore.ProjectComponentSimulation, at); err != nil {
			return err
		}
		return nil
	}
	result, err := c.simulatePrimary(ctx, base.Creator, contract, wethPair, usdtPair, simulationState)
	if err != nil {
		return err
	}
	item := appstore.ProjectSimulationResult{ChainID: chainID, ProjectContract: contract, Result: appstore.SimulateResult(result), FetchedAt: nowUTC()}
	if err := c.store.UpsertProjectSimulationResult(ctx, item); err != nil {
		return err
	}
	if c.cache != nil {
		if err := c.cache.SetSimulation(ctx, item); err != nil {
			return err
		}
	}
	if err := appcomponents.MarkComponentSuccess(ctx, c.store, chainID, contract, appstore.ProjectComponentSimulation, item.FetchedAt); err != nil {
		return err
	}
	return nil
}

func nowUTC() time.Time {
	return time.Now().UTC()
}
