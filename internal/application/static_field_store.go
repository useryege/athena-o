package application

import (
	"context"

	"github.com/google/uuid"
)

type StaticFieldStore interface {
	GetStaticState(ctx context.Context, projectID uuid.UUID) (*ProjectStaticState, error)
	SaveStaticState(ctx context.Context, projectID uuid.UUID, static *ProjectStaticState) error
}

type staticFieldStoreImpl struct {
}

func NewStaticFieldStore() StaticFieldStore {
	return &staticFieldStoreImpl{}
}

func (s *staticFieldStoreImpl) GetStaticState(ctx context.Context, projectID uuid.UUID) (*ProjectStaticState, error) {
	panic("not implemented")
}

func (s *staticFieldStoreImpl) SaveStaticState(ctx context.Context, projectID uuid.UUID, static *ProjectStaticState) error {
	panic("not implemented")
}
