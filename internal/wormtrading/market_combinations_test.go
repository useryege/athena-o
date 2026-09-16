package wormtrading

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	gogoproto "github.com/gogo/protobuf/proto"
	"github.com/stretchr/testify/require"
	"github.com/useryege/athena/internal/accountaccess"
	"github.com/useryege/athena/internal/wormtrading/apiclient"
	wormstore "github.com/useryege/athena/internal/wormtrading/store"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protowire"
)

type combinationStoreSpy struct {
	wormstore.Store
	saved          []wormstore.MarketCombinationItemInput
	updated        []wormstore.MarketCombinationItemInput
	updateRevision int64
}

func (s *combinationStoreSpy) CreateMarketCombination(_ context.Context, owner, name string, items []wormstore.MarketCombinationItemInput) (*wormstore.MarketCombination, error) {
	s.saved = append([]wormstore.MarketCombinationItemInput(nil), items...)
	return &wormstore.MarketCombination{OwnerAccountID: owner, Name: name, Revision: 1, Items: combinationStoredItems(items)}, nil
}

func (s *combinationStoreSpy) UpdateMarketCombination(_ context.Context, owner, id, name string, expectedRevision int64, items []wormstore.MarketCombinationItemInput) (*wormstore.MarketCombination, error) {
	s.updated = append([]wormstore.MarketCombinationItemInput(nil), items...)
	s.updateRevision = expectedRevision
	return &wormstore.MarketCombination{ID: id, OwnerAccountID: owner, Name: name, Revision: expectedRevision + 1, Items: combinationStoredItems(items)}, nil
}

func combinationStoredItems(inputs []wormstore.MarketCombinationItemInput) []wormstore.MarketCombinationItem {
	items := make([]wormstore.MarketCombinationItem, 0, len(inputs))
	for index, input := range inputs {
		items = append(items, wormstore.MarketCombinationItem{Ordinal: int32(index + 1), MarketCombinationItemInput: input})
	}
	return items
}

type combinationCatalogReaderFunc func(context.Context, *apiclient.GetOrderEventCatalogRequest) (*apiclient.GetOrderEventCatalogResponse, error)

func (f combinationCatalogReaderFunc) GetOrderEventCatalog(ctx context.Context, req *apiclient.GetOrderEventCatalogRequest) (*apiclient.GetOrderEventCatalogResponse, error) {
	return f(ctx, req)
}

func TestMarketCombinationRejectsEmptySelection(t *testing.T) {
	s := &Service{wormCatalogBudget: 45 * time.Second}
	_, err := s.resolveMarketCombinationItems(context.Background(), nil)
	require.Equal(t, codes.InvalidArgument, status.Code(err))
}

func TestMarketCombinationResolvesTrustedSnapshotsInSelectionOrder(t *testing.T) {
	eventA, eventB := catalogConditionID(31), catalogConditionID(32)
	marketA, marketB, marketC := catalogConditionID(41), catalogConditionID(42), catalogConditionID(43)
	var mu sync.Mutex
	calls := map[string]int{}
	s := &Service{wormCatalogBudget: time.Second, catalogReader: combinationCatalogReaderFunc(func(_ context.Context, req *apiclient.GetOrderEventCatalogRequest) (*apiclient.GetOrderEventCatalogResponse, error) {
		mu.Lock()
		calls[req.GetEventConditionId()]++
		mu.Unlock()
		switch req.GetEventConditionId() {
		case eventA:
			return combinationCatalog(eventA, "Event A", marketA, "Market A", true, "A yes", marketC, "Market C", true, "C yes"), nil
		case eventB:
			return combinationCatalog(eventB, "Event B", marketB, "Market B", true, "B yes"), nil
		default:
			return nil, status.Error(codes.NotFound, "missing")
		}
	})}

	items, err := s.resolveMarketCombinationItems(context.Background(), []*apiclient.MarketCombinationItemInput{
		{EventConditionId: eventA, MarketConditionId: marketC, IsYes: true},
		{EventConditionId: eventB, MarketConditionId: marketB, IsYes: true},
		{EventConditionId: eventA, MarketConditionId: marketA, IsYes: true},
	})

	require.NoError(t, err)
	require.Equal(t, map[string]int{eventA: 1, eventB: 1}, calls)
	require.Equal(t, []string{marketC, marketB, marketA}, []string{items[0].MarketConditionID, items[1].MarketConditionID, items[2].MarketConditionID})
	require.Equal(t, []string{"Market C", "Market B", "Market A"}, []string{items[0].MarketTitle, items[1].MarketTitle, items[2].MarketTitle})
}

func TestMarketCombinationIgnoresRemovedSnapshotFields(t *testing.T) {
	eventID, marketID := catalogConditionID(51), catalogConditionID(52)
	wire, err := gogoproto.Marshal(&apiclient.MarketCombinationItemInput{EventConditionId: eventID, MarketConditionId: marketID, IsYes: true})
	require.NoError(t, err)
	for _, field := range []protowire.Number{2, 3, 5, 6, 8} {
		wire = protowire.AppendTag(wire, field, protowire.BytesType)
		wire = protowire.AppendString(wire, "tampered")
	}
	selection := new(apiclient.MarketCombinationItemInput)
	require.NoError(t, gogoproto.Unmarshal(wire, selection))
	s := &Service{wormCatalogBudget: time.Second, catalogReader: combinationCatalogReaderFunc(func(context.Context, *apiclient.GetOrderEventCatalogRequest) (*apiclient.GetOrderEventCatalogResponse, error) {
		return combinationCatalog(eventID, "Trusted event", marketID, "Trusted market", true, "Trusted outcome"), nil
	})}

	items, err := s.resolveMarketCombinationItems(context.Background(), []*apiclient.MarketCombinationItemInput{selection})

	require.NoError(t, err)
	require.Equal(t, []wormstore.MarketCombinationItemInput{{
		EventConditionID: eventID, EventTitle: "Trusted event", EventLogo: "https://worm.example/event.png",
		MarketConditionID: marketID, MarketTitle: "Trusted market", MarketLogo: "https://worm.example/market.png",
		IsYes: true, OutcomeLabel: "Trusted outcome",
	}}, items)
}

func TestMarketCombinationRejectsInvalidSelectionsBeforeCatalog(t *testing.T) {
	eventID, marketID := catalogConditionID(61), catalogConditionID(62)
	tests := []struct {
		name       string
		selections []*apiclient.MarketCombinationItemInput
	}{
		{name: "nil item", selections: []*apiclient.MarketCombinationItemInput{nil}},
		{name: "invalid event", selections: []*apiclient.MarketCombinationItemInput{{EventConditionId: "bad", MarketConditionId: marketID}}},
		{name: "invalid market", selections: []*apiclient.MarketCombinationItemInput{{EventConditionId: eventID, MarketConditionId: "bad"}}},
		{name: "duplicate market", selections: []*apiclient.MarketCombinationItemInput{{EventConditionId: eventID, MarketConditionId: marketID}, {EventConditionId: eventID, MarketConditionId: marketID, IsYes: true}}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			s := &Service{wormCatalogBudget: time.Second, catalogReader: combinationCatalogReaderFunc(func(context.Context, *apiclient.GetOrderEventCatalogRequest) (*apiclient.GetOrderEventCatalogResponse, error) {
				t.Fatal("invalid selection reached catalog")
				return nil, nil
			})}
			_, err := s.resolveMarketCombinationItems(context.Background(), tc.selections)
			require.Equal(t, codes.InvalidArgument, status.Code(err))
		})
	}
}

func TestMarketCombinationRejectsStaleOrUnselectableSelection(t *testing.T) {
	eventID, otherEventID := catalogConditionID(71), catalogConditionID(72)
	marketID := catalogConditionID(73)
	wrongEventCatalog := combinationCatalog(eventID, "Event", marketID, "Market", true, "Yes")
	wrongEventCatalog.Event.Markets[0].EventConditionId = otherEventID
	tests := []struct {
		name    string
		catalog *apiclient.GetOrderEventCatalogResponse
	}{
		{name: "missing market", catalog: combinationCatalog(eventID, "Event", catalogConditionID(74), "Other", true, "Yes")},
		{name: "market belongs to another event", catalog: wrongEventCatalog},
		{name: "direction unavailable", catalog: combinationCatalog(eventID, "Event", marketID, "Market", false, "Yes")},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			s := &Service{wormCatalogBudget: time.Second, catalogReader: combinationCatalogReaderFunc(func(context.Context, *apiclient.GetOrderEventCatalogRequest) (*apiclient.GetOrderEventCatalogResponse, error) {
				return tc.catalog, nil
			})}
			_, err := s.resolveMarketCombinationItems(context.Background(), []*apiclient.MarketCombinationItemInput{{EventConditionId: eventID, MarketConditionId: marketID, IsYes: true}})
			require.Equal(t, codes.FailedPrecondition, status.Code(err))
		})
	}
}

func TestMarketCombinationMapsCatalogFailuresWithoutSaving(t *testing.T) {
	eventID, marketID := catalogConditionID(81), catalogConditionID(82)
	tests := []struct {
		name string
		err  error
		want codes.Code
	}{
		{name: "not found", err: status.Error(codes.NotFound, "gone"), want: codes.FailedPrecondition},
		{name: "dependency", err: status.Error(codes.Unavailable, "offline"), want: codes.Unavailable},
		{name: "canceled", err: context.Canceled, want: codes.Canceled},
		{name: "deadline", err: context.DeadlineExceeded, want: codes.DeadlineExceeded},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			store := &combinationStoreSpy{}
			s := newCombinationService(store, accountaccess.AccessLevelReadWrite, nil)
			s.catalogReader = combinationCatalogReaderFunc(func(context.Context, *apiclient.GetOrderEventCatalogRequest) (*apiclient.GetOrderEventCatalogResponse, error) {
				return nil, tc.err
			})
			_, err := s.CreateMarketCombination(combinationAccountContext(context.Background()), &apiclient.CreateMarketCombinationRequest{
				OwnerAccountId: catalogAccountID, Name: "Failure",
				Items: []*apiclient.MarketCombinationItemInput{{EventConditionId: eventID, MarketConditionId: marketID}},
			})
			require.Equal(t, tc.want, status.Code(err))
			require.Empty(t, store.saved)
		})
	}
}

func TestMarketCombinationUsesOneBudgetAndFourEventConcurrency(t *testing.T) {
	var active, peak atomic.Int32
	reader := combinationCatalogReaderFunc(func(ctx context.Context, _ *apiclient.GetOrderEventCatalogRequest) (*apiclient.GetOrderEventCatalogResponse, error) {
		current := active.Add(1)
		defer active.Add(-1)
		for old := peak.Load(); current > old && !peak.CompareAndSwap(old, current); old = peak.Load() {
		}
		<-ctx.Done()
		return nil, ctx.Err()
	})
	selections := make([]*apiclient.MarketCombinationItemInput, 5)
	for index := range selections {
		selections[index] = &apiclient.MarketCombinationItemInput{EventConditionId: catalogConditionID(byte(90 + index)), MarketConditionId: catalogConditionID(byte(100 + index)), IsYes: true}
	}
	s := &Service{wormCatalogBudget: 30 * time.Millisecond, catalogReader: reader}

	_, err := s.resolveMarketCombinationItems(context.Background(), selections)

	require.Equal(t, codes.DeadlineExceeded, status.Code(err))
	require.Equal(t, int32(4), peak.Load())
}

func TestMarketCombinationCreateAuthorizesAndSavesResolvedSnapshot(t *testing.T) {
	eventID, marketID := catalogConditionID(111), catalogConditionID(112)
	store := &combinationStoreSpy{}
	s := newCombinationService(store, accountaccess.AccessLevelReadWrite, combinationCatalog(eventID, "Trusted event", marketID, "Trusted market", true, "Yes"))

	response, err := s.CreateMarketCombination(combinationAccountContext(context.Background()), &apiclient.CreateMarketCombinationRequest{
		OwnerAccountId: catalogAccountID, Name: "Trusted",
		Items: []*apiclient.MarketCombinationItemInput{{EventConditionId: eventID, MarketConditionId: marketID, IsYes: true}},
	})

	require.NoError(t, err)
	require.Equal(t, "Trusted event", response.GetCombination().GetItems()[0].GetEventTitle())
	require.Equal(t, "Trusted market", store.saved[0].MarketTitle)
}

func TestMarketCombinationWriteAuthorizationPrecedesCatalogAndStore(t *testing.T) {
	store := &combinationStoreSpy{}
	eventID, marketID := catalogConditionID(121), catalogConditionID(122)
	s := newCombinationService(store, accountaccess.AccessLevelRead, combinationCatalog(eventID, "Event", marketID, "Market", true, "Yes"))

	_, err := s.CreateMarketCombination(combinationAccountContext(context.Background()), &apiclient.CreateMarketCombinationRequest{
		OwnerAccountId: catalogAccountID, Name: "Denied",
		Items: []*apiclient.MarketCombinationItemInput{{EventConditionId: eventID, MarketConditionId: marketID}},
	})

	require.Equal(t, codes.PermissionDenied, status.Code(err))
	require.Empty(t, store.saved)
}

func TestMarketCombinationUpdateRequiresPositiveRevisionBeforeCatalog(t *testing.T) {
	store := &combinationStoreSpy{}
	s := newCombinationService(store, accountaccess.AccessLevelReadWrite, nil)

	_, err := s.UpdateMarketCombination(combinationAccountContext(context.Background()), &apiclient.UpdateMarketCombinationRequest{
		OwnerAccountId: catalogAccountID, Id: "cb9e4741-0b85-4891-a453-e13dfa4636c6", Name: "Invalid revision",
	})

	require.Equal(t, codes.InvalidArgument, status.Code(err))
	require.Empty(t, store.updated)
}

func TestMarketCombinationUpdateSavesResolvedSnapshotAtExpectedRevision(t *testing.T) {
	eventID, marketID := catalogConditionID(123), catalogConditionID(124)
	store := &combinationStoreSpy{}
	s := newCombinationService(store, accountaccess.AccessLevelReadWrite, combinationCatalog(eventID, "Current event", marketID, "Current market", true, "Current yes"))

	response, err := s.UpdateMarketCombination(combinationAccountContext(context.Background()), &apiclient.UpdateMarketCombinationRequest{
		OwnerAccountId: catalogAccountID, Id: "cb9e4741-0b85-4891-a453-e13dfa4636c6", Name: "Updated", ExpectedRevision: 3,
		Items: []*apiclient.MarketCombinationItemInput{{EventConditionId: eventID, MarketConditionId: marketID, IsYes: true}},
	})

	require.NoError(t, err)
	require.Equal(t, int64(4), response.GetCombination().GetRevision())
	require.Equal(t, int64(3), store.updateRevision)
	require.Equal(t, "Current event", store.updated[0].EventTitle)
	require.Equal(t, "Current market", store.updated[0].MarketTitle)
	require.Equal(t, "Current yes", store.updated[0].OutcomeLabel)
}

func newCombinationService(store wormstore.Store, level accountaccess.AccessLevel, catalog *apiclient.GetOrderEventCatalogResponse) *Service {
	return &Service{
		credentialStore: store, started: true, wormCatalogBudget: time.Second,
		accountAccessReader: accountAccessFunc(func(context.Context, string) (accountaccess.Access, error) { return catalogAccess(level), nil }),
		catalogReader: combinationCatalogReaderFunc(func(context.Context, *apiclient.GetOrderEventCatalogRequest) (*apiclient.GetOrderEventCatalogResponse, error) {
			if catalog == nil {
				return nil, errors.New("catalog must not be called")
			}
			return catalog, nil
		}),
	}
}

func combinationAccountContext(ctx context.Context) context.Context {
	return metadata.NewIncomingContext(ctx, metadata.Pairs(AccountIDMetadataKey, catalogAccountID))
}

func combinationCatalog(eventID, eventTitle string, marketFields ...any) *apiclient.GetOrderEventCatalogResponse {
	markets := make([]*apiclient.OrderEventCatalogMarket, 0, len(marketFields)/4)
	for index := 0; index < len(marketFields); index += 4 {
		marketID, marketTitle := marketFields[index].(string), marketFields[index+1].(string)
		selectable, outcomeLabel := marketFields[index+2].(bool), marketFields[index+3].(string)
		markets = append(markets, &apiclient.OrderEventCatalogMarket{
			EventConditionId: eventID, MarketConditionId: marketID, Title: marketTitle, Logo: "https://worm.example/market.png",
			Outcomes: []*apiclient.OrderEventCatalogOutcome{{IsYes: true, Label: outcomeLabel, Selectable: selectable}, {IsYes: false, Label: "No", Selectable: selectable}},
		})
	}
	return &apiclient.GetOrderEventCatalogResponse{Event: &apiclient.OrderEventCatalog{
		EventConditionId: eventID, Title: eventTitle, Logo: "https://worm.example/event.png", Markets: markets,
	}, FetchedAt: 1}
}
