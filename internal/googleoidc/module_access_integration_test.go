//go:build integration

package googleoidc

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
	"github.com/useryege/athena/internal/accountcredentials"
	"github.com/useryege/athena/internal/moduleaccess"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"testing"
	"time"
)

// Use an explicitly supplied, isolated Redis. Real store.consume must delete
// trusted state before admission failure; no OIDC exchange or lease is issued.
func TestModuleAccessGooglePurposesConsumeStateAndPreserveReason(t *testing.T) {
	addr := os.Getenv("ATHENA_TEST_REDIS_ADDR")
	if addr == "" {
		t.Skip("ATHENA_TEST_REDIS_ADDR required")
	}
	client := redis.NewClient(&redis.Options{Addr: addr})
	defer client.Close()
	ctx := context.Background()
	require.NoError(t, client.Ping(ctx).Err())
	const id = "22222222-2222-4222-8222-222222222222"
	digest := sha256.Sum256([]byte("verified-session"))
	digestHex := hex.EncodeToString(digest[:])
	for _, reason := range []string{moduleaccess.ClosedReason, moduleaccess.UnavailableReason} {
		for _, purpose := range []string{"credentials", "execution", "cash_out", "batch"} {
			t.Run(reason+"/"+purpose, func(t *testing.T) {
				h := &Handler{baseHRef: "/athena"}
				calls := 0
				authenticate := func(r *http.Request) (context.Context, accountcredentials.AuthenticatedCredential, error) {
					calls++
					return r.Context(), accountcredentials.AuthenticatedCredential{}, moduleaccess.Error(reason, moduleaccess.Worm)
				}
				var state, key, cookie string
				var value any
				var err error
				switch purpose {
				case "credentials":
					state, err = newWormCredentialState()
					key = wormCredentialTransactionKey(state)
					cookie = wormCredentialStateCookieName
					value = wormCredentialTransaction{Nonce: "nonce", Verifier: "verifier", ReturnTo: "/worm-trading", AccountID: id, SessionJTIDigest: digestHex, AccessRevision: 1, CreatedAt: time.Now()}
					h.wormCredentials = &wormCredentialReauthentication{google: h, store: &wormCredentialTransactionStore{redis: client}, authenticate: authenticate}
				case "execution":
					state, err = newWormExecutionState()
					key = wormExecutionTransactionKey(state)
					cookie = wormExecutionStateCookieName
					value = wormExecutionTransaction{Nonce: "nonce", Verifier: "verifier", ReturnTo: validateWormExecutionReturnTo("", id), RunID: id, CommandID: id, ExpectedRevision: 1, AccountID: id, SessionJTIDigestSHA256: digestHex, AccessRevision: 1, CreatedAt: time.Now()}
					h.wormExecutions = &wormExecutionAuthorization{google: h, store: &wormExecutionTransactionStore{redis: client}, authenticate: authenticate}
				case "cash_out":
					state, err = newWormPositionCashOutState()
					key = wormPositionCashOutTransactionKey(state)
					cookie = wormPositionCashOutStateCookieName
					value = wormPositionCashOutTransaction{Nonce: "nonce", Verifier: "verifier", ReturnTo: "/worm-trading", CashOutID: id, CommandID: id, ExpectedRevision: 1, AccountID: id, SessionJTIDigestSHA256: digestHex, IntentDigestSHA256: digestHex, AccessRevision: 1, CreatedAt: time.Now()}
					h.wormPositionCashOuts = &wormPositionCashOutAuthorization{google: h, store: &wormPositionCashOutTransactionStore{redis: client}, authenticate: authenticate}
				case "batch":
					state, err = newWormPositionCashOutBatchState()
					key = wormPositionCashOutBatchTransactionKey(state)
					cookie = wormPositionCashOutBatchStateCookieName
					value = wormPositionCashOutBatchTransaction{Nonce: "nonce", Verifier: "verifier", ReturnTo: "/worm-trading", BatchID: id, CommandID: id, ExpectedRevision: 1, AccountID: id, SessionJTIDigestSHA256: digestHex, IntentDigestSHA256: digestHex, AccessRevision: 1, CreatedAt: time.Now()}
					h.wormPositionCashOutBatches = &wormPositionCashOutBatchAuthorization{google: h, store: &wormPositionCashOutBatchTransactionStore{redis: client}, authenticate: authenticate}
				}
				require.NoError(t, err)
				raw, err := json.Marshal(value)
				require.NoError(t, err)
				require.NoError(t, client.Set(ctx, key, raw, time.Minute).Err())
				t.Cleanup(func() { client.Del(ctx, key) })
				request := func() *http.Request {
					r := httptest.NewRequest("GET", "http://localhost/athena/auth/google/callback?state="+url.QueryEscape(state)+"&code=proof", nil)
					r.AddCookie(&http.Cookie{Name: cookie, Value: state})
					return r
				}
				w := httptest.NewRecorder()
				h.Callback(w, request())
				require.Equal(t, 303, w.Code)
				require.Contains(t, w.Header().Get("Location"), reason)
				require.Contains(t, w.Header().Get("Location"), "/athena/worm-trading")
				require.Equal(t, 1, calls)
				remaining, err := client.Exists(ctx, key).Result()
				require.NoError(t, err)
				require.Zero(t, remaining)
				w = httptest.NewRecorder()
				h.Callback(w, request())
				require.Equal(t, 1, calls, "consumed state cannot be restored on reopening")
				require.NotContains(t, w.Header().Get("Location"), reason)
			})
		}
	}
}
