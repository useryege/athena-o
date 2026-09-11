package polymarket

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestComboDirectoryUsesIndependentHostAndBounds(t *testing.T) {
	gamma := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { t.Error("directory sent to gamma"); w.WriteHeader(500) }))
	defer gamma.Close()
	combo := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/rfq/combo-markets" || r.URL.Query().Get("cursor") != "a+/=" || r.URL.Query().Get("limit") != "100" {
			t.Error(r.URL)
		}
		fmt.Fprint(w, `{"markets":[{"id":"3977566","condition_id":"0x02","position_ids":["99999999999999999999999999999999999999999999999999"]}],"next_cursor":"next"}`)
	}))
	defer combo.Close()
	c, e := NewGammaClient(GammaConfig{GammaBaseURL: gamma.URL, ComboBaseURL: combo.URL})
	if e != nil {
		t.Fatal(e)
	}
	page, e := c.ListComboMarkets(context.Background(), "a+/=", 100)
	if e != nil || len(page.Markets) != 1 || page.Markets[0].PositionIDs[0] != "99999999999999999999999999999999999999999999999999" || page.NextCursor != "next" {
		t.Fatal(page, e)
	}
	if _, e = c.ListComboMarkets(context.Background(), "", 101); e == nil {
		t.Fatal("accepted oversized page")
	}
}
func TestMarketPositionIDsPresencePreservesLegacyText(t *testing.T) {
	for _, tc := range []struct {
		raw     string
		present bool
		n       int
	}{{`{}`, false, 0}, {`{"positionIds":null}`, false, 0}, {`{"positionIds":[]}`, true, 0}, {`{"positionIds":["999999999999999999999999999999999999999"],"outcomes":"[\"Yes\",\"No\"]","clobTokenIds":"[\"1\",\"2\"]"}`, true, 1}} {
		var m Market
		if e := json.Unmarshal([]byte(tc.raw), &m); e != nil {
			t.Fatal(e)
		}
		if (m.PositionIDs != nil) != tc.present {
			t.Fatal(tc.raw, m)
		}
		if tc.present && len(*m.PositionIDs) != tc.n {
			t.Fatal(m)
		}
		if tc.n == 1 && (m.Outcomes == nil || *m.Outcomes != `["Yes","No"]` || *m.ClobTokenIDs != `["1","2"]`) {
			t.Fatal("legacy text changed", m)
		}
	}
	var m Market
	if json.Unmarshal([]byte(`{"positionIds":[123]}`), &m) == nil {
		t.Fatal("accepted numeric position")
	}
}

type comboDeadlineTransport struct{ t *testing.T }

func (d comboDeadlineTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	deadline, ok := r.Context().Deadline()
	if !ok || time.Until(deadline) > 5*time.Second {
		d.t.Errorf("directory request deadline exceeds five seconds: %v", deadline)
	}
	return nil, errors.New("controlled transport error")
}
func TestComboDirectoryCapsHTTPDeadline(t *testing.T) {
	c, e := NewGammaClient(GammaConfig{HTTPClient: &http.Client{Transport: comboDeadlineTransport{t}}, Timeout: time.Minute})
	if e != nil {
		t.Fatal(e)
	}
	if _, e = c.ListComboMarkets(context.Background(), "", 100); e == nil {
		t.Fatal("transport error hidden")
	}
}
