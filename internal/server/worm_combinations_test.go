package server

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
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
