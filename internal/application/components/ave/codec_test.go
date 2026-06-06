package ave

import (
	"encoding/json"
	"testing"
	"time"

	utilave "github.com/useryege/athena/util/ave"
)

func TestDetailFromResponseKeepsRawResponseAndMapsDetail(t *testing.T) {
	fetchedAt := time.Date(2026, time.May, 30, 12, 0, 0, 0, time.UTC)
	resp := &utilave.TokenDetailResponse{
		Status:   1,
		Msg:      "SUCCESS",
		DataType: 1,
		Data: utilave.TokenDetailData{
			Token:     utilave.Token{LogoURL: " https://example.com/logo.png ", Token: "token", Chain: "bsc"},
			Pairs:     []utilave.Pair{{Pair: "pair-1", Chain: "bsc"}},
			IsAudited: true,
		},
	}

	detail, err := DetailFromResponse(resp, fetchedAt)
	if err != nil {
		t.Fatalf("detail from response: %v", err)
	}
	if detail == nil {
		t.Fatal("detail is nil")
	}
	if detail.Token.LogoURL != "https://example.com/logo.png" || len(detail.Pairs) != 1 || detail.Pairs[0].Pair != "pair-1" {
		t.Fatalf("detail = %#v, want trimmed logo and pair", detail)
	}
	if !detail.FetchedAt.Equal(fetchedAt) {
		t.Fatalf("FetchedAt = %s, want %s", detail.FetchedAt, fetchedAt)
	}
	if len(detail.RawResponse) == 0 {
		t.Fatal("RawResponse is empty")
	}
	var raw utilave.TokenDetailResponse
	if err := json.Unmarshal(detail.RawResponse, &raw); err != nil {
		t.Fatalf("unmarshal raw response: %v", err)
	}
	if raw.Status != resp.Status || raw.Data.Token.LogoURL != resp.Data.Token.LogoURL {
		t.Fatalf("raw response = %#v, want original response values", raw)
	}
}
