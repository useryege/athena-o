package creatorhistory

import (
	"context"

	"github.com/ethereum/go-ethereum/common"
	appcache "github.com/useryege/athena/internal/application/cache"
	appcomponents "github.com/useryege/athena/internal/application/components"
	appstore "github.com/useryege/athena/internal/application/store"
)

type Options struct {
	ChainID int64
	Store   appstore.Store
	Cache   appcache.ProjectComponentCache
}

type Component struct {
	chainID int64
	store   appstore.Store
	cache   appcache.ProjectComponentCache
}

func NewComponent(opts Options) *Component {
	if opts.Store == nil {
		return nil
	}
	return &Component{chainID: opts.ChainID, store: opts.Store, cache: opts.Cache}
}

func (c *Component) Start(context.Context) error { return nil }

func (c *Component) Stop() error { return nil }

func (c *Component) Collect(ctx context.Context, chainID int64, contract common.Address) error {
	if c == nil || c.store == nil {
		return nil
	}
	if contract == (common.Address{}) {
		return nil
	}
	if err := c.refresh(ctx, chainID, contract); err != nil {
		_ = appcomponents.MarkComponentFailed(ctx, c.store, chainID, contract, appstore.ProjectComponentCreatorHistory, err, appcomponents.NowUTC())
		return err
	}
	return nil
}

func (c *Component) refresh(ctx context.Context, chainID int64, contract common.Address) error {
	base, err := appcomponents.LoadProject(ctx, c.cache, c.store, chainID, contract)
	if err != nil || base == nil {
		return err
	}
	if err := appcomponents.MarkComponentRunning(ctx, c.store, chainID, contract, appstore.ProjectComponentCreatorHistory, appcomponents.NowUTC()); err != nil {
		return err
	}
	projects, err := c.store.ListProjectMetasByCreatorBefore(ctx, chainID, base.Creator, base.BlockNumber, base.TxIndex)
	if err != nil {
		return err
	}
	items := make([]appstore.ProjectCreatorHistoricalProject, 0, len(projects))
	seen := map[common.Address]struct{}{}
	for _, meta := range projects {
		if meta.Contract == (common.Address{}) || meta.Contract == contract {
			continue
		}
		if _, ok := seen[meta.Contract]; ok {
			continue
		}
		seen[meta.Contract] = struct{}{}
		items = append(items, appstore.ProjectCreatorHistoricalProject{
			ChainID:                   chainID,
			ProjectContract:           contract,
			HistoricalProjectContract: meta.Contract,
			RankIndex:                 int32(len(items)),
		})
	}
	if err := c.store.ReplaceProjectCreatorHistoricalProjects(ctx, chainID, contract, items); err != nil {
		return err
	}
	if c.cache != nil {
		if err := c.cache.SetCreatorHistory(ctx, chainID, contract, items); err != nil {
			return err
		}
	}
	at := appcomponents.NowUTC()
	if err := appcomponents.MarkComponentSuccess(ctx, c.store, chainID, contract, appstore.ProjectComponentCreatorHistory, at); err != nil {
		return err
	}
	return nil
}

