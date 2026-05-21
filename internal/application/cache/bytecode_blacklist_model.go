package cache

import (
	"context"
	"errors"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/useryege/athena/internal/application/store"
)

var _ BytecodeBlacklistModel = &bytecodeBlacklistModel{}

type bytecodeBlacklistModel struct {
	store          store.BytecodeBlacklistContractStore
	cache          BytecodeBlacklistCache
	writePublisher BytecodeBlacklistWritePublisher
}

func NewBytecodeBlacklistModel(
	store store.BytecodeBlacklistContractStore,
	cache BytecodeBlacklistCache,
	writePublisher BytecodeBlacklistWritePublisher,
) BytecodeBlacklistModel {
	if cache == nil {
		cache = NewLayeredBytecodeBlacklistCache(NewLocalBytecodeBlacklistCache(), nil)
	}
	return &bytecodeBlacklistModel{
		store:          store,
		cache:          cache,
		writePublisher: writePublisher,
	}
}

func (m *bytecodeBlacklistModel) Load(ctx context.Context) error {
	return m.refresh(ctx)
}

func (m *bytecodeBlacklistModel) List(ctx context.Context) ([]store.BytecodeBlacklistContract, error) {
	if m == nil || m.cache == nil {
		return nil, nil
	}
	return m.cache.Take(ctx, func(ctx context.Context) ([]store.BytecodeBlacklistContract, error) {
		if m.store == nil {
			return nil, nil
		}
		return m.store.ListBytecodeBlacklistContracts(ctx)
	})
}

func (m *bytecodeBlacklistModel) Add(ctx context.Context, item store.BytecodeBlacklistContract) error {
	if m == nil {
		return nil
	}
	if item.Contract == (common.Address{}) {
		return nil
	}
	item.Note = normalizeNote(item.Note)
	if m.writePublisher == nil {
		return errors.New("bytecode blacklist write publisher is not configured")
	}

	items, err := m.List(ctx)
	if err != nil {
		return err
	}
	for _, current := range items {
		if current.Contract == item.Contract {
			return store.ErrBytecodeBlacklistContractAlreadyExists
		}
	}
	if err := m.writePublisher.PublishAdd(ctx, item); err != nil {
		return err
	}
	if item.CreatedAt.IsZero() {
		item.CreatedAt = time.Now().UTC()
	}
	next := append([]store.BytecodeBlacklistContract{item}, items...)
	return m.cache.Set(ctx, next)
}

func (m *bytecodeBlacklistModel) UpdateNote(ctx context.Context, contract common.Address, note string) error {
	if m == nil || contract == (common.Address{}) {
		return nil
	}
	if m.writePublisher == nil {
		return errors.New("bytecode blacklist write publisher is not configured")
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
		return store.ErrBytecodeBlacklistContractNotFound
	}
	note = normalizeNote(note)
	if err := m.writePublisher.PublishUpdateNote(ctx, contract, note); err != nil {
		return err
	}
	items[target].Note = note
	return m.cache.Set(ctx, items)
}

func (m *bytecodeBlacklistModel) Delete(ctx context.Context, contract common.Address) error {
	if m == nil || contract == (common.Address{}) {
		return nil
	}
	if m.writePublisher == nil {
		return errors.New("bytecode blacklist write publisher is not configured")
	}

	items, err := m.List(ctx)
	if err != nil {
		return err
	}
	found := false
	filtered := make([]store.BytecodeBlacklistContract, 0, len(items))
	for _, item := range items {
		if item.Contract == contract {
			found = true
			continue
		}
		filtered = append(filtered, item)
	}
	if !found {
		return store.ErrBytecodeBlacklistContractNotFound
	}
	if err := m.writePublisher.PublishDelete(ctx, contract); err != nil {
		return err
	}
	return m.cache.Set(ctx, filtered)
}

func (m *bytecodeBlacklistModel) refresh(ctx context.Context) error {
	if m.cache == nil {
		return nil
	}
	if m.store == nil {
		return m.cache.Del(ctx)
	}
	items, err := m.store.ListBytecodeBlacklistContracts(ctx)
	if err != nil {
		return err
	}
	return m.cache.Set(ctx, items)
}
