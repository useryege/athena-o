package phantomauth

import (
	"context"
	"github.com/stretchr/testify/require"
	"github.com/useryege/athena/internal/accountcredentials"
	"github.com/useryege/athena/internal/moduleaccess"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestModuleAccessPhantomChallengesPreserveAdmissionReason(t *testing.T) {
	const id = "22222222-2222-4222-8222-222222222222"
	for _, reason := range []string{moduleaccess.ClosedReason, moduleaccess.UnavailableReason} {
		h := &Handler{publicOrigin: "http://localhost"}
		auth := func(r *http.Request) (context.Context, accountcredentials.AuthenticatedCredential, error) {
			return r.Context(), accountcredentials.AuthenticatedCredential{AccountID: id, Capability: accountcredentials.CapabilityLogin, JTI: "verified-session", AccessRevision: 1}, nil
		}
		admit := func(context.Context) error { return moduleaccess.Error(reason, moduleaccess.Worm) }
		h.wormCredentials = &wormCredentialReauthentication{phantom: h, authenticate: auth, admit: admit}
		h.wormExecutions = &wormExecutionAuthorization{phantom: h, authenticate: auth, admit: admit}
		h.wormPositionCashOuts = &wormPositionCashOutAuthorization{phantom: h, authenticate: auth, admit: admit}
		h.wormPositionCashOutBatches = &wormPositionCashOutBatchAuthorization{phantom: h, authenticate: auth, admit: admit}
		for _, tc := range []struct {
			path string
			call http.HandlerFunc
		}{
			{"/auth/worm-trading/solana/challenge", h.WormCredentialChallenge},
			{"/auth/worm-trading/executions/" + id + "/solana/challenge", h.WormExecutionChallenge},
			{"/auth/worm-trading/position-cash-outs/" + id + "/solana/challenge", h.WormPositionCashOutChallenge},
			{"/auth/worm-trading/position-cash-out-batches/" + id + "/solana/challenge", h.WormPositionCashOutBatchChallenge},
		} {
			t.Run(reason+tc.path, func(t *testing.T) {
				r := httptest.NewRequest("POST", "http://localhost"+tc.path, strings.NewReader(`{"commandId":"`+id+`","expectedRevision":1}`))
				r.Header.Set("Origin", "http://localhost")
				r.Header.Set("Content-Type", "application/json")
				// The credentials challenge does not accept operation fields.
				if tc.path == "/auth/worm-trading/solana/challenge" {
					r = httptest.NewRequest("POST", "http://localhost"+tc.path, strings.NewReader(`{}`))
					r.Header.Set("Origin", "http://localhost")
					r.Header.Set("Content-Type", "application/json")
				}
				w := httptest.NewRecorder()
				tc.call(w, r)
				require.Equal(t, 503, w.Code, w.Body.String())
				require.Equal(t, reason, w.Header().Get("X-Athena-Error-Reason"))
				require.Contains(t, w.Body.String(), "athena.module_access")
			})
		}
	}
}
