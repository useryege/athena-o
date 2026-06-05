package ave

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	log "github.com/sirupsen/logrus"
	appstore "github.com/useryege/athena/internal/application/store"
	utilave "github.com/useryege/athena/util/ave"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	DefaultDetailTTL         = 24 * time.Hour
	DefaultPollInterval      = 30 * time.Second
	DefaultFailureRetryDelay = 10 * time.Minute
	DefaultBatchSize         = 20
)

type Store interface {
	appstore.ProjectAveDetailStore
	appstore.ProjectAveRefreshStore
}

type Cache interface {
	SetAveDetail(ctx context.Context, chainID int64, contract common.Address, item appstore.ProjectAveDetail) error
	GetAveDetail(ctx context.Context, chainID int64, contract common.Address) (*appstore.ProjectAveDetail, bool, error)
}

type Options struct {
	Config            utilave.Config
	ChainID           int64
	Store             Store
	Cache             Cache
	Fetcher           Fetcher
	Chain             string
	DetailTTL         time.Duration
	PollInterval      time.Duration
	FailureRetryDelay time.Duration
	BatchSize         int32
	Now               func() time.Time
}

type State struct {
	Contract        common.Address
	Detail          *appstore.ProjectAveDetail
	ComponentState  *appstore.ProjectComponentState
	DetailAvailable bool
	Stale           bool
}

type Component struct {
	chainID           int64
	store             Store
	cache             Cache
	fetcher           Fetcher
	chain             string
	detailTTL         time.Duration
	pollInterval      time.Duration
	failureRetryDelay time.Duration
	batchSize         int32
	now               func() time.Time

	startStopMu sync.Mutex
	cancel      context.CancelFunc
	wg          sync.WaitGroup
	started     bool
}

func NewComponent(opts Options) (*Component, error) {
	if opts.Store == nil {
		return nil, nil
	}
	opts = opts.withDefaults()
	if opts.Fetcher == nil {
		if strings.TrimSpace(opts.Config.APIKey) != "" {
			chain, ok := ChainNameForChainID(opts.ChainID)
			if !ok {
				log.WithField("chainID", opts.ChainID).Warn("Ave component disabled for unsupported chain")
			} else {
				client, err := utilave.NewClient(opts.Config)
				if err != nil {
					return nil, fmt.Errorf("configure Ave component: %w", err)
				}
				opts.Fetcher = NewFetcher(client)
				opts.Chain = chain
			}
		}
	} else if strings.TrimSpace(opts.Chain) == "" {
		chain, ok := ChainNameForChainID(opts.ChainID)
		if ok {
			opts.Chain = chain
		}
	}
	component := &Component{
		store:             opts.Store,
		chainID:           opts.ChainID,
		cache:             opts.Cache,
		fetcher:           opts.Fetcher,
		chain:             strings.TrimSpace(opts.Chain),
		detailTTL:         opts.DetailTTL,
		pollInterval:      opts.PollInterval,
		failureRetryDelay: opts.FailureRetryDelay,
		batchSize:         opts.BatchSize,
		now:               opts.Now,
	}
	return component, nil
}

func (o Options) withDefaults() Options {
	if o.DetailTTL <= 0 {
		o.DetailTTL = DefaultDetailTTL
	}
	if o.PollInterval <= 0 {
		o.PollInterval = DefaultPollInterval
	}
	if o.FailureRetryDelay <= 0 {
		o.FailureRetryDelay = DefaultFailureRetryDelay
	}
	if o.BatchSize <= 0 {
		o.BatchSize = DefaultBatchSize
	}
	if o.Now == nil {
		o.Now = func() time.Time { return time.Now().UTC() }
	}
	return o
}

func (c *Component) Start(ctx context.Context) error {
	if c == nil || c.store == nil || c.fetcher == nil || strings.TrimSpace(c.chain) == "" {
		return nil
	}
	c.startStopMu.Lock()
	defer c.startStopMu.Unlock()
	if c.started {
		return nil
	}
	runCtx, cancel := context.WithCancel(ctx)
	c.cancel = cancel
	c.started = true
	c.wg.Add(1)
	go c.run(runCtx)
	return nil
}

func (c *Component) Stop() {
	if c == nil {
		return
	}
	c.startStopMu.Lock()
	cancel := c.cancel
	c.cancel = nil
	c.started = false
	c.startStopMu.Unlock()
	if cancel != nil {
		cancel()
	}
	c.wg.Wait()
}

func (c *Component) ScheduleRefresh(ctx context.Context, contract common.Address) error {
	if c == nil || c.store == nil {
		return status.Error(codes.FailedPrecondition, "Ave component is not configured")
	}
	if c.fetcher == nil || strings.TrimSpace(c.chain) == "" {
		return status.Error(codes.FailedPrecondition, "Ave refresh is not configured")
	}
	if contract == (common.Address{}) {
		return status.Error(codes.InvalidArgument, "Ave refresh contract is empty")
	}
	return c.store.ScheduleProjectAveRefresh(ctx, c.chainID, contract, c.now())
}

func (c *Component) State(ctx context.Context, contract common.Address) (*State, error) {
	if c == nil || c.store == nil {
		return nil, status.Error(codes.FailedPrecondition, "Ave component is not configured")
	}
	if contract == (common.Address{}) {
		return nil, status.Error(codes.InvalidArgument, "Ave state contract is empty")
	}
	detail, err := c.detail(ctx, contract)
	if err != nil {
		return nil, err
	}
	componentState, err := c.store.GetProjectAveComponentState(ctx, c.chainID, contract)
	if err != nil {
		return nil, err
	}
	return &State{
		Contract:        contract,
		Detail:          detail,
		ComponentState:  componentState,
		DetailAvailable: detail != nil,
		Stale:           detail == nil || detail.FetchedAt.Before(c.now().Add(-c.detailTTL)),
	}, nil
}

func (c *Component) RefreshDetail(ctx context.Context, contract common.Address) error {
	if c == nil || c.store == nil || c.fetcher == nil || strings.TrimSpace(c.chain) == "" {
		return nil
	}
	if contract == (common.Address{}) {
		return nil
	}
	return c.refreshOne(ctx, contract)
}

func (c *Component) run(ctx context.Context) {
	defer c.wg.Done()
	c.runOnce(ctx)
	ticker := time.NewTicker(c.pollInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			c.runOnce(ctx)
		}
	}
}

func (c *Component) runOnce(ctx context.Context) {
	now := c.now()
	candidates, err := c.store.ListProjectAveRefreshCandidates(ctx, c.chainID, now.Add(-c.detailTTL), now, c.batchSize)
	if err != nil {
		if !errors.Is(err, context.Canceled) {
			log.WithError(err).Warn("failed to list Ave refresh candidates")
		}
		return
	}
	for _, contract := range candidates {
		if err := ctx.Err(); err != nil {
			return
		}
		if err := c.refreshOne(ctx, contract); err != nil {
			log.WithFields(log.Fields{"contract": contract.Hex(), "error": err.Error()}).Warn("failed to refresh Ave detail")
		}
	}
}

func (c *Component) refreshOne(ctx context.Context, contract common.Address) error {
	if contract == (common.Address{}) {
		return nil
	}
	attemptAt := c.now()
	if err := c.store.MarkProjectAveRefreshRunning(ctx, c.chainID, contract, attemptAt); err != nil {
		return err
	}
	response, err := c.fetcher.FetchDetail(ctx, tokenID(contract, c.chain))
	if err != nil {
		_ = c.store.MarkProjectAveRefreshFailed(ctx, c.chainID, contract, attemptAt, attemptAt.Add(c.failureRetryDelay), err.Error())
		return err
	}
	detail := DetailFromResponse(response, c.now())
	if detail == nil {
		err := errors.New("Ave token detail response is empty")
		_ = c.store.MarkProjectAveRefreshFailed(ctx, c.chainID, contract, attemptAt, attemptAt.Add(c.failureRetryDelay), err.Error())
		return err
	}
	if err := c.store.UpsertProjectAveDetail(ctx, c.chainID, contract, *detail); err != nil {
		_ = c.store.MarkProjectAveRefreshFailed(ctx, c.chainID, contract, attemptAt, attemptAt.Add(c.failureRetryDelay), err.Error())
		return err
	}
	if c.cache != nil {
		if err := c.cache.SetAveDetail(ctx, c.chainID, contract, *detail); err != nil {
			return err
		}
	}
	if err := c.store.MarkProjectAveRefreshSuccess(ctx, c.chainID, contract, detail.FetchedAt, detail.FetchedAt.Add(c.detailTTL)); err != nil {
		return err
	}
	return nil
}

func (c *Component) detail(ctx context.Context, contract common.Address) (*appstore.ProjectAveDetail, error) {
	if c.cache != nil {
		if item, ok, err := c.cache.GetAveDetail(ctx, c.chainID, contract); err != nil {
			return nil, err
		} else if ok && item != nil {
			return item, nil
		}
	}
	item, err := c.store.GetProjectAveDetail(ctx, c.chainID, contract)
	if err != nil || item == nil {
		return item, err
	}
	if c.cache != nil {
		if err := c.cache.SetAveDetail(ctx, c.chainID, contract, *item); err != nil {
			return nil, err
		}
	}
	return item, nil
}

func tokenID(contract common.Address, chain string) string {
	return strings.ToLower(contract.Hex()) + "-" + strings.TrimSpace(chain)
}
