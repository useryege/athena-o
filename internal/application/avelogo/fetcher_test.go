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

func TestFetchLogo(t *testing.T) {
	client := &fakeAveClient{response: &ave.TokenDetailResponse{
		Data: ave.TokenDetailData{
			Token: ave.Token{LogoURL: " https://example.com/logo.png "},
		},
	}}
	fetcher := NewFetcher(client)

	logo, err := fetcher.FetchLogo(context.Background(), " token-bsc ")
	if err != nil {
		t.Fatalf("FetchLogo: %v", err)
	}
	if logo != "https://example.com/logo.png" {
		t.Fatalf("logo = %q, want trimmed logo", logo)
	}
	if client.tokenID != "token-bsc" {
		t.Fatalf("token id = %q, want trimmed token id", client.tokenID)
	}
}

func TestFetchLogoRequiresClient(t *testing.T) {
	_, err := (*fetcherImpl)(nil).FetchLogo(context.Background(), "token-bsc")
	if err == nil || !strings.Contains(err.Error(), "not configured") {
		t.Fatalf("error = %v, want not configured", err)
	}
}

func TestFetchLogoRequiresTokenID(t *testing.T) {
	fetcher := NewFetcher(&fakeAveClient{})
	_, err := fetcher.FetchLogo(context.Background(), " ")
	if err == nil || !strings.Contains(err.Error(), "token id is empty") {
		t.Fatalf("error = %v, want token id error", err)
	}
}

func TestFetchLogoClientError(t *testing.T) {
	fetcher := NewFetcher(&fakeAveClient{err: errors.New("ave down")})
	_, err := fetcher.FetchLogo(context.Background(), "token-bsc")
	if err == nil || !strings.Contains(err.Error(), "ave down") {
		t.Fatalf("error = %v, want wrapped Ave error", err)
	}
}
