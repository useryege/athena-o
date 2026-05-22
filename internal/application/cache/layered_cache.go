package cache

import (
	"context"
	"sync"
)

var _ SourceCodeBlacklistCache = &LayeredBlacklistCache{}

type LayeredBlacklistCache struct {
	local  *LocalBlacklistCache
	remote SourceCodeBlacklistRemoteCache
	loadMu sync.Mutex
}

func NewLayeredBlacklistCache(local *LocalBlacklistCache, remote SourceCodeBlacklistRemoteCache) *LayeredBlacklistCache {
	if local == nil {
		local = NewLocalBlacklistCache()
	}
	return &LayeredBlacklistCache{
		local:  local,
		remote: remote,
	}
}

func (c *LayeredBlacklistCache) Take(ctx context.Context, loader func(context.Context) ([]string, error)) ([]string, error) {
	if c == nil {
		return loadSourceCodeBlacklistFields(ctx, loader)
	}
	if fields, version, ok := c.local.GetSnapshot(); ok {
		if c.remote == nil {
			return fields, nil
		}
		remoteVersion, versionOK, err := c.remote.Version(ctx)
		if err != nil {
			return nil, err
		}
		if versionOK && remoteVersion == version {
			return fields, nil
		}
	}

	c.loadMu.Lock()
	defer c.loadMu.Unlock()
	if fields, version, ok := c.local.GetSnapshot(); ok {
		if c.remote == nil {
			return fields, nil
		}
		remoteVersion, versionOK, err := c.remote.Version(ctx)
		if err != nil {
			return nil, err
		}
		if versionOK && remoteVersion == version {
			return fields, nil
		}
	}

	if c.remote != nil {
		fields, version, ok, err := c.remote.Get(ctx)
		if err != nil {
			return nil, err
		}
		if ok {
			c.local.SetWithVersion(fields, version)
			fields, _ = c.local.Get()
			return fields, nil
		}
	}

	fields, err := loadSourceCodeBlacklistFields(ctx, loader)
	if err != nil {
		return nil, err
	}
	if c.remote != nil {
		version, err := c.remote.Set(ctx, fields)
		if err != nil {
			return nil, err
		}
		c.local.SetWithVersion(fields, version)
		fields, _ = c.local.Get()
		return fields, nil
	}
	c.local.Set(fields)
	fields, _ = c.local.Get()
	return fields, nil
}

func (c *LayeredBlacklistCache) Set(ctx context.Context, fields []string) error {
	if c == nil {
		return nil
	}
	if c.remote != nil {
		version, err := c.remote.Set(ctx, fields)
		if err != nil {
			return err
		}
		c.local.SetWithVersion(fields, version)
		return nil
	}
	c.local.Set(fields)
	return nil
}

func (c *LayeredBlacklistCache) Del(ctx context.Context) error {
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

func (c *LayeredBlacklistCache) Version(ctx context.Context) (string, error) {
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

func loadSourceCodeBlacklistFields(ctx context.Context, loader func(context.Context) ([]string, error)) ([]string, error) {
	if loader == nil {
		return nil, nil
	}
	return loader(ctx)
}
