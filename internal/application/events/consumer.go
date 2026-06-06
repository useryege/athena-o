package events

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/useryege/athena/internal/application/model"
	appstore "github.com/useryege/athena/internal/application/store"
)

const (
	// ReasonDexSwap is an alias for model.ProjectCollectionReasonDexSwap for use within this package.
	ReasonDexSwap = model.ProjectCollectionReasonDexSwap
)

type ProjectEventStore interface {
	appstore.ProjectIntakeStore
	ListProjectMetasByPairAddresses(ctx context.Context, chainID int64, pairs []common.Address) ([]model.Project, error)
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
	codeHash, err := parseContractCreatedCodeHash(payload.CodeHash)
	if err != nil {
		return err
	}
	candidate := model.DiscoveredProjectCandidate{
		ChainID:     envelope.ChainID,
		Contract:    common.HexToAddress(payload.Contract),
		BlockNumber: uint64(payload.BlockNumber),
		BlockTime:   uint64(payload.BlockTime),
		TxIndex:     uint64(payload.TxIndex),
		CodeHash:    codeHash,
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

func parseContractCreatedCodeHash(value string) (common.Hash, error) {
	trimmed := strings.TrimSpace(value)
	raw := common.FromHex(trimmed)
	if trimmed == "" || len(raw) != common.HashLength {
		return common.Hash{}, fmt.Errorf("contract created payload code_hash is invalid: %q", value)
	}
	return common.BytesToHash(raw), nil
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
	if pair == (common.Address{}) {
		return errors.New("dex swap payload pair cannot be zero address")
	}
	if !common.IsHexAddress(payload.Token0) {
		return fmt.Errorf("dex swap payload token0 is invalid: %q", payload.Token0)
	}
	token0 := common.HexToAddress(payload.Token0)
	if token0 == (common.Address{}) {
		return errors.New("dex swap payload token0 cannot be zero address")
	}
	if !common.IsHexAddress(payload.Token1) {
		return fmt.Errorf("dex swap payload token1 is invalid: %q", payload.Token1)
	}
	token1 := common.HexToAddress(payload.Token1)
	if token1 == (common.Address{}) {
		return errors.New("dex swap payload token1 cannot be zero address")
	}
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
