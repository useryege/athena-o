package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/jackc/pgx/v5"
	q "github.com/useryege/athena/internal/tradersync/store/sqlc"
	tm "github.com/useryege/athena/internal/tradersync/types"
	pm "github.com/useryege/athena/util/polymarket"
	"math/big"
	"slices"
	"strings"
	"time"
)

func (s *SQLStore) LoadMetadata(ctx context.Context, key string) (tm.TradeMetadata, bool, error) {
	var result tm.TradeMetadata
	b, e := q.New(s.pool).GetTradeMetadata(ctx, key)
	if errors.Is(e, pgx.ErrNoRows) {
		return result, false, nil
	}
	if e != nil {
		return result, false, e
	}
	e = json.Unmarshal(b, &result)
	return result, e == nil, e
}
func (s *SQLStore) SaveMetadata(ctx context.Context, key string, result tm.TradeMetadata) error {
	b, e := json.Marshal(result)
	if e != nil {
		return e
	}
	return q.New(s.pool).SaveTradeMetadata(ctx, q.SaveTradeMetadataParams{CacheKey: key, MetadataJson: b})
}

// LookupComboPosition returns directory claims, not verified MarketRefs.
// The resolver must check the exact Gamma market before marking available.
func (s *SQLStore) LookupComboPosition(ctx context.Context, position string) ([]pm.ComboMarket, error) {
	rows, e := q.New(s.pool).LookupComboPosition(ctx, position)
	if e != nil {
		return nil, e
	}
	result := make([]pm.ComboMarket, 0, len(rows))
	for _, row := range rows {
		m := pm.ComboMarket{ID: row.MarketID, ConditionID: row.ConditionID}
		if e = json.Unmarshal(row.PositionIds, &m.PositionIDs); e != nil {
			return nil, e
		}
		result = append(result, m)
	}
	return result, nil
}

// RefreshComboPage serializes only the directory state row, never an account or
// wallet gate. The request has a five-second deadline and its mappings/cursor
// commit together. Provider failures preserve the cursor but commit page pacing.
func (s *SQLStore) RefreshComboPage(ctx context.Context, fetch func(context.Context, string, int) (pm.ComboMarketPage, error)) (time.Time, error) {
	tx, e := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if e != nil {
		return time.Time{}, e
	}
	defer tx.Rollback(context.Background())
	queries := q.New(tx)
	if e = queries.EnsureComboDirectory(ctx); e != nil {
		return time.Time{}, e
	}
	state, e := queries.LockComboDirectory(ctx)
	if e != nil {
		return time.Time{}, e
	}
	now, ok := state.DatabaseNow.(time.Time)
	if !ok {
		return time.Time{}, fmt.Errorf("database clock unavailable")
	}
	if state.NextPageAt.Time.After(now) {
		return state.NextPageAt.Time, tx.Commit(ctx)
	}
	if e = queries.StartComboRound(ctx); e != nil {
		return time.Time{}, e
	}
	// Re-read under the same row lock after resetting a new round's visited list.
	state, e = queries.LockComboDirectory(ctx)
	if e != nil {
		return time.Time{}, e
	}
	callCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	page, pageErr := fetch(callCtx, state.Cursor, 100)
	cancel()
	if pageErr == nil {
		pageErr = validateComboPage(page, state.Cursor, state.VisitedCursors)
	}
	if pageErr != nil {
		at, e := queries.DelayComboPage(ctx)
		if e != nil {
			return time.Time{}, e
		}
		if e = tx.Commit(ctx); e != nil {
			return time.Time{}, e
		}
		return at.Time, pageErr
	}
	for _, market := range page.Markets {
		ids, e := json.Marshal(market.PositionIDs)
		if e != nil {
			return time.Time{}, e
		}
		for _, position := range market.PositionIDs {
			if e = queries.UpsertComboPosition(ctx, q.UpsertComboPositionParams{PositionID: position, MarketID: market.ID, ConditionID: market.ConditionID, PositionIds: ids}); e != nil {
				return time.Time{}, e
			}
		}
	}
	at, e := queries.AdvanceComboPage(ctx, q.AdvanceComboPageParams{NextCursor: page.NextCursor, ExpectedCursor: state.Cursor})
	if e != nil {
		return time.Time{}, e
	}
	return at.Time, tx.Commit(ctx)
}
func validateComboPage(page pm.ComboMarketPage, cursor string, visited []string) error {
	if page.Markets == nil || len(page.Markets) > 100 {
		return fmt.Errorf("incomplete or oversized directory page")
	}
	if page.NextCursor != "" && (page.NextCursor == cursor || slices.Contains(visited, page.NextCursor)) {
		return fmt.Errorf("directory cursor cycle")
	}
	markets := map[string]pm.ComboMarket{}
	for _, m := range page.Markets {
		if !positiveDecimal(m.ID) || strings.TrimSpace(m.ConditionID) == "" || len(m.PositionIDs) == 0 {
			return fmt.Errorf("invalid directory market")
		}
		if old, ok := markets[m.ID]; ok && (old.ConditionID != m.ConditionID || !slices.Equal(old.PositionIDs, m.PositionIDs)) {
			return fmt.Errorf("conflicting directory market")
		}
		markets[m.ID] = m
		seen := map[string]bool{}
		for _, id := range m.PositionIDs {
			n, ok := new(big.Int).SetString(id, 10)
			if !positiveDecimal(id) || !ok || n.BitLen() > 256 || seen[id] {
				return fmt.Errorf("invalid directory position")
			}
			seen[id] = true
		}
	}
	return nil
}
func positiveDecimal(s string) bool {
	if len(s) == 0 || s[0] == '0' {
		return false
	}
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}
