package store

import (
	"testing"
	"time"
)

func TestProjectAveDetailFromRawResponseRestoresAveData(t *testing.T) {
	chainID := int64(56)
	fetchedAt := time.Date(2026, time.May, 30, 12, 0, 0, 0, time.UTC)
	raw := []byte(`{"status":1,"msg":"SUCCESS","data_type":1,"data":{"token":{"logo_url":"https://example.com/logo.png","token":"token","chain":"bsc"},"pairs":[{"pair":"pair-1","chain":"bsc"}],"is_audited":true}}`)

	detail, err := projectAveDetailFromRawResponse(raw, chainID, fetchedAt)
	if err != nil {
		t.Fatalf("detail from raw response: %v", err)
	}
	if detail == nil {
		t.Fatal("detail is nil")
	}
	if detail.ChainID != chainID || !detail.FetchedAt.Equal(fetchedAt) {
		t.Fatalf("detail metadata = %#v, want chain id and fetched at", detail)
	}
	if !detail.Data.IsAudited {
		t.Fatalf("detail data = %#v, want audited flag from Ave data", detail.Data)
	}
	if detail.Data.Token.LogoURL != "https://example.com/logo.png" || detail.Data.Token.Token != "token" || detail.Data.Token.Chain != "bsc" {
		t.Fatalf("detail token = %#v, want Ave token data", detail.Data.Token)
	}
	if len(detail.Data.Pairs) != 1 || detail.Data.Pairs[0].Pair != "pair-1" {
		t.Fatalf("detail pairs = %#v, want Ave pair data", detail.Data.Pairs)
	}
}
