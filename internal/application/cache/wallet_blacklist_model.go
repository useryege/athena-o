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
	store          store.WalletBlacklistStore
	cache          WalletBlacklistCache
	writePublisher WalletBlacklistWritePublisher
}

func NewWalletBlacklistModel(
	store store.WalletBlacklistStore,
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

func (m *walletBlacklistModel) List(ctx context.Context) ([]store.WalletBlacklistEntry, error) {
	if m == nil || m.cache == nil {
		return nil, nil
	}
	return m.cache.Take(ctx, func(ctx context.Context) ([]store.WalletBlacklistEntry, error) {
		if m.store == nil {
			return nil, nil
		}
		return m.store.ListWalletBlacklistEntries(ctx)
	})
}

func (m *walletBlacklistModel) Add(ctx context.Context, item store.WalletBlacklistEntry) error {
	if m == nil {
		return nil
	}
	if item.Wallet == (common.Address{}) {
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
		if current.Wallet == item.Wallet {
			return store.ErrWalletBlacklistEntryAlreadyExists
		}
	}
	if err := m.writePublisher.PublishAdd(ctx, item); err != nil {
		return err
	}
	if item.CreatedAt.IsZero() {
		item.CreatedAt = time.Now().UTC()
	}
	next := append([]store.WalletBlacklistEntry{item}, items...)
	return m.cache.Set(ctx, next)
}

func (m *walletBlacklistModel) UpdateNote(ctx context.Context, wallet common.Address, note string) error {
	if m == nil || wallet == (common.Address{}) {
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
		if items[i].Wallet == wallet {
			target = i
			break
		}
	}
	if target < 0 {
		return store.ErrWalletBlacklistEntryNotFound
	}
	note = normalizeNote(note)
	if err := m.writePublisher.PublishUpdateNote(ctx, wallet, note); err != nil {
		return err
	}
	items[target].Note = note
	return m.cache.Set(ctx, items)
}

func (m *walletBlacklistModel) Delete(ctx context.Context, wallet common.Address) error {
	if m == nil || wallet == (common.Address{}) {
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
	filtered := make([]store.WalletBlacklistEntry, 0, len(items))
	for _, item := range items {
		if item.Wallet == wallet {
			found = true
			continue
		}
		filtered = append(filtered, item)
	}
	if !found {
		return store.ErrWalletBlacklistEntryNotFound
	}
	if err := m.writePublisher.PublishDelete(ctx, wallet); err != nil {
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
	items, err := m.store.ListWalletBlacklistEntries(ctx)
	if err != nil {
		return err
	}
	return m.cache.Set(ctx, items)
}
