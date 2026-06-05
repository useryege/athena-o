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
	ChainID int64
	Store   appstore.Store
	Cache   appcache.ProjectComponentCache
}

type Component struct {
	chainID int64
	store   appstore.ProjectBaseStore
	intake  appstore.ProjectIntakeStore
	state   appstore.ProjectComponentStateStore
	cache   appcache.ProjectComponentCache
}

func NewComponent(opts Options) *Component {
	if opts.Store == nil {
		return nil
	}
	return &Component{
		chainID: opts.ChainID,
		store:   opts.Store,
		intake:  opts.Store,
		state:   opts.Store,
		cache:   opts.Cache,
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
		if candidate.ChainID <= 0 {
			candidate.ChainID = c.chainID
		}
		base := appcomponents.ProjectBaseFromCandidate(candidate)
		if err := appcomponents.MarkComponentRunning(ctx, c.state, candidate.ChainID, candidate.Contract, appstore.ProjectComponentInitializer, nowUTC()); err != nil {
			return err
		}
		if err := c.store.SaveProjectBase(ctx, base); err != nil {
			_ = appcomponents.MarkComponentFailed(ctx, c.state, candidate.ChainID, candidate.Contract, appstore.ProjectComponentInitializer, err, nowUTC())
			return err
		}
		if c.cache != nil {
			if err := c.cache.SetBase(ctx, base); err != nil {
				_ = appcomponents.MarkComponentFailed(ctx, c.state, candidate.ChainID, candidate.Contract, appstore.ProjectComponentInitializer, err, nowUTC())
				return err
			}
		}
		if err := appcomponents.MarkComponentSuccess(ctx, c.state, candidate.ChainID, candidate.Contract, appstore.ProjectComponentInitializer, nowUTC()); err != nil {
			return err
		}
	}
	return nil
}

func (c *Component) ScheduleProjects(ctx context.Context, items []model.DiscoveredProjectCandidate) error {
	if c == nil || c.intake == nil {
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
		chainID := candidate.ChainID
		if chainID <= 0 {
			chainID = c.chainID
		}
		if err := c.intake.EnqueueProjectCollection(ctx, model.ProjectRef{ChainID: chainID, Contract: candidate.Contract}, string(source)); err != nil {
			return err
		}
	}
	return nil
}

func nowUTC() time.Time {
	return time.Now().UTC()
}
