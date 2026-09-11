package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/useryege/athena/internal/tradersync/activity"
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
	tx, e := s.pool.Begin(ctx)
	if e != nil {
		return e
	}
	defer tx.Rollback(context.Background())
	if e = s.saveMetadataTx(ctx, tx, key, result); e != nil {
		return e
	}
	return tx.Commit(ctx)
}
func (s *SQLStore) saveMetadataTx(ctx context.Context, tx pgx.Tx, key string, result tm.TradeMetadata) error {
	queries := q.New(tx)
	if e := queries.LockTradeMetadata(ctx, key); e != nil {
		return e
	}
	raw, e := queries.GetTradeMetadata(ctx, key)
	if e != nil && !errors.Is(e, pgx.ErrNoRows) {
		return e
	}
	if e == nil {
		var old tm.TradeMetadata
		if e = json.Unmarshal(raw, &old); e != nil {
			return e
		}
		result = activity.MergeMetadata(old, result)
	}
	b, e := json.Marshal(result)
	if e != nil {
		return e
	}
	return queries.SaveTradeMetadata(ctx, q.SaveTradeMetadataParams{CacheKey: key, MetadataJson: b})
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

// RefreshComboPage reserves a shared six-second recovery window before HTTP.
// The second transaction retains the directory row lock and the original five-
// second budget. Returned times are local monotonic wakeups computed from a
// confirmed database delta, not assumptions about synchronized wall clocks.
func (s *SQLStore) RefreshComboPage(ctx context.Context, fetch func(context.Context, string, int) (pm.ComboMarketPage, error)) (next time.Time, resultErr error) {
	if fetch == nil {
		return time.Time{}, &tm.DirectoryError{Phase: "configuration", CommitKnown: true, Err: errors.New("directory fetch required")}
	}
	phase := "begin_admission"
	sent, commitKnown := false, true
	var tx pgx.Tx
	defer func() {
		var cleanup error
		if tx != nil {
			closeCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), time.Second)
			cleanup = tx.Rollback(closeCtx)
			cancel()
			if errors.Is(cleanup, pgx.ErrTxClosed) {
				cleanup = nil
			}
		}
		if resultErr != nil || cleanup != nil {
			resultErr = &tm.DirectoryError{Phase: phase, HTTPAttempted: sent, CommitKnown: commitKnown, Err: resultErr, CleanupErr: cleanup}
		}
	}()
	var e error
	tx, e = s.beginDirectoryTx(ctx)
	if e != nil {
		return time.Time{}, e
	}
	queries := q.New(tx)
	phase = "lock_admission"
	if e = queries.EnsureComboDirectory(ctx); e != nil {
		return time.Time{}, e
	}
	state, e := queries.LockComboDirectory(ctx)
	if e != nil {
		return time.Time{}, e
	}
	if state.NextPageAt.Time.After(state.DatabaseNow.Time) {
		next = directoryWakeup(state.NextPageAt.Time, state.DatabaseNow.Time)
		// This transaction did not mutate state or consume an admission.
		e = tx.Rollback(ctx)
		tx = nil
		return next, e
	}
	if e = queries.StartComboRound(ctx); e != nil {
		return time.Time{}, e
	}
	state, e = queries.LockComboDirectory(ctx)
	if e != nil {
		return time.Time{}, e
	}
	token := pgtype.UUID{Bytes: uuid.New(), Valid: true}
	// Establish this deadline BEFORE the admission SQL's database timestamp.
	// Neither a slow commit acknowledgement nor a second lock wait resets it.
	deadlineCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	phase = "reserve_admission"
	if _, e = queries.AdmitComboPage(deadlineCtx, q.AdmitComboPageParams{AdmissionID: token, ExpectedCursor: state.Cursor}); e != nil {
		return time.Time{}, e
	}
	phase = "commit_admission"
	commitKnown = false
	if e = tx.Commit(deadlineCtx); e != nil {
		return time.Time{}, e
	}
	commitKnown = true
	tx = nil
	if e = deadlineCtx.Err(); e != nil {
		return time.Time{}, e
	}
	phase = "begin_page"
	tx, e = s.beginDirectoryTx(deadlineCtx)
	if e != nil {
		return time.Time{}, e
	}
	queries = q.New(tx)
	phase = "lock_page"
	current, e := queries.LockComboDirectory(deadlineCtx)
	if e != nil {
		return time.Time{}, e
	}
	if current.AdmissionID != token || current.Cursor != state.Cursor {
		return time.Time{}, errors.New("directory admission superseded")
	}
	if e = deadlineCtx.Err(); e != nil {
		return time.Time{}, e
	}
	phase = "fetch"
	sent = true
	page, pageErr := fetch(deadlineCtx, state.Cursor, 100)
	pageErr = errors.Join(pageErr, deadlineCtx.Err())
	// Only the already-attempted page is allowed bounded detached settlement.
	// A failure leaves TX1's reservation; no new connection replays old results.
	cleanupCtx, cleanupCancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
	defer cleanupCancel()
	if pageErr == nil {
		pageErr = validateComboPage(page, state.Cursor, state.VisitedCursors)
	}
	if pageErr != nil {
		phase = "persist_pacing"
		at, e := queries.DelayComboPage(cleanupCtx, q.DelayComboPageParams{AdmissionID: token, ExpectedCursor: state.Cursor})
		if e != nil {
			return time.Time{}, errors.Join(pageErr, e)
		}
		phase = "commit_pacing"
		commitKnown = false
		if e = tx.Commit(cleanupCtx); e != nil {
			return time.Time{}, errors.Join(pageErr, e)
		}
		commitKnown = true
		tx = nil
		return directoryWakeup(at.NextPageAt.Time, at.DatabaseNow.Time), pageErr
	}
	phase = "persist_mapping"
	for _, market := range page.Markets {
		ids, e := json.Marshal(market.PositionIDs)
		if e != nil {
			return time.Time{}, e
		}
		for _, position := range market.PositionIDs {
			if e = queries.UpsertComboPosition(cleanupCtx, q.UpsertComboPositionParams{PositionID: position, MarketID: market.ID, ConditionID: market.ConditionID, PositionIds: ids}); e != nil {
				return time.Time{}, e
			}
		}
	}
	phase = "advance_page"
	at, e := queries.AdvanceComboPage(cleanupCtx, q.AdvanceComboPageParams{NextCursor: page.NextCursor, ExpectedCursor: state.Cursor, AdmissionID: token})
	if e != nil {
		return time.Time{}, e
	}
	phase = "commit_page"
	commitKnown = false
	if e = tx.Commit(cleanupCtx); e != nil {
		return time.Time{}, e
	}
	commitKnown = true
	tx = nil
	return directoryWakeup(at.NextPageAt.Time, at.DatabaseNow.Time), nil
}
func (s *SQLStore) beginDirectoryTx(ctx context.Context) (pgx.Tx, error) {
	if s.directoryTransactions != nil {
		return s.directoryTransactions.BeginTx(ctx, pgx.TxOptions{})
	}
	return s.pool.BeginTx(ctx, pgx.TxOptions{})
}
func directoryWakeup(next, now time.Time) time.Time { return time.Now().Add(next.Sub(now)) }
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
