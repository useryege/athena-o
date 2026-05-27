package cache

import (
	"context"
	"sort"
	"strings"
	"sync"

	"github.com/ethereum/go-ethereum/common"
	"github.com/useryege/athena/internal/application/store"
)

var _ WalletBlacklistCache = &LayeredWalletBlacklistCache{}

type LocalWalletBlacklistCache struct {
	mu      sync.RWMutex
	items   []store.WalletBlacklistEntry
	ready   bool
	version string
}

func NewLocalWalletBlacklistCache(items ...store.WalletBlacklistEntry) *LocalWalletBlacklistCache {
	c := &LocalWalletBlacklistCache{}
	if len(items) > 0 {
		c.Set(items)
	}
	return c
}

func (c *LocalWalletBlacklistCache) Get() ([]store.WalletBlacklistEntry, bool) {
	if c == nil {
		return nil, false
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	if !c.ready {
		return nil, false
	}
	return append([]store.WalletBlacklistEntry(nil), c.items...), true
}

func (c *LocalWalletBlacklistCache) Set(items []store.WalletBlacklistEntry) {
	c.SetWithVersion(items, newBlacklistVersion())
}

func (c *LocalWalletBlacklistCache) GetSnapshot() ([]store.WalletBlacklistEntry, string, bool) {
	if c == nil {
		return nil, "", false
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	if !c.ready {
		return nil, "", false
	}
	return append([]store.WalletBlacklistEntry(nil), c.items...), c.version, true
}

func (c *LocalWalletBlacklistCache) SetWithVersion(items []store.WalletBlacklistEntry, version string) {
	if c == nil {
		return
	}
	normalized := normalizeWalletBlacklistEntries(items)
	if version == "" {
		version = newBlacklistVersion()
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.items = normalized
	c.ready = true
	c.version = version
}

func (c *LocalWalletBlacklistCache) Del() {
	if c == nil {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.items = nil
	c.ready = false
	c.version = ""
}

func (c *LocalWalletBlacklistCache) Version() (string, bool) {
	if c == nil {
		return "", false
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	if !c.ready {
		return "", false
	}
	return c.version, true
}

type LayeredWalletBlacklistCache struct {
	local  *LocalWalletBlacklistCache
	remote WalletBlacklistRemoteCache
	loadMu sync.Mutex
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

func (c *LayeredWalletBlacklistCache) Take(ctx context.Context, loader func(context.Context) ([]store.WalletBlacklistEntry, error)) ([]store.WalletBlacklistEntry, error) {
	if c == nil {
		return loadWalletBlacklistEntries(ctx, loader)
	}
	if items, version, ok := c.local.GetSnapshot(); ok {
		if c.remote == nil {
			return items, nil
		}
		remoteVersion, versionOK, err := c.remote.Version(ctx)
		if err != nil {
			return nil, err
		}
		if versionOK && remoteVersion == version {
			return items, nil
		}
	}

	c.loadMu.Lock()
	defer c.loadMu.Unlock()
	if items, version, ok := c.local.GetSnapshot(); ok {
		if c.remote == nil {
			return items, nil
		}
		remoteVersion, versionOK, err := c.remote.Version(ctx)
		if err != nil {
			return nil, err
		}
		if versionOK && remoteVersion == version {
			return items, nil
		}
	}

	if c.remote != nil {
		items, version, ok, err := c.remote.Get(ctx)
		if err != nil {
			return nil, err
		}
		if ok {
			c.local.SetWithVersion(items, version)
			items, _ = c.local.Get()
			return items, nil
		}
	}

	items, err := loadWalletBlacklistEntries(ctx, loader)
	if err != nil {
		return nil, err
	}
	if c.remote != nil {
		version, err := c.remote.Set(ctx, items)
		if err != nil {
			return nil, err
		}
		c.local.SetWithVersion(items, version)
		items, _ = c.local.Get()
		return items, nil
	}
	c.local.Set(items)
	items, _ = c.local.Get()
	return items, nil
}

func (c *LayeredWalletBlacklistCache) Set(ctx context.Context, items []store.WalletBlacklistEntry) error {
	if c == nil {
		return nil
	}
	if c.remote != nil {
		version, err := c.remote.Set(ctx, items)
		if err != nil {
			return err
		}
		c.local.SetWithVersion(items, version)
		return nil
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

func (c *LayeredWalletBlacklistCache) Version(ctx context.Context) (string, error) {
	if c == nil {
		return "", nil
	}
	if c.remote != nil {
		version, ok, err := c.remote.Version(ctx)
		if err != nil {
			return "", err
		}
		if ok {
			return version, nil
		}
	}
	version, _ := c.local.Version()
	return version, nil
}

func loadWalletBlacklistEntries(ctx context.Context, loader func(context.Context) ([]store.WalletBlacklistEntry, error)) ([]store.WalletBlacklistEntry, error) {
	if loader == nil {
		return nil, nil
	}
	items, err := loader(ctx)
	if err != nil {
		return nil, err
	}
	return normalizeWalletBlacklistEntries(items), nil
}

func normalizeWalletBlacklistEntries(items []store.WalletBlacklistEntry) []store.WalletBlacklistEntry {
	if len(items) == 0 {
		return nil
	}
	seen := make(map[common.Address]struct{}, len(items))
	normalized := make([]store.WalletBlacklistEntry, 0, len(items))
	for _, item := range items {
		if item.Wallet == (common.Address{}) {
			continue
		}
		if _, ok := seen[item.Wallet]; ok {
			continue
		}
		seen[item.Wallet] = struct{}{}
		item.Note = normalizeNote(item.Note)
		normalized = append(normalized, item)
	}
	sort.SliceStable(normalized, func(i, j int) bool {
		if !normalized[i].CreatedAt.Equal(normalized[j].CreatedAt) {
			return normalized[i].CreatedAt.After(normalized[j].CreatedAt)
		}
		return normalized[i].Wallet.Hex() < normalized[j].Wallet.Hex()
	})
	return normalized
}

func normalizeNote(note string) string {
	return strings.TrimSpace(note)
}
