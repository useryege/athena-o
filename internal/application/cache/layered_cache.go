package cache

import "context"

var _ SourceCodeBlacklistCache = &LayeredBlacklistCache{}

type LayeredBlacklistCache struct {
	local  *LocalBlacklistCache
	remote SourceCodeBlacklistRemoteCache
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
	if fields, ok := c.local.Get(); ok {
		return fields, nil
	}
	if c.remote != nil {
		fields, ok, err := c.remote.Get(ctx)
		if err != nil {
			return nil, err
		}
		if ok {
			c.local.Set(fields)
			return fields, nil
		}
	}

	fields, err := loadSourceCodeBlacklistFields(ctx, loader)
	if err != nil {
		return nil, err
	}
	if c.remote != nil {
		if err := c.remote.Set(ctx, fields); err != nil {
			return nil, err
		}
	}
	c.local.Set(fields)
	return fields, nil
}

func (c *LayeredBlacklistCache) Set(ctx context.Context, fields []string) error {
	if c == nil {
		return nil
	}
	if c.remote != nil {
		if err := c.remote.Set(ctx, fields); err != nil {
			return err
		}
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

func loadSourceCodeBlacklistFields(ctx context.Context, loader func(context.Context) ([]string, error)) ([]string, error) {
	if loader == nil {
		return nil, nil
	}
	return loader(ctx)
}
