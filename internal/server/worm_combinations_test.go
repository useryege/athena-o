package server

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestWormCombinationHTTPForwardsSelectionWithoutCatalogPreResolution(t *testing.T) {
	server, client := newCatalogHTTPServer(t)
	mux := http.NewServeMux()
	registerWormCombinationHandlers(mux, server)
	response := httptest.NewRecorder()

	mux.ServeHTTP(response, catalogHTTPRequest("POST", "/api/v1/worm-trading/combinations", `{"name":"Selection","items":[{"eventConditionId":"`+httpCatalogEventID+`","marketConditionId":"`+httpCatalogEventID+`","side":"YES"}]}`))

	require.Equal(t, http.StatusCreated, response.Code, response.Body.String())
	require.Zero(t, client.calls)
	require.Equal(t, 1, client.createCalls)
	require.Equal(t, httpCatalogAccountID, client.createReq.GetOwnerAccountId())
	require.Equal(t, httpCatalogEventID, client.createReq.GetItems()[0].GetEventConditionId())
	require.Equal(t, httpCatalogEventID, client.createReq.GetItems()[0].GetMarketConditionId())
	require.True(t, client.createReq.GetItems()[0].GetIsYes())
}

func TestWormCombinationHTTPPreservesTradingWriteErrorSemantics(t *testing.T) {
	tests := []struct {
		code       codes.Code
		httpStatus int
		message    string
	}{
		{code: codes.FailedPrecondition, httpStatus: http.StatusConflict, message: "Worm market selection is no longer available"},
		{code: codes.Unauthenticated, httpStatus: http.StatusUnauthorized, message: "Worm Trading authentication is required"},
		{code: codes.PermissionDenied, httpStatus: http.StatusForbidden, message: "Worm Trading access denied"},
		{code: codes.Canceled, httpStatus: http.StatusInternalServerError, message: "Worm Trading combination request was canceled"},
		{code: codes.DeadlineExceeded, httpStatus: http.StatusInternalServerError, message: "Worm Trading combination request timed out"},
	}
	for _, method := range []string{http.MethodPost, http.MethodPut} {
		for _, tc := range tests {
			t.Run(method+"/"+tc.code.String(), func(t *testing.T) {
				server, client := newCatalogHTTPServer(t)
				client.mutationErr = status.Error(tc.code, "sensitive dependency detail")
				mux := http.NewServeMux()
				registerWormCombinationHandlers(mux, server)
				path := "/api/v1/worm-trading/combinations"
				body := `{"name":"Selection","items":[{"eventConditionId":"` + httpCatalogEventID + `","marketConditionId":"` + httpCatalogEventID + `","side":"YES"}]}`
				if method == http.MethodPut {
					path += "/965f7c56-5e65-450d-8faf-8bd9b918aeeb"
					body = `{"name":"Selection","expectedRevision":1,"items":[{"eventConditionId":"` + httpCatalogEventID + `","marketConditionId":"` + httpCatalogEventID + `","side":"YES"}]}`
				}
				response := httptest.NewRecorder()

				mux.ServeHTTP(response, catalogHTTPRequest(method, path, body))

				require.Equal(t, tc.httpStatus, response.Code, response.Body.String())
				require.Contains(t, response.Body.String(), fmt.Sprintf(`"code":%d`, tc.code))
				require.Contains(t, response.Body.String(), `"message":"`+tc.message+`"`)
				require.NotContains(t, response.Body.String(), "sensitive dependency detail")
			})
		}
	}
}

func TestWormCombinationHTTPUpdateForwardsSelectionWithoutCatalogPreResolution(t *testing.T) {
	server, client := newCatalogHTTPServer(t)
	mux := http.NewServeMux()
	registerWormCombinationHandlers(mux, server)
	response := httptest.NewRecorder()
	combinationID := "965f7c56-5e65-450d-8faf-8bd9b918aeeb"

	mux.ServeHTTP(response, catalogHTTPRequest("PUT", "/api/v1/worm-trading/combinations/"+combinationID, `{"name":"Selection","expectedRevision":1,"items":[{"eventConditionId":"`+httpCatalogEventID+`","marketConditionId":"`+httpCatalogEventID+`","side":"NO"}]}`))

	require.Equal(t, http.StatusOK, response.Code, response.Body.String())
	require.Zero(t, client.calls)
	require.Equal(t, 1, client.updateCalls)
	require.Equal(t, httpCatalogAccountID, client.updateReq.GetOwnerAccountId())
	require.Equal(t, combinationID, client.updateReq.GetId())
	require.Equal(t, int64(1), client.updateReq.GetExpectedRevision())
	require.False(t, client.updateReq.GetItems()[0].GetIsYes())
}

func TestWormCombinationHTTPRejectsSelectionSyntaxBeforeTrading(t *testing.T) {
	for _, body := range []string{
		`{"name":"Empty","items":[]}`,
		`{"name":"Bad ID","items":[{"eventConditionId":"bad","marketConditionId":"` + httpCatalogEventID + `","side":"YES"}]}`,
		`{"name":"Bad side","items":[{"eventConditionId":"` + httpCatalogEventID + `","marketConditionId":"` + httpCatalogEventID + `","side":"MAYBE"}]}`,
		`{"name":"Duplicate","items":[{"eventConditionId":"` + httpCatalogEventID + `","marketConditionId":"` + httpCatalogEventID + `","side":"YES"},{"eventConditionId":"` + httpCatalogEventID + `","marketConditionId":"` + httpCatalogEventID + `","side":"NO"}]}`,
	} {
		server, client := newCatalogHTTPServer(t)
		mux := http.NewServeMux()
		registerWormCombinationHandlers(mux, server)
		response := httptest.NewRecorder()

		mux.ServeHTTP(response, catalogHTTPRequest("POST", "/api/v1/worm-trading/combinations", body))

		require.Equal(t, http.StatusBadRequest, response.Code, response.Body.String())
		require.Zero(t, client.calls)
		require.Zero(t, client.createCalls)
	}
}
