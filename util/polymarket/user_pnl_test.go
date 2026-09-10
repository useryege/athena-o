package polymarket

import (
	"context"
	"net/http"
	"testing"
)

func TestUserPNLRawQueryAndPrecision(t *testing.T) {
	a := localProfileAdapter(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/user-pnl" || r.URL.Query().Get("user_address") != "0xabc" || r.URL.Query().Get("interval") != "all" || r.URL.Query().Get("fidelity") != "1d" {
			t.Error(r.URL)
		}
		w.Write([]byte(`[{"t":1,"p":-9007199254740993.0100},{"t":1,"p":0}]`))
	})
	p, _, e := a.UserPNL(context.Background(), "0xabc", "all", "1d")
	if e != nil || len(p) != 2 || *p[0].P.Value != "-9007199254740993.0100" || p[1].T != 1 {
		t.Fatal(p, e)
	}
}

func TestUserPNLRejectsMalformedSeries(t *testing.T) {
	for _, body := range []string{`null`, `{}`, `[]`, `[{"t":1,"p":0}]`, `[{"t":1,"p":null},{"t":2,"p":0}]`, `[{"p":0},{"t":2,"p":0}]`, `[{"t":2,"p":0},{"t":1,"p":0}]`, `[{"t":1.2,"p":0},{"t":2,"p":0}]`, `[{"t":1,"p":"0"},{"t":2,"p":0}]`} {
		a := localProfileAdapter(t, func(w http.ResponseWriter, r *http.Request) { w.Write([]byte(body)) })
		if _, _, e := a.UserPNL(context.Background(), "0xabc", "all", "1d"); e == nil {
			t.Fatal(body)
		}
	}
}
