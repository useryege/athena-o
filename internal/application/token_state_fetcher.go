package application

import "context"

type TokenMetadataFetcher interface {
	FetchTokenMetadata(ctx context.Context, project *Project) (*TokenMetadata, error)
}

type tokenMetadataFetcher struct {
}

func NewTokenMetadataFetcher() TokenMetadataFetcher {
	return &tokenMetadataFetcher{}
}

func (f *tokenMetadataFetcher) FetchTokenMetadata(ctx context.Context, project *Project) (*TokenMetadata, error) {
	return nil, nil
}
