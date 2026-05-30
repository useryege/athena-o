package creatorhistory

import (
	"context"
	"time"

	"github.com/ethereum/go-ethereum/common"
	appcache "github.com/useryege/athena/internal/application/cache"
	appcomponents "github.com/useryege/athena/internal/application/components"
	appstore "github.com/useryege/athena/internal/application/store"
)

type Options struct {
	Store appstore.Store
	Cache appcache.ProjectComponentCache
	Bus   appcomponents.EventBus
}

type Component struct {
	store    appstore.Store
	cache    appcache.ProjectComponentCache
	bus      appcomponents.EventBus
	consumer *appcomponents.Consumer
}

func NewComponent(opts Options) *Component {
	if opts.Store == nil {
		return nil
	}
	c := &Component{store: opts.Store, cache: opts.Cache, bus: opts.Bus}
	c.consumer = appcomponents.NewConsumer(appstore.ProjectComponentCreatorHistory, opts.Bus, c.handleEvent)
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
	if event.Type != appcomponents.EventProjectRefresh {
		done, err := appcomponents.ComponentSucceeded(ctx, c.store, contract, appstore.ProjectComponentCreatorHistory)
		if err != nil || done {
			return nil
		}
	}
	if err := c.refresh(ctx, contract); err != nil {
		_ = appcomponents.MarkComponentFailed(ctx, c.store, contract, appstore.ProjectComponentCreatorHistory, err, nowUTC())
		if c.bus != nil {
			_ = c.bus.Publish(ctx, appcomponents.ComponentFailedEvent(contract, appstore.ProjectComponentCreatorHistory, err))
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
	if err := appcomponents.MarkComponentRunning(ctx, c.store, contract, appstore.ProjectComponentCreatorHistory, nowUTC()); err != nil {
		return err
	}
	metas, err := c.store.ListProjectMetasByCreatorBefore(ctx, base.Creator, base.BlockNumber, base.TxIndex)
	if err != nil {
		return err
	}
	items := make([]appstore.ProjectCreatorHistoricalProject, 0, len(metas))
	seen := map[common.Address]struct{}{}
	for _, meta := range metas {
		if meta.Contract == (common.Address{}) || meta.Contract == contract {
			continue
		}
		if _, ok := seen[meta.Contract]; ok {
			continue
		}
		seen[meta.Contract] = struct{}{}
		items = append(items, appstore.ProjectCreatorHistoricalProject{
			ProjectContract:           contract,
			HistoricalProjectContract: meta.Contract,
			RankIndex:                 int32(len(items)),
		})
	}
	if err := c.store.ReplaceProjectCreatorHistoricalProjects(ctx, contract, items); err != nil {
		return err
	}
	if c.cache != nil {
		if err := c.cache.SetCreatorHistory(ctx, contract, items); err != nil {
			return err
		}
	}
	at := nowUTC()
	if err := appcomponents.MarkComponentSuccess(ctx, c.store, contract, appstore.ProjectComponentCreatorHistory, at); err != nil {
		return err
	}
	if c.bus != nil {
		return c.bus.Publish(ctx, appcomponents.ComponentCompletedEvent(contract, appstore.ProjectComponentCreatorHistory))
	}
	return nil
}

func nowUTC() time.Time {
	return time.Now().UTC()
}
