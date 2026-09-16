package wormtrading

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/useryege/athena/internal/wormtrading/apiclient"
	wormstore "github.com/useryege/athena/internal/wormtrading/store"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func executionCatalogResponse(eventID, marketID string) *apiclient.GetOrderEventCatalogResponse {
	return &apiclient.GetOrderEventCatalogResponse{FetchedAt: 1, Event: &apiclient.OrderEventCatalog{
		EventConditionId: eventID, Title: "Fixture event", Markets: []*apiclient.OrderEventCatalogMarket{{
			EventConditionId: eventID, MarketConditionId: marketID, Backend: "polymarket", Outcomes: []*apiclient.OrderEventCatalogOutcome{
				{IsYes: true, Label: "Yes", Selectable: true, MaxLeverage: "2"},
				{IsYes: false, Label: "No", Selectable: true, MaxLeverage: "2"},
			},
		}},
	}}
}

func TestExecutionCatalogUsesReaderOncePerEvent(t *testing.T) {
	eventID, marketID := catalogConditionID(1), catalogConditionID(2)
	calls := 0
	s := &Service{wormCatalogBudget: time.Second, catalogReader: catalogReaderFunc(func(ctx context.Context, req *apiclient.GetOrderEventCatalogRequest) (*apiclient.GetOrderEventCatalogResponse, error) {
		calls++
		require.Equal(t, eventID, req.EventConditionId)
		_, incoming := metadata.FromIncomingContext(ctx)
		_, outgoing := metadata.FromOutgoingContext(ctx)
		require.False(t, incoming)
		require.False(t, outgoing)
		deadline, ok := ctx.Deadline()
		require.True(t, ok)
		require.LessOrEqual(t, time.Until(deadline), time.Second)
		return executionCatalogResponse(eventID, marketID), nil
	})}
	result, err := s.readExecutionPlanCatalogs(context.Background(), []wormstore.ExecutionPlanItem{
		{Ordinal: 1, EventConditionID: eventID, MarketConditionID: marketID, IsYes: true},
		{Ordinal: 2, EventConditionID: eventID, MarketConditionID: marketID, IsYes: false},
	})
	require.NoError(t, err)
	require.Len(t, result, 2)
	require.Equal(t, 1, calls)
	for i, item := range result {
		require.True(t, item.input.Selectable)
		require.Equal(t, "polymarket", item.input.Backend)
		require.Equal(t, "5", item.input.Funds)
		require.EqualValues(t, i+1, item.observation.Ordinal)
	}
	require.True(t, result[0].input.IsYes)
	require.False(t, result[1].input.IsYes)
}

func TestExecutionCatalogPropagatesCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	called := false
	s := &Service{wormCatalogBudget: time.Second, catalogReader: catalogReaderFunc(func(c context.Context, _ *apiclient.GetOrderEventCatalogRequest) (*apiclient.GetOrderEventCatalogResponse, error) {
		called = true
		return nil, status.FromContextError(c.Err()).Err()
	})}
	_, err := s.readExecutionPlanCatalogs(ctx, []wormstore.ExecutionPlanItem{{EventConditionID: catalogConditionID(1)}})
	require.Error(t, err)
	require.True(t, called)
	require.ErrorContains(t, err, "Canceled")
	require.Equal(t, executionPlanFailureMarketsUnavailable, executionPlanFailureCode(err))
}

func TestExecutionCatalogFailureClassification(t *testing.T) {
	for _, tc := range []struct {
		name string
		code codes.Code
		want string
	}{
		{"unavailable", codes.Unavailable, executionPlanFailureMarketsUnavailable},
		{"invalid", codes.InvalidArgument, executionPlanFailureMarketsInvalid},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := &Service{wormCatalogBudget: time.Second, catalogReader: catalogReaderFunc(func(context.Context, *apiclient.GetOrderEventCatalogRequest) (*apiclient.GetOrderEventCatalogResponse, error) {
				return nil, status.Error(tc.code, "catalog failure")
			})}
			_, err := s.readExecutionPlanCatalogs(context.Background(), []wormstore.ExecutionPlanItem{{EventConditionID: catalogConditionID(1)}})
			require.ErrorContains(t, err, "catalog failure")
			require.Equal(t, tc.want, executionPlanFailureCode(err))
		})
	}
}

func TestExecutionCatalogSharesBudgetAndHonorsCallerDeadline(t *testing.T) {
	for _, callerDeadline := range []bool{false, true} {
		t.Run(map[bool]string{false: "budget", true: "caller"}[callerDeadline], func(t *testing.T) {
			ctx := context.Background()
			if callerDeadline {
				var cancel context.CancelFunc
				ctx, cancel = context.WithTimeout(ctx, 20*time.Millisecond)
				defer cancel()
			}
			var firstDeadline time.Time
			calls := 0
			s := &Service{wormCatalogBudget: time.Second, catalogReader: catalogReaderFunc(func(c context.Context, req *apiclient.GetOrderEventCatalogRequest) (*apiclient.GetOrderEventCatalogResponse, error) {
				calls++
				deadline, ok := c.Deadline()
				require.True(t, ok)
				if calls == 1 {
					firstDeadline = deadline
					return executionCatalogResponse(req.EventConditionId, catalogConditionID(2)), nil
				}
				require.Equal(t, firstDeadline, deadline)
				if callerDeadline {
					<-c.Done()
					return nil, status.FromContextError(c.Err()).Err()
				}
				return executionCatalogResponse(req.EventConditionId, catalogConditionID(2)), nil
			})}
			_, err := s.readExecutionPlanCatalogs(ctx, []wormstore.ExecutionPlanItem{
				{EventConditionID: catalogConditionID(1), MarketConditionID: catalogConditionID(2)},
				{EventConditionID: catalogConditionID(3), MarketConditionID: catalogConditionID(2)},
			})
			require.Equal(t, 2, calls)
			if callerDeadline {
				require.ErrorContains(t, err, "DeadlineExceeded")
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestExecutionCatalogBudgetExpires(t *testing.T) {
	s := &Service{wormCatalogBudget: 10 * time.Millisecond, catalogReader: catalogReaderFunc(func(ctx context.Context, _ *apiclient.GetOrderEventCatalogRequest) (*apiclient.GetOrderEventCatalogResponse, error) {
		<-ctx.Done()
		return nil, status.FromContextError(ctx.Err()).Err()
	})}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	_, err := s.readExecutionPlanCatalogs(ctx, []wormstore.ExecutionPlanItem{{EventConditionID: catalogConditionID(1)}})
	require.ErrorContains(t, err, "DeadlineExceeded")
	require.NoError(t, ctx.Err(), "catalog budget must expire before the worker context")
	require.Equal(t, executionPlanFailureMarketsUnavailable, executionPlanFailureCode(err))
}

func TestExecutionCatalogRejectsMalformedResponse(t *testing.T) {
	for _, tc := range []struct {
		name   string
		mutate func(*apiclient.GetOrderEventCatalogResponse) *apiclient.GetOrderEventCatalogResponse
	}{
		{"nil", func(*apiclient.GetOrderEventCatalogResponse) *apiclient.GetOrderEventCatalogResponse { return nil }},
		{"missing timestamp", func(r *apiclient.GetOrderEventCatalogResponse) *apiclient.GetOrderEventCatalogResponse {
			r.FetchedAt = 0
			return r
		}},
		{"wrong event", func(r *apiclient.GetOrderEventCatalogResponse) *apiclient.GetOrderEventCatalogResponse {
			r.Event.EventConditionId = catalogConditionID(3)
			return r
		}},
		{"duplicate market", func(r *apiclient.GetOrderEventCatalogResponse) *apiclient.GetOrderEventCatalogResponse {
			r.Event.Markets = append(r.Event.Markets, r.Event.Markets[0])
			return r
		}},
		{"duplicate direction", func(r *apiclient.GetOrderEventCatalogResponse) *apiclient.GetOrderEventCatalogResponse {
			r.Event.Markets[0].Outcomes[1].IsYes = true
			return r
		}},
		{"invalid leverage", func(r *apiclient.GetOrderEventCatalogResponse) *apiclient.GetOrderEventCatalogResponse {
			r.Event.Markets[0].Outcomes[0].MaxLeverage = "0.5"
			return r
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := &Service{wormCatalogBudget: time.Second, catalogReader: catalogReaderFunc(func(context.Context, *apiclient.GetOrderEventCatalogRequest) (*apiclient.GetOrderEventCatalogResponse, error) {
				return tc.mutate(executionCatalogResponse(catalogConditionID(1), catalogConditionID(2))), nil
			})}
			_, err := s.readExecutionPlanCatalogs(context.Background(), []wormstore.ExecutionPlanItem{{EventConditionID: catalogConditionID(1), MarketConditionID: catalogConditionID(2), IsYes: true}})
			require.Error(t, err)
			require.Equal(t, executionPlanFailureMarketsInvalid, executionPlanFailureCode(err))
		})
	}
}
