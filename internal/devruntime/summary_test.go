package devruntime

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestAccessSummaryKeepsReadFailureSeparateFromReadiness(t *testing.T) {
	for _, code := range []int{200, 401, 500} {
		t.Run(fmt.Sprint(code), func(t *testing.T) {
			calls := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls++
				if r.Method != "GET" || r.URL.Path != "/root/api/v1/admin/module-access-settings" || r.Header.Get("X-Athena-Application-Realm") != "admin" {
					t.Errorf("wrong read request: %s %s", r.Method, r.URL.Path)
				}
				w.WriteHeader(code)
				if code == 200 {
					fmt.Fprint(w, `{"settings":[{"module_key":"market-radar","state":2}]}`)
				}
			}))
			defer server.Close()
			env := map[string]string{"ATHENA_SERVER_ROOTPATH": "/root"}
			result := readAccessSettings(context.Background(), strings.TrimPrefix(server.URL, "http://"), env)
			if calls != 1 || (result.Error == "") != (code == 200) {
				t.Fatal(result, calls)
			}
			if code == 200 && string(result.Settings) == "" {
				t.Fatal("lost closed setting")
			}
		})
	}
}
