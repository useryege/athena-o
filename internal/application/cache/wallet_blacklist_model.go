package cache

import (
	"context"
	"errors"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/useryege/athena/internal/application/store"
)

var _ WalletBlacklistModel = &walletBlacklistModel{}

type walletBlacklistModel struct {
	store          store.WalletBlacklistContractStore
	cache          WalletBlacklistCache
	writePublisher WalletBlacklistWritePublisher
}

func NewWalletBlacklistModel(
	store store.WalletBlacklistContractStore,
	cache WalletBlacklistCache,
	writePublisher WalletBlacklistWritePublisher,
) WalletBlacklistModel {
	if cache == nil {
		cache = NewLayeredWalletBlacklistCache(NewLocalWalletBlacklistCache(), nil)
	}
	return &walletBlacklistModel{
		store:          store,
		cache:          cache,
		writePublisher: writePublisher,
	}
}

func (m *walletBlacklistModel) Load(ctx context.Context) error {
	return m.refresh(ctx)
}

func (m *walletBlacklistModel) List(ctx context.Context) ([]store.WalletBlacklistContract, error) {
	if m == nil || m.cache == nil {
		return nil, nil
	}
	return m.cache.Take(ctx, func(ctx context.Context) ([]store.WalletBlacklistContract, error) {
		if m.store == nil {
			return nil, nil
		}
		return m.store.ListWalletBlacklistContracts(ctx)
	})
}

func (m *walletBlacklistModel) Add(ctx context.Context, item store.WalletBlacklistContract) error {
	if m == nil {
		return nil
	}
	if item.Contract == (common.Address{}) {
		return nil
	}
	item.Note = normalizeNote(item.Note)
	if m.writePublisher == nil {
		return errors.New("wallet blacklist write publisher is not configured")
	}

	items, err := m.List(ctx)
	if err != nil {
		return err
	}
	for _, current := range items {
		if current.Contract == item.Contract {
			return store.ErrWalletBlacklistContractAlreadyExists
		}
	}
	if err := m.writePublisher.PublishAdd(ctx, item); err != nil {
		return err
	}
	if item.CreatedAt.IsZero() {
		item.CreatedAt = time.Now().UTC()
	}
	next := append([]store.WalletBlacklistContract{item}, items...)
	return m.cache.Set(ctx, next)
}

func (m *walletBlacklistModel) UpdateNote(ctx context.Context, contract common.Address, note string) error {
	if m == nil || contract == (common.Address{}) {
		return nil
	}
	if m.writePublisher == nil {
		return errors.New("wallet blacklist write publisher is not configured")
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
		return store.ErrWalletBlacklistContractNotFound
	}
	note = normalizeNote(note)
	if err := m.writePublisher.PublishUpdateNote(ctx, contract, note); err != nil {
		return err
	}
	items[target].Note = note
	return m.cache.Set(ctx, items)
}

func (m *walletBlacklistModel) Delete(ctx context.Context, contract common.Address) error {
	if m == nil || contract == (common.Address{}) {
		return nil
	}
	if m.writePublisher == nil {
		return errors.New("wallet blacklist write publisher is not configured")
	}

	items, err := m.List(ctx)
	if err != nil {
		return err
	}
	found := false
	filtered := make([]store.WalletBlacklistContract, 0, len(items))
	for _, item := range items {
		if item.Contract == contract {
			found = true
			continue
		}
		filtered = append(filtered, item)
	}
	if !found {
		return store.ErrWalletBlacklistContractNotFound
	}
	if err := m.writePublisher.PublishDelete(ctx, contract); err != nil {
		return err
	}
	return m.cache.Set(ctx, filtered)
}

func (m *walletBlacklistModel) refresh(ctx context.Context) error {
	if m.cache == nil {
		return nil
	}
	if m.store == nil {
		return m.cache.Del(ctx)
	}
	items, err := m.store.ListWalletBlacklistContracts(ctx)
	if err != nil {
		return err
	}
	return m.cache.Set(ctx, items)
}
