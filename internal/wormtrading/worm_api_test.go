package wormtrading

import (
	"context"
	"errors"
	"reflect"
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
