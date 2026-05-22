package cache

import (
	"context"
	"errors"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/useryege/athena/internal/application/store"
)

var _ SourcecodeBlacklistContractModel = &sourcecodeBlacklistContractModel{}

type sourcecodeBlacklistContractModel struct {
	store          store.SourcecodeBlacklistContractStore
	cache          SourcecodeBlacklistContractCache
	writePublisher SourcecodeBlacklistContractWritePublisher
}

func NewSourcecodeBlacklistContractModel(
	store store.SourcecodeBlacklistContractStore,
	cache SourcecodeBlacklistContractCache,
	writePublisher SourcecodeBlacklistContractWritePublisher,
) SourcecodeBlacklistContractModel {
	if cache == nil {
		cache = NewLayeredSourcecodeBlacklistContractCache(NewLocalSourcecodeBlacklistContractCache(), nil)
	}
	return &sourcecodeBlacklistContractModel{
		store:          store,
		cache:          cache,
		writePublisher: writePublisher,
	}
}

func (m *sourcecodeBlacklistContractModel) Load(ctx context.Context) error {
	return m.refresh(ctx)
}

func (m *sourcecodeBlacklistContractModel) List(ctx context.Context) ([]store.SourcecodeBlacklistContract, error) {
	if m == nil || m.cache == nil {
		return nil, nil
	}
	return m.cache.Take(ctx, func(ctx context.Context) ([]store.SourcecodeBlacklistContract, error) {
		if m.store == nil {
			return nil, nil
		}
		return m.store.ListSourcecodeBlacklistContracts(ctx)
	})
}

func (m *sourcecodeBlacklistContractModel) Version(ctx context.Context) (string, error) {
	if m == nil || m.cache == nil {
		return "", nil
	}
	return m.cache.Version(ctx)
}

func (m *sourcecodeBlacklistContractModel) Add(ctx context.Context, item store.SourcecodeBlacklistContract) error {
	if m == nil {
		return nil
	}
	if item.Contract == (common.Address{}) || item.SourceHash == (common.Hash{}) {
		return nil
	}
	item.Note = normalizeNote(item.Note)
	if m.writePublisher == nil {
		return errors.New("sourcecode blacklist contract write publisher is not configured")
	}

	items, err := m.List(ctx)
	if err != nil {
		return err
	}
	for _, current := range items {
		if current.Contract == item.Contract || current.SourceHash == item.SourceHash {
			return store.ErrSourcecodeBlacklistContractAlreadyExists
		}
	}
	if err := m.writePublisher.PublishAdd(ctx, item); err != nil {
		return err
	}
	if item.CreatedAt.IsZero() {
		item.CreatedAt = time.Now().UTC()
	}
	next := append([]store.SourcecodeBlacklistContract{item}, items...)
	return m.cache.Set(ctx, next)
}

func (m *sourcecodeBlacklistContractModel) UpdateNote(ctx context.Context, contract common.Address, note string) error {
	if m == nil || contract == (common.Address{}) {
		return nil
	}
	if m.writePublisher == nil {
		return errors.New("sourcecode blacklist contract write publisher is not configured")
	}

	items, err := m.List(ctx)
	if err != nil {
		return err
	}
	target := -1
	for i := range items {
		if items[i].Contract == contract {
			target = i
			break
		}
	}
	if target < 0 {
		return store.ErrSourcecodeBlacklistContractNotFound
	}
	note = normalizeNote(note)
	if err := m.writePublisher.PublishUpdateNote(ctx, contract, note); err != nil {
		return err
	}
	items[target].Note = note
	return m.cache.Set(ctx, items)
}

func (m *sourcecodeBlacklistContractModel) Delete(ctx context.Context, contract common.Address) error {
	if m == nil || contract == (common.Address{}) {
		return nil
	}
	if m.writePublisher == nil {
		return errors.New("sourcecode blacklist contract write publisher is not configured")
	}

	items, err := m.List(ctx)
	if err != nil {
		return err
	}
	found := false
	filtered := make([]store.SourcecodeBlacklistContract, 0, len(items))
	for _, item := range items {
		if item.Contract == contract {
			found = true
			continue
		}
		filtered = append(filtered, item)
	}
	if !found {
		return store.ErrSourcecodeBlacklistContractNotFound
	}
	if err := m.writePublisher.PublishDelete(ctx, contract); err != nil {
		return err
	}
	return m.cache.Set(ctx, filtered)
}

func (m *sourcecodeBlacklistContractModel) refresh(ctx context.Context) error {
	if m.cache == nil {
		return nil
	}
	if m.store == nil {
		return m.cache.Del(ctx)
	}
	items, err := m.store.ListSourcecodeBlacklistContracts(ctx)
	if err != nil {
		return err
	}
	return m.cache.Set(ctx, items)
}
