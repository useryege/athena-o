package cache

import (
	"sync"

	"github.com/useryege/athena/internal/application/sourcecode"
)

type LocalBlacklistCache struct {
	mu      sync.RWMutex
	fields  []string
	ready   bool
	version string
}

func NewLocalBlacklistCache(fields ...string) *LocalBlacklistCache {
	c := &LocalBlacklistCache{}
	if len(fields) > 0 {
		c.Set(fields)
	}
	return c
}

func (c *LocalBlacklistCache) Get() ([]string, bool) {
	if c == nil {
		return nil, false
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	if !c.ready {
		return nil, false
	}
	return append([]string(nil), c.fields...), true
}

func (c *LocalBlacklistCache) GetSnapshot() ([]string, string, bool) {
	if c == nil {
		return nil, "", false
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	if !c.ready {
		return nil, "", false
	}
	return append([]string(nil), c.fields...), c.version, true
}

func (c *LocalBlacklistCache) Set(fields []string) {
	c.SetWithVersion(fields, newBlacklistVersion())
}

func (c *LocalBlacklistCache) SetWithVersion(fields []string, version string) {
	if c == nil {
		return
	}
	fields = sourcecode.NormalizeBlacklistFields(fields)
	if version == "" {
		version = newBlacklistVersion()
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.fields = fields
	c.ready = true
	c.version = version
}

func (c *LocalBlacklistCache) Del() {
	if c == nil {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.fields = nil
	c.ready = false
	c.version = ""
}

func (c *LocalBlacklistCache) Version() (string, bool) {
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
