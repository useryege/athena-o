package components

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
	"github.com/useryege/athena/internal/application/model"
	"github.com/useryege/athena/internal/application/redisport"
	"github.com/useryege/athena/internal/application/redisrepo"
)

const (
	eventReadCount    = int64(64)
	eventReadBlock    = 2 * time.Second
	eventPendingPause = 200 * time.Millisecond
)

var componentEventKeyspace = redisrepo.NewKeyspace("application")

type EventType string

const (
	EventProjectInitialized EventType = "project.initialized"
	EventProjectRefresh     EventType = "project.refresh_requested"
	EventComponentCompleted EventType = "component.completed"
	EventComponentFailed    EventType = "component.failed"
)

type ProjectCandidate struct {
	BlockTime   uint64                       `json:"block_time"`
	BlockNumber uint64                       `json:"block_number"`
	TxIndex     uint64                       `json:"tx_index"`
	Contract    string                       `json:"contract"`
	Creator     string                       `json:"creator"`
	TxHash      string                       `json:"tx_hash"`
	Source      model.ProjectDiscoverySource `json:"source"`
}

type Event struct {
	Type       EventType                    `json:"type"`
	Contract   string                       `json:"contract,omitempty"`
	Component  string                       `json:"component,omitempty"`
	Source     model.ProjectDiscoverySource `json:"source,omitempty"`
	Candidate  ProjectCandidate             `json:"candidate,omitempty"`
	Error      string                       `json:"error,omitempty"`
	OccurredAt time.Time                    `json:"occurred_at"`
}

type Handler func(context.Context, Event) error

type EventBus interface {
	Publish(ctx context.Context, event Event) error
	Subscribe(ctx context.Context, group string, consumer string, handler Handler) error
}

type RedisEventBus struct {
	client redisport.StreamClient
	stream string
}

func NewRedisEventBus(client redisport.StreamClient) *RedisEventBus {
	if client == nil {
		return nil
	}
	return &RedisEventBus{client: client, stream: componentEventKeyspace.ProjectComponentStream()}
}

func NewProjectCandidate(item model.DiscoveredProjectCandidate) ProjectCandidate {
	return ProjectCandidate{
		BlockTime:   item.BlockTime,
		BlockNumber: item.BlockNumber,
		TxIndex:     item.TxIndex,
		Contract:    item.Contract.Hex(),
		Creator:     item.Creator.Hex(),
		TxHash:      item.TxHash.Hex(),
		Source:      item.Source,
	}
}

func (c ProjectCandidate) ToModel() model.DiscoveredProjectCandidate {
	item := model.DiscoveredProjectCandidate{
		BlockTime:   c.BlockTime,
		BlockNumber: c.BlockNumber,
		TxIndex:     c.TxIndex,
		Source:      c.Source,
	}
	if common.IsHexAddress(c.Contract) {
		item.Contract = common.HexToAddress(c.Contract)
	}
	if common.IsHexAddress(c.Creator) {
		item.Creator = common.HexToAddress(c.Creator)
	}
	if strings.TrimSpace(c.TxHash) != "" {
		item.TxHash = common.HexToHash(c.TxHash)
	}
	return item
}

func (e Event) ProjectContract() common.Address {
	if common.IsHexAddress(e.Contract) {
		return common.HexToAddress(e.Contract)
	}
	if common.IsHexAddress(e.Candidate.Contract) {
		return common.HexToAddress(e.Candidate.Contract)
	}
	return common.Address{}
}

func ProjectInitializedEvent(item model.DiscoveredProjectCandidate) Event {
	candidate := NewProjectCandidate(item)
	return Event{
		Type:      EventProjectInitialized,
		Contract:  candidate.Contract,
		Source:    item.Source,
		Candidate: candidate,
	}
}

func ProjectRefreshEvent(contract common.Address, source model.ProjectDiscoverySource) Event {
	return Event{
		Type:     EventProjectRefresh,
		Contract: contract.Hex(),
		Source:   source,
	}
}

func ComponentCompletedEvent(contract common.Address, component string) Event {
	return Event{
		Type:      EventComponentCompleted,
		Contract:  contract.Hex(),
		Component: component,
	}
}

func ComponentFailedEvent(contract common.Address, component string, err error) Event {
	event := Event{
		Type:      EventComponentFailed,
		Contract:  contract.Hex(),
		Component: component,
	}
	if err != nil {
		event.Error = err.Error()
	}
	return event
}

func (b *RedisEventBus) Publish(ctx context.Context, event Event) error {
	if b == nil || b.client == nil {
		return nil
	}
	if event.Type == "" {
		return errors.New("component event type is required")
	}
	if event.OccurredAt.IsZero() {
		event.OccurredAt = time.Now().UTC()
	}
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal component event: %w", err)
	}
	return b.client.XAdd(ctx, redisport.XAddInput{
		Stream: b.stream,
		Values: map[string]any{"event": string(data)},
	})
}

func (b *RedisEventBus) Subscribe(ctx context.Context, group string, consumer string, handler Handler) error {
	if b == nil || b.client == nil {
		return nil
	}
	if strings.TrimSpace(group) == "" || strings.TrimSpace(consumer) == "" {
		return errors.New("component event consumer group and consumer are required")
	}
	if handler == nil {
		return errors.New("component event handler is required")
	}
	if err := b.ensureConsumerGroup(ctx, group); err != nil {
		return err
	}
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		processed, err := b.consume(ctx, group, consumer, "0", 0, handler)
		if err != nil {
			return err
		}
		if processed == 0 {
			break
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(eventPendingPause):
		}
	}
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		if _, err := b.consume(ctx, group, consumer, ">", eventReadBlock, handler); err != nil {
			return err
		}
	}
}

func (b *RedisEventBus) consume(ctx context.Context, group string, consumer string, id string, block time.Duration, handler Handler) (int, error) {
	streams, err := b.client.XReadGroup(ctx, redisport.XReadGroupInput{
		Group:    group,
		Consumer: consumer,
		Streams:  []string{b.stream, id},
		Count:    eventReadCount,
		Block:    block,
	})
	if err != nil {
		if errors.Is(err, redisport.ErrNotFound) {
			return 0, nil
		}
		if strings.Contains(err.Error(), "NOGROUP") {
			if ensureErr := b.ensureConsumerGroup(ctx, group); ensureErr != nil {
				return 0, ensureErr
			}
			return 0, nil
		}
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return 0, err
		}
		return 0, fmt.Errorf("xreadgroup component events: %w", err)
	}
	processed := 0
	for _, stream := range streams {
		for _, message := range stream.Messages {
			event, err := eventFromMessage(message)
			if err != nil {
				return processed, err
			}
			if err := handler(ctx, event); err != nil {
				return processed, err
			}
			if err := b.client.XAck(ctx, b.stream, group, message.ID); err != nil {
				return processed, fmt.Errorf("ack component event %s: %w", message.ID, err)
			}
			processed++
		}
	}
	return processed, nil
}

func eventFromMessage(message redisport.XMessage) (Event, error) {
	value, ok := message.Values["event"]
	if !ok {
		return Event{}, fmt.Errorf("component event %s missing payload", message.ID)
	}
	raw, ok := value.(string)
	if !ok {
		return Event{}, fmt.Errorf("component event %s payload type %T is not string", message.ID, value)
	}
	var event Event
	if err := json.Unmarshal([]byte(raw), &event); err != nil {
		return Event{}, fmt.Errorf("unmarshal component event %s: %w", message.ID, err)
	}
	return event, nil
}

func (b *RedisEventBus) ensureConsumerGroup(ctx context.Context, group string) error {
	err := b.client.XGroupCreateMkStream(ctx, b.stream, group, "0")
	if err == nil || strings.Contains(err.Error(), "BUSYGROUP") {
		return nil
	}
	return fmt.Errorf("create component event consumer group %q: %w", group, err)
}

type Consumer struct {
	name    string
	group   string
	bus     EventBus
	handler Handler

	mu      sync.Mutex
	cancel  context.CancelFunc
	started bool
	wg      sync.WaitGroup
}

func NewConsumer(name string, bus EventBus, handler Handler) *Consumer {
	if bus == nil {
		return nil
	}
	return &Consumer{
		name:    name,
		group:   "application:component:" + name,
		bus:     bus,
		handler: handler,
	}
}

func (c *Consumer) Start(ctx context.Context) error {
	if c == nil || c.bus == nil || c.handler == nil {
		return nil
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.started {
		return nil
	}
	runCtx, cancel := context.WithCancel(ctx)
	c.cancel = cancel
	c.started = true
	c.wg.Add(1)
	go func() {
		defer c.wg.Done()
		for {
			err := c.bus.Subscribe(runCtx, c.group, c.name, c.handler)
			if err == nil || errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				return
			}
			log.WithFields(log.Fields{"component": c.name, "error": err.Error()}).Warn("component event consumer stopped; retrying")
			select {
			case <-runCtx.Done():
				return
			case <-time.After(time.Second):
			}
		}
	}()
	return nil
}

func (c *Consumer) Stop() error {
	if c == nil {
		return nil
	}
	c.mu.Lock()
	cancel := c.cancel
	c.cancel = nil
	c.started = false
	c.mu.Unlock()
	if cancel != nil {
		cancel()
	}
	c.wg.Wait()
	return nil
}
