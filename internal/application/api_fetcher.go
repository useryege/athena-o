package application

import (
	"context"

	"github.com/ethereum/go-ethereum/common"
)

type APIFetcher interface {
	FetchSourceCode(ctx context.Context, Contract common.Address) (string, error)
	FetchSourceCodeABI(ctx context.Context, Contract common.Address) (string, error)
}

type apiFetcherImpl struct {
}

func NewAPIFetcher() APIFetcher {
	return &apiFetcherImpl{}
}

func (f *apiFetcherImpl) FetchSourceCode(ctx context.Context, Contract common.Address) (string, error) {
	return "SourceCode", nil
}

func (f *apiFetcherImpl) FetchSourceCodeABI(ctx context.Context, Contract common.Address) (string, error) {
	return "SourceCodeABI", nil
}
