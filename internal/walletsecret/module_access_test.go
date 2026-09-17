package walletsecret

import (
	"encoding/json"
	"github.com/stretchr/testify/require"
	"github.com/useryege/athena/internal/moduleaccess"
	"net/http/httptest"
	"testing"
)

func TestModuleAccessErrorPreservesJSONDetails(t *testing.T) {
	for _, reason := range []string{moduleaccess.ClosedReason, moduleaccess.UnavailableReason} {
		w := httptest.NewRecorder()
		WriteError(w, moduleaccess.Error(reason, moduleaccess.Worm))
		require.Equal(t, 503, w.Code)
		require.Equal(t, reason, w.Header().Get("X-Athena-Error-Reason"))
		require.Contains(t, w.Header().Get("Cache-Control"), "no-store")
		var result map[string]any
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &result))
		body := result["error"].(map[string]any)
		require.Equal(t, reason, body["reason"])
		require.Equal(t, "worm", body["module_key"])
		details := body["details"].([]any)
		info := details[0].(map[string]any)
		require.Equal(t, "type.googleapis.com/google.rpc.ErrorInfo", info["@type"])
		require.Equal(t, moduleaccess.Domain, info["domain"])
	}
}
