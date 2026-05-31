package persistence

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	log "github.com/sirupsen/logrus"
	appstore "github.com/useryege/athena/internal/application/store"
)

const defaultFlushInterval = time.Minute

type Flusher struct {
	store    *RedisBufferedStore
	interval time.Duration

	cancel context.CancelFunc
	done   chan struct{}
	mu     sync.Mutex
}

func NewFlusher(store *RedisBufferedStore, interval time.Duration) *Flusher {
	if store == nil {
		return nil
	}
	if interval <= 0 {
		interval = defaultFlushInterval
	}
	return &Flusher{store: store, interval: interval}
}

func (f *Flusher) Start(ctx context.Context) error {
	if f == nil {
		return nil
	}
	f.mu.Lock()
	if f.cancel != nil {
		f.mu.Unlock()
		return nil
	}
	runCtx, cancel := context.WithCancel(ctx)
	f.cancel = cancel
	f.done = make(chan struct{})
	f.mu.Unlock()

	go f.loop(runCtx)
	return nil
}

func (f *Flusher) Stop() error {
	if f == nil {
		return nil
	}
	f.mu.Lock()
	cancel := f.cancel
	done := f.done
	f.cancel = nil
	f.done = nil
	f.mu.Unlock()
	if cancel != nil {
		cancel()
	}
	if done != nil {
		<-done
	}
	ctx, cancelFlush := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancelFlush()
	return f.Flush(ctx)
}

func (f *Flusher) loop(ctx context.Context) {
	defer close(f.done)
	ticker := time.NewTicker(f.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := f.Flush(ctx); err != nil && !errors.Is(err, context.Canceled) {
				log.WithError(err).Warn("application redis buffered persistence flush failed")
			}
		}
	}
}

func (f *Flusher) Flush(ctx context.Context) error {
	if f == nil || f.store == nil {
		return nil
	}
	errs := make([]error, 0)
	for _, kind := range bufferedFlushOrder {
		if err := f.flushKind(ctx, kind); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

func (f *Flusher) flushKind(ctx context.Context, kind string) error {
	members, err := f.store.dirtyMembers(ctx, kind)
	if err != nil {
		return err
	}
	errs := make([]error, 0)
	for _, member := range members {
		if err := ctx.Err(); err != nil {
			return err
		}
		if err := f.flushMember(ctx, kind, member); err != nil {
			errs = append(errs, fmt.Errorf("%s %s: %w", kind, member, err))
			continue
		}
		if err := f.store.clearDirty(ctx, kind, member); err != nil {
			errs = append(errs, fmt.Errorf("clear dirty %s %s: %w", kind, member, err))
		}
	}
	return errors.Join(errs...)
}

func (f *Flusher) flushMember(ctx context.Context, kind string, member string) error {
	switch kind {
	case bufferedBaseKind:
		item, ok, err := getJSON[appstore.ProjectBase](ctx, f.store.client, bufferedItemKey(kind, member))
		if err != nil || !ok {
			return err
		}
		return f.store.db.SaveProjectBase(ctx, item)
	case bufferedChainStateKind:
		item, ok, err := getJSON[appstore.ProjectChainState](ctx, f.store.client, bufferedItemKey(kind, member))
		if err != nil || !ok {
			return err
		}
		return f.store.db.UpsertProjectChainState(ctx, item)
	case bufferedSimulationKind:
		item, ok, err := getJSON[appstore.ProjectSimulationResult](ctx, f.store.client, bufferedItemKey(kind, member))
		if err != nil || !ok {
			return err
		}
		return f.store.db.UpsertProjectSimulationResult(ctx, item)
	case bufferedReportKind:
		item, ok, err := getJSON[appstore.ProjectReportState](ctx, f.store.client, bufferedItemKey(kind, member))
		if err != nil || !ok {
			return err
		}
		return f.store.db.UpsertProjectReportState(ctx, item)
	case bufferedBytecodeKind:
		item, ok, err := getJSON[appstore.ProjectBytecodeFact](ctx, f.store.client, bufferedItemKey(kind, member))
		if err != nil || !ok {
			return err
		}
		return f.store.db.UpsertProjectBytecodeFact(ctx, item)
	case bufferedAveKind:
		item, ok, err := getJSON[appstore.ProjectAveDetail](ctx, f.store.client, bufferedItemKey(kind, member))
		if err != nil || !ok {
			return err
		}
		return f.store.db.UpsertProjectAveDetail(ctx, common.HexToAddress(member), item)
	case bufferedGenesisKind:
		items, ok, err := getJSON[[]appstore.ProjectGenesisWallet](ctx, f.store.client, bufferedItemKey(kind, member))
		if err != nil || !ok {
			return err
		}
		return f.store.db.ReplaceProjectGenesisWallets(ctx, common.HexToAddress(member), items)
	case bufferedCreatorHistoryKind:
		items, ok, err := getJSON[[]appstore.ProjectCreatorHistoricalProject](ctx, f.store.client, bufferedItemKey(kind, member))
		if err != nil || !ok {
			return err
		}
		return f.store.db.ReplaceProjectCreatorHistoricalProjects(ctx, common.HexToAddress(member), items)
	case bufferedComponentStateKind:
		item, ok, err := getJSON[appstore.ProjectComponentState](ctx, f.store.client, bufferedItemKey(kind, member))
		if err != nil || !ok {
			return err
		}
		return f.store.db.UpsertProjectComponentState(ctx, item)
	default:
		return fmt.Errorf("unsupported dirty kind %q", kind)
	}
}
