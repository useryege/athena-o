package cache

import (
	"context"

	"github.com/useryege/athena/internal/application/sourcecode"
	"github.com/useryege/athena/internal/application/store"
)

var _ SourceCodeBlacklistModel = &sourceCodeBlacklistModel{}

type sourceCodeBlacklistModel struct {
	store store.SourceCodeBlacklistStore
	cache SourceCodeBlacklistCache
}

func NewSourceCodeBlacklistModel(store store.SourceCodeBlacklistStore, cache SourceCodeBlacklistCache) SourceCodeBlacklistModel {
	if cache == nil {
		cache = NewLayeredBlacklistCache(NewLocalBlacklistCache(), nil)
	}
	return &sourceCodeBlacklistModel{
		store: store,
		cache: cache,
	}
}

func (m *sourceCodeBlacklistModel) Load(ctx context.Context) error {
	return m.refresh(ctx)
}

func (m *sourceCodeBlacklistModel) List(ctx context.Context) ([]string, error) {
	if m == nil || m.cache == nil {
		return nil, nil
	}
	return m.cache.Take(ctx, func(ctx context.Context) ([]string, error) {
		if m.store == nil {
			return nil, nil
		}
		return m.store.ListSourceCodeBlacklistFields(ctx)
	})
}

func (m *sourceCodeBlacklistModel) Add(ctx context.Context, field string) error {
	if m == nil {
		return nil
	}
	field = sourcecode.NormalizeBlacklistField(field)
	if field == "" {
		return nil
	}
	if m.store != nil {
		if err := m.store.AddSourceCodeBlacklistField(ctx, field); err != nil {
			return err
		}
		return m.refresh(ctx)
	}
	fields, err := m.List(ctx)
	if err != nil {
		return err
	}
	fields = append(fields, field)
	return m.cache.Set(ctx, fields)
}

func (m *sourceCodeBlacklistModel) Delete(ctx context.Context, field string) error {
	if m == nil {
		return nil
	}
	field = sourcecode.NormalizeBlacklistField(field)
	if field == "" {
		return nil
	}
	if m.store != nil {
		if err := m.store.DeleteSourceCodeBlacklistField(ctx, field); err != nil {
			return err
		}
		return m.refresh(ctx)
	}
	fields, err := m.List(ctx)
	if err != nil {
		return err
	}
	filtered := fields[:0]
	for _, current := range fields {
		if current != field {
			filtered = append(filtered, current)
		}
	}
	return m.cache.Set(ctx, filtered)
}

func (m *sourceCodeBlacklistModel) refresh(ctx context.Context) error {
	if m.cache == nil {
		return nil
	}
	if m.store == nil {
		m.cache.Del(ctx)
		return nil
	}
	fields, err := m.store.ListSourceCodeBlacklistFields(ctx)
	if err != nil {
		return err
	}
	return m.cache.Set(ctx, fields)
}
