//go:build integration

package phantomauth

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
	"github.com/useryege/athena/internal/accountcredentials"
	"github.com/useryege/athena/internal/authregistration"
	"github.com/useryege/athena/internal/moduleaccess"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
)

func TestModuleAccessPhantomVerifyConsumesChallenge(t *testing.T) {
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
		for _, family := range []string{"credentials", "executions", "position-cash-outs", "position-cash-out-batches"} {
			for _, binding := range []string{"bound", "account", "jti", "revision"} {
				t.Run(reason+family+"/"+binding, func(t *testing.T) {
					h := &Handler{publicOrigin: "http://localhost"}
					calls := 0
					admitCalls := 0
					admit := func(context.Context) error { admitCalls++; return moduleaccess.Error(reason, moduleaccess.Worm) }
					credential := accountcredentials.AuthenticatedCredential{AccountID: id, Capability: accountcredentials.CapabilityLogin, JTI: "verified-session", AccessRevision: 1}
					switch binding {
					case "account":
						credential.AccountID = "33333333-3333-4333-8333-333333333333"
					case "jti":
						credential.JTI = "different-session"
					case "revision":
						credential.AccessRevision = 2
					}
					expected := reason
					if binding != "bound" {
						expected = map[string]string{"credentials": "WORM_TRADING_REAUTH_REQUIRED", "executions": WormExecutionAuthorizationRequiredReason, "position-cash-outs": WormPositionCashOutAuthorizationRequiredReason, "position-cash-out-batches": WormPositionCashOutBatchAuthorizationRequiredReason}[family]
					}
					auth := func(r *http.Request) (context.Context, accountcredentials.AuthenticatedCredential, error) {
						calls++
						return r.Context(), credential, nil
					}
					opaque, err := authregistration.RandomOpaqueValue()
					require.NoError(t, err)
					now := time.Now().UTC()
					value := map[string]any{"address": "So11111111111111111111111111111111111111112", "message": "bound SIWS proof", "nonce": strings.Repeat("ab", 16), "returnTo": "/worm-trading", "runId": id, "cashOutId": id, "batchId": id, "commandId": id, "expectedRevision": 1, "accountId": id, "sessionJtiDigest": digestHex, "sessionJtiDigestSha256": digestHex, "planDigestSha256": strings.Repeat("ab", 32), "intentDigestSha256": strings.Repeat("ab", 32), "accessRevision": 1, "createdAt": now, "expiresAt": now.Add(5 * time.Minute)}
					var cookie, key, path string
					var call http.HandlerFunc
					switch family {
					case "credentials":
						cookie = wormCredentialChallengeCookieName
						key = wormCredentialChallengeKey(opaque)
						path = "/auth/worm-trading/solana/verify"
						h.wormCredentials = &wormCredentialReauthentication{phantom: h, authenticate: auth, admit: admit, store: &wormCredentialChallengeStore{redis: client}}
						call = h.WormCredentialVerify
					case "executions":
						cookie = wormExecutionChallengeCookieName
						key = wormExecutionChallengeKey(opaque)
						value["returnTo"] = validateWormExecutionReturnTo("", id)
						h.wormExecutions = &wormExecutionAuthorization{phantom: h, authenticate: auth, admit: admit, store: &wormExecutionChallengeStore{redis: client}}
						call = h.WormExecutionVerify
					case "position-cash-outs":
						cookie = wormPositionCashOutChallengeCookieName
						key = wormPositionCashOutChallengeKey(opaque)
						h.wormPositionCashOuts = &wormPositionCashOutAuthorization{phantom: h, authenticate: auth, admit: admit, store: &wormPositionCashOutChallengeStore{redis: client}}
						call = h.WormPositionCashOutVerify
					case "position-cash-out-batches":
						cookie = wormPositionCashOutBatchChallengeCookieName
						key = wormPositionCashOutBatchChallengeKey(opaque)
						h.wormPositionCashOutBatches = &wormPositionCashOutBatchAuthorization{phantom: h, authenticate: auth, admit: admit, store: &wormPositionCashOutBatchChallengeStore{redis: client}}
						call = h.WormPositionCashOutBatchVerify
					}
					if path == "" {
						path = "/auth/worm-trading/" + family + "/" + id + "/solana/verify"
					}
					raw, err := json.Marshal(value)
					require.NoError(t, err)
					require.NoError(t, client.Set(ctx, key, raw, time.Minute).Err())
					t.Cleanup(func() { client.Del(ctx, key) })
					request := func() *http.Request {
						r := httptest.NewRequest("POST", "http://localhost"+path, strings.NewReader(`{}`))
						r.Header.Set("Origin", "http://localhost")
						r.Header.Set("Content-Type", "application/json")
						r.AddCookie(&http.Cookie{Name: cookie, Value: opaque})
						return r
					}
					w := httptest.NewRecorder()
					call(w, request())
					expectedStatus := 503
					if binding != "bound" {
						expectedStatus = 401
					}
					require.Equal(t, expectedStatus, w.Code, w.Body.String())
					require.Equal(t, expected, w.Header().Get("X-Athena-Error-Reason"))
					require.Equal(t, 1, calls)
					expectedAdmissions := 0
					if binding == "bound" {
						expectedAdmissions = 1
					}
					require.Equal(t, expectedAdmissions, admitCalls)
					remaining, err := client.Exists(ctx, key).Result()
					require.NoError(t, err)
					require.Zero(t, remaining)
					call(httptest.NewRecorder(), request())
					require.Equal(t, 1, calls, "consumed challenge cannot be reused")
					require.Equal(t, expectedAdmissions, admitCalls, "replay cannot reach admission")
				})
			}
		}
	}
}
