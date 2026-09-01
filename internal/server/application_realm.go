package server

import (
	"context"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/useryege/athena/common"
	"github.com/useryege/athena/internal/accountcredentials"
	httputil "github.com/useryege/athena/util/http"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func applicationRealmHeaderMatcher(key string) (string, bool) {
	if strings.EqualFold(key, common.ApplicationRealmHeader) {
		return strings.ToLower(common.ApplicationRealmHeader), true
	}
	return runtime.DefaultHeaderMatcher(key)
}

// adaptApplicationRealmQuery removes the browser-only realm query before the
// request reaches grpc-gateway's protobuf query parser. Header-bearing GETs
// must agree with it. EventSource is the only gateway client allowed to use
// the query as its realm transport because the browser API cannot set headers.
func adaptApplicationRealmQuery(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		query := request.URL.Query()
		if _, present := query[common.ApplicationRealmQueryParameter]; !present {
			next.ServeHTTP(w, request)
			return
		}
		if request.Method != http.MethodGet {
			http.Error(w, "application realm query requires GET", http.StatusUnauthorized)
			return
		}

		realm, err := applicationRealmFromHTTPRequest(request, true)
		if err != nil {
			http.Error(w, "application realm is invalid", http.StatusUnauthorized)
			return
		}
		if len(request.Header.Values(common.ApplicationRealmHeader)) == 0 && !acceptsEventStream(request.Header.Values("Accept")) {
			http.Error(w, "application realm header is required", http.StatusUnauthorized)
			return
		}

		adapted := request.Clone(request.Context())
		adapted.Header = request.Header.Clone()
		adapted.Header.Set(common.ApplicationRealmHeader, string(realm))
		adaptedURL := *request.URL
		query.Del(common.ApplicationRealmQueryParameter)
		adaptedURL.RawQuery = query.Encode()
		adapted.URL = &adaptedURL
		next.ServeHTTP(w, adapted)
	})
}

func acceptsEventStream(values []string) bool {
	for _, value := range values {
		for _, candidate := range strings.Split(value, ",") {
			mediaType := strings.TrimSpace(strings.SplitN(candidate, ";", 2)[0])
			if strings.EqualFold(mediaType, "text/event-stream") {
				return true
			}
		}
	}
	return false
}

func applicationRealmFromIncomingContext(ctx context.Context) (accountcredentials.ApplicationRealm, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return "", applicationRealmRequiredError()
	}
	realm, present, err := parseApplicationRealmValues(md.Get(common.ApplicationRealmHeader))
	if err != nil {
		return "", err
	}
	if !present {
		return "", applicationRealmRequiredError()
	}
	return realm, nil
}

// applicationRealmFromHTTPRequest accepts the realm header on every native
// request. Query transport is reserved for private GET resources that a browser
// renders directly and therefore cannot attach the header itself.
func applicationRealmFromHTTPRequest(request *http.Request, allowQuery bool) (accountcredentials.ApplicationRealm, error) {
	if request == nil {
		return "", applicationRealmRequiredError()
	}
	headerRealm, headerPresent, err := parseApplicationRealmValues(request.Header.Values(common.ApplicationRealmHeader))
	if err != nil {
		return "", err
	}

	var queryRealm accountcredentials.ApplicationRealm
	queryPresent := false
	if allowQuery {
		queryRealm, queryPresent, err = parseApplicationRealmValues(request.URL.Query()[common.ApplicationRealmQueryParameter])
		if err != nil {
			return "", err
		}
	} else if _, present := request.URL.Query()[common.ApplicationRealmQueryParameter]; present {
		return "", status.Error(codes.Unauthenticated, "application realm query is not accepted for this request")
	}

	if headerPresent && queryPresent && headerRealm != queryRealm {
		return "", status.Error(codes.Unauthenticated, "application realm transports do not match")
	}
	if headerPresent {
		return headerRealm, nil
	}
	if queryPresent {
		return queryRealm, nil
	}
	return "", applicationRealmRequiredError()
}

func parseApplicationRealmValues(values []string) (accountcredentials.ApplicationRealm, bool, error) {
	if len(values) == 0 {
		return "", false, nil
	}
	if len(values) != 1 {
		return "", true, status.Error(codes.Unauthenticated, "application realm must be supplied exactly once")
	}
	realm, err := accountcredentials.ParseApplicationRealm(values[0])
	if err != nil {
		return "", true, status.Error(codes.Unauthenticated, "application realm is invalid")
	}
	return realm, true, nil
}

func applicationRealmRequiredError() error {
	return status.Error(codes.Unauthenticated, "application realm is required")
}

func (server *AthenaServer) developmentAccountIDForRealm(realm accountcredentials.ApplicationRealm) (string, error) {
	if server == nil || !server.DisableAuth {
		return "", status.Error(codes.Internal, "development identity selection is unavailable")
	}
	if _, err := accountcredentials.ParseApplicationRealm(string(realm)); err != nil {
		return "", status.Error(codes.Unauthenticated, "application realm is invalid")
	}
	accountID := server.developmentAccountIDs[realm]
	if accountID == "" {
		return "", status.Error(codes.Internal, "development identity is not configured for the application realm")
	}
	return accountID, nil
}

func (server *AthenaServer) developmentAccountIDFromIncomingContext(ctx context.Context) (string, error) {
	realm, err := applicationRealmFromIncomingContext(ctx)
	if err != nil {
		return "", err
	}
	return server.developmentAccountIDForRealm(realm)
}

func (server *AthenaServer) developmentAccountIDFromHTTPRequest(request *http.Request, allowQuery bool) (string, error) {
	realm, err := applicationRealmFromHTTPRequest(request, allowQuery)
	if err != nil {
		return "", err
	}
	return server.developmentAccountIDForRealm(realm)
}

// authenticationContextFromHTTPRequest projects native HTTP authentication
// headers into the same incoming metadata shape used by grpc-gateway. The
// selected realm is validated before either disabled-auth or normal session
// authentication sees the request.
func authenticationContextFromHTTPRequest(request *http.Request, allowRealmQuery bool) (context.Context, error) {
	if request == nil {
		return context.Background(), applicationRealmRequiredError()
	}
	md := metadata.MD{}
	if authorization := request.Header.Get("Authorization"); authorization != "" {
		headerPresent := len(request.Header.Values(common.ApplicationRealmHeader)) > 0
		_, queryPresent := request.URL.Query()[common.ApplicationRealmQueryParameter]
		if headerPresent || queryPresent {
			realm, err := applicationRealmFromHTTPRequest(request, allowRealmQuery)
			if err != nil {
				return request.Context(), err
			}
			md.Set(strings.ToLower(common.ApplicationRealmHeader), string(realm))
		}
		md.Set("authorization", authorization)
		return metadata.NewIncomingContext(request.Context(), md), nil
	}
	realm, err := applicationRealmFromHTTPRequest(request, allowRealmQuery)
	if err != nil {
		return request.Context(), err
	}
	md.Set(strings.ToLower(common.ApplicationRealmHeader), string(realm))
	if cookie := request.Header.Get("Cookie"); cookie != "" {
		md.Set("grpcgateway-cookie", cookie)
	}
	return metadata.NewIncomingContext(request.Context(), md), nil
}

// authenticateRealmLoginCookie validates the exact realm-selected browser
// cookie and then checks that the signed account's persisted role belongs to
// the same realm. A client-controlled header can select a cookie slot but can
// never reinterpret a member token as administrator authority.
func (server *AthenaServer) authenticateRealmLoginCookie(request *http.Request, allowRealmQuery bool) (jwt.Claims, accountcredentials.AuthenticatedCredential, error) {
	realm, err := applicationRealmFromHTTPRequest(request, allowRealmQuery)
	if err != nil {
		return nil, accountcredentials.AuthenticatedCredential{}, err
	}
	cookieName, err := httputil.RealmAuthCookieName(realm)
	if err != nil {
		return nil, accountcredentials.AuthenticatedCredential{}, err
	}
	token, err := httputil.JoinCookies(cookieName, request.Cookies())
	if err != nil || token == "" {
		return nil, accountcredentials.AuthenticatedCredential{}, status.Error(codes.Unauthenticated, "realm login cookie is required")
	}
	claims, credential, err := server.sessionMgr.AuthenticateToken(token)
	if err != nil || credential.Capability != accountcredentials.CapabilityLogin {
		return nil, accountcredentials.AuthenticatedCredential{}, status.Error(codes.Unauthenticated, "realm login cookie is invalid")
	}
	account, err := server.credentialMgr.Get(credential.AccountID)
	if err != nil || account.ApplicationRealm() != realm {
		return nil, accountcredentials.AuthenticatedCredential{}, status.Error(codes.Unauthenticated, "realm login cookie does not match the application realm")
	}
	return claims, credential, nil
}
