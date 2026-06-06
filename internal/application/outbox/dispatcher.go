package outbox

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	log "github.com/sirupsen/logrus"
	"github.com/useryege/athena/internal/application/store"
	"github.com/useryege/athena/internal/application/workflows"
)

const (
	DefaultBatchSize            int32 = 20
	DefaultPollInterval               = 2 * time.Second
	DefaultRetryInitialInterval       = 5 * time.Second
	DefaultRetryMaxInterval           = 5 * time.Minute
)

var dispatchTypes = []string{
	store.OutboxTypeProjectCollectionRequested,
}

type WorkflowStarter interface {
	StartProjectCollection(ctx context.Context, input workflows.ProjectCollectionInput) (string, error)
}

type Store interface {
	store.OutboxStore
	store.ProjectCollectionStateStore
}

type Options struct {
	Store                Store
	Starter              WorkflowStarter
	LockedBy             string
	BatchSize            int32
	PollInterval         time.Duration
	RetryInitialInterval time.Duration
	RetryMaxInterval     time.Duration
	Now                  func() time.Time
}

type Dispatcher struct {
	store                Store
	starter              WorkflowStarter
	lockedBy             string
	batchSize            int32
	pollInterval         time.Duration
	retryInitialInterval time.Duration
	retryMaxInterval     time.Duration
	now                  func() time.Time

	mu      sync.Mutex
	cancel  context.CancelFunc
	wg      sync.WaitGroup
	started bool
}

func NewDispatcher(opts Options) (*Dispatcher, error) {
	if opts.Store == nil {
		return nil, errors.New("application outbox store is required")
	}
	if opts.Starter == nil {
		return nil, errors.New("application outbox workflow starter is required")
	}
	if strings.TrimSpace(opts.LockedBy) == "" {
		opts.LockedBy = fmt.Sprintf("application-outbox-%d", time.Now().UnixNano())
	}
	if opts.BatchSize <= 0 {
		opts.BatchSize = DefaultBatchSize
	}
	if opts.PollInterval <= 0 {
		opts.PollInterval = DefaultPollInterval
	}
	if opts.RetryInitialInterval <= 0 {
		opts.RetryInitialInterval = DefaultRetryInitialInterval
	}
	if opts.RetryMaxInterval <= 0 {
		opts.RetryMaxInterval = DefaultRetryMaxInterval
	}
	if opts.Now == nil {
		opts.Now = func() time.Time { return time.Now().UTC() }
	}
	return &Dispatcher{
		store:                opts.Store,
		starter:              opts.Starter,
		lockedBy:             strings.TrimSpace(opts.LockedBy),
		batchSize:            opts.BatchSize,
		pollInterval:         opts.PollInterval,
		retryInitialInterval: opts.RetryInitialInterval,
		retryMaxInterval:     opts.RetryMaxInterval,
		now:                  opts.Now,
	}, nil
}

func (d *Dispatcher) Start(ctx context.Context) error {
	if d == nil {
		return nil
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.started {
		return nil
	}
	runCtx, cancel := context.WithCancel(ctx)
	d.cancel = cancel
	d.started = true
	d.wg.Add(1)
	go d.run(runCtx)
	return nil
}

func (d *Dispatcher) Stop() {
	if d == nil {
		return
	}
	d.mu.Lock()
	cancel := d.cancel
	d.cancel = nil
	d.started = false
	d.mu.Unlock()
	if cancel != nil {
		cancel()
	}
	d.wg.Wait()
}

func (d *Dispatcher) ProcessOnce(ctx context.Context) (int, error) {
	if d == nil {
		return 0, nil
	}
	items, err := d.store.ClaimOutboxEventsByTypes(ctx, d.lockedBy, d.now(), d.batchSize, dispatchTypes)
	if err != nil {
		return 0, err
	}
	var errs []error
	for _, item := range items {
		if err := d.process(ctx, item); err != nil {
			errs = append(errs, err)
		}
	}
	return len(items), errors.Join(errs...)
}

func (d *Dispatcher) run(ctx context.Context) {
	defer d.wg.Done()
	for {
		count, err := d.ProcessOnce(ctx)
		if err != nil && !errors.Is(err, context.Canceled) {
			log.WithError(err).Warn("application outbox dispatch failed")
		}
		wait := d.pollInterval
		if count > 0 && err == nil {
			wait = 0
		}
		if !sleep(ctx, wait) {
			return
		}
	}
}

func (d *Dispatcher) process(ctx context.Context, item store.OutboxEvent) error {
	switch item.Type {
	case store.OutboxTypeProjectCollectionRequested:
		input, err := collectionInput(item)
		if err != nil {
			return d.discard(ctx, item, err)
		}
		workflowID, err := d.starter.StartProjectCollection(ctx, input)
		if err != nil {
			return d.fail(ctx, item, err)
		}
		if err := d.store.MarkProjectCollectionRunning(ctx, input.Project.ChainID, input.Project.Contract, workflowID, d.now()); err != nil {
			return d.fail(ctx, item, err)
		}
	default:
		return d.discard(ctx, item, fmt.Errorf("unsupported application outbox type %q", item.Type))
	}
	if err := d.store.MarkOutboxEventProcessed(ctx, item.ID); err != nil {
		return fmt.Errorf("mark application outbox event %d processed: %w", item.ID, err)
	}
	return nil
}

func (d *Dispatcher) discard(ctx context.Context, item store.OutboxEvent, cause error) error {
	message := errorMessage(cause)
	if err := d.store.MarkOutboxEventDiscarded(ctx, item.ID, message); err != nil {
		return fmt.Errorf("discard application outbox event %d: %w", item.ID, err)
	}
	log.WithFields(log.Fields{"event_id": item.ID, "type": item.Type, "error": message}).Warn("discarded application outbox event")
	return nil
}

func (d *Dispatcher) fail(ctx context.Context, item store.OutboxEvent, cause error) error {
	next := d.now().Add(d.nextRetryDelay(item.Attempts))
	message := errorMessage(cause)
	if err := d.store.MarkOutboxEventFailed(ctx, item.ID, next, message); err != nil {
		return fmt.Errorf("mark application outbox event %d failed: %w", item.ID, err)
	}
	return fmt.Errorf("dispatch application outbox event %d: %w", item.ID, cause)
}

func (d *Dispatcher) nextRetryDelay(attempts int32) time.Duration {
	if attempts <= 1 {
		return d.retryInitialInterval
	}
	delay := d.retryInitialInterval
	for i := int32(1); i < attempts; i++ {
		delay *= 2
		if delay >= d.retryMaxInterval {
			return d.retryMaxInterval
		}
	}
	return delay
}

type collectionPayload struct {
	Project workflows.ProjectRef `json:"project"`
	Reason  string               `json:"reason,omitempty"`
}

func collectionInput(item store.OutboxEvent) (workflows.ProjectCollectionInput, error) {
	var payload collectionPayload
	if err := json.Unmarshal(item.Payload, &payload); err != nil {
		return workflows.ProjectCollectionInput{}, fmt.Errorf("decode project collection payload: %w", err)
	}
	if payload.Project.ChainID <= 0 {
		payload.Project.ChainID = item.ChainID
	}
	if payload.Project.ChainID <= 0 {
		return workflows.ProjectCollectionInput{}, errors.New("project collection payload chain_id is empty")
	}
	if payload.Project.Contract == (common.Address{}) {
		return workflows.ProjectCollectionInput{}, errors.New("project collection payload contract is empty")
	}
	return workflows.ProjectCollectionInput{Project: payload.Project, Reason: strings.TrimSpace(payload.Reason)}, nil
}

func sleep(ctx context.Context, delay time.Duration) bool {
	if delay <= 0 {
		select {
		case <-ctx.Done():
			return false
		default:
			return true
		}
	}
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}

func errorMessage(err error) string {
	if err == nil {
		return ""
	}
	return strings.TrimSpace(err.Error())
}
