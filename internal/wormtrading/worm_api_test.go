package wormtrading

import (
	"context"
	"errors"
	"io"
	"net/http"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/useryege/athena/util/ratelimit"
	utilworm "github.com/useryege/athena/util/worm"
)

type countingWormLimiter struct {
	mu    sync.Mutex
	calls int
	err   error
}

func (l *countingWormLimiter) Wait(context.Context) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.calls++
	return l.err
}

func (l *countingWormLimiter) count() int {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.calls
}

var _ ratelimit.Limiter = (*countingWormLimiter)(nil)

type wormRoundTripFunc func(*http.Request) (*http.Response, error)

func (f wormRoundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return f(request)
}

func TestWormAPIClientFactorySharesCatalogAndEstimateLimiter(t *testing.T) {
	waitErr := errors.New("stop before HTTP")
	unauthenticatedLimiter := &countingWormLimiter{err: waitErr}
	authenticatedLimiter := &countingWormLimiter{err: waitErr}
	factory := &officialWormAPIClientFactory{
		attemptTimeout:             time.Second,
		unauthenticatedRateLimiter: unauthenticatedLimiter,
		authenticatedRateLimiter:   authenticatedLimiter,
	}

	catalogClient, err := factory.NewCatalogClient()
	require.NoError(t, err)
	estimateClient, err := factory.NewUnauthenticatedClient()
	require.NoError(t, err)
	authenticatedClient, err := factory.NewAuthenticatedClient("key", "secret")
	require.NoError(t, err)

	_, err = catalogClient.GetEvent(context.Background(), catalogConditionID(1))
	require.ErrorIs(t, err, waitErr)
	_, err = estimateClient.EstimateMarginPosition(context.Background(), utilworm.EstimateMarginPositionOptions{})
	require.ErrorIs(t, err, waitErr)
	_, err = authenticatedClient.RevokeAPIKey(context.Background(), "key-id")
	require.ErrorIs(t, err, waitErr)

	require.Equal(t, 2, unauthenticatedLimiter.count())
	require.Equal(t, 1, authenticatedLimiter.count())
}

func TestWormCatalogClientExposesOnlyReadMethods(t *testing.T) {
	typeOfClient := reflect.TypeOf((*WormCatalogClient)(nil)).Elem()
	require.Equal(t, 2, typeOfClient.NumMethod())
	require.Equal(t, "GetEvent", typeOfClient.Method(0).Name)
	require.Equal(t, "GetMarket", typeOfClient.Method(1).Name)

	typeOfFactory := reflect.TypeOf(NewOfficialWormAPIClientFactory)
	require.Equal(t, 1, typeOfFactory.NumIn(), "production factory must not accept a base URL")
	require.Equal(t, reflect.TypeOf(time.Duration(0)), typeOfFactory.In(0))
}

func TestWormCatalogClientUsesOnlyReadHTTPRoutes(t *testing.T) {
	eventID := catalogConditionID(10)
	marketID := catalogConditionID(11)
	type requestRecord struct {
		method string
		url    string
	}
	var requests []requestRecord
	originalTransport := http.DefaultTransport
	http.DefaultTransport = wormRoundTripFunc(func(request *http.Request) (*http.Response, error) {
		requests = append(requests, requestRecord{method: request.Method, url: request.URL.String()})
		body := ""
		switch request.URL.Path {
		case "/events/" + eventID + "/":
			body = `{"data":{"condition_id":"` + eventID + `"}}`
		case "/markets/" + marketID + "/":
			body = `{"data":{"condition_id":"` + marketID + `"}}`
		default:
			return &http.Response{
				StatusCode: http.StatusNotFound,
				Status:     "404 Not Found",
				Header:     make(http.Header),
				Body:       io.NopCloser(strings.NewReader(`{"error":{"message":"unexpected route"}}`)),
			}, nil
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Status:     "200 OK",
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader(body)),
		}, nil
	})
	t.Cleanup(func() { http.DefaultTransport = originalTransport })

	factory := &officialWormAPIClientFactory{
		attemptTimeout:             time.Second,
		unauthenticatedRateLimiter: ratelimit.Noop(),
		authenticatedRateLimiter:   ratelimit.Noop(),
	}
	client, err := factory.NewCatalogClient()
	require.NoError(t, err)

	event, err := client.GetEvent(context.Background(), eventID)
	require.NoError(t, err)
	require.Equal(t, eventID, event.ConditionID)
	market, err := client.GetMarket(context.Background(), marketID)
	require.NoError(t, err)
	require.Equal(t, marketID, market.ConditionID)
	require.Equal(t, []requestRecord{
		{method: http.MethodGet, url: OfficialWormAPIBaseURL + "/events/" + eventID + "/"},
		{method: http.MethodGet, url: OfficialWormAPIBaseURL + "/markets/" + marketID + "/"},
	}, requests)
}
