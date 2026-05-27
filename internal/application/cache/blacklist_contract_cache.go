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
	mu      sync.RWMutex
	items   []store.BytecodeBlacklistContract
	ready   bool
	version string
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
	c.SetWithVersion(items, newBlacklistVersion())
}

func (c *LocalBytecodeBlacklistCache) GetSnapshot() ([]store.BytecodeBlacklistContract, string, bool) {
	if c == nil {
		return nil, "", false
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	if !c.ready {
		return nil, "", false
	}
	return append([]store.BytecodeBlacklistContract(nil), c.items...), c.version, true
}

func (c *LocalBytecodeBlacklistCache) SetWithVersion(items []store.BytecodeBlacklistContract, version string) {
	if c == nil {
		return
	}
	normalized := normalizeBytecodeBlacklistContracts(items)
	if version == "" {
		version = newBlacklistVersion()
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.items = normalized
	c.ready = true
	c.version = version
}

func (c *LocalBytecodeBlacklistCache) Del() {
	if c == nil {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.items = nil
	c.ready = false
	c.version = ""
}

func (c *LocalBytecodeBlacklistCache) Version() (string, bool) {
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

type LayeredBytecodeBlacklistCache struct {
	local  *LocalBytecodeBlacklistCache
	remote BytecodeBlacklistRemoteCache
	loadMu sync.Mutex
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

	items, err := loadBytecodeBlacklistContracts(ctx, loader)
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

func (c *LayeredBytecodeBlacklistCache) Set(ctx context.Context, items []store.BytecodeBlacklistContract) error {
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

func (c *LayeredBytecodeBlacklistCache) Version(ctx context.Context) (string, error) {
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
