package events

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/twmb/franz-go/pkg/kgo"
	"github.com/useryege/athena/internal/application/model"
	appstore "github.com/useryege/athena/internal/application/store"
)

const (
	ReasonDexSwap = "dex_swap"
)

type ProjectEventStore interface {
	appstore.ProjectIntakeStore
	ListProjectMetasByPairAddresses(ctx context.Context, chainID int64, pairs []common.Address) ([]appstore.ProjectMeta, error)
}

type Processor struct {
	store ProjectEventStore
}

func NewProcessor(store ProjectEventStore) *Processor {
	return &Processor{store: store}
}

func (p *Processor) ProcessEnvelope(ctx context.Context, envelope Envelope) error {
	if p == nil || p.store == nil {
		return nil
	}
	if envelope.SchemaVersion != 0 && envelope.SchemaVersion != SchemaVersionV1 {
		return fmt.Errorf("unsupported application event schema version %d", envelope.SchemaVersion)
	}
	if envelope.ChainID <= 0 {
		return errors.New("application event chain_id must be positive")
	}
	switch envelope.EventType {
	case EventTypeContractCreated:
		return p.processContractCreated(ctx, envelope)
	case EventTypeDexSwap:
		return p.processDexSwap(ctx, envelope)
	default:
		return fmt.Errorf("unsupported application event type %q", envelope.EventType)
	}
}

func (p *Processor) processContractCreated(ctx context.Context, envelope Envelope) error {
	var payload ContractCreatedPayload
	if err := json.Unmarshal(envelope.Payload, &payload); err != nil {
		return fmt.Errorf("decode contract created payload: %w", err)
	}
	if !common.IsHexAddress(payload.Contract) {
		return fmt.Errorf("contract created payload contract is invalid: %q", payload.Contract)
	}
	if payload.BlockNumber < 0 || payload.BlockTime < 0 || payload.TxIndex < 0 {
		return errors.New("contract created payload block fields must be non-negative")
	}
	candidate := model.DiscoveredProjectCandidate{
		ChainID:     envelope.ChainID,
		Contract:    common.HexToAddress(payload.Contract),
		BlockNumber: uint64(payload.BlockNumber),
		BlockTime:   uint64(payload.BlockTime),
		TxIndex:     uint64(payload.TxIndex),
		Source:      model.ProjectDiscoverySourceFollowHeads,
	}
	if common.IsHexAddress(payload.Creator) {
		candidate.Creator = common.HexToAddress(payload.Creator)
	}
	if strings.TrimSpace(payload.TxHash) != "" {
		candidate.TxHash = common.HexToHash(payload.TxHash)
	}
	if common.IsHexAddress(payload.WethPair) {
		candidate.WethPair = common.HexToAddress(payload.WethPair)
	}
	if common.IsHexAddress(payload.UsdtPair) {
		candidate.UsdtPair = common.HexToAddress(payload.UsdtPair)
	}
	return p.store.UpsertProjectCandidateAndEnqueueQualification(ctx, candidate)
}

func (p *Processor) processDexSwap(ctx context.Context, envelope Envelope) error {
	var payload DexSwapPayload
	if err := json.Unmarshal(envelope.Payload, &payload); err != nil {
		return fmt.Errorf("decode dex swap payload: %w", err)
	}
	if !common.IsHexAddress(payload.Pair) {
		return fmt.Errorf("dex swap payload pair is invalid: %q", payload.Pair)
	}
	pair := common.HexToAddress(payload.Pair)
	metas, err := p.store.ListProjectMetasByPairAddresses(ctx, envelope.ChainID, []common.Address{pair})
	if err != nil {
		return err
	}
	for _, meta := range metas {
		if meta.Contract == (common.Address{}) {
			continue
		}
		ref := model.ProjectRef{ChainID: envelope.ChainID, Contract: meta.Contract}
		if err := p.store.EnqueueProjectCollection(ctx, ref, ReasonDexSwap); err != nil {
			return err
		}
	}
	return nil
}

type KafkaConsumer struct {
	client    *kgo.Client
	processor *Processor

	mu      sync.Mutex
	cancel  context.CancelFunc
	wg      sync.WaitGroup
	started bool
}

func NewKafkaConsumer(brokers []string, group string, store ProjectEventStore) (*KafkaConsumer, error) {
	if len(brokers) == 0 {
		return nil, errors.New("application kafka brokers are required")
	}
	group = strings.TrimSpace(group)
	if group == "" {
		return nil, errors.New("application kafka consumer group is required")
	}
	client, err := kgo.NewClient(
		kgo.SeedBrokers(brokers...),
		kgo.ConsumerGroup(group),
		kgo.ConsumeTopics(TopicContractCreatedV1, TopicDexSwapV1),
	)
	if err != nil {
		return nil, fmt.Errorf("create application kafka consumer: %w", err)
	}
	return &KafkaConsumer{client: client, processor: NewProcessor(store)}, nil
}

func (c *KafkaConsumer) Start(ctx context.Context) error {
	if c == nil || c.client == nil {
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
	go c.run(runCtx)
	return nil
}

func (c *KafkaConsumer) Stop() {
	if c == nil {
		return
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
	if c.client != nil {
		c.client.Close()
	}
}

func (c *KafkaConsumer) run(ctx context.Context) {
	defer c.wg.Done()
	for {
		fetches := c.client.PollFetches(ctx)
		if err := fetches.Err(); err != nil {
			if ctx.Err() != nil {
				return
			}
			time.Sleep(time.Second)
			continue
		}
		fetches.EachRecord(func(record *kgo.Record) {
			var envelope Envelope
			if err := json.Unmarshal(record.Value, &envelope); err != nil {
				return
			}
			_ = c.processor.ProcessEnvelope(ctx, envelope)
		})
	}
}
