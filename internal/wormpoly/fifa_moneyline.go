package wormpoly

import (
	"context"
	"encoding/json"
	"errors"
	"net/url"
	"strconv"
	"strings"

	"github.com/useryege/athena/pkg/apis/application/v1alpha1"
	utilpolymarket "github.com/useryege/athena/util/polymarket"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	fifaOutcomeHome = "home"
	fifaOutcomeDraw = "draw"
	fifaOutcomeAway = "away"
	clobSideBuy     = "BUY"
	clobSideSell    = "SELL"
)

type fifaTeam struct {
	Name         string `json:"name,omitempty"`
	Logo         string `json:"logo,omitempty"`
	Abbreviation string `json:"abbreviation,omitempty"`
	Alias        string `json:"alias,omitempty"`
	Ordering     string `json:"ordering,omitempty"`
	League       string `json:"league,omitempty"`
}

func (s *Service) getFIFAMoneylineEvent(ctx context.Context, eventRef string) (*v1alpha1.PolymarketFIFAMoneylineEventItem, int64, error) {
	eventRef = normalizePolymarketEventRef(eventRef)
	if eventRef == "" {
		return nil, 0, status.Error(codes.InvalidArgument, "event_ref is required")
	}
	if s.gammaClient == nil || s.clobClient == nil {
		return nil, 0, status.Error(codes.FailedPrecondition, "polymarket clients are required")
	}

	event, err := s.getPolymarketEvent(ctx, eventRef)
	if err != nil {
		return nil, 0, err
	}

	teams := fifaTeams(event.Teams)
	if !isFIFAEvent(event, teams) {
		return nil, 0, status.Error(codes.InvalidArgument, "event is not a FIFA football event")
	}

	options, tokenIDs, err := buildFIFAMoneylineOptions(event, teams)
	if err != nil {
		return nil, 0, err
	}
	if err := s.hydrateFIFAMoneylineQuotes(ctx, options, tokenIDs); err != nil {
		return nil, 0, err
	}

	fetchedAt := s.nowUnix()
	item := &v1alpha1.PolymarketFIFAMoneylineEventItem{
		EventID:       strings.TrimSpace(event.ID),
		EventSlug:     strings.TrimSpace(stringValue(event.Slug)),
		Title:         strings.TrimSpace(stringValue(event.Title)),
		Image:         strings.TrimSpace(stringValue(event.Image)),
		Sport:         fifaSportCode(event.Sport),
		Score:         strings.TrimSpace(stringValue(event.Score)),
		GameStatus:    strings.TrimSpace(stringValue(event.GameStatus)),
		StartTime:     formatTimeRFC3339(event.StartTime),
		UpdatedAt:     formatTimeRFC3339(event.UpdatedAt),
		Active:        boolValue(event.Active),
		Closed:        boolValue(event.Closed),
		Live:          boolValue(event.Live),
		Ended:         boolValue(event.Ended),
		Teams:         fifaTeamItems(teams),
		Options:       options,
		PolymarketURL: polymarketEventURL(event),
	}

	return item, fetchedAt, nil
}

func normalizePolymarketEventRef(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	if parsed, err := url.Parse(value); err == nil && parsed.Host != "" {
		parts := strings.Split(strings.Trim(parsed.Path, "/"), "/")
		for i := range parts {
			if parts[i] == "event" && i+1 < len(parts) {
				return strings.TrimSpace(parts[i+1])
			}
		}
	}
	return strings.Trim(value, "/")
}

func (s *Service) getPolymarketEvent(ctx context.Context, eventRef string) (*utilpolymarket.Event, error) {
	var (
		event *utilpolymarket.Event
		err   error
	)
	if id, parseErr := strconv.ParseInt(eventRef, 10, 64); parseErr == nil && id > 0 {
		event, err = s.gammaClient.GetEventByID(ctx, id, utilpolymarket.GetEventOptions{})
	} else {
		event, err = s.gammaClient.GetEventBySlug(ctx, eventRef, utilpolymarket.GetEventOptions{})
	}
	if err == nil {
		return event, nil
	}

	var apiErr *utilpolymarket.APIError
	if errors.As(err, &apiErr) && apiErr.StatusCode == 404 {
		return nil, status.Errorf(codes.NotFound, "polymarket event %q was not found", eventRef)
	}
	return nil, status.Errorf(codes.Unavailable, "failed to query polymarket event %q: %v", eventRef, err)
}

func buildFIFAMoneylineOptions(event *utilpolymarket.Event, teams []fifaTeam) ([]*v1alpha1.PolymarketFIFAMoneylineOptionItem, []string, error) {
	optionsByKey := make(map[string]*v1alpha1.PolymarketFIFAMoneylineOptionItem, 3)
	tokenIDs := make([]string, 0, 6)
	tokenIDSeen := make(map[string]bool, 6)
	for _, market := range event.Markets {
		if strings.ToLower(strings.TrimSpace(stringValue(market.SportsMarketType))) != "moneyline" {
			continue
		}
		outcomeKey := fifaMoneylineOutcomeKey(market, teams)
		if outcomeKey == "" {
			continue
		}
		if _, exists := optionsByKey[outcomeKey]; exists {
			continue
		}
		option, ok := fifaMoneylineOption(market, teams, outcomeKey)
		if !ok {
			continue
		}
		optionsByKey[outcomeKey] = option
		tokenIDs = appendFIFAMoneylineTokenID(tokenIDs, tokenIDSeen, option.Yes.TokenID)
		tokenIDs = appendFIFAMoneylineTokenID(tokenIDs, tokenIDSeen, option.No.TokenID)
	}

	ordered := make([]*v1alpha1.PolymarketFIFAMoneylineOptionItem, 0, 3)
	for _, key := range []string{fifaOutcomeHome, fifaOutcomeDraw, fifaOutcomeAway} {
		option := optionsByKey[key]
		if option == nil {
			return nil, nil, status.Error(codes.FailedPrecondition, "event does not contain a complete FIFA moneyline market")
		}
		ordered = append(ordered, option)
	}
	return ordered, tokenIDs, nil
}

func fifaMoneylineOption(market utilpolymarket.Market, teams []fifaTeam, outcomeKey string) (*v1alpha1.PolymarketFIFAMoneylineOptionItem, bool) {
	tokenIDs := parseHotMarketStringList(market.ClobTokenIDs)
	outcomes := parseHotMarketStringList(market.Outcomes)
	yesIndex := 0
	for i, outcome := range outcomes {
		if strings.EqualFold(strings.TrimSpace(outcome), "yes") {
			yesIndex = i
			break
		}
	}
	if yesIndex >= len(tokenIDs) || strings.TrimSpace(tokenIDs[yesIndex]) == "" {
		return nil, false
	}

	noTokenID := ""
	for i, tokenID := range tokenIDs {
		if i != yesIndex {
			noTokenID = strings.TrimSpace(tokenID)
			break
		}
	}
	yesTokenID := strings.TrimSpace(tokenIDs[yesIndex])
	if noTokenID == "" {
		return nil, false
	}

	return &v1alpha1.PolymarketFIFAMoneylineOptionItem{
		OutcomeKey:      outcomeKey,
		OutcomeLabel:    fifaMoneylineLabel(outcomeKey, teams),
		MarketID:        strings.TrimSpace(market.ID),
		MarketSlug:      strings.TrimSpace(stringValue(market.Slug)),
		Question:        strings.TrimSpace(stringValue(market.Question)),
		ConditionID:     strings.TrimSpace(stringValue(market.ConditionID)),
		Yes:             &v1alpha1.PolymarketFIFAMoneylineDirectionItem{TokenID: yesTokenID},
		No:              &v1alpha1.PolymarketFIFAMoneylineDirectionItem{TokenID: noTokenID},
		OrderMinSize:    float64Value(market.OrderMinSize),
		TickSize:        float64Value(market.OrderPriceMinTickSize),
		EnableOrderBook: boolValue(market.EnableOrderBook),
		AcceptingOrders: boolValue(market.AcceptingOrders),
		NegRisk:         boolValue(market.NegRisk),
	}, true
}

func appendFIFAMoneylineTokenID(tokenIDs []string, seen map[string]bool, tokenID string) []string {
	tokenID = strings.TrimSpace(tokenID)
	if tokenID == "" || seen[tokenID] {
		return tokenIDs
	}
	seen[tokenID] = true
	return append(tokenIDs, tokenID)
}

func (s *Service) hydrateFIFAMoneylineQuotes(ctx context.Context, options []*v1alpha1.PolymarketFIFAMoneylineOptionItem, tokenIDs []string) error {
	if len(tokenIDs) == 0 {
		return nil
	}
	midpoints, err := s.clobClient.GetMidpointPrices(ctx, tokenIDs)
	if err != nil {
		return status.Errorf(codes.Unavailable, "failed to query polymarket clob midpoint prices: %v", err)
	}

	priceTokenIDs := make([]string, 0, len(tokenIDs)*2)
	priceSides := make([]string, 0, len(tokenIDs)*2)
	spreadRequests := make([]utilpolymarket.CLOBBookRequest, 0, len(tokenIDs))
	for _, tokenID := range tokenIDs {
		priceTokenIDs = append(priceTokenIDs, tokenID, tokenID)
		priceSides = append(priceSides, clobSideBuy, clobSideSell)
		spreadRequests = append(spreadRequests, utilpolymarket.CLOBBookRequest{TokenID: tokenID})
	}

	prices, err := s.clobClient.GetMarketPrices(ctx, priceTokenIDs, priceSides)
	if err != nil {
		return status.Errorf(codes.Unavailable, "failed to query polymarket clob market prices: %v", err)
	}
	spreads, err := s.clobClient.GetSpreads(ctx, spreadRequests)
	if err != nil {
		return status.Errorf(codes.Unavailable, "failed to query polymarket clob spreads: %v", err)
	}

	for _, option := range options {
		applyFIFAMoneylineQuote(option.Yes, midpoints, prices, spreads)
		applyFIFAMoneylineQuote(option.No, midpoints, prices, spreads)
	}
	return nil
}

func applyFIFAMoneylineQuote(
	item *v1alpha1.PolymarketFIFAMoneylineDirectionItem,
	midpoints map[string]string,
	prices map[string]map[string]string,
	spreads map[string]string,
) {
	if item == nil {
		return
	}
	item.MidPrice = parseFloatOrZero(midpoints[item.TokenID])
	item.BestBid = parseFloatOrZero(prices[item.TokenID][clobSideBuy])
	item.BestAsk = parseFloatOrZero(prices[item.TokenID][clobSideSell])
	item.Spread = parseFloatOrZero(spreads[item.TokenID])
}

func fifaMoneylineOutcomeKey(market utilpolymarket.Market, teams []fifaTeam) string {
	groupTitle := normalizeFIFAKey(stringValue(market.GroupItemTitle))
	if strings.Contains(groupTitle, "draw") || strings.Contains(groupTitle, "tie") {
		return fifaOutcomeDraw
	}
	if groupTitle != "" {
		if matchesFIFATeam(groupTitle, fifaTeamByOrdering(teams, fifaOutcomeHome)) {
			return fifaOutcomeHome
		}
		if matchesFIFATeam(groupTitle, fifaTeamByOrdering(teams, fifaOutcomeAway)) {
			return fifaOutcomeAway
		}
	}

	questionText := normalizeFIFAKey(stringValue(market.Question))
	if strings.Contains(questionText, " draw") || strings.Contains(questionText, "draw ") {
		return fifaOutcomeDraw
	}
	if matchesFIFATeam(questionText, fifaTeamByOrdering(teams, fifaOutcomeHome)) {
		return fifaOutcomeHome
	}
	if matchesFIFATeam(questionText, fifaTeamByOrdering(teams, fifaOutcomeAway)) {
		return fifaOutcomeAway
	}
	slug := normalizeFIFAKey(stringValue(market.Slug))
	if strings.HasSuffix(slug, "-draw") {
		return fifaOutcomeDraw
	}
	if slugHasTeamSuffix(slug, fifaTeamByOrdering(teams, fifaOutcomeHome)) {
		return fifaOutcomeHome
	}
	if slugHasTeamSuffix(slug, fifaTeamByOrdering(teams, fifaOutcomeAway)) {
		return fifaOutcomeAway
	}
	return ""
}

func fifaMoneylineLabel(outcomeKey string, teams []fifaTeam) string {
	switch outcomeKey {
	case fifaOutcomeHome:
		if team := fifaTeamByOrdering(teams, fifaOutcomeHome); team.Name != "" {
			return team.Name
		}
		return "Home"
	case fifaOutcomeAway:
		if team := fifaTeamByOrdering(teams, fifaOutcomeAway); team.Name != "" {
			return team.Name
		}
		return "Away"
	default:
		return "Draw"
	}
}

func matchesFIFATeam(text string, team fifaTeam) bool {
	for _, key := range []string{team.Name, team.Abbreviation, team.Alias} {
		key = normalizeFIFAKey(key)
		if key != "" && (text == key || strings.Contains(text, key)) {
			return true
		}
	}
	return false
}

func slugHasTeamSuffix(slug string, team fifaTeam) bool {
	for _, key := range []string{team.Abbreviation, team.Alias, team.Name} {
		key = normalizeFIFAKey(key)
		if key != "" && strings.HasSuffix(slug, "-"+key) {
			return true
		}
	}
	return false
}

func normalizeFIFAKey(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func fifaTeamByOrdering(teams []fifaTeam, ordering string) fifaTeam {
	for _, team := range teams {
		if strings.EqualFold(strings.TrimSpace(team.Ordering), ordering) {
			return team
		}
	}
	switch ordering {
	case fifaOutcomeHome:
		if len(teams) > 0 {
			return teams[0]
		}
	case fifaOutcomeAway:
		if len(teams) > 1 {
			return teams[1]
		}
	}
	return fifaTeam{}
}

func fifaTeams(rawTeams []json.RawMessage) []fifaTeam {
	teams := make([]fifaTeam, 0, len(rawTeams))
	for _, raw := range rawTeams {
		var team fifaTeam
		if err := json.Unmarshal(raw, &team); err != nil {
			continue
		}
		teams = append(teams, team)
	}
	return teams
}

func fifaTeamItems(teams []fifaTeam) []*v1alpha1.PolymarketSportsLiveTeamItem {
	items := make([]*v1alpha1.PolymarketSportsLiveTeamItem, 0, len(teams))
	for _, team := range teams {
		items = append(items, &v1alpha1.PolymarketSportsLiveTeamItem{
			Name:         strings.TrimSpace(team.Name),
			Logo:         strings.TrimSpace(team.Logo),
			Abbreviation: strings.TrimSpace(team.Abbreviation),
			Alias:        strings.TrimSpace(team.Alias),
		})
	}
	return items
}

func isFIFAEvent(event *utilpolymarket.Event, teams []fifaTeam) bool {
	sport := normalizeFIFAKey(fifaSportCode(event.Sport))
	if strings.HasPrefix(sport, "fif") {
		return true
	}
	slug := normalizeFIFAKey(stringValue(event.Slug))
	if strings.HasPrefix(slug, "fif-") || strings.HasPrefix(slug, "fifwc-") {
		return true
	}
	for _, team := range teams {
		if strings.HasPrefix(normalizeFIFAKey(team.League), "fif") {
			return true
		}
	}
	return false
}

func fifaSportCode(raw json.RawMessage) string {
	var sport struct {
		Sport string `json:"sport,omitempty"`
	}
	if err := json.Unmarshal(raw, &sport); err == nil {
		return strings.TrimSpace(sport.Sport)
	}
	return ""
}

func polymarketEventURL(event *utilpolymarket.Event) string {
	slug := strings.TrimSpace(stringValue(event.Slug))
	if slug == "" {
		return ""
	}
	return "https://polymarket.com/event/" + slug
}
