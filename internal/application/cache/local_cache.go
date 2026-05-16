package cache

import (
	"sync"

	"github.com/useryege/athena/internal/application/sourcecode"
)

type LocalBlacklistCache struct {
	mu     sync.RWMutex
	fields []string
	ready  bool
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
	return c.fields, true
}

func (c *LocalBlacklistCache) Set(fields []string) {
	if c == nil {
		return
	}
	fields = sourcecode.NormalizeBlacklistFields(fields)
	c.mu.Lock()
	defer c.mu.Unlock()
	c.fields = fields
	c.ready = true
}

func (c *LocalBlacklistCache) Del() {
	if c == nil {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.fields = nil
	c.ready = false
}
