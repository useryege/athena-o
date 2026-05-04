package application

import (
	"context"

	"github.com/google/uuid"
)

type StaticFieldStore interface {
	GetStaticState(ctx context.Context, projectID uuid.UUID) (*ProjectStaticState, error)
	SaveStaticState(ctx context.Context, projectID uuid.UUID, static *ProjectStaticState) error
}

type StaticFieldStoreImpl struct {
}

func NewStaticFieldStore() StaticFieldStore {
	return &StaticFieldStoreImpl{}
}

func (s *StaticFieldStoreImpl) GetStaticState(ctx context.Context, projectID uuid.UUID) (*ProjectStaticState, error) {
	panic("not implemented")
}

func (s *StaticFieldStoreImpl) SaveStaticState(ctx context.Context, projectID uuid.UUID, static *ProjectStaticState) error {
	panic("not implemented")
}
