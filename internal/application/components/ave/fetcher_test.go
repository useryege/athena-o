package ave

import (
	"context"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	utilave "github.com/useryege/athena/util/ave"
)

type fetcherClientFake struct {
	contract common.Address
	chainID  int64
	calls    int
	resp     *utilave.TokenDetailResponse
}

func (f *fetcherClientFake) GetTokenDetail(_ context.Context, contract common.Address, chainID int64) (*utilave.TokenDetailResponse, error) {
	f.calls++
	f.contract = contract
	f.chainID = chainID
	return f.resp, nil
}

func TestFetcherFetchDetailPassesContractAndChainID(t *testing.T) {
	contract := common.HexToAddress("0x79a11e727d00ef6333845b660c94c3e1478e6a41")
	client := &fetcherClientFake{resp: &utilave.TokenDetailResponse{
		Data: utilave.TokenDetailData{
			Token: utilave.Token{LogoURL: " https://example.com/logo.png "},
		},
	}}
	fetcher := NewFetcher(client)

	resp, err := fetcher.FetchDetail(context.Background(), contract, 56)
	if err != nil {
		t.Fatalf("FetchDetail: %v", err)
	}
	if client.calls != 1 || client.contract != contract || client.chainID != 56 {
		t.Fatalf("client call = (%d, %s, %d), want one call with contract and chain id", client.calls, client.contract, client.chainID)
	}
	if resp.Data.Token.LogoURL != "https://example.com/logo.png" {
		t.Fatalf("LogoURL = %q, want trimmed logo url", resp.Data.Token.LogoURL)
	}
}

func TestFetcherFetchDetailRequiresContract(t *testing.T) {
	client := &fetcherClientFake{resp: &utilave.TokenDetailResponse{}}
	fetcher := NewFetcher(client)

	_, err := fetcher.FetchDetail(context.Background(), common.Address{}, 56)
	if err == nil || !strings.Contains(err.Error(), "contract") {
		t.Fatalf("error = %v, want contract error", err)
	}
	if client.calls != 0 {
		t.Fatalf("client calls = %d, want 0", client.calls)
	}
}
