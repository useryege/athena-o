package simulation

import (
	"context"
	"time"

	"github.com/ethereum/go-ethereum/common"
	appcache "github.com/useryege/athena/internal/application/cache"
	appcomponents "github.com/useryege/athena/internal/application/components"
	"github.com/useryege/athena/internal/application/evm"
	appsimulate "github.com/useryege/athena/internal/application/simulate"
	appstore "github.com/useryege/athena/internal/application/store"
)

type Options struct {
	Store     appstore.Store
	Cache     appcache.ProjectComponentCache
	Fetcher   evm.AthenaFetcher
	Simulator appsimulate.ProjectSimulator
	Bus       appcomponents.EventBus
}

type Component struct {
	store     appstore.Store
	cache     appcache.ProjectComponentCache
	fetcher   evm.AthenaFetcher
	simulator appsimulate.ProjectSimulator
	bus       appcomponents.EventBus
	consumer  *appcomponents.Consumer
}

func NewComponent(opts Options) *Component {
	if opts.Store == nil || opts.Fetcher == nil || opts.Simulator == nil {
		return nil
	}
	c := &Component{store: opts.Store, cache: opts.Cache, fetcher: opts.Fetcher, simulator: opts.Simulator, bus: opts.Bus}
	c.consumer = appcomponents.NewConsumer(appstore.ProjectComponentSimulation, opts.Bus, c.handleEvent)
	return c
}

func (c *Component) Start(ctx context.Context) error {
	if c == nil || c.consumer == nil {
		return nil
	}
	return c.consumer.Start(ctx)
}

func (c *Component) Stop() error {
	if c == nil || c.consumer == nil {
		return nil
	}
	return c.consumer.Stop()
}

func (c *Component) handleEvent(ctx context.Context, event appcomponents.Event) error {
	if event.Type == appcomponents.EventComponentCompleted && event.Component != appstore.ProjectComponentChainState {
		return nil
	}
	if event.Type != appcomponents.EventComponentCompleted && event.Type != appcomponents.EventProjectRefresh {
		return nil
	}
	contract := event.ProjectContract()
	if contract == (common.Address{}) {
		return nil
	}
	if err := c.refresh(ctx, contract); err != nil {
		_ = appcomponents.MarkComponentFailed(ctx, c.store, contract, appstore.ProjectComponentSimulation, err, nowUTC())
		if c.bus != nil {
			_ = c.bus.Publish(ctx, appcomponents.ComponentFailedEvent(contract, appstore.ProjectComponentSimulation, err))
		}
		return nil
	}
	return nil
}

func (c *Component) refresh(ctx context.Context, contract common.Address) error {
	base, err := appcomponents.LoadProjectBase(ctx, c.cache, c.store, contract)
	if err != nil || base == nil {
		return err
	}
	chainState, err := appcomponents.LoadProjectChainState(ctx, c.cache, c.store, contract)
	if err != nil || chainState == nil {
		return err
	}
	if err := appcomponents.MarkComponentRunning(ctx, c.store, contract, appstore.ProjectComponentSimulation, nowUTC()); err != nil {
		return err
	}
	simulationState, err := c.fetcher.FetchSimulationState(ctx, appcomponents.ProjectQuery(*base, nil))
	if err != nil {
		return err
	}
	wethPair := chainState.WethPair
	usdtPair := chainState.UsdtPair
	if wethPair == (common.Address{}) || usdtPair == (common.Address{}) {
		return nil
	}
	result, err := c.simulator.SimulatePrimary(ctx, base.Creator, contract, wethPair, usdtPair, simulationState)
	if err != nil {
		return err
	}
	item := appstore.ProjectSimulationResult{ProjectContract: contract, Result: appstore.SimulateResult(result), FetchedAt: nowUTC()}
	if err := c.store.UpsertProjectSimulationResult(ctx, item); err != nil {
		return err
	}
	if c.cache != nil {
		if err := c.cache.SetSimulation(ctx, item); err != nil {
			return err
		}
	}
	if err := appcomponents.MarkComponentSuccess(ctx, c.store, contract, appstore.ProjectComponentSimulation, item.FetchedAt); err != nil {
		return err
	}
	if c.bus != nil {
		return c.bus.Publish(ctx, appcomponents.ComponentCompletedEvent(contract, appstore.ProjectComponentSimulation))
	}
	return nil
}

func nowUTC() time.Time {
	return time.Now().UTC()
}
