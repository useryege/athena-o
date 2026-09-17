package googleoidc

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/useryege/athena/internal/accountcredentials"
	"github.com/useryege/athena/internal/moduleaccess"
)

func TestModuleAccessGoogleBeginsPreserveAdmissionReason(t *testing.T) {
	const id = "22222222-2222-4222-8222-222222222222"
	for _, reason := range []string{moduleaccess.ClosedReason, moduleaccess.UnavailableReason} {
		h := &Handler{publicOrigin: "http://localhost"}
		auth := func(r *http.Request) (context.Context, accountcredentials.AuthenticatedCredential, error) {
			return r.Context(), accountcredentials.AuthenticatedCredential{AccountID: id, Capability: accountcredentials.CapabilityLogin, JTI: "verified-session", AccessRevision: 1}, nil
		}
		calls := 0
		admit := func(context.Context) error { calls++; return moduleaccess.Error(reason, moduleaccess.Worm) }
		h.wormCredentials = &wormCredentialReauthentication{google: h, authenticate: auth, admit: admit}
		h.wormExecutions = &wormExecutionAuthorization{google: h, authenticate: auth, admit: admit}
		h.wormPositionCashOuts = &wormPositionCashOutAuthorization{google: h, authenticate: auth, admit: admit}
		h.wormPositionCashOutBatches = &wormPositionCashOutBatchAuthorization{google: h, authenticate: auth, admit: admit}
		for _, tc := range []struct {
			name, method string
			call         http.HandlerFunc
		}{
			{"credentials", "GET", h.WormCredentialReauthentication},
			{"execution", "GET", h.WormExecutionAuthorization},
			{"cash_out", "POST", h.WormPositionCashOutAuthorization},
			{"batch", "POST", h.WormPositionCashOutBatchAuthorization},
		} {
			t.Run(reason+"/"+tc.name, func(t *testing.T) {
				calls = 0
				r := httptest.NewRequest(tc.method, "http://localhost/auth?athenaRealm=member&runId="+id+"&cashOutId="+id+"&batchId="+id+"&commandId="+id+"&expectedRevision=1", nil)
				r.Header.Set("Origin", "http://localhost")
				w := httptest.NewRecorder()
				tc.call(w, r)
				require.Equal(t, http.StatusSeeOther, w.Code)
				require.Contains(t, w.Header().Get("Location"), reason)
				require.Equal(t, 1, calls)
			})
		}
	}
}
