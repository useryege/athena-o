package initializer

import (
	"context"
	"time"

	"github.com/ethereum/go-ethereum/common"
	appcache "github.com/useryege/athena/internal/application/cache"
	appcomponents "github.com/useryege/athena/internal/application/components"
	"github.com/useryege/athena/internal/application/model"
	appstore "github.com/useryege/athena/internal/application/store"
)

type Options struct {
	Store appstore.Store
	Cache appcache.ProjectComponentCache
	Bus   appcomponents.EventBus
}

type Component struct {
	store appstore.ProjectBaseStore
	state appstore.ProjectComponentStateStore
	cache appcache.ProjectComponentCache
	bus   appcomponents.EventBus
}

func NewComponent(opts Options) *Component {
	if opts.Store == nil {
		return nil
	}
	return &Component{
		store: opts.Store,
		state: opts.Store,
		cache: opts.Cache,
		bus:   opts.Bus,
	}
}

func (c *Component) Start(context.Context) error { return nil }

func (c *Component) Stop() error { return nil }

func (c *Component) IntakeCandidates(ctx context.Context, items []model.DiscoveredProjectCandidate) error {
	if c == nil || c.store == nil {
		return nil
	}
	for _, candidate := range items {
		if candidate.Contract == (common.Address{}) {
			continue
		}
		if candidate.Source == "" {
			candidate.Source = model.ProjectDiscoverySourceCatchUp
		}
		base := appcomponents.ProjectBaseFromCandidate(candidate)
		if err := appcomponents.MarkComponentRunning(ctx, c.state, candidate.Contract, appstore.ProjectComponentInitializer, nowUTC()); err != nil {
			return err
		}
		if err := c.store.SaveProjectBase(ctx, base); err != nil {
			_ = appcomponents.MarkComponentFailed(ctx, c.state, candidate.Contract, appstore.ProjectComponentInitializer, err, nowUTC())
			if c.bus != nil {
				_ = c.bus.Publish(ctx, appcomponents.ComponentFailedEvent(candidate.Contract, appstore.ProjectComponentInitializer, err))
			}
			return err
		}
		if c.cache != nil {
			if err := c.cache.SetBase(ctx, base); err != nil {
				_ = appcomponents.MarkComponentFailed(ctx, c.state, candidate.Contract, appstore.ProjectComponentInitializer, err, nowUTC())
				return err
			}
		}
		if err := appcomponents.MarkComponentSuccess(ctx, c.state, candidate.Contract, appstore.ProjectComponentInitializer, nowUTC()); err != nil {
			return err
		}
		if c.bus != nil {
			if err := c.bus.Publish(ctx, appcomponents.ProjectInitializedEvent(candidate)); err != nil {
				return err
			}
			if err := c.bus.Publish(ctx, appcomponents.ComponentCompletedEvent(candidate.Contract, appstore.ProjectComponentInitializer)); err != nil {
				return err
			}
		}
	}
	return nil
}

func (c *Component) ScheduleProjects(ctx context.Context, items []model.DiscoveredProjectCandidate) error {
	if c == nil || c.bus == nil {
		return nil
	}
	for _, candidate := range items {
		if candidate.Contract == (common.Address{}) {
			continue
		}
		source := candidate.Source
		if source == "" {
			source = model.ProjectDiscoverySourcePairSwap
		}
		if err := c.bus.Publish(ctx, appcomponents.ProjectRefreshEvent(candidate.Contract, source)); err != nil {
			return err
		}
	}
	return nil
}

func nowUTC() time.Time {
	return time.Now().UTC()
}
