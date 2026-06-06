package chainstate

import (
	"context"

	"github.com/ethereum/go-ethereum/common"
	appcache "github.com/useryege/athena/internal/application/cache"
	appcomponents "github.com/useryege/athena/internal/application/components"
	"github.com/useryege/athena/internal/application/evm"
	appstore "github.com/useryege/athena/internal/application/store"
)

type Options struct {
	ChainID int64
	Store   appstore.Store
	Cache   appcache.ProjectComponentCache
	Fetcher evm.AthenaFetcher
}

type Component struct {
	chainID int64
	store   appstore.Store
	cache   appcache.ProjectComponentCache
	fetcher evm.AthenaFetcher
}

func NewComponent(opts Options) *Component {
	if opts.Store == nil || opts.Fetcher == nil {
		return nil
	}
	return &Component{chainID: opts.ChainID, store: opts.Store, cache: opts.Cache, fetcher: opts.Fetcher}
}

func (c *Component) Start(context.Context) error { return nil }

func (c *Component) Stop() error { return nil }

func (c *Component) Collect(ctx context.Context, chainID int64, contract common.Address) error {
	if c == nil || c.store == nil || c.fetcher == nil {
		return nil
	}
	if contract == (common.Address{}) {
		return nil
	}
	if err := c.refresh(ctx, chainID, contract); err != nil {
		_ = appcomponents.MarkComponentFailed(ctx, c.store, chainID, contract, appstore.ProjectComponentChainState, err, appcomponents.NowUTC())
		return err
	}
	return nil
}

func (c *Component) refresh(ctx context.Context, chainID int64, contract common.Address) error {
	base, err := appcomponents.LoadProject(ctx, c.cache, c.store, chainID, contract)
	if err != nil || base == nil {
		return err
	}
	if err := appcomponents.MarkComponentRunning(ctx, c.store, chainID, contract, appstore.ProjectComponentChainState, appcomponents.NowUTC()); err != nil {
		return err
	}
	snapshot, err := c.fetcher.FetchProject(ctx, appcomponents.ProjectQuery(*base, nil))
	if err != nil {
		return err
	}
	item, err := appcomponents.ChainStateFromSnapshot(chainID, contract, snapshot, appcomponents.NowUTC())
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
	if err := appcomponents.MarkComponentSuccess(ctx, c.store, chainID, contract, appstore.ProjectComponentChainState, item.FetchedAt); err != nil {
		return err
	}
	return nil
}

