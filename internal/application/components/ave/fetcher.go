package ave

import (
	"context"
	"fmt"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	utilave "github.com/useryege/athena/util/ave"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Fetcher interface {
	FetchDetail(ctx context.Context, contract common.Address, chainID int64) (*utilave.TokenDetailResponse, error)
}

type fetcherImpl struct {
	client utilave.Client
}

func NewFetcher(client utilave.Client) Fetcher {
	return &fetcherImpl{client: client}
}

func (f *fetcherImpl) FetchDetail(ctx context.Context, contract common.Address, chainID int64) (*utilave.TokenDetailResponse, error) {
	if f == nil || f.client == nil {
		return nil, status.Error(codes.FailedPrecondition, "Ave detail fetcher is not configured")
	}
	if contract == (common.Address{}) {
		return nil, status.Error(codes.InvalidArgument, "Ave token contract is empty")
	}
	response, err := f.client.GetTokenDetail(ctx, contract, chainID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch Ave token detail: %w", err)
	}
	response.Data.Token.LogoURL = strings.TrimSpace(response.Data.Token.LogoURL)
	return response, nil
}
