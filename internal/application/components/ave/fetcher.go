package ave

import (
	"context"
	"fmt"
	"strings"

	utilave "github.com/useryege/athena/util/ave"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Fetcher interface {
	FetchDetail(ctx context.Context, tokenID string) (*utilave.TokenDetailResponse, error)
}

type fetcherImpl struct {
	client utilave.Client
}

func NewFetcher(client utilave.Client) Fetcher {
	return &fetcherImpl{client: client}
}

func (f *fetcherImpl) FetchDetail(ctx context.Context, tokenID string) (*utilave.TokenDetailResponse, error) {
	if f == nil || f.client == nil {
		return nil, status.Error(codes.FailedPrecondition, "Ave detail fetcher is not configured")
	}
	tokenID = strings.TrimSpace(tokenID)
	if tokenID == "" {
		return nil, status.Error(codes.InvalidArgument, "Ave token id is empty")
	}
	response, err := f.client.GetTokenDetail(ctx, tokenID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch Ave token detail: %w", err)
	}
	response.Data.Token.LogoURL = strings.TrimSpace(response.Data.Token.LogoURL)
	return response, nil
}

func ChainNameForChainID(chainID int64) (string, bool) {
	switch chainID {
	case 1:
		return "eth", true
	case 56:
		return "bsc", true
	case 137:
		return "polygon", true
	case 42161:
		return "arbitrum", true
	case 10:
		return "optimism", true
	case 8453:
		return "base", true
	default:
		return "", false
	}
}
