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

type staticFieldFetcherImpl struct {
}

func NewStaticFieldFetcher() StaticFieldFetcher {
	return &staticFieldFetcherImpl{}
}

func (f *staticFieldFetcherImpl) FetchName(ctx context.Context, projectID uuid.UUID) (string, error) {
	panic("not implemented")
}

func (f *staticFieldFetcherImpl) FetchSymbol(ctx context.Context, projectID uuid.UUID) (string, error) {
	panic("not implemented")
}

func (f *staticFieldFetcherImpl) FetchDecimals(ctx context.Context, projectID uuid.UUID) (uint8, error) {
	panic("not implemented")
}

func (f *staticFieldFetcherImpl) FetchSourceCode(ctx context.Context, projectID uuid.UUID) (string, error) {
	panic("not implemented")
}

func (f *staticFieldFetcherImpl) FetchSourceCodeABI(ctx context.Context, projectID uuid.UUID) (string, error) {
	panic("not implemented")
}
