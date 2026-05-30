package components

import (
	"context"
	"time"

	"github.com/ethereum/go-ethereum/common"
	appcache "github.com/useryege/athena/internal/application/cache"
	"github.com/useryege/athena/internal/application/model"
	appstore "github.com/useryege/athena/internal/application/store"
)

type Initializer struct {
	store appstore.ProjectBaseStore
	state appstore.ProjectComponentStateStore
	cache appcache.ProjectComponentCache
	bus   EventBus
}

func NewInitializer(store appstore.Store, cache appcache.ProjectComponentCache, bus EventBus) *Initializer {
	if store == nil {
		return nil
	}
	return &Initializer{
		store: store,
		state: store,
		cache: cache,
		bus:   bus,
	}
}

func (c *Initializer) Start(context.Context) error { return nil }

func (c *Initializer) Stop() error { return nil }

func (c *Initializer) IntakeCandidates(ctx context.Context, items []model.DiscoveredProjectCandidate) error {
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
		base := ProjectBaseFromCandidate(candidate)
		if err := MarkComponentRunning(ctx, c.state, candidate.Contract, appstore.ProjectComponentInitializer, nowUTC()); err != nil {
			return err
		}
		if err := c.store.SaveProjectBase(ctx, base); err != nil {
			_ = MarkComponentFailed(ctx, c.state, candidate.Contract, appstore.ProjectComponentInitializer, err, nowUTC())
			if c.bus != nil {
				_ = c.bus.Publish(ctx, ComponentFailedEvent(candidate.Contract, appstore.ProjectComponentInitializer, err))
			}
			return err
		}
		if c.cache != nil {
			if err := c.cache.SetBase(ctx, base); err != nil {
				_ = MarkComponentFailed(ctx, c.state, candidate.Contract, appstore.ProjectComponentInitializer, err, nowUTC())
				return err
			}
		}
		if err := MarkComponentSuccess(ctx, c.state, candidate.Contract, appstore.ProjectComponentInitializer, nowUTC()); err != nil {
			return err
		}
		if c.bus != nil {
			if err := c.bus.Publish(ctx, ProjectInitializedEvent(candidate)); err != nil {
				return err
			}
			if err := c.bus.Publish(ctx, ComponentCompletedEvent(candidate.Contract, appstore.ProjectComponentInitializer)); err != nil {
				return err
			}
		}
	}
	return nil
}

func (c *Initializer) ScheduleProjects(ctx context.Context, items []model.DiscoveredProjectCandidate) error {
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
		if err := c.bus.Publish(ctx, ProjectRefreshEvent(candidate.Contract, source)); err != nil {
			return err
		}
	}
	return nil
}

func nowUTC() time.Time {
	return time.Now().UTC()
}
