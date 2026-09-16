package server

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/useryege/athena/common"
	"github.com/useryege/athena/internal/accountaccess"
	"github.com/useryege/athena/internal/accountcredentials"
	trading "github.com/useryege/athena/internal/wormtrading/apiclient"
	httputil "github.com/useryege/athena/util/http"
	utilsession "github.com/useryege/athena/util/session"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

const httpCatalogAccountID = "d30fa35d-78a6-43d6-8faf-ed5b1b9e63d1"
const httpCatalogEventID = "So11111111111111111111111111111111111111112"

type catalogHTTPClientset struct {
	trading.Clientset
	client trading.WormTradingServiceClient
}

func (c catalogHTTPClientset) WormTrading() trading.WormTradingServiceClient { return c.client }

type catalogHTTPClient struct {
	trading.WormTradingServiceClient
	calls       int
	createCalls int
	createReq   *trading.CreateMarketCombinationRequest
	updateCalls int
	updateReq   *trading.UpdateMarketCombinationRequest
	mutationErr error
	err         error
	t           *testing.T
}

func (c *catalogHTTPClient) UpdateMarketCombination(ctx context.Context, req *trading.UpdateMarketCombinationRequest, _ ...grpc.CallOption) (*trading.UpdateMarketCombinationResponse, error) {
	c.updateCalls++
	c.updateReq = req
	md, _ := metadata.FromOutgoingContext(ctx)
	require.Equal(c.t, []string{httpCatalogAccountID}, md.Get("x-athena-account-id"))
	if c.mutationErr != nil {
		return nil, c.mutationErr
	}
	item := req.GetItems()[0]
	return &trading.UpdateMarketCombinationResponse{Combination: &trading.MarketCombination{
		Id: req.GetId(), OwnerAccountId: req.GetOwnerAccountId(), Name: req.GetName(), Revision: req.GetExpectedRevision() + 1, CreatedAt: 1, UpdatedAt: 2,
		Items: []*trading.MarketCombinationItem{{Ordinal: 1, EventConditionId: item.GetEventConditionId(), EventTitle: "Trusted event", MarketConditionId: item.GetMarketConditionId(), MarketTitle: "Trusted market", IsYes: item.GetIsYes(), OutcomeLabel: "YES"}},
	}}, nil
}

func (c *catalogHTTPClient) CreateMarketCombination(ctx context.Context, req *trading.CreateMarketCombinationRequest, _ ...grpc.CallOption) (*trading.CreateMarketCombinationResponse, error) {
	c.createCalls++
	c.createReq = req
	md, _ := metadata.FromOutgoingContext(ctx)
	require.Equal(c.t, []string{httpCatalogAccountID}, md.Get("x-athena-account-id"))
	if c.mutationErr != nil {
		return nil, c.mutationErr
	}
	item := req.GetItems()[0]
	return &trading.CreateMarketCombinationResponse{Combination: &trading.MarketCombination{
		Id: "965f7c56-5e65-450d-8faf-8bd9b918aeeb", OwnerAccountId: req.GetOwnerAccountId(), Name: req.GetName(), Revision: 1, CreatedAt: 1, UpdatedAt: 1,
		Items: []*trading.MarketCombinationItem{{Ordinal: 1, EventConditionId: item.GetEventConditionId(), EventTitle: "Trusted event", MarketConditionId: item.GetMarketConditionId(), MarketTitle: "Trusted market", IsYes: item.GetIsYes(), OutcomeLabel: "YES"}},
	}}, nil
}

func (c *catalogHTTPClient) GetOrderEventCatalog(ctx context.Context, req *trading.GetOrderEventCatalogRequest, _ ...grpc.CallOption) (*trading.GetOrderEventCatalogResponse, error) {
	c.calls++
	if c.err != nil {
		return nil, c.err
	}
	md, _ := metadata.FromOutgoingContext(ctx)
	require.Equal(c.t, []string{httpCatalogAccountID}, md.Get("x-athena-account-id"))
	require.Equal(c.t, []string{"keep"}, md.Get("trace-test"))
	require.Equal(c.t, httpCatalogEventID, req.EventConditionId)
	return &trading.GetOrderEventCatalogResponse{Event: &trading.OrderEventCatalog{EventConditionId: httpCatalogEventID, Title: "Catalog event"}, FetchedAt: 123}, nil
}
func newCatalogHTTPServer(t *testing.T) (*AthenaServer, *catalogHTTPClient) {
	t.Helper()
	a := accountaccess.Access{LoginEnabled: true, APIKeyEnabled: true, Revision: 1, Modules: accountaccess.NoModuleAccess()}
	a.Modules[accountaccess.ModuleWormTrading] = accountaccess.AccessLevelReadWrite
	controller, err := accountaccess.NewController(context.Background(), traderAuthStore{httpCatalogAccountID: a})
	require.NoError(t, err)
	client := &catalogHTTPClient{t: t}
	return &AthenaServer{AthenaServerOpts: AthenaServerOpts{DisableAuth: true, WormTradingClientset: catalogHTTPClientset{client: client}}, accessController: controller, developmentAccountIDs: map[accountcredentials.ApplicationRealm]string{accountcredentials.ApplicationRealmMember: httpCatalogAccountID}, walletSecretPublicOrigin: "http://localhost"}, client
}
func catalogHTTPRequest(method, path, body string) *http.Request {
	req := httptest.NewRequest(method, "http://localhost"+path, strings.NewReader(body))
	req.RemoteAddr = "127.0.0.1:3456"
	req.Header.Set(common.ApplicationRealmHeader, string(accountcredentials.ApplicationRealmMember))
	req.Header.Set("x-athena-account-id", "forged-browser-account")
	req.Header.Set("Origin", developmentWormPublicOrigin)
	req.Header.Set("Content-Type", "application/json")
	return req.WithContext(metadata.NewOutgoingContext(req.Context(), metadata.Pairs("x-athena-account-id", "forged-internal", "x-athena-account-id", "duplicate", "trace-test", "keep")))
}
func TestWormCatalogHTTPUsesTrustedAccountIdentity(t *testing.T) {
	s, c := newCatalogHTTPServer(t)
	mux := http.NewServeMux()
	registerWormCombinationHandlers(mux, s)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, catalogHTTPRequest("GET", "/api/v1/worm-trading/events/"+httpCatalogEventID, ""))
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	require.JSONEq(t, `{"eventConditionId":"`+httpCatalogEventID+`","title":"Catalog event","logo":"","markets":[],"fetchedAt":123}`, w.Body.String())
	require.Equal(t, 1, c.calls)
}
func TestWormCatalogHTTPRejectsAPIKey(t *testing.T) {
	s, c := newCatalogHTTPServer(t)
	s.DisableAuth = false
	mux := http.NewServeMux()
	registerWormCombinationHandlers(mux, s)
	req := catalogHTTPRequest("GET", "/api/v1/worm-trading/events/"+httpCatalogEventID, "")
	req.Header.Set("Authorization", "Bearer api-key")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	require.Equal(t, http.StatusUnauthorized, w.Code, w.Body.String())
	require.Zero(t, c.calls)
}
func TestWormCatalogHTTPPreservesRevocationAndCancellation(t *testing.T) {
	for _, tc := range []struct {
		code     codes.Code
		httpCode int
	}{{codes.PermissionDenied, http.StatusForbidden}, {codes.Unauthenticated, http.StatusUnauthorized}, {codes.Canceled, http.StatusInternalServerError}, {codes.DeadlineExceeded, http.StatusInternalServerError}} {
		t.Run(tc.code.String(), func(t *testing.T) {
			s, c := newCatalogHTTPServer(t)
			c.err = status.Error(tc.code, "current access rejected")
			mux := http.NewServeMux()
			registerWormCombinationHandlers(mux, s)
			w := httptest.NewRecorder()
			mux.ServeHTTP(w, catalogHTTPRequest("GET", "/api/v1/worm-trading/events/"+httpCatalogEventID, ""))
			require.Equal(t, tc.httpCode, w.Code, w.Body.String())
			require.Contains(t, w.Body.String(), fmt.Sprintf(`"code":%d`, tc.code))
		})
	}
}

type catalogCredentialStore struct{ accountcredentials.Store }

func (catalogCredentialStore) ListCredentialAccounts(context.Context) (map[string]accountcredentials.Account, error) {
	return map[string]accountcredentials.Account{httpCatalogAccountID: {ID: httpCatalogAccountID, Username: "catalog-user", IdentityProvider: accountcredentials.IdentityProviderGoogle, IdentitySubject: "catalog-google", VerifiedEmail: "catalog@example.test"}}, nil
}
func (catalogCredentialStore) CreateAPIKeyMetadata(context.Context, string, accountcredentials.Token) error {
	return nil
}

type catalogRevocations struct{ utilsession.UserStateStorage }

func (catalogRevocations) IsTokenRevoked(string) bool { return false }
func TestWormCatalogHTTPRequiresSignedLoginCookie(t *testing.T) {
	s, c := newCatalogHTTPServer(t)
	s.DisableAuth = false
	codec, err := accountcredentials.NewJWTCodec([]byte(strings.Repeat("j", 32)))
	require.NoError(t, err)
	credentials, err := accountcredentials.NewCredentialManager(context.Background(), catalogCredentialStore{}, codec)
	require.NoError(t, err)
	s.credentialMgr = credentials
	s.sessionMgr = utilsession.NewSessionManager(credentials, codec, catalogRevocations{}, s.accessController)
	login, err := credentials.IssueLoginSession(httpCatalogAccountID, accountcredentials.ApplicationRealmMember, accountcredentials.IdentityProviderGoogle, "catalog-google", "catalog-login", 3600)
	require.NoError(t, err)
	key, err := credentials.IssueAPIKey(context.Background(), httpCatalogAccountID, "catalog-api-key", 3600)
	require.NoError(t, err)
	_, parsedKey, err := s.sessionMgr.AuthenticateToken(key)
	require.NoError(t, err)
	require.Equal(t, accountcredentials.CapabilityAPIKey, parsedKey.Capability)
	name, err := httputil.RealmAuthCookieName(accountcredentials.ApplicationRealmMember)
	require.NoError(t, err)
	mux := http.NewServeMux()
	registerWormCombinationHandlers(mux, s)
	for _, tc := range []struct {
		name, token string
		want        int
	}{{"signed login", login, http.StatusOK}, {"API key in cookie", key, http.StatusUnauthorized}} {
		t.Run(tc.name, func(t *testing.T) {
			c.t = t
			req := catalogHTTPRequest("GET", "/api/v1/worm-trading/events/"+httpCatalogEventID, "")
			req.AddCookie(&http.Cookie{Name: name, Value: tc.token})
			req.Header.Set("Authorization", "Bearer "+key)
			w := httptest.NewRecorder()
			before := c.calls
			mux.ServeHTTP(w, req)
			require.Equal(t, tc.want, w.Code, w.Body.String())
			if tc.want == http.StatusOK {
				require.Equal(t, before+1, c.calls)
			} else {
				require.Equal(t, before, c.calls)
			}
		})
	}
}
