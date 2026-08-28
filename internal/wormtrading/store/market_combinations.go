package store

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/gagliardetto/solana-go"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	wormtradingsqlc "github.com/useryege/athena/internal/wormtrading/store/sqlc"
)

const (
	maxMarketCombinationNameLength = 80
	maxMarketSnapshotTitleLength   = 500
	maxMarketSnapshotLogoLength    = 2048
	maxMarketOutcomeLabelLength    = 100
	maxMarketConditionIDLength     = 64
	minMarketConditionIDLength     = 32
	maxMarketCombinationPageSize   = 100
)

func (s *SQLStore) CreateMarketCombination(
	ctx context.Context,
	ownerAccountID string,
	name string,
	items []MarketCombinationItemInput,
) (*MarketCombination, error) {
	ownerUUID, ownerAccountID, err := marketCombinationUUID(ownerAccountID, "owner account ID")
	if err != nil {
		return nil, invalidMarketCombination(err)
	}
	name, items, err = normalizeMarketCombination(name, items)
	if err != nil {
		return nil, err
	}
	if err := s.requireDatabase(); err != nil {
		return nil, err
	}

	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, fmt.Errorf("begin create market combination transaction: %w", err)
	}
	defer rollbackWalletTransaction(tx)
	queries := wormtradingsqlc.New(tx)
	now := canonicalNow(time.Now())
	id := uuid.New()
	combinationID := pgtype.UUID{Bytes: [16]byte(id), Valid: true}
	row, err := queries.CreateMarketCombination(ctx, wormtradingsqlc.CreateMarketCombinationParams{
		ID:             combinationID,
		OwnerAccountID: ownerUUID,
		Name:           name,
		Now:            timestampParam(now),
	})
	if err != nil {
		if isMarketCombinationConstraint(err, "worm_market_combinations_owner_name_key_unique") {
			return nil, ErrMarketCombinationExists
		}
		return nil, fmt.Errorf("create market combination: %w", err)
	}
	if err := insertMarketCombinationItems(ctx, queries, combinationID, items); err != nil {
		return nil, err
	}
	created, err := loadMarketCombination(ctx, queries, row)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit create market combination: %w", err)
	}
	created.OwnerAccountID = ownerAccountID
	return &created, nil
}

func (s *SQLStore) GetMarketCombination(
	ctx context.Context,
	ownerAccountID string,
	combinationID string,
) (*MarketCombination, error) {
	ownerUUID, ownerAccountID, err := marketCombinationUUID(ownerAccountID, "owner account ID")
	if err != nil {
		return nil, invalidMarketCombination(err)
	}
	id, _, err := marketCombinationUUID(combinationID, "market combination ID")
	if err != nil {
		return nil, invalidMarketCombination(err)
	}
	if err := s.requireDatabase(); err != nil {
		return nil, err
	}
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if err != nil {
		return nil, fmt.Errorf("begin get market combination transaction: %w", err)
	}
	defer rollbackWalletTransaction(tx)
	queries := wormtradingsqlc.New(tx)
	row, err := queries.GetMarketCombination(ctx, wormtradingsqlc.GetMarketCombinationParams{
		ID:             id,
		OwnerAccountID: ownerUUID,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrMarketCombinationNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get market combination: %w", err)
	}
	combination, err := loadMarketCombination(ctx, queries, row)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit get market combination: %w", err)
	}
	combination.OwnerAccountID = ownerAccountID
	return &combination, nil
}

func (s *SQLStore) ListMarketCombinations(
	ctx context.Context,
	ownerAccountID string,
	page int32,
	pageSize int32,
) ([]MarketCombination, int64, error) {
	ownerUUID, ownerAccountID, err := marketCombinationUUID(ownerAccountID, "owner account ID")
	if err != nil {
		return nil, 0, invalidMarketCombination(err)
	}
	if page <= 0 || pageSize <= 0 || pageSize > maxMarketCombinationPageSize {
		return nil, 0, invalidMarketCombination(fmt.Errorf("page must be positive and page size must be between 1 and %d", maxMarketCombinationPageSize))
	}
	pageOffset := int64(page-1) * int64(pageSize)
	if pageOffset < 0 {
		return nil, 0, invalidMarketCombination(fmt.Errorf("page offset is invalid"))
	}
	if err := s.requireDatabase(); err != nil {
		return nil, 0, err
	}
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if err != nil {
		return nil, 0, fmt.Errorf("begin list market combinations transaction: %w", err)
	}
	defer rollbackWalletTransaction(tx)
	queries := wormtradingsqlc.New(tx)
	total, err := queries.CountMarketCombinations(ctx, ownerUUID)
	if err != nil {
		return nil, 0, fmt.Errorf("count market combinations: %w", err)
	}
	rows, err := queries.ListMarketCombinations(ctx, wormtradingsqlc.ListMarketCombinationsParams{
		OwnerAccountID: ownerUUID,
		PageSize:       pageSize,
		PageOffset:     pageOffset,
	})
	if err != nil {
		return nil, 0, fmt.Errorf("list market combinations: %w", err)
	}
	combinations := make([]MarketCombination, 0, len(rows))
	for _, row := range rows {
		combination, err := loadMarketCombination(ctx, queries, row)
		if err != nil {
			return nil, 0, err
		}
		combination.OwnerAccountID = ownerAccountID
		combinations = append(combinations, combination)
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, 0, fmt.Errorf("commit list market combinations: %w", err)
	}
	return combinations, total, nil
}

func (s *SQLStore) UpdateMarketCombination(
	ctx context.Context,
	ownerAccountID string,
	combinationID string,
	name string,
	expectedRevision int64,
	items []MarketCombinationItemInput,
) (*MarketCombination, error) {
	ownerUUID, ownerAccountID, err := marketCombinationUUID(ownerAccountID, "owner account ID")
	if err != nil {
		return nil, invalidMarketCombination(err)
	}
	id, _, err := marketCombinationUUID(combinationID, "market combination ID")
	if err != nil {
		return nil, invalidMarketCombination(err)
	}
	if expectedRevision <= 0 {
		return nil, invalidMarketCombination(fmt.Errorf("expected revision must be positive"))
	}
	name, items, err = normalizeMarketCombination(name, items)
	if err != nil {
		return nil, err
	}
	if err := s.requireDatabase(); err != nil {
		return nil, err
	}

	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, fmt.Errorf("begin update market combination transaction: %w", err)
	}
	defer rollbackWalletTransaction(tx)
	queries := wormtradingsqlc.New(tx)
	current, err := queries.GetMarketCombinationForUpdate(ctx, wormtradingsqlc.GetMarketCombinationForUpdateParams{
		ID:             id,
		OwnerAccountID: ownerUUID,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrMarketCombinationNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("lock market combination: %w", err)
	}
	if current.Revision != expectedRevision {
		return nil, ErrMarketCombinationRevision
	}
	now := canonicalNow(time.Now())
	updatedRow, err := queries.UpdateMarketCombination(ctx, wormtradingsqlc.UpdateMarketCombinationParams{
		Name:             name,
		Now:              timestampParam(now),
		ID:               id,
		OwnerAccountID:   ownerUUID,
		ExpectedRevision: expectedRevision,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrMarketCombinationRevision
	}
	if err != nil {
		if isMarketCombinationConstraint(err, "worm_market_combinations_owner_name_key_unique") {
			return nil, ErrMarketCombinationExists
		}
		return nil, fmt.Errorf("update market combination: %w", err)
	}
	if err := queries.DeleteMarketCombinationItems(ctx, id); err != nil {
		return nil, fmt.Errorf("delete prior market combination items: %w", err)
	}
	if err := insertMarketCombinationItems(ctx, queries, id, items); err != nil {
		return nil, err
	}
	updated, err := loadMarketCombination(ctx, queries, updatedRow)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit update market combination: %w", err)
	}
	updated.OwnerAccountID = ownerAccountID
	return &updated, nil
}

func (s *SQLStore) DeleteMarketCombination(
	ctx context.Context,
	ownerAccountID string,
	combinationID string,
	expectedRevision int64,
) error {
	ownerUUID, _, err := marketCombinationUUID(ownerAccountID, "owner account ID")
	if err != nil {
		return invalidMarketCombination(err)
	}
	id, _, err := marketCombinationUUID(combinationID, "market combination ID")
	if err != nil {
		return invalidMarketCombination(err)
	}
	if expectedRevision <= 0 {
		return invalidMarketCombination(fmt.Errorf("expected revision must be positive"))
	}
	if err := s.requireDatabase(); err != nil {
		return err
	}
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("begin delete market combination transaction: %w", err)
	}
	defer rollbackWalletTransaction(tx)
	queries := wormtradingsqlc.New(tx)
	current, err := queries.GetMarketCombinationForUpdate(ctx, wormtradingsqlc.GetMarketCombinationForUpdateParams{
		ID:             id,
		OwnerAccountID: ownerUUID,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrMarketCombinationNotFound
	}
	if err != nil {
		return fmt.Errorf("lock market combination for delete: %w", err)
	}
	if current.Revision != expectedRevision {
		return ErrMarketCombinationRevision
	}
	if _, err := queries.DeleteMarketCombination(ctx, wormtradingsqlc.DeleteMarketCombinationParams{
		ID:               id,
		OwnerAccountID:   ownerUUID,
		ExpectedRevision: expectedRevision,
	}); errors.Is(err, pgx.ErrNoRows) {
		return ErrMarketCombinationRevision
	} else if err != nil {
		return fmt.Errorf("delete market combination: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit delete market combination: %w", err)
	}
	return nil
}

func normalizeMarketCombination(name string, items []MarketCombinationItemInput) (string, []MarketCombinationItemInput, error) {
	name = strings.TrimSpace(name)
	if name == "" || utf8.RuneCountInString(name) > maxMarketCombinationNameLength {
		return "", nil, invalidMarketCombination(fmt.Errorf("name must contain between 1 and %d characters", maxMarketCombinationNameLength))
	}
	if len(items) == 0 {
		return "", nil, invalidMarketCombination(fmt.Errorf("at least one market is required"))
	}
	if len(items) > math.MaxInt32 {
		return "", nil, invalidMarketCombination(fmt.Errorf("too many market items"))
	}
	normalized := make([]MarketCombinationItemInput, 0, len(items))
	seenMarkets := make(map[string]struct{}, len(items))
	for index, item := range items {
		item.EventConditionID = strings.TrimSpace(item.EventConditionID)
		item.EventTitle = strings.TrimSpace(item.EventTitle)
		item.EventLogo = strings.TrimSpace(item.EventLogo)
		item.MarketConditionID = strings.TrimSpace(item.MarketConditionID)
		item.MarketTitle = strings.TrimSpace(item.MarketTitle)
		item.MarketLogo = strings.TrimSpace(item.MarketLogo)
		item.OutcomeLabel = strings.TrimSpace(item.OutcomeLabel)
		if !validMarketConditionID(item.EventConditionID) {
			return "", nil, invalidMarketCombination(fmt.Errorf("item %d event condition ID is invalid", index+1))
		}
		if !validMarketConditionID(item.MarketConditionID) {
			return "", nil, invalidMarketCombination(fmt.Errorf("item %d market condition ID is invalid", index+1))
		}
		if item.EventTitle == "" || utf8.RuneCountInString(item.EventTitle) > maxMarketSnapshotTitleLength {
			return "", nil, invalidMarketCombination(fmt.Errorf("item %d event title is invalid", index+1))
		}
		if item.MarketTitle == "" || utf8.RuneCountInString(item.MarketTitle) > maxMarketSnapshotTitleLength {
			return "", nil, invalidMarketCombination(fmt.Errorf("item %d market title is invalid", index+1))
		}
		if utf8.RuneCountInString(item.EventLogo) > maxMarketSnapshotLogoLength || utf8.RuneCountInString(item.MarketLogo) > maxMarketSnapshotLogoLength {
			return "", nil, invalidMarketCombination(fmt.Errorf("item %d logo is invalid", index+1))
		}
		if item.OutcomeLabel == "" || utf8.RuneCountInString(item.OutcomeLabel) > maxMarketOutcomeLabelLength {
			return "", nil, invalidMarketCombination(fmt.Errorf("item %d outcome label is invalid", index+1))
		}
		if _, exists := seenMarkets[item.MarketConditionID]; exists {
			return "", nil, invalidMarketCombination(fmt.Errorf("market condition ID %q is duplicated", item.MarketConditionID))
		}
		seenMarkets[item.MarketConditionID] = struct{}{}
		normalized = append(normalized, item)
	}
	return name, normalized, nil
}

func validMarketConditionID(value string) bool {
	if len(value) < minMarketConditionIDLength || len(value) > maxMarketConditionIDLength {
		return false
	}
	publicKey, err := solana.PublicKeyFromBase58(value)
	return err == nil && publicKey.String() == value
}

func marketCombinationUUID(value, label string) (pgtype.UUID, string, error) {
	parsed, err := uuid.Parse(strings.TrimSpace(value))
	if err != nil || parsed == uuid.Nil {
		return pgtype.UUID{}, "", fmt.Errorf("%s must be a non-zero UUID", label)
	}
	return pgtype.UUID{Bytes: [16]byte(parsed), Valid: true}, parsed.String(), nil
}

func invalidMarketCombination(err error) error {
	return fmt.Errorf("%w: %v", ErrInvalidMarketCombination, err)
}

func insertMarketCombinationItems(
	ctx context.Context,
	queries *wormtradingsqlc.Queries,
	combinationID pgtype.UUID,
	items []MarketCombinationItemInput,
) error {
	for index, item := range items {
		err := queries.CreateMarketCombinationItem(ctx, wormtradingsqlc.CreateMarketCombinationItemParams{
			CombinationID:     combinationID,
			Ordinal:           int32(index + 1),
			EventConditionID:  item.EventConditionID,
			EventTitle:        item.EventTitle,
			EventLogo:         item.EventLogo,
			MarketConditionID: item.MarketConditionID,
			MarketTitle:       item.MarketTitle,
			MarketLogo:        item.MarketLogo,
			IsYes:             item.IsYes,
			OutcomeLabel:      item.OutcomeLabel,
		})
		if err != nil {
			if isMarketCombinationConstraint(err, "worm_market_combination_items_market_unique") {
				return invalidMarketCombination(fmt.Errorf("market condition IDs must be unique"))
			}
			return fmt.Errorf("create market combination item %d: %w", index+1, err)
		}
	}
	return nil
}

func loadMarketCombination(
	ctx context.Context,
	queries *wormtradingsqlc.Queries,
	row wormtradingsqlc.WormMarketCombination,
) (MarketCombination, error) {
	itemRows, err := queries.ListMarketCombinationItems(ctx, row.ID)
	if err != nil {
		return MarketCombination{}, fmt.Errorf("list market combination items: %w", err)
	}
	combination := MarketCombination{
		ID:             uuidValue(row.ID),
		OwnerAccountID: uuidValue(row.OwnerAccountID),
		Name:           row.Name,
		Revision:       row.Revision,
		Items:          make([]MarketCombinationItem, 0, len(itemRows)),
		CreatedAt:      timestampValue(row.CreatedAt),
		UpdatedAt:      timestampValue(row.UpdatedAt),
	}
	for _, itemRow := range itemRows {
		combination.Items = append(combination.Items, MarketCombinationItem{
			Ordinal: itemRow.Ordinal,
			MarketCombinationItemInput: MarketCombinationItemInput{
				EventConditionID:  itemRow.EventConditionID,
				EventTitle:        itemRow.EventTitle,
				EventLogo:         itemRow.EventLogo,
				MarketConditionID: itemRow.MarketConditionID,
				MarketTitle:       itemRow.MarketTitle,
				MarketLogo:        itemRow.MarketLogo,
				IsYes:             itemRow.IsYes,
				OutcomeLabel:      itemRow.OutcomeLabel,
			},
		})
	}
	return combination, nil
}

func isMarketCombinationConstraint(err error, name string) bool {
	var postgresError *pgconn.PgError
	return errors.As(err, &postgresError) && postgresError.ConstraintName == name
}
