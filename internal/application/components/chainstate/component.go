package chainstate

import (
	"context"
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/common"
	appcache "github.com/useryege/athena/internal/application/cache"
	appcomponents "github.com/useryege/athena/internal/application/components"
	"github.com/useryege/athena/internal/application/evm"
	appstore "github.com/useryege/athena/internal/application/store"
)

type Options struct {
	Store   appstore.Store
	Cache   appcache.ProjectComponentCache
	Fetcher evm.AthenaFetcher
	Bus     appcomponents.EventBus
}

type Component struct {
	store    appstore.Store
	cache    appcache.ProjectComponentCache
	fetcher  evm.AthenaFetcher
	bus      appcomponents.EventBus
	consumer *appcomponents.Consumer
}

func NewComponent(opts Options) *Component {
	if opts.Store == nil || opts.Fetcher == nil {
		return nil
	}
	c := &Component{store: opts.Store, cache: opts.Cache, fetcher: opts.Fetcher, bus: opts.Bus}
	c.consumer = appcomponents.NewConsumer(appstore.ProjectComponentChainState, opts.Bus, c.handleEvent)
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
	if event.Type != appcomponents.EventProjectInitialized && event.Type != appcomponents.EventProjectRefresh {
		return nil
	}
	contract := event.ProjectContract()
	if contract == (common.Address{}) {
		return nil
	}
	if err := c.refresh(ctx, contract); err != nil {
		_ = appcomponents.MarkComponentFailed(ctx, c.store, contract, appstore.ProjectComponentChainState, err, nowUTC())
		if c.bus != nil {
			_ = c.bus.Publish(ctx, appcomponents.ComponentFailedEvent(contract, appstore.ProjectComponentChainState, err))
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
	if err := appcomponents.MarkComponentRunning(ctx, c.store, contract, appstore.ProjectComponentChainState, nowUTC()); err != nil {
		return err
	}
	snapshot, err := c.fetcher.FetchProject(ctx, appcomponents.ProjectQuery(*base, nil))
	if err != nil {
		return err
	}
	if snapshot.TokenContract != (common.Address{}) && snapshot.TokenContract != contract {
		return fmt.Errorf("athena project token contract = %s, want %s", snapshot.TokenContract.Hex(), contract.Hex())
	}
	if !snapshot.Token.IsValidERC20 {
		return fmt.Errorf("project token is not a valid ERC20")
	}
	item, err := appcomponents.ChainStateFromSnapshot(contract, snapshot, nowUTC())
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
	if err := appcomponents.MarkComponentSuccess(ctx, c.store, contract, appstore.ProjectComponentChainState, item.FetchedAt); err != nil {
		return err
	}
	if c.bus != nil {
		return c.bus.Publish(ctx, appcomponents.ComponentCompletedEvent(contract, appstore.ProjectComponentChainState))
	}
	return nil
}

func nowUTC() time.Time {
	return time.Now().UTC()
}
