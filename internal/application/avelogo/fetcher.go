package avelogo

import (
	"context"
	"fmt"
	"strings"

	"github.com/useryege/athena/util/ave"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Fetcher interface {
	FetchDetail(ctx context.Context, tokenID string) (*ave.TokenDetailResponse, error)
}

type fetcherImpl struct {
	client ave.Client
}

func NewFetcher(client ave.Client) Fetcher {
	return &fetcherImpl{client: client}
}

func (f *fetcherImpl) FetchDetail(ctx context.Context, tokenID string) (*ave.TokenDetailResponse, error) {
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
