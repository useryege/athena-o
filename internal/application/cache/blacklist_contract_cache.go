package cache

import (
	"context"
	"sort"
	"strings"
	"sync"

	"github.com/ethereum/go-ethereum/common"
	"github.com/useryege/athena/internal/application/store"
)

var _ BytecodeBlacklistCache = &LayeredBytecodeBlacklistCache{}
var _ WalletBlacklistCache = &LayeredWalletBlacklistCache{}

type LocalBytecodeBlacklistCache struct {
	mu    sync.RWMutex
	items []store.BytecodeBlacklistContract
	ready bool
}

func NewLocalBytecodeBlacklistCache(items ...store.BytecodeBlacklistContract) *LocalBytecodeBlacklistCache {
	c := &LocalBytecodeBlacklistCache{}
	if len(items) > 0 {
		c.Set(items)
	}
	return c
}

func (c *LocalBytecodeBlacklistCache) Get() ([]store.BytecodeBlacklistContract, bool) {
	if c == nil {
		return nil, false
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	if !c.ready {
		return nil, false
	}
	return append([]store.BytecodeBlacklistContract(nil), c.items...), true
}

func (c *LocalBytecodeBlacklistCache) Set(items []store.BytecodeBlacklistContract) {
	if c == nil {
		return
	}
	normalized := normalizeBytecodeBlacklistContracts(items)
	c.mu.Lock()
	defer c.mu.Unlock()
	c.items = normalized
	c.ready = true
}

func (c *LocalBytecodeBlacklistCache) Del() {
	if c == nil {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.items = nil
	c.ready = false
}

type LocalWalletBlacklistCache struct {
	mu    sync.RWMutex
	items []store.WalletBlacklistContract
	ready bool
}

func NewLocalWalletBlacklistCache(items ...store.WalletBlacklistContract) *LocalWalletBlacklistCache {
	c := &LocalWalletBlacklistCache{}
	if len(items) > 0 {
		c.Set(items)
	}
	return c
}

func (c *LocalWalletBlacklistCache) Get() ([]store.WalletBlacklistContract, bool) {
	if c == nil {
		return nil, false
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	if !c.ready {
		return nil, false
	}
	return append([]store.WalletBlacklistContract(nil), c.items...), true
}

func (c *LocalWalletBlacklistCache) Set(items []store.WalletBlacklistContract) {
	if c == nil {
		return
	}
	normalized := normalizeWalletBlacklistContracts(items)
	c.mu.Lock()
	defer c.mu.Unlock()
	c.items = normalized
	c.ready = true
}

func (c *LocalWalletBlacklistCache) Del() {
	if c == nil {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.items = nil
	c.ready = false
}

type LayeredBytecodeBlacklistCache struct {
	local  *LocalBytecodeBlacklistCache
	remote BytecodeBlacklistRemoteCache
}

func NewLayeredBytecodeBlacklistCache(local *LocalBytecodeBlacklistCache, remote BytecodeBlacklistRemoteCache) *LayeredBytecodeBlacklistCache {
	if local == nil {
		local = NewLocalBytecodeBlacklistCache()
	}
	return &LayeredBytecodeBlacklistCache{
		local:  local,
		remote: remote,
	}
}

func (c *LayeredBytecodeBlacklistCache) Take(ctx context.Context, loader func(context.Context) ([]store.BytecodeBlacklistContract, error)) ([]store.BytecodeBlacklistContract, error) {
	if c == nil {
		return loadBytecodeBlacklistContracts(ctx, loader)
	}
	if items, ok := c.local.Get(); ok {
		return items, nil
	}
	if c.remote != nil {
		items, ok, err := c.remote.Get(ctx)
		if err != nil {
			return nil, err
		}
		if ok {
			c.local.Set(items)
			return items, nil
		}
	}

	items, err := loadBytecodeBlacklistContracts(ctx, loader)
	if err != nil {
		return nil, err
	}
	if c.remote != nil {
		if err := c.remote.Set(ctx, items); err != nil {
			return nil, err
		}
	}
	c.local.Set(items)
	return items, nil
}

func (c *LayeredBytecodeBlacklistCache) Set(ctx context.Context, items []store.BytecodeBlacklistContract) error {
	if c == nil {
		return nil
	}
	if c.remote != nil {
		if err := c.remote.Set(ctx, items); err != nil {
			return err
		}
	}
	c.local.Set(items)
	return nil
}

func (c *LayeredBytecodeBlacklistCache) Del(ctx context.Context) error {
	if c == nil {
		return nil
	}
	if c.remote != nil {
		if err := c.remote.Del(ctx); err != nil {
			return err
		}
	}
	c.local.Del()
	return nil
}

type LayeredWalletBlacklistCache struct {
	local  *LocalWalletBlacklistCache
	remote WalletBlacklistRemoteCache
}

func NewLayeredWalletBlacklistCache(local *LocalWalletBlacklistCache, remote WalletBlacklistRemoteCache) *LayeredWalletBlacklistCache {
	if local == nil {
		local = NewLocalWalletBlacklistCache()
	}
	return &LayeredWalletBlacklistCache{
		local:  local,
		remote: remote,
	}
}

func (c *LayeredWalletBlacklistCache) Take(ctx context.Context, loader func(context.Context) ([]store.WalletBlacklistContract, error)) ([]store.WalletBlacklistContract, error) {
	if c == nil {
		return loadWalletBlacklistContracts(ctx, loader)
	}
	if items, ok := c.local.Get(); ok {
		return items, nil
	}
	if c.remote != nil {
		items, ok, err := c.remote.Get(ctx)
		if err != nil {
			return nil, err
		}
		if ok {
			c.local.Set(items)
			return items, nil
		}
	}

	items, err := loadWalletBlacklistContracts(ctx, loader)
	if err != nil {
		return nil, err
	}
	if c.remote != nil {
		if err := c.remote.Set(ctx, items); err != nil {
			return nil, err
		}
	}
	c.local.Set(items)
	return items, nil
}

func (c *LayeredWalletBlacklistCache) Set(ctx context.Context, items []store.WalletBlacklistContract) error {
	if c == nil {
		return nil
	}
	if c.remote != nil {
		if err := c.remote.Set(ctx, items); err != nil {
			return err
		}
	}
	c.local.Set(items)
	return nil
}

func (c *LayeredWalletBlacklistCache) Del(ctx context.Context) error {
	if c == nil {
		return nil
	}
	if c.remote != nil {
		if err := c.remote.Del(ctx); err != nil {
			return err
		}
	}
	c.local.Del()
	return nil
}

func loadBytecodeBlacklistContracts(ctx context.Context, loader func(context.Context) ([]store.BytecodeBlacklistContract, error)) ([]store.BytecodeBlacklistContract, error) {
	if loader == nil {
		return nil, nil
	}
	items, err := loader(ctx)
	if err != nil {
		return nil, err
	}
	return normalizeBytecodeBlacklistContracts(items), nil
}

func loadWalletBlacklistContracts(ctx context.Context, loader func(context.Context) ([]store.WalletBlacklistContract, error)) ([]store.WalletBlacklistContract, error) {
	if loader == nil {
		return nil, nil
	}
	items, err := loader(ctx)
	if err != nil {
		return nil, err
	}
	return normalizeWalletBlacklistContracts(items), nil
}

func normalizeBytecodeBlacklistContracts(items []store.BytecodeBlacklistContract) []store.BytecodeBlacklistContract {
	if len(items) == 0 {
		return nil
	}
	seen := make(map[common.Address]struct{}, len(items))
	normalized := make([]store.BytecodeBlacklistContract, 0, len(items))
	for _, item := range items {
		if item.Contract == (common.Address{}) {
			continue
		}
		if _, ok := seen[item.Contract]; ok {
			continue
		}
		seen[item.Contract] = struct{}{}
		item.Note = normalizeNote(item.Note)
		normalized = append(normalized, item)
	}
	sort.SliceStable(normalized, func(i, j int) bool {
		if !normalized[i].CreatedAt.Equal(normalized[j].CreatedAt) {
			return normalized[i].CreatedAt.After(normalized[j].CreatedAt)
		}
		return normalized[i].Contract.Hex() < normalized[j].Contract.Hex()
	})
	return normalized
}

func normalizeWalletBlacklistContracts(items []store.WalletBlacklistContract) []store.WalletBlacklistContract {
	if len(items) == 0 {
		return nil
	}
	seen := make(map[common.Address]struct{}, len(items))
	normalized := make([]store.WalletBlacklistContract, 0, len(items))
	for _, item := range items {
		if item.Contract == (common.Address{}) {
			continue
		}
		if _, ok := seen[item.Contract]; ok {
			continue
		}
		seen[item.Contract] = struct{}{}
		item.Note = normalizeNote(item.Note)
		normalized = append(normalized, item)
	}
	sort.SliceStable(normalized, func(i, j int) bool {
		if !normalized[i].CreatedAt.Equal(normalized[j].CreatedAt) {
			return normalized[i].CreatedAt.After(normalized[j].CreatedAt)
		}
		return normalized[i].Contract.Hex() < normalized[j].Contract.Hex()
	})
	return normalized
}

func normalizeNote(note string) string {
	return strings.TrimSpace(note)
}
