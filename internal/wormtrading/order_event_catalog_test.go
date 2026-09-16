package wormtrading

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gagliardetto/solana-go"
	"github.com/stretchr/testify/require"
	"github.com/useryege/athena/internal/wormtrading/apiclient"
	utilworm "github.com/useryege/athena/util/worm"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func catalogConditionID(seed byte) string {
	var key solana.PublicKey
	key[len(key)-1] = seed
	return key.String()
}

type catalogClientFunc struct {
	event  func(context.Context, string) (*utilworm.Event, error)
	market func(context.Context, string) (*utilworm.Market, error)
}

func (f catalogClientFunc) GetEvent(ctx context.Context, id string) (*utilworm.Event, error) {
	return f.event(ctx, id)
}

func (f catalogClientFunc) GetMarket(ctx context.Context, id string) (*utilworm.Market, error) {
	return f.market(ctx, id)
}

func TestOrderEventCatalogRejectsInvalidIDBeforeProvider(t *testing.T) {
	client := catalogClientFunc{
		event: func(context.Context, string) (*utilworm.Event, error) {
			t.Fatal("invalid ID reached provider")
			return nil, nil
		},
		market: func(context.Context, string) (*utilworm.Market, error) {
			t.Fatal("unexpected market read")
			return nil, nil
		},
	}
	reader, err := NewOrderEventCatalogReader(client, 45*time.Second, nil)
	require.NoError(t, err)

	_, err = reader.GetOrderEventCatalog(context.Background(), &apiclient.GetOrderEventCatalogRequest{EventConditionId: "invalid"})
	require.Equal(t, codes.InvalidArgument, status.Code(err))
}

func TestOrderEventCatalogConstructorValidatesDependencies(t *testing.T) {
	client := catalogClientFunc{
		event:  func(context.Context, string) (*utilworm.Event, error) { return nil, nil },
		market: func(context.Context, string) (*utilworm.Market, error) { return nil, nil },
	}

	reader, err := NewOrderEventCatalogReader(nil, time.Second, nil)
	require.Nil(t, reader)
	require.EqualError(t, err, "worm catalog client is required")
	reader, err = NewOrderEventCatalogReader(client, 0, nil)
	require.Nil(t, reader)
	require.EqualError(t, err, "worm catalog budget must be positive")
}

func TestOrderEventCatalogReadsCanonicalEventOnce(t *testing.T) {
	eventID := catalogConditionID(1)
	logo := "/images/event.png"
	var eventCalls int
	client := catalogClientFunc{
		event: func(_ context.Context, id string) (*utilworm.Event, error) {
			eventCalls++
			require.Equal(t, eventID, id)
			return &utilworm.Event{ConditionID: eventID, Title: " Event title ", Logo: &logo}, nil
		},
		market: func(context.Context, string) (*utilworm.Market, error) {
			t.Fatal("empty event reached market provider")
			return nil, nil
		},
	}
	var observed []error
	reader, err := NewOrderEventCatalogReader(client, time.Second, func(err error) { observed = append(observed, err) })
	require.NoError(t, err)

	response, err := reader.GetOrderEventCatalog(context.Background(), &apiclient.GetOrderEventCatalogRequest{EventConditionId: eventID})
	require.NoError(t, err)
	require.Equal(t, 1, eventCalls)
	require.Len(t, observed, 1)
	require.NoError(t, observed[0])
	require.Equal(t, eventID, response.GetEvent().GetEventConditionId())
	require.Equal(t, "Event title", response.GetEvent().GetTitle())
	require.Equal(t, OfficialWormAPIBaseURL+logo, response.GetEvent().GetLogo())
	require.Empty(t, response.GetEvent().GetMarkets())
	require.NotZero(t, response.GetFetchedAt())
}

func TestOrderEventCatalogRejectsInvalidEventShapeBeforeMarkets(t *testing.T) {
	eventID := catalogConditionID(2)
	marketID := catalogConditionID(3)
	tests := []struct {
		name  string
		event *utilworm.Event
	}{
		{name: "nil event"},
		{name: "mismatched event", event: &utilworm.Event{ConditionID: catalogConditionID(4)}},
		{name: "noncanonical child", event: &utilworm.Event{ConditionID: eventID, Markets: []utilworm.MarketSummary{{ConditionID: " " + marketID}}}},
		{name: "invalid child", event: &utilworm.Event{ConditionID: eventID, Markets: []utilworm.MarketSummary{{ConditionID: "invalid"}}}},
		{name: "duplicate child", event: &utilworm.Event{ConditionID: eventID, Markets: []utilworm.MarketSummary{{ConditionID: marketID}, {ConditionID: marketID}}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var observed []error
			client := catalogClientFunc{
				event: func(context.Context, string) (*utilworm.Event, error) { return tt.event, nil },
				market: func(context.Context, string) (*utilworm.Market, error) {
					t.Fatal("invalid event shape reached market provider")
					return nil, nil
				},
			}
			reader, err := NewOrderEventCatalogReader(client, time.Second, func(err error) { observed = append(observed, err) })
			require.NoError(t, err)

			_, err = reader.GetOrderEventCatalog(context.Background(), &apiclient.GetOrderEventCatalogRequest{EventConditionId: eventID})
			require.Equal(t, codes.Unavailable, status.Code(err))
			require.Len(t, observed, 1)
			require.Error(t, observed[0])
			require.Equal(t, wormErrorInvalidResponse, classifyWormError(observed[0]))
		})
	}
}

func TestOrderEventCatalogMapsEventProviderErrors(t *testing.T) {
	eventID := catalogConditionID(5)
	tests := []struct {
		name string
		err  error
		code codes.Code
	}{
		{name: "cancelled", err: context.Canceled, code: codes.Canceled},
		{name: "deadline", err: context.DeadlineExceeded, code: codes.DeadlineExceeded},
		{name: "not found", err: &utilworm.Error{StatusCode: http.StatusNotFound}, code: codes.NotFound},
		{name: "dependency", err: errors.New("network down"), code: codes.Unavailable},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var mu sync.Mutex
			var observed []error
			client := catalogClientFunc{
				event: func(context.Context, string) (*utilworm.Event, error) { return nil, tt.err },
				market: func(context.Context, string) (*utilworm.Market, error) {
					t.Fatal("failed event reached market provider")
					return nil, nil
				},
			}
			reader, err := NewOrderEventCatalogReader(client, time.Second, func(err error) {
				mu.Lock()
				defer mu.Unlock()
				observed = append(observed, err)
			})
			require.NoError(t, err)

			_, err = reader.GetOrderEventCatalog(context.Background(), &apiclient.GetOrderEventCatalogRequest{EventConditionId: eventID})
			require.Equal(t, tt.code, status.Code(err))
			require.Equal(t, []error{tt.err}, observed)
		})
	}
}

func catalogString(value string) *string { return &value }

func validCatalogMarket(eventID, marketID string) *utilworm.Market {
	return &utilworm.Market{
		MarketSummary: utilworm.MarketSummary{
			ConditionID:    marketID,
			Title:          " Market " + marketID,
			State:          "OPEN",
			MarginEnabled:  true,
			LastTradePrice: catalogString("0.25"),
			Event:          &utilworm.EventMini{ConditionID: eventID},
		},
		Outcomes: []utilworm.Outcome{{IsYes: true, Text: "Yes"}, {IsYes: false, Text: "No"}},
		Config: &utilworm.MarketConfig{
			Kind:           "Polymarket",
			MaxLeverageYes: catalogString("1"),
			MaxLeverageNo:  catalogString("2.5"),
		},
	}
}

func readCatalog(t *testing.T, client WormCatalogClient, eventID string) (*apiclient.GetOrderEventCatalogResponse, error) {
	t.Helper()
	reader, err := NewOrderEventCatalogReader(client, time.Second, nil)
	require.NoError(t, err)
	return reader.GetOrderEventCatalog(context.Background(), &apiclient.GetOrderEventCatalogRequest{EventConditionId: eventID})
}

func TestOrderEventCatalogPreservesProviderOrderAndReadsEachDetailOnce(t *testing.T) {
	eventID := catalogConditionID(10)
	firstID := catalogConditionID(11)
	secondID := catalogConditionID(12)
	secondStarted := make(chan struct{})
	var mu sync.Mutex
	eventCalls := 0
	marketCalls := map[string]int{}
	client := catalogClientFunc{
		event: func(context.Context, string) (*utilworm.Event, error) {
			mu.Lock()
			eventCalls++
			mu.Unlock()
			return &utilworm.Event{
				ConditionID: eventID,
				Markets: []utilworm.MarketSummary{
					{ConditionID: firstID, Title: "first"},
					{ConditionID: secondID, Title: "second"},
				},
			}, nil
		},
		market: func(ctx context.Context, id string) (*utilworm.Market, error) {
			mu.Lock()
			marketCalls[id]++
			mu.Unlock()
			if id == firstID {
				select {
				case <-secondStarted:
				case <-ctx.Done():
					return nil, ctx.Err()
				}
			} else {
				close(secondStarted)
			}
			return validCatalogMarket(eventID, id), nil
		},
	}

	response, err := readCatalog(t, client, eventID)
	require.NoError(t, err)
	require.Len(t, response.GetEvent().GetMarkets(), 2)
	require.Equal(t, []string{firstID, secondID}, []string{
		response.GetEvent().GetMarkets()[0].GetMarketConditionId(),
		response.GetEvent().GetMarkets()[1].GetMarketConditionId(),
	})
	require.Equal(t, 1, eventCalls)
	require.Equal(t, map[string]int{firstID: 1, secondID: 1}, marketCalls)
}

func TestOrderEventCatalogRetainsUnavailableMarketDetails(t *testing.T) {
	eventID := catalogConditionID(20)
	marketID := catalogConditionID(21)
	tests := []struct {
		name string
		err  error
	}{
		{name: "not found", err: &utilworm.Error{StatusCode: http.StatusNotFound}},
		{name: "network", err: errors.New("network down")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var observed []error
			client := catalogClientFunc{
				event: func(context.Context, string) (*utilworm.Event, error) {
					return &utilworm.Event{ConditionID: eventID, Markets: []utilworm.MarketSummary{{
						ConditionID: marketID,
						Title:       "summary title",
						Outcomes:    []utilworm.Outcome{{IsYes: true, Text: "Up"}, {IsYes: false, Text: "Down"}},
					}}}, nil
				},
				market: func(context.Context, string) (*utilworm.Market, error) { return nil, tt.err },
			}
			reader, err := NewOrderEventCatalogReader(client, time.Second, func(err error) { observed = append(observed, err) })
			require.NoError(t, err)

			response, err := reader.GetOrderEventCatalog(context.Background(), &apiclient.GetOrderEventCatalogRequest{EventConditionId: eventID})
			require.NoError(t, err)
			require.Len(t, response.GetEvent().GetMarkets(), 1)
			market := response.GetEvent().GetMarkets()[0]
			require.Equal(t, marketID, market.GetMarketConditionId())
			require.Equal(t, "summary title", market.GetTitle())
			require.Equal(t, "MARKET_DETAIL_UNAVAILABLE", market.GetUnavailableCode())
			require.Equal(t, []string{"Up", "Down"}, []string{market.GetOutcomes()[0].GetLabel(), market.GetOutcomes()[1].GetLabel()})
			require.False(t, market.GetOutcomes()[0].GetSelectable())
			require.False(t, market.GetOutcomes()[1].GetSelectable())
			require.Len(t, observed, 2)
			require.NoError(t, observed[0])
			require.ErrorIs(t, observed[1], tt.err)
		})
	}
}

func TestOrderEventCatalogAppliesMarketAndOutcomeRules(t *testing.T) {
	eventID := catalogConditionID(30)
	marketID := catalogConditionID(31)
	tests := []struct {
		name           string
		mutate         func(*utilworm.Market)
		marketCode     string
		yesSelectable  bool
		noSelectable   bool
		yesUnavailable string
		noUnavailable  string
	}{
		{name: "valid polymarket", yesSelectable: true, noSelectable: true},
		{name: "valid hyperliquid", mutate: func(m *utilworm.Market) { m.Config.Kind = " HYPERLIQUID " }, yesSelectable: true, noSelectable: true},
		{name: "not open", mutate: func(m *utilworm.Market) { m.State = "closed" }, marketCode: "MARKET_NOT_OPEN", yesUnavailable: "MARKET_NOT_OPEN", noUnavailable: "MARKET_NOT_OPEN"},
		{name: "margin disabled", mutate: func(m *utilworm.Market) { m.MarginEnabled = false }, marketCode: "MARGIN_DISABLED", yesUnavailable: "MARGIN_DISABLED", noUnavailable: "MARGIN_DISABLED"},
		{name: "config missing", mutate: func(m *utilworm.Market) { m.Config = nil }, marketCode: "CONFIG_MISSING", yesUnavailable: "CONFIG_MISSING", noUnavailable: "CONFIG_MISSING"},
		{name: "backend unsupported", mutate: func(m *utilworm.Market) { m.Config.Kind = "other" }, marketCode: "BACKEND_UNSUPPORTED", yesUnavailable: "BACKEND_UNSUPPORTED", noUnavailable: "BACKEND_UNSUPPORTED"},
		{name: "outcomes invalid", mutate: func(m *utilworm.Market) { m.Outcomes = []utilworm.Outcome{{IsYes: true, Text: "Yes"}} }, marketCode: "OUTCOMES_INVALID", yesUnavailable: "OUTCOMES_INVALID", noUnavailable: "OUTCOMES_INVALID"},
		{name: "market id mismatch", mutate: func(m *utilworm.Market) { m.ConditionID = catalogConditionID(32) }, marketCode: "MARKET_ID_MISMATCH", yesUnavailable: "MARKET_ID_MISMATCH", noUnavailable: "MARKET_ID_MISMATCH"},
		{name: "event mismatch", mutate: func(m *utilworm.Market) { m.Event.ConditionID = catalogConditionID(33) }, marketCode: "MARKET_EVENT_MISMATCH", yesUnavailable: "MARKET_EVENT_MISMATCH", noUnavailable: "MARKET_EVENT_MISMATCH"},
		{name: "yes leverage missing", mutate: func(m *utilworm.Market) { m.Config.MaxLeverageYes = nil }, noSelectable: true, yesUnavailable: "MAX_LEVERAGE_MISSING"},
		{name: "yes leverage invalid", mutate: func(m *utilworm.Market) { m.Config.MaxLeverageYes = catalogString("nan") }, noSelectable: true, yesUnavailable: "MAX_LEVERAGE_INVALID"},
		{name: "yes leverage below one", mutate: func(m *utilworm.Market) { m.Config.MaxLeverageYes = catalogString("0.999") }, noSelectable: true, yesUnavailable: "MAX_LEVERAGE_BELOW_ONE"},
		{name: "both leverages below one", mutate: func(m *utilworm.Market) {
			m.Config.MaxLeverageYes = catalogString("0")
			m.Config.MaxLeverageNo = catalogString("0")
		}, marketCode: "NO_SELECTABLE_OUTCOMES", yesUnavailable: "MAX_LEVERAGE_BELOW_ONE", noUnavailable: "MAX_LEVERAGE_BELOW_ONE"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			detail := validCatalogMarket(eventID, marketID)
			if tt.mutate != nil {
				tt.mutate(detail)
			}
			client := catalogClientFunc{
				event: func(context.Context, string) (*utilworm.Event, error) {
					return &utilworm.Event{ConditionID: eventID, Markets: []utilworm.MarketSummary{{ConditionID: marketID}}}, nil
				},
				market: func(context.Context, string) (*utilworm.Market, error) { return detail, nil },
			}

			response, err := readCatalog(t, client, eventID)
			require.NoError(t, err)
			require.Len(t, response.GetEvent().GetMarkets(), 1)
			market := response.GetEvent().GetMarkets()[0]
			require.Equal(t, tt.marketCode, market.GetUnavailableCode())
			require.Equal(t, tt.yesSelectable, market.GetOutcomes()[0].GetSelectable())
			require.Equal(t, tt.noSelectable, market.GetOutcomes()[1].GetSelectable())
			require.Equal(t, tt.yesUnavailable, market.GetOutcomes()[0].GetUnavailableCode())
			require.Equal(t, tt.noUnavailable, market.GetOutcomes()[1].GetUnavailableCode())
		})
	}
}

func TestOrderEventCatalogObservesInvalidMarketResponse(t *testing.T) {
	eventID := catalogConditionID(35)
	marketID := catalogConditionID(36)
	tests := []struct {
		name   string
		market *utilworm.Market
	}{
		{name: "nil market"},
		{name: "market id mismatch", market: func() *utilworm.Market {
			market := validCatalogMarket(eventID, marketID)
			market.ConditionID = catalogConditionID(37)
			return market
		}()},
		{name: "event mismatch", market: func() *utilworm.Market {
			market := validCatalogMarket(eventID, marketID)
			market.Event.ConditionID = catalogConditionID(38)
			return market
		}()},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var observed []error
			client := catalogClientFunc{
				event: func(context.Context, string) (*utilworm.Event, error) {
					return &utilworm.Event{ConditionID: eventID, Markets: []utilworm.MarketSummary{{ConditionID: marketID}}}, nil
				},
				market: func(context.Context, string) (*utilworm.Market, error) { return tt.market, nil },
			}
			reader, err := NewOrderEventCatalogReader(client, time.Second, func(err error) { observed = append(observed, err) })
			require.NoError(t, err)

			response, err := reader.GetOrderEventCatalog(context.Background(), &apiclient.GetOrderEventCatalogRequest{EventConditionId: eventID})
			require.NoError(t, err)
			require.Len(t, response.GetEvent().GetMarkets(), 1)
			require.Len(t, observed, 2)
			require.NoError(t, observed[0])
			require.Error(t, observed[1])
			require.Equal(t, wormErrorInvalidResponse, classifyWormError(observed[1]))
		})
	}
}

func TestOrderEventCatalogPreservesExactPricesWithoutAffectingSelection(t *testing.T) {
	eventID := catalogConditionID(40)
	marketID := catalogConditionID(41)
	longFraction := strings.Repeat("0", 99) + "1"
	tests := []struct {
		name     string
		price    *string
		yesPrice string
		noPrice  string
	}{
		{name: "zero", price: catalogString("0"), yesPrice: "0", noPrice: "1"},
		{name: "one", price: catalogString("1"), yesPrice: "1", noPrice: "0"},
		{name: "long exact decimal", price: catalogString("0." + longFraction), yesPrice: "0." + longFraction, noPrice: "0." + strings.Repeat("9", 100)},
		{name: "missing"},
		{name: "longer than bound", price: catalogString("0." + strings.Repeat("1", 127))},
		{name: "above one", price: catalogString("1.0001")},
		{name: "not plain decimal", price: catalogString("1e-1")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			detail := validCatalogMarket(eventID, marketID)
			detail.LastTradePrice = tt.price
			client := catalogClientFunc{
				event: func(context.Context, string) (*utilworm.Event, error) {
					return &utilworm.Event{ConditionID: eventID, Markets: []utilworm.MarketSummary{{ConditionID: marketID}}}, nil
				},
				market: func(context.Context, string) (*utilworm.Market, error) { return detail, nil },
			}

			response, err := readCatalog(t, client, eventID)
			require.NoError(t, err)
			outcomes := response.GetEvent().GetMarkets()[0].GetOutcomes()
			require.Equal(t, tt.yesPrice, outcomes[0].GetLastTradePrice())
			require.Equal(t, tt.noPrice, outcomes[1].GetLastTradePrice())
			require.True(t, outcomes[0].GetSelectable())
			require.True(t, outcomes[1].GetSelectable())
		})
	}
}

func TestOrderEventCatalogLimitsMarketConcurrencyToEight(t *testing.T) {
	eventID := catalogConditionID(50)
	markets := make([]utilworm.MarketSummary, 20)
	for i := range markets {
		markets[i].ConditionID = catalogConditionID(byte(51 + i))
	}
	var active atomic.Int32
	var maximum atomic.Int32
	release := make(chan struct{})
	client := catalogClientFunc{
		event: func(context.Context, string) (*utilworm.Event, error) {
			return &utilworm.Event{ConditionID: eventID, Markets: markets}, nil
		},
		market: func(ctx context.Context, id string) (*utilworm.Market, error) {
			current := active.Add(1)
			defer active.Add(-1)
			for {
				old := maximum.Load()
				if current <= old || maximum.CompareAndSwap(old, current) {
					break
				}
			}
			if current == orderEventCatalogMarketConcurrency {
				select {
				case <-release:
				default:
					close(release)
				}
			}
			select {
			case <-release:
			case <-ctx.Done():
				return nil, ctx.Err()
			}
			return validCatalogMarket(eventID, id), nil
		},
	}

	response, err := readCatalog(t, client, eventID)
	require.NoError(t, err)
	require.Len(t, response.GetEvent().GetMarkets(), len(markets))
	require.Equal(t, int32(orderEventCatalogMarketConcurrency), maximum.Load())
}

func TestOrderEventCatalogCancellationStopsDispatchingNewMarkets(t *testing.T) {
	eventID := catalogConditionID(80)
	markets := make([]utilworm.MarketSummary, 20)
	for i := range markets {
		markets[i].ConditionID = catalogConditionID(byte(81 + i))
	}
	var calls atomic.Int32
	eightStarted := make(chan struct{})
	client := catalogClientFunc{
		event: func(context.Context, string) (*utilworm.Event, error) {
			return &utilworm.Event{ConditionID: eventID, Markets: markets}, nil
		},
		market: func(ctx context.Context, _ string) (*utilworm.Market, error) {
			if calls.Add(1) == orderEventCatalogMarketConcurrency {
				close(eightStarted)
			}
			<-ctx.Done()
			return nil, ctx.Err()
		},
	}
	reader, err := NewOrderEventCatalogReader(client, time.Second, nil)
	require.NoError(t, err)
	ctx, cancel := context.WithCancel(context.Background())
	result := make(chan error, 1)
	go func() {
		_, err := reader.GetOrderEventCatalog(ctx, &apiclient.GetOrderEventCatalogRequest{EventConditionId: eventID})
		result <- err
	}()
	select {
	case <-eightStarted:
	case <-time.After(time.Second):
		t.Fatal("first market batch was not dispatched")
	}
	cancel()

	require.Equal(t, codes.Canceled, status.Code(<-result))
	require.Equal(t, int32(orderEventCatalogMarketConcurrency), calls.Load())
}

func TestOrderEventCatalogDoesNotDispatchProviderAfterCancellation(t *testing.T) {
	eventID := catalogConditionID(105)
	client := catalogClientFunc{
		event: func(context.Context, string) (*utilworm.Event, error) {
			t.Fatal("cancelled request reached event provider")
			return nil, nil
		},
		market: func(context.Context, string) (*utilworm.Market, error) {
			t.Fatal("cancelled request reached market provider")
			return nil, nil
		},
	}
	reader, err := NewOrderEventCatalogReader(client, time.Second, nil)
	require.NoError(t, err)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err = reader.GetOrderEventCatalog(ctx, &apiclient.GetOrderEventCatalogRequest{EventConditionId: eventID})
	require.Equal(t, codes.Canceled, status.Code(err))
}

func TestOrderEventCatalogBudgetIncludesProviderWait(t *testing.T) {
	eventID := catalogConditionID(110)
	var observed []error
	client := catalogClientFunc{
		event: func(ctx context.Context, _ string) (*utilworm.Event, error) {
			<-ctx.Done()
			return nil, ctx.Err()
		},
		market: func(context.Context, string) (*utilworm.Market, error) {
			t.Fatal("timed-out event reached market provider")
			return nil, nil
		},
	}
	reader, err := NewOrderEventCatalogReader(client, 20*time.Millisecond, func(err error) { observed = append(observed, err) })
	require.NoError(t, err)

	_, err = reader.GetOrderEventCatalog(context.Background(), &apiclient.GetOrderEventCatalogRequest{EventConditionId: eventID})
	require.Equal(t, codes.DeadlineExceeded, status.Code(err))
	require.Len(t, observed, 1)
	require.ErrorIs(t, observed[0], context.DeadlineExceeded)
}

func TestOrderEventCatalogRejectsProviderSuccessAfterBudget(t *testing.T) {
	eventID := catalogConditionID(111)
	client := catalogClientFunc{
		event: func(ctx context.Context, _ string) (*utilworm.Event, error) {
			<-ctx.Done()
			return &utilworm.Event{ConditionID: eventID}, nil
		},
		market: func(context.Context, string) (*utilworm.Market, error) {
			t.Fatal("expired event reached market provider")
			return nil, nil
		},
	}
	var observed []error
	reader, err := NewOrderEventCatalogReader(client, 20*time.Millisecond, func(err error) { observed = append(observed, err) })
	require.NoError(t, err)

	_, err = reader.GetOrderEventCatalog(context.Background(), &apiclient.GetOrderEventCatalogRequest{EventConditionId: eventID})
	require.Equal(t, codes.DeadlineExceeded, status.Code(err))
	require.Len(t, observed, 1)
	require.ErrorIs(t, observed[0], context.DeadlineExceeded)
}

func TestOrderEventCatalogObservesMarketSuccessAfterBudgetAsTimeout(t *testing.T) {
	eventID := catalogConditionID(112)
	marketID := catalogConditionID(113)
	var observed []error
	client := catalogClientFunc{
		event: func(context.Context, string) (*utilworm.Event, error) {
			return &utilworm.Event{ConditionID: eventID, Markets: []utilworm.MarketSummary{{ConditionID: marketID}}}, nil
		},
		market: func(ctx context.Context, _ string) (*utilworm.Market, error) {
			<-ctx.Done()
			return validCatalogMarket(eventID, marketID), nil
		},
	}
	reader, err := NewOrderEventCatalogReader(client, 20*time.Millisecond, func(err error) { observed = append(observed, err) })
	require.NoError(t, err)

	_, err = reader.GetOrderEventCatalog(context.Background(), &apiclient.GetOrderEventCatalogRequest{EventConditionId: eventID})
	require.Equal(t, codes.DeadlineExceeded, status.Code(err))
	require.Len(t, observed, 2)
	require.NoError(t, observed[0])
	require.ErrorIs(t, observed[1], context.DeadlineExceeded)
}

func TestOrderEventCatalogDoesNotObserveInvalidUserInput(t *testing.T) {
	observations := 0
	client := catalogClientFunc{
		event:  func(context.Context, string) (*utilworm.Event, error) { t.Fatal("provider called"); return nil, nil },
		market: func(context.Context, string) (*utilworm.Market, error) { t.Fatal("provider called"); return nil, nil },
	}
	reader, err := NewOrderEventCatalogReader(client, time.Second, func(error) { observations++ })
	require.NoError(t, err)

	_, err = reader.GetOrderEventCatalog(context.Background(), &apiclient.GetOrderEventCatalogRequest{EventConditionId: "invalid"})
	require.Equal(t, codes.InvalidArgument, status.Code(err))
	require.Zero(t, observations)
}
