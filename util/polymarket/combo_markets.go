package polymarket

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

type ComboMarketPage struct {
	Markets    []ComboMarket `json:"markets"`
	NextCursor string        `json:"next_cursor"`
}
type ComboMarket struct {
	ID          string   `json:"id"`
	PositionIDs []string `json:"position_ids"`
	ConditionID string   `json:"condition_id"`
}

func (c *gammaClientImpl) ListComboMarkets(ctx context.Context, cursor string, limit int) (ComboMarketPage, error) {
	var page ComboMarketPage
	if limit < 1 || limit > 100 {
		return page, fmt.Errorf("combo page limit must be 1..100")
	}
	endpoint, err := url.Parse(strings.TrimRight(c.config.ComboBaseURL, "/") + "/v1/rfq/combo-markets")
	if err != nil {
		return page, err
	}
	q := url.Values{"limit": {strconv.Itoa(limit)}}
	if cursor != "" {
		q.Set("cursor", cursor)
	}
	endpoint.RawQuery = q.Encode()
	ctx, cancel := context.WithTimeout(ctx, min(c.config.Timeout, 5*time.Second))
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), http.NoBody)
	if err != nil {
		return page, err
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return page, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return page, decodeHTTPError(resp)
	}
	if err = json.NewDecoder(resp.Body).Decode(&page); err != nil {
		return page, err
	}
	if page.Markets == nil || len(page.Markets) > limit {
		return ComboMarketPage{}, fmt.Errorf("incomplete or oversized combo page")
	}
	return page, nil
}
