package application

import "context"

type ProjectMetaStore interface {
	SaveProjectMeta(ctx context.Context, meta ProjectMeta) error
}

type noopProjectMetaStore struct{}

func NewNoopProjectMetaStore() ProjectMetaStore {
	return noopProjectMetaStore{}
}

func (noopProjectMetaStore) SaveProjectMeta(ctx context.Context, meta ProjectMeta) error {
	return nil
}
