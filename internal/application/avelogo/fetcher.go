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
	FetchLogo(ctx context.Context, tokenID string) (string, error)
}

type fetcherImpl struct {
	client ave.Client
}

func NewFetcher(client ave.Client) Fetcher {
	return &fetcherImpl{client: client}
}

func (f *fetcherImpl) FetchLogo(ctx context.Context, tokenID string) (string, error) {
	if f == nil || f.client == nil {
		return "", status.Error(codes.FailedPrecondition, "Ave logo fetcher is not configured")
	}
	tokenID = strings.TrimSpace(tokenID)
	if tokenID == "" {
		return "", status.Error(codes.InvalidArgument, "Ave token id is empty")
	}
	response, err := f.client.GetTokenDetail(ctx, tokenID)
	if err != nil {
		return "", fmt.Errorf("failed to fetch Ave token detail: %w", err)
	}
	return strings.TrimSpace(response.Data.Token.LogoURL), nil
}
