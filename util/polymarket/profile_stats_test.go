package polymarket

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestProfileStatsPreserveRawNumbersAndAbsence(t *testing.T) {
	for _, tt := range []struct{ body, want string }{{`{"traded":9007199254740993}`, "9007199254740993"}, {`{"traded":0}`, "0"}, {`{"traded":null}`, ""}, {`{}`, ""}} {
		s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/traded" || r.URL.Query().Get("user") != "0xabc" {
				t.Error(r.URL)
			}
			w.Write([]byte(tt.body))
		}))
		c, _ := NewDataClient(DataConfig{DataBaseURL: s.URL})
		v, e := c.GetTotalMarketsTraded(context.Background(), "0xabc")
		s.Close()
		if e != nil {
			t.Fatal(e)
		}
		if tt.want == "" {
			if v.Traded.Value != nil {
				t.Fatal(v)
			}
		} else if v.Traded.Value == nil || *v.Traded.Value != tt.want {
			t.Fatal(v)
		}
	}
	var v DataUserValue
	if e := json.Unmarshal([]byte(`{"value":-9007199254740993.0100}`), &v); e != nil || v.Value.Value == nil || *v.Value.Value != "-9007199254740993.0100" {
		t.Fatal(v, e)
	}
}

func TestProfileAuxiliaryFieldsAreIndependent(t *testing.T) {
	a := localProfileAdapter(t, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"joinDate":"2024-06-19T21:35:45.083000Z","largestWin":"bad","createdAt":"2000-01-01"}`))
	})
	stats, _, e := a.UserStats(context.Background(), "0x1111111111111111111111111111111111111111")
	if e != nil || stats.JoinDate == nil || *stats.JoinDate != "2024-06-19T21:35:45.083000Z" || stats.LargestWin.Value != nil {
		t.Fatal(stats, e)
	}
	for _, body := range []string{`[{"user":"0x2222222222222222222222222222222222222222","value":0}]`, `[]`, `null`, `{"value":0}`, `[{"user":"0x1111111111111111111111111111111111111111","value":"0"}]`} {
		a := localProfileAdapter(t, func(w http.ResponseWriter, r *http.Request) { w.Write([]byte(body)) })
		if _, _, e := a.PositionValue(context.Background(), "0x1111111111111111111111111111111111111111"); e == nil {
			t.Fatal(body)
		}
	}
}

func TestPositionValueRecordedFixture(t *testing.T) {
	a := localProfileAdapter(t, func(w http.ResponseWriter, r *http.Request) { w.Write(profileFixture(t, "value.json")) })
	value, source, e := a.PositionValue(context.Background(), "0x5c52d767c32cc18100c72d7471209d9618216058")
	if e != nil || value.Value == nil || *value.Value != "0" || source != "https://data-api.polymarket.com/value?user=0x5c52d767c32cc18100c72d7471209d9618216058" {
		t.Fatal(value, source, e)
	}
}
