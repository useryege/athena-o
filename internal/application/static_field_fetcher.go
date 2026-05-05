package application

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
)

type StaticFieldFetcher interface {
	FetchName(ctx context.Context, projectID uuid.UUID) (string, error)
	FetchSymbol(ctx context.Context, projectID uuid.UUID) (string, error)
	FetchDecimals(ctx context.Context, projectID uuid.UUID) (uint8, error)
	FetchSourceCode(ctx context.Context, projectID uuid.UUID) (string, error)
	FetchSourceCodeABI(ctx context.Context, projectID uuid.UUID) (string, error)
}

type staticFieldFetcherImpl struct{}

var ErrStaticFetcherNotImplemented = errors.New("static field fetcher is not implemented")

func NewStaticFieldFetcher() StaticFieldFetcher {
	return &staticFieldFetcherImpl{}
}

func (f *staticFieldFetcherImpl) FetchName(ctx context.Context, projectID uuid.UUID) (string, error) {
	return "", wrapStaticFetcherNotImplemented(StaticFieldName, projectID)
}

func (f *staticFieldFetcherImpl) FetchSymbol(ctx context.Context, projectID uuid.UUID) (string, error) {
	return "", wrapStaticFetcherNotImplemented(StaticFieldSymbol, projectID)
}

func (f *staticFieldFetcherImpl) FetchDecimals(ctx context.Context, projectID uuid.UUID) (uint8, error) {
	return 0, wrapStaticFetcherNotImplemented(StaticFieldDecimals, projectID)
}

func (f *staticFieldFetcherImpl) FetchSourceCode(ctx context.Context, projectID uuid.UUID) (string, error) {
	return "", wrapStaticFetcherNotImplemented(StaticFieldSourceCode, projectID)
}

func (f *staticFieldFetcherImpl) FetchSourceCodeABI(ctx context.Context, projectID uuid.UUID) (string, error) {
	return "", wrapStaticFetcherNotImplemented(StaticFieldSourceCodeABI, projectID)
}

func wrapStaticFetcherNotImplemented(field StaticField, projectID uuid.UUID) error {
	return fmt.Errorf("%w: field=%s project_id=%s", ErrStaticFetcherNotImplemented, field, projectID)
}
