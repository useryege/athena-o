package application

import (
	"context"

	"github.com/google/uuid"
)

type StaticFieldFetcher interface {
	FetchName(ctx context.Context, projectID uuid.UUID) (string, error)
	FetchSymbol(ctx context.Context, projectID uuid.UUID) (string, error)
	FetchDecimals(ctx context.Context, projectID uuid.UUID) (uint8, error)
	FetchSourceCode(ctx context.Context, projectID uuid.UUID) (string, error)
	FetchSourceCodeABI(ctx context.Context, projectID uuid.UUID) (string, error)
}

type StaticFieldFetcherImpl struct {
}

func NewStaticFieldFetcher() StaticFieldFetcher {
	return &StaticFieldFetcherImpl{}
}

func (f *StaticFieldFetcherImpl) FetchName(ctx context.Context, projectID uuid.UUID) (string, error) {
	panic("not implemented")
}

func (f *StaticFieldFetcherImpl) FetchSymbol(ctx context.Context, projectID uuid.UUID) (string, error) {
	panic("not implemented")
}

func (f *StaticFieldFetcherImpl) FetchDecimals(ctx context.Context, projectID uuid.UUID) (uint8, error) {
	panic("not implemented")
}

func (f *StaticFieldFetcherImpl) FetchSourceCode(ctx context.Context, projectID uuid.UUID) (string, error) {
	panic("not implemented")
}

func (f *StaticFieldFetcherImpl) FetchSourceCodeABI(ctx context.Context, projectID uuid.UUID) (string, error) {
	panic("not implemented")
}
