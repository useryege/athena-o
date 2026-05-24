package avelogo

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/useryege/athena/util/ave"
)

type fakeAveClient struct {
	response *ave.TokenDetailResponse
	err      error
	tokenID  string
}

func (f *fakeAveClient) GetTokenDetail(_ context.Context, tokenID string) (*ave.TokenDetailResponse, error) {
	f.tokenID = tokenID
	return f.response, f.err
}

func TestFetchDetail(t *testing.T) {
	client := &fakeAveClient{response: &ave.TokenDetailResponse{
		Data: ave.TokenDetailData{
			Token: ave.Token{LogoURL: " https://example.com/logo.png "},
		},
	}}
	fetcher := NewFetcher(client)

	detail, err := fetcher.FetchDetail(context.Background(), " token-bsc ")
	if err != nil {
		t.Fatalf("FetchDetail: %v", err)
	}
	if detail.Data.Token.LogoURL != "https://example.com/logo.png" {
		t.Fatalf("logo = %q, want trimmed logo", detail.Data.Token.LogoURL)
	}
	if client.tokenID != "token-bsc" {
		t.Fatalf("token id = %q, want trimmed token id", client.tokenID)
	}
}

func TestFetchDetailRequiresClient(t *testing.T) {
	_, err := (*fetcherImpl)(nil).FetchDetail(context.Background(), "token-bsc")
	if err == nil || !strings.Contains(err.Error(), "not configured") {
		t.Fatalf("error = %v, want not configured", err)
	}
}

func TestFetchDetailRequiresTokenID(t *testing.T) {
	fetcher := NewFetcher(&fakeAveClient{})
	_, err := fetcher.FetchDetail(context.Background(), " ")
	if err == nil || !strings.Contains(err.Error(), "token id is empty") {
		t.Fatalf("error = %v, want token id error", err)
	}
}

func TestFetchDetailClientError(t *testing.T) {
	fetcher := NewFetcher(&fakeAveClient{err: errors.New("ave down")})
	_, err := fetcher.FetchDetail(context.Background(), "token-bsc")
	if err == nil || !strings.Contains(err.Error(), "ave down") {
		t.Fatalf("error = %v, want wrapped Ave error", err)
	}
}
