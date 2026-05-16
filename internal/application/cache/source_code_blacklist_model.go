package cache

import (
	"context"
	"errors"

	"github.com/useryege/athena/internal/application/sourcecode"
	"github.com/useryege/athena/internal/application/store"
)

var _ SourceCodeBlacklistModel = &sourceCodeBlacklistModel{}

type sourceCodeBlacklistModel struct {
	store          store.SourceCodeBlacklistStore
	cache          SourceCodeBlacklistCache
	writePublisher SourceCodeBlacklistWritePublisher
}

func NewSourceCodeBlacklistModel(store store.SourceCodeBlacklistStore, cache SourceCodeBlacklistCache, writePublisher SourceCodeBlacklistWritePublisher) SourceCodeBlacklistModel {
	if cache == nil {
		cache = NewLayeredBlacklistCache(NewLocalBlacklistCache(), nil)
	}
	return &sourceCodeBlacklistModel{
		store:          store,
		cache:          cache,
		writePublisher: writePublisher,
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
	if m.writePublisher == nil {
		return errors.New("source code blacklist write publisher is not configured")
	}

	if err := m.writePublisher.PublishAdd(ctx, field); err != nil {
		return err
	}
	fields, err := m.List(ctx)
	if err != nil {
		return err
	}
	for _, current := range fields {
		if current == field {
			return nil
		}
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
	if m.writePublisher == nil {
		return errors.New("source code blacklist write publisher is not configured")
	}

	if err := m.writePublisher.PublishDelete(ctx, field); err != nil {
		return err
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
