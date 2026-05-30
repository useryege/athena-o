package components

import (
	"context"

	"github.com/ethereum/go-ethereum/common"
	appcache "github.com/useryege/athena/internal/application/cache"
	"github.com/useryege/athena/internal/application/evm"
	"github.com/useryege/athena/internal/application/simulate"
	appstore "github.com/useryege/athena/internal/application/store"
)

type SimulationComponent struct {
	store     appstore.Store
	cache     appcache.ProjectComponentCache
	fetcher   evm.AthenaFetcher
	simulator simulate.ProjectSimulator
	bus       EventBus
	consumer  *Consumer
}

func NewSimulationComponent(store appstore.Store, cache appcache.ProjectComponentCache, fetcher evm.AthenaFetcher, simulator simulate.ProjectSimulator, bus EventBus) *SimulationComponent {
	if store == nil || fetcher == nil || simulator == nil {
		return nil
	}
	c := &SimulationComponent{store: store, cache: cache, fetcher: fetcher, simulator: simulator, bus: bus}
	c.consumer = NewConsumer(appstore.ProjectComponentSimulation, bus, c.handleEvent)
	return c
}

func (c *SimulationComponent) Start(ctx context.Context) error {
	if c == nil || c.consumer == nil {
		return nil
	}
	return c.consumer.Start(ctx)
}

func (c *SimulationComponent) Stop() error {
	if c == nil || c.consumer == nil {
		return nil
	}
	return c.consumer.Stop()
}

func (c *SimulationComponent) handleEvent(ctx context.Context, event Event) error {
	if event.Type == EventComponentCompleted && event.Component != appstore.ProjectComponentChainState {
		return nil
	}
	if event.Type != EventComponentCompleted && event.Type != EventProjectRefresh {
		return nil
	}
	contract := event.ProjectContract()
	if contract == (common.Address{}) {
		return nil
	}
	if err := c.refresh(ctx, contract); err != nil {
		_ = MarkComponentFailed(ctx, c.store, contract, appstore.ProjectComponentSimulation, err, nowUTC())
		if c.bus != nil {
			_ = c.bus.Publish(ctx, ComponentFailedEvent(contract, appstore.ProjectComponentSimulation, err))
		}
		return nil
	}
	return nil
}

func (c *SimulationComponent) refresh(ctx context.Context, contract common.Address) error {
	base, err := LoadProjectBase(ctx, c.cache, c.store, contract)
	if err != nil || base == nil {
		return err
	}
	chainState, err := LoadProjectChainState(ctx, c.cache, c.store, contract)
	if err != nil || chainState == nil {
		return err
	}
	if err := MarkComponentRunning(ctx, c.store, contract, appstore.ProjectComponentSimulation, nowUTC()); err != nil {
		return err
	}
	simulationState, err := c.fetcher.FetchSimulationState(ctx, ProjectQuery(*base, nil))
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
	if err := MarkComponentSuccess(ctx, c.store, contract, appstore.ProjectComponentSimulation, item.FetchedAt); err != nil {
		return err
	}
	if c.bus != nil {
		return c.bus.Publish(ctx, ComponentCompletedEvent(contract, appstore.ProjectComponentSimulation))
	}
	return nil
}
