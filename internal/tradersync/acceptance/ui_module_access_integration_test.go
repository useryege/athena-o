//go:build integration && uiharness

package acceptance

import (
	"encoding/json"
	"github.com/stretchr/testify/require"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"
)

// Exercise actual cookie authentication, HTTP/gRPC registration, admission and
// PostgreSQL settings. The harness only explicitly opens Trader Sync.
func TestUIHarnessModuleAccessContract(t *testing.T) {
	t.Setenv("ATHENA_UI_E2E_DIR", t.TempDir())
	h := newHarness(t, 0, 0)
	info := h.StartUI(t, os.Getenv("ATHENA_UI_DIST"))
	request := func(stateFile, realm, method, path, body string) (int, http.Header, []byte) {
		raw, err := os.ReadFile(stateFile)
		require.NoError(t, err)
		var state struct {
			Cookies []struct{ Name, Value string }
		}
		require.NoError(t, json.Unmarshal(raw, &state))
		req, err := http.NewRequest(method, info.BaseURL+info.PathPrefix+path, strings.NewReader(body))
		require.NoError(t, err)
		req.Header.Set("X-Athena-Application-Realm", realm)
		req.Header.Set("Content-Type", "application/json")
		for _, cookie := range state.Cookies {
			req.AddCookie(&http.Cookie{Name: cookie.Name, Value: cookie.Value})
		}
		response, err := (&http.Client{Timeout: 5 * time.Second}).Do(req)
		require.NoError(t, err)
		defer response.Body.Close()
		data, err := io.ReadAll(response.Body)
		require.NoError(t, err)
		return response.StatusCode, response.Header, data
	}
	code, _, data := request(info.MemberAState, "member", "GET", "/api/v1/module-access-states", "")
	require.Equal(t, 200, code, string(data))
	var states struct {
		States []struct {
			ModuleKey string `json:"module_key"`
			State     int
		}
	}
	require.NoError(t, json.Unmarshal(data, &states))
	require.Len(t, states.States, 6)
	for _, row := range states.States {
		expected := 2
		if row.ModuleKey == "trader_sync" {
			expected = 1
		}
		require.Equal(t, expected, row.State, row.ModuleKey)
	}
	code, _, data = request(info.AdminState, "admin", "GET", "/api/v1/admin/module-access-settings", "")
	require.Equal(t, 200, code, string(data))
	require.Contains(t, string(data), `"updated_by_username":"admin"`)
	code, _, data = request(info.AdminState, "admin", "PUT", "/api/v1/admin/module-access-settings/trader_sync", `{"state":2}`)
	require.Equal(t, 200, code, string(data))
	code, headers, data := request(info.MemberAState, "member", "GET", "/api/v1/trader-sync/subscriptions", "")
	require.Equal(t, 503, code, string(data))
	require.Equal(t, "MODULE_ACCESS_CLOSED", headers.Get("X-Athena-Error-Reason"))
	require.Contains(t, headers.Get("Cache-Control"), "no-store")
	require.Contains(t, string(data), `"module_key":"trader_sync"`)
	require.Contains(t, string(data), `athena.module_access`)
	code, _, data = request(info.AdminState, "admin", "PUT", "/api/v1/admin/module-access-settings/trader_sync", `{"state":1}`)
	require.Equal(t, 200, code, string(data))
	code, _, data = request(info.MemberAState, "member", "GET", "/api/v1/trader-sync/subscriptions", "")
	require.Equal(t, 200, code, string(data))
}
