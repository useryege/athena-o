package wormtrading

import (
	"context"
	"errors"
	"strings"
	"sync"

	"golang.org/x/sync/errgroup"

	"github.com/useryege/athena/internal/wormtrading/apiclient"
	wormstore "github.com/useryege/athena/internal/wormtrading/store"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const marketCombinationCatalogConcurrency = 4

type normalizedMarketCombinationSelection struct {
	eventConditionID  string
	marketConditionID string
	isYes             bool
}

func (s *Service) resolveMarketCombinationItems(
	ctx context.Context,
	selections []*apiclient.MarketCombinationItemInput,
) ([]wormstore.MarketCombinationItemInput, error) {
	if len(selections) == 0 {
		return nil, status.Error(codes.InvalidArgument, "at least one market is required")
	}
	normalized := make([]normalizedMarketCombinationSelection, len(selections))
	eventIDs := make([]string, 0, len(selections))
	seenEvents := make(map[string]struct{}, len(selections))
	seenMarkets := make(map[string]struct{}, len(selections))
	for index, selection := range selections {
		if selection == nil {
			return nil, status.Errorf(codes.InvalidArgument, "item %d is required", index+1)
		}
		eventConditionID := selection.GetEventConditionId()
		if err := validateOrderConditionID(eventConditionID); err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "item %d event condition ID is invalid", index+1)
		}
		marketConditionID := selection.GetMarketConditionId()
		if err := validateOrderConditionID(marketConditionID); err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "item %d market condition ID is invalid", index+1)
		}
		if _, exists := seenMarkets[marketConditionID]; exists {
			return nil, status.Errorf(codes.InvalidArgument, "market condition ID at item %d is duplicated", index+1)
		}
		seenMarkets[marketConditionID] = struct{}{}
		normalized[index] = normalizedMarketCombinationSelection{
			eventConditionID: eventConditionID, marketConditionID: marketConditionID, isYes: selection.GetIsYes(),
		}
		if _, exists := seenEvents[eventConditionID]; !exists {
			seenEvents[eventConditionID] = struct{}{}
			eventIDs = append(eventIDs, eventConditionID)
		}
	}
	if s.catalogReader == nil || s.wormCatalogBudget <= 0 {
		return nil, status.Error(codes.FailedPrecondition, "Worm catalog resolver is not configured")
	}

	resolveCtx, cancel := context.WithTimeout(ctx, s.wormCatalogBudget)
	defer cancel()
	catalogs := make(map[string]*apiclient.OrderEventCatalog, len(eventIDs))
	var catalogsMu sync.Mutex
	group, groupCtx := errgroup.WithContext(resolveCtx)
	group.SetLimit(marketCombinationCatalogConcurrency)
	for _, eventConditionID := range eventIDs {
		eventConditionID := eventConditionID
		group.Go(func() error {
			response, err := s.catalogReader.GetOrderEventCatalog(groupCtx, &apiclient.GetOrderEventCatalogRequest{EventConditionId: eventConditionID})
			if err != nil {
				return err
			}
			if response == nil || response.GetEvent() == nil {
				return status.Error(codes.Internal, "Worm catalog returned an incomplete event")
			}
			catalogsMu.Lock()
			catalogs[eventConditionID] = response.GetEvent()
			catalogsMu.Unlock()
			return nil
		})
	}
	if err := group.Wait(); err != nil {
		return nil, marketCombinationCatalogError(resolveCtx, err)
	}

	resolved := make([]wormstore.MarketCombinationItemInput, 0, len(normalized))
	for index, selection := range normalized {
		event := catalogs[selection.eventConditionID]
		if event == nil {
			return nil, status.Error(codes.Internal, "Worm catalog returned an incomplete event set")
		}
		if event.GetEventConditionId() != selection.eventConditionID {
			return nil, status.Errorf(codes.FailedPrecondition, "item %d selected event is no longer available", index+1)
		}
		eventTitle := strings.TrimSpace(event.GetTitle())
		if eventTitle == "" {
			return nil, status.Error(codes.Internal, "Worm catalog returned an event without a title")
		}
		var selectedMarket *apiclient.OrderEventCatalogMarket
		for _, market := range event.GetMarkets() {
			if market != nil && market.GetMarketConditionId() == selection.marketConditionID {
				if selectedMarket != nil {
					return nil, status.Error(codes.Internal, "Worm catalog returned duplicate market IDs")
				}
				selectedMarket = market
			}
		}
		if selectedMarket == nil || selectedMarket.GetEventConditionId() != selection.eventConditionID {
			return nil, status.Errorf(codes.FailedPrecondition, "item %d market is not part of the selected event", index+1)
		}
		marketTitle := strings.TrimSpace(selectedMarket.GetTitle())
		if marketTitle == "" {
			return nil, status.Error(codes.Internal, "Worm catalog returned a market without a title")
		}
		var selectedOutcome *apiclient.OrderEventCatalogOutcome
		for _, outcome := range selectedMarket.GetOutcomes() {
			if outcome != nil && outcome.GetIsYes() == selection.isYes {
				if selectedOutcome != nil {
					return nil, status.Error(codes.Internal, "Worm catalog returned duplicate market directions")
				}
				selectedOutcome = outcome
			}
		}
		if selectedOutcome == nil || !selectedOutcome.GetSelectable() {
			return nil, status.Errorf(codes.FailedPrecondition, "item %d market direction is not selectable", index+1)
		}
		outcomeLabel := strings.TrimSpace(selectedOutcome.GetLabel())
		if outcomeLabel == "" {
			return nil, status.Error(codes.Internal, "Worm catalog returned a direction without a label")
		}
		resolved = append(resolved, wormstore.MarketCombinationItemInput{
			EventConditionID: selection.eventConditionID, EventTitle: eventTitle, EventLogo: strings.TrimSpace(event.GetLogo()),
			MarketConditionID: selection.marketConditionID, MarketTitle: marketTitle, MarketLogo: strings.TrimSpace(selectedMarket.GetLogo()),
			IsYes: selection.isYes, OutcomeLabel: outcomeLabel,
		})
	}
	return resolved, nil
}

func marketCombinationCatalogError(ctx context.Context, err error) error {
	if ctxErr := ctx.Err(); ctxErr != nil {
		return status.FromContextError(ctxErr).Err()
	}
	switch {
	case errors.Is(err, context.Canceled), errors.Is(err, context.DeadlineExceeded):
		return status.FromContextError(err).Err()
	case status.Code(err) == codes.NotFound:
		return status.Error(codes.FailedPrecondition, "a selected Worm event is no longer available")
	default:
		return err
	}
}
