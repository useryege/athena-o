package store

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestProjectAveDetailFromRawResponseRestoresBusinessDetail(t *testing.T) {
	fetchedAt := time.Date(2026, time.May, 30, 12, 0, 0, 0, time.UTC)
	raw := []byte(`{"status":1,"msg":"SUCCESS","data_type":1,"data":{"token":{"logo_url":" https://example.com/logo.png ","token":"token","chain":"bsc"},"pairs":[{"pair":"pair-1","chain":"bsc"}],"is_audited":true}}`)

	detail, err := projectAveDetailFromRawResponse(raw, fetchedAt)
	if err != nil {
		t.Fatalf("detail from raw response: %v", err)
	}
	if detail == nil {
		t.Fatal("detail is nil")
	}
	if detail.Status != 1 || detail.Msg != "SUCCESS" || detail.DataType != 1 || !detail.IsAudited {
		t.Fatalf("detail status fields = %#v, want Ave response status fields", detail)
	}
	if detail.Token.LogoURL != "https://example.com/logo.png" || detail.Token.Token != "token" || detail.Token.Chain != "bsc" {
		t.Fatalf("detail token = %#v, want mapped token", detail.Token)
	}
	if len(detail.Pairs) != 1 || detail.Pairs[0].Pair != "pair-1" {
		t.Fatalf("detail pairs = %#v, want mapped pair", detail.Pairs)
	}
	if !detail.FetchedAt.Equal(fetchedAt) {
		t.Fatalf("FetchedAt = %s, want %s", detail.FetchedAt, fetchedAt)
	}
	if string(detail.RawResponse) != string(raw) {
		t.Fatalf("RawResponse = %s, want original raw response", string(detail.RawResponse))
	}
}

func TestProjectAveDetailRawResponseIsOmittedFromJSON(t *testing.T) {
	detail := ProjectAveDetail{Status: 1, RawResponse: json.RawMessage(`{"status":1}`)}

	data, err := json.Marshal(detail)
	if err != nil {
		t.Fatalf("marshal detail: %v", err)
	}
	if strings.Contains(string(data), "RawResponse") || strings.Contains(string(data), "status\":1") {
		t.Fatalf("json detail = %s, want raw response omitted", string(data))
	}
}
