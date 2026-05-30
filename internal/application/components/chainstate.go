package components

import (
	"context"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	appcache "github.com/useryege/athena/internal/application/cache"
	"github.com/useryege/athena/internal/application/evm"
	appstore "github.com/useryege/athena/internal/application/store"
)

type ChainStateComponent struct {
	store    appstore.Store
	cache    appcache.ProjectComponentCache
	fetcher  evm.AthenaFetcher
	bus      EventBus
	consumer *Consumer
}

func NewChainStateComponent(store appstore.Store, cache appcache.ProjectComponentCache, fetcher evm.AthenaFetcher, bus EventBus) *ChainStateComponent {
	if store == nil || fetcher == nil {
		return nil
	}
	c := &ChainStateComponent{store: store, cache: cache, fetcher: fetcher, bus: bus}
	c.consumer = NewConsumer(appstore.ProjectComponentChainState, bus, c.handleEvent)
	return c
}

func (c *ChainStateComponent) Start(ctx context.Context) error {
	if c == nil || c.consumer == nil {
		return nil
	}
	return c.consumer.Start(ctx)
}

func (c *ChainStateComponent) Stop() error {
	if c == nil || c.consumer == nil {
		return nil
	}
	return c.consumer.Stop()
}

func (c *ChainStateComponent) handleEvent(ctx context.Context, event Event) error {
	if event.Type != EventProjectInitialized && event.Type != EventProjectRefresh {
		return nil
	}
	contract := event.ProjectContract()
	if contract == (common.Address{}) {
		return nil
	}
	if err := c.refresh(ctx, contract); err != nil {
		_ = MarkComponentFailed(ctx, c.store, contract, appstore.ProjectComponentChainState, err, nowUTC())
		if c.bus != nil {
			_ = c.bus.Publish(ctx, ComponentFailedEvent(contract, appstore.ProjectComponentChainState, err))
		}
		return nil
	}
	return nil
}

func (c *ChainStateComponent) refresh(ctx context.Context, contract common.Address) error {
	base, err := LoadProjectBase(ctx, c.cache, c.store, contract)
	if err != nil || base == nil {
		return err
	}
	if err := MarkComponentRunning(ctx, c.store, contract, appstore.ProjectComponentChainState, nowUTC()); err != nil {
		return err
	}
	snapshot, err := c.fetcher.FetchProject(ctx, ProjectQuery(*base, nil))
	if err != nil {
		return err
	}
	if snapshot.TokenContract != (common.Address{}) && snapshot.TokenContract != contract {
		return fmt.Errorf("athena project token contract = %s, want %s", snapshot.TokenContract.Hex(), contract.Hex())
	}
	if !snapshot.Token.IsValidERC20 {
		return fmt.Errorf("project token is not a valid ERC20")
	}
	item, err := ChainStateFromSnapshot(contract, snapshot, nowUTC())
	if err != nil {
		return err
	}
	if err := c.store.UpsertProjectChainState(ctx, item); err != nil {
		return err
	}
	if c.cache != nil {
		if err := c.cache.SetChainState(ctx, item); err != nil {
			return err
		}
	}
	if err := MarkComponentSuccess(ctx, c.store, contract, appstore.ProjectComponentChainState, item.FetchedAt); err != nil {
		return err
	}
	if c.bus != nil {
		return c.bus.Publish(ctx, ComponentCompletedEvent(contract, appstore.ProjectComponentChainState))
	}
	return nil
}
