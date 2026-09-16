//go:build integration

package wormtrading

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/useryege/athena/internal/accountaccess"
	"github.com/useryege/athena/internal/testutil/pgtest"
	"github.com/useryege/athena/internal/wormtrading/apiclient"
	wormstore "github.com/useryege/athena/internal/wormtrading/store"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestProviderFailurePreservesStoredCombinationAndHistoryReads(t *testing.T) {
	db := pgtest.New(t, wormstore.Migrations(), "migrations")
	store := wormstore.NewSQLStore(db.Pool)
	event, market := catalogConditionID(91), catalogConditionID(92)
	service := newCombinationService(store, accountaccess.AccessLevelReadWrite, combinationCatalog(event, "Persisted event", market, "Persisted market", true, "Yes"))
	ctx := combinationAccountContext(context.Background())
	saved, err := service.CreateMarketCombination(ctx, &apiclient.CreateMarketCombinationRequest{OwnerAccountId: catalogAccountID, Name: "Saved before outage", Items: []*apiclient.MarketCombinationItemInput{{EventConditionId: event, MarketConditionId: market, IsYes: true}}})
	require.NoError(t, err)
	service.catalogReader = combinationCatalogReaderFunc(func(context.Context, *apiclient.GetOrderEventCatalogRequest) (*apiclient.GetOrderEventCatalogResponse, error) {
		return nil, status.Error(codes.Unavailable, "controlled provider outage")
	})
	_, err = service.GetOrderEventCatalog(ctx, &apiclient.GetOrderEventCatalogRequest{EventConditionId: event})
	require.Equal(t, codes.Unavailable, status.Code(err))
	combinations, err := service.ListMarketCombinations(ctx, &apiclient.ListMarketCombinationsRequest{OwnerAccountId: catalogAccountID, Page: 1, PageSize: 10})
	require.NoError(t, err)
	require.Len(t, combinations.Items, 1)
	require.Equal(t, saved.Combination.Id, combinations.Items[0].Id)
	detail, err := service.GetMarketCombination(ctx, &apiclient.GetMarketCombinationRequest{OwnerAccountId: catalogAccountID, Id: saved.Combination.Id})
	require.NoError(t, err)
	require.Equal(t, "Persisted event", detail.Combination.Items[0].EventTitle)
	runs, err := service.ListExecutionRuns(ctx, &apiclient.ListExecutionRunsRequest{OwnerAccountId: catalogAccountID, Page: 1, PageSize: 10})
	require.NoError(t, err)
	require.Empty(t, runs.Items)
}
