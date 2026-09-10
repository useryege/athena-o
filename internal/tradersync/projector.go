package tradersync

import (
	"context"
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	et "github.com/ethereum/go-ethereum/core/types"
	"github.com/useryege/athena/internal/tradersync/activity"
	tm "github.com/useryege/athena/internal/tradersync/types"
	"sync"
	"sync/atomic"
	"time"
)

type ProjectionStore interface {
	ProjectionSources(context.Context, int) ([]tm.ProjectionSource, error)
	SaveProjectionEvidence(context.Context, int64, tm.CanonicalEvidence, *tm.Trade) error
	Project(context.Context, tm.Projection) (int64, bool, error)
	SaveMetadata(context.Context, string, tm.TradeMetadata) error
	CompleteProjectionMetadata(context.Context, int64, bool) error
}
type ProjectionVersion interface {
	Verify(context.Context, et.Log) (string, error)
}
type ProjectionMetadata interface {
	ResolveProgress(context.Context, tm.Trade, common.Hash, func(tm.TradeMetadata)) tm.TradeMetadata
}
type ProjectorConfig struct {
	Interval, MetadataWait, MetadataTimeout time.Duration
	MaxInFlightSources                      int
}
type Projector struct {
	store    ProjectionStore
	node     CanonicalRPC
	version  ProjectionVersion
	metadata ProjectionMetadata
	config   ProjectorConfig
	running  atomic.Bool
}

func NewProjector(store ProjectionStore, node CanonicalRPC, version ProjectionVersion, metadata ProjectionMetadata, config ProjectorConfig) (*Projector, error) {
	if store == nil || node == nil || version == nil || metadata == nil {
		return nil, fmt.Errorf("projector dependencies required")
	}
	if config.Interval == 0 {
		config.Interval = 2 * time.Second
	}
	if config.MetadataWait == 0 {
		config.MetadataWait = 2 * time.Second
	}
	if config.MetadataTimeout == 0 {
		config.MetadataTimeout = 30 * time.Second
	}
	if config.MaxInFlightSources == 0 {
		config.MaxInFlightSources = 100
	}
	if config.Interval < 0 || config.MetadataWait < 0 || config.MetadataWait > 2*time.Second || config.MetadataTimeout < 0 || config.MaxInFlightSources < 1 {
		return nil, fmt.Errorf("invalid projector resource limits")
	}
	return &Projector{store: store, node: node, version: version, metadata: metadata, config: config}, nil
}
func (p *Projector) Run(ctx context.Context) error {
	if !p.running.CompareAndSwap(false, true) {
		return fmt.Errorf("projector already running")
	}
	defer p.running.Store(false)
	ctx, cancel := context.WithCancel(ctx)
	var workers sync.WaitGroup
	defer func() { cancel(); workers.Wait() }()
	type completion struct {
		id  int64
		err error
	}
	done := make(chan completion, p.config.MaxInFlightSources)
	active := map[int64]bool{}
	tick := time.NewTicker(p.config.Interval)
	defer tick.Stop()
	schedule := func() error {
		free := p.config.MaxInFlightSources - len(active)
		if free <= 0 {
			return nil
		}
		sources, e := p.store.ProjectionSources(ctx, p.config.MaxInFlightSources)
		if e != nil {
			return e
		}
		for _, source := range sources {
			if active[source.ID] {
				continue
			}
			if len(active) >= p.config.MaxInFlightSources {
				break
			}
			active[source.ID] = true
			workers.Add(1)
			go func(source tm.ProjectionSource) {
				defer workers.Done()
				err := p.process(ctx, source)
				done <- completion{source.ID, err}
			}(source)
		}
		return nil
	}
	if e := schedule(); e != nil {
		return e
	}
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case result := <-done:
			delete(active, result.id)
			if result.err != nil && ctx.Err() == nil {
				return result.err
			}
		case <-tick.C:
			if e := schedule(); e != nil {
				return e
			}
		}
	}
}

func (p *Projector) process(ctx context.Context, source tm.ProjectionSource) error {
	jobCtx, cancel := context.WithCancel(ctx)
	var workers sync.WaitGroup
	defer func() { cancel(); workers.Wait() }()
	// All spawned work is owned by this one bounded source slot, including late metadata.
	type versionResult struct {
		trade tm.Trade
		err   error
	}
	versions := make(chan versionResult, 1)
	type confirmationResult struct {
		evidence tm.CanonicalEvidence
		at       time.Time
	}
	confirmations := make(chan confirmationResult, 1)
	workers.Add(2)
	go func() {
		defer workers.Done()
		version, e := p.version.Verify(jobCtx, source.Raw)
		var trade tm.Trade
		if e == nil {
			trade, e = DecodeOwnTrade(source.Raw, version)
		}
		versions <- versionResult{trade, e}
	}()
	go func() {
		defer workers.Done()
		e, _ := ConfirmReceived(jobCtx, p.node, source.Raw)
		confirmations <- confirmationResult{e, time.Now()}
	}()
	var trade tm.Trade
	var confirmation confirmationResult
	versionReady, confirmationReady := false, false
	var latest tm.TradeMetadata
	var latestMu sync.Mutex
	metadataDone := make(chan tm.TradeMetadata, 1)
	metadataStarted := false
	createdAt := time.Now()
	for !versionReady || !confirmationReady {
		select {
		case <-ctx.Done():
			cancel()
			return ctx.Err()
		case v := <-versions:
			versionReady = true
			if v.err != nil {
				cancel()
				return p.store.SaveProjectionEvidence(ctx, source.ID, tm.CanonicalEvidence{Status: "unverified", BlockHash: source.Raw.BlockHash, CheckedAt: time.Now().UTC(), Reason: "source_version_or_decode_unverified"}, nil)
			}
			trade = v.trade
			latest = tm.TradeMetadata{Market: missingMarket(trade.PositionID, "metadata_pending", "metadata"), LegsEvidence: metadataEvidence("unavailable", "metadata_pending", "metadata")}
			if trade.SourceVersion != ComboExchangeVersion {
				latest.LegsEvidence.ReasonCode = "not_combo"
			}
			metadataStarted = true
			workers.Add(1)
			go func() {
				defer workers.Done()
				metadataCtx, stop := context.WithDeadline(jobCtx, createdAt.Add(p.config.MetadataTimeout))
				defer stop()
				result := p.metadata.ResolveProgress(metadataCtx, trade, source.Raw.BlockHash, func(v tm.TradeMetadata) {
					latestMu.Lock()
					latest = activity.MergeMetadata(latest, v)
					latestMu.Unlock()
				})
				metadataDone <- result
			}()
		case c := <-confirmations:
			confirmation = c
			confirmationReady = true
			if c.evidence.Status != "confirmed" {
				cancel()
				return p.store.SaveProjectionEvidence(ctx, source.ID, c.evidence, nil)
			}
		}
	}
	if e := p.store.SaveProjectionEvidence(ctx, source.ID, confirmation.evidence, &trade); e != nil {
		cancel()
		return e
	}
	deadline := confirmation.at.Add(p.config.MetadataWait)
	timer := time.NewTimer(max(time.Duration(0), time.Until(deadline)))
	defer timer.Stop()
	var final tm.TradeMetadata
	finished := false
	select {
	case <-ctx.Done():
		cancel()
		return ctx.Err()
	case final = <-metadataDone:
		finished = true
		latestMu.Lock()
		latest = activity.MergeMetadata(latest, final)
		latestMu.Unlock()
	case <-timer.C:
	}
	latestMu.Lock()
	snapshot := activity.CloneMetadata(latest)
	latestMu.Unlock()
	for _, candidate := range source.Candidates {
		if _, _, e := p.store.Project(ctx, tm.Projection{Candidate: candidate, Trade: trade, Confirmation: confirmation.evidence, Metadata: snapshot}); e != nil {
			cancel()
			return e
		}
	}
	if metadataStarted && !finished {
		select {
		case <-ctx.Done():
			cancel()
			return ctx.Err()
		case final = <-metadataDone:
		}
	}
	final = activity.MergeMetadata(snapshot, final)
	if e := p.store.SaveMetadata(ctx, activity.MetadataKey(trade, source.Raw.BlockHash.Hex()), final); e != nil {
		return e
	}
	return p.store.CompleteProjectionMetadata(ctx, source.ID, completeMetadata(final, trade.SourceVersion == ComboExchangeVersion))
}
