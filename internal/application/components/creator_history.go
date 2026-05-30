package components

import (
	"context"

	"github.com/ethereum/go-ethereum/common"
	appcache "github.com/useryege/athena/internal/application/cache"
	appstore "github.com/useryege/athena/internal/application/store"
)

type CreatorHistoryComponent struct {
	store    appstore.Store
	cache    appcache.ProjectComponentCache
	bus      EventBus
	consumer *Consumer
}

func NewCreatorHistoryComponent(store appstore.Store, cache appcache.ProjectComponentCache, bus EventBus) *CreatorHistoryComponent {
	if store == nil {
		return nil
	}
	c := &CreatorHistoryComponent{store: store, cache: cache, bus: bus}
	c.consumer = NewConsumer(appstore.ProjectComponentCreatorHistory, bus, c.handleEvent)
	return c
}

func (c *CreatorHistoryComponent) Start(ctx context.Context) error {
	if c == nil || c.consumer == nil {
		return nil
	}
	return c.consumer.Start(ctx)
}

func (c *CreatorHistoryComponent) Stop() error {
	if c == nil || c.consumer == nil {
		return nil
	}
	return c.consumer.Stop()
}

func (c *CreatorHistoryComponent) handleEvent(ctx context.Context, event Event) error {
	if event.Type != EventProjectInitialized && event.Type != EventProjectRefresh {
		return nil
	}
	contract := event.ProjectContract()
	if contract == (common.Address{}) {
		return nil
	}
	if event.Type != EventProjectRefresh {
		done, err := ComponentSucceeded(ctx, c.store, contract, appstore.ProjectComponentCreatorHistory)
		if err != nil || done {
			return nil
		}
	}
	if err := c.refresh(ctx, contract); err != nil {
		_ = MarkComponentFailed(ctx, c.store, contract, appstore.ProjectComponentCreatorHistory, err, nowUTC())
		if c.bus != nil {
			_ = c.bus.Publish(ctx, ComponentFailedEvent(contract, appstore.ProjectComponentCreatorHistory, err))
		}
		return nil
	}
	return nil
}

func (c *CreatorHistoryComponent) refresh(ctx context.Context, contract common.Address) error {
	base, err := LoadProjectBase(ctx, c.cache, c.store, contract)
	if err != nil || base == nil {
		return err
	}
	if err := MarkComponentRunning(ctx, c.store, contract, appstore.ProjectComponentCreatorHistory, nowUTC()); err != nil {
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
	if err := MarkComponentSuccess(ctx, c.store, contract, appstore.ProjectComponentCreatorHistory, at); err != nil {
		return err
	}
	if c.bus != nil {
		return c.bus.Publish(ctx, ComponentCompletedEvent(contract, appstore.ProjectComponentCreatorHistory))
	}
	return nil
}
