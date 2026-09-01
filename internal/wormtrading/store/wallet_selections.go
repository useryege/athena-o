package store

import (
	"context"
	"errors"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/gagliardetto/solana-go"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	wormtradingsqlc "github.com/useryege/athena/internal/wormtrading/store/sqlc"
)

func (s *SQLStore) GetWalletSelection(ctx context.Context, ownerAccountID string) (*WalletSelection, error) {
	ownerUUID, ownerAccountID, err := walletSelectionOwner(ownerAccountID)
	if err != nil {
		return nil, err
	}
	if err := s.requireDatabase(); err != nil {
		return nil, err
	}
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if err != nil {
		return nil, fmt.Errorf("begin get Wallet selection transaction: %w", err)
	}
	defer rollbackWalletTransaction(tx)
	queries := wormtradingsqlc.New(tx)
	row, err := queries.GetWalletSelection(ctx, ownerUUID)
	if errors.Is(err, pgx.ErrNoRows) {
		if err := tx.Commit(ctx); err != nil {
			return nil, fmt.Errorf("commit unconfigured Wallet selection read: %w", err)
		}
		return &WalletSelection{OwnerAccountID: ownerAccountID}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get Wallet selection: %w", err)
	}
	selection, err := loadWalletSelection(ctx, queries, row)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit get Wallet selection: %w", err)
	}
	selection.OwnerAccountID = ownerAccountID
	return &selection, nil
}

func (s *SQLStore) ReplaceWalletSelection(
	ctx context.Context,
	req ReplaceWalletSelectionRequest,
) (*WalletSelection, error) {
	ownerUUID, ownerAccountID, err := walletSelectionOwner(req.OwnerAccountID)
	if err != nil {
		return nil, err
	}
	if req.ExpectedRevision < 0 || req.ExpectedRevision == math.MaxInt64 {
		return nil, invalidWalletSelection(fmt.Errorf("expected revision must be an incrementable non-negative integer"))
	}
	selected, retiring, err := normalizeWalletSelectionInputs(req.SelectedItems, req.RetiringWallets)
	if err != nil {
		return nil, err
	}
	if err := s.requireDatabase(); err != nil {
		return nil, err
	}

	// Read only to determine the advisory-lock set. The authoritative revision
	// and rows are checked again after all Wallet locks are held.
	prior, err := s.GetWalletSelection(ctx, ownerAccountID)
	if err != nil {
		return nil, err
	}
	lockIDs := walletSelectionOperationIDs(prior, selected, retiring)
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, fmt.Errorf("begin replace Wallet selection transaction: %w", err)
	}
	defer rollbackWalletTransaction(tx)
	if len(lockIDs) > 0 {
		if err := lockWalletOperations(ctx, tx, lockIDs); err != nil {
			return nil, err
		}
	}
	queries := wormtradingsqlc.New(tx)
	current, getErr := queries.GetWalletSelectionForUpdate(ctx, ownerUUID)
	configured := getErr == nil
	if getErr != nil && !errors.Is(getErr, pgx.ErrNoRows) {
		return nil, fmt.Errorf("lock Wallet selection: %w", getErr)
	}
	if (!configured && req.ExpectedRevision != 0) ||
		(configured && current.Revision != req.ExpectedRevision) {
		return nil, ErrWalletSelectionRevision
	}

	oldItems := []wormtradingsqlc.WormTradingWalletSelectionItem{}
	if configured {
		oldItems, err = queries.ListWalletSelectionItems(ctx, ownerUUID)
		if err != nil {
			return nil, fmt.Errorf("list prior Wallet selection items: %w", err)
		}
		if int32(len(oldItems)) != current.SelectedWalletCount {
			return nil, invalidWalletSelection(fmt.Errorf("stored selected Wallet count is inconsistent"))
		}
	}

	selectedByID := make(map[int64]WalletSelectionInput, len(selected))
	for _, item := range selected {
		selectedByID[item.WalletID] = item
	}
	removed := make([]wormtradingsqlc.WormTradingWalletSelectionItem, 0, len(oldItems))
	removedIDs := make([]int64, 0, len(oldItems))
	for _, item := range oldItems {
		if selectedItem, remainsSelected := selectedByID[item.WalletID]; remainsSelected {
			if selectedItem.Address != item.Address {
				return nil, invalidWalletSelection(fmt.Errorf("a selected Wallet address cannot change"))
			}
			continue
		}
		removed = append(removed, item)
		removedIDs = append(removedIDs, item.WalletID)
	}
	blockedIDSet := make(map[int64]struct{}, len(removedIDs)+len(retiring))
	for _, walletID := range removedIDs {
		blockedIDSet[walletID] = struct{}{}
	}
	for _, item := range retiring {
		blockedIDSet[item.WalletID] = struct{}{}
	}
	blockedIDs := make([]int64, 0, len(blockedIDSet))
	for walletID := range blockedIDSet {
		blockedIDs = append(blockedIDs, walletID)
	}
	sort.Slice(blockedIDs, func(left, right int) bool { return blockedIDs[left] < blockedIDs[right] })
	if len(blockedIDs) > 0 {
		blockers, err := listWalletRetirementBlockers(ctx, queries, ownerUUID, blockedIDs)
		if err != nil {
			return nil, err
		}
		if len(blockers) > 0 {
			return nil, &WalletSelectionRetirementBlockedError{
				WalletID: blockers[0].WalletID, ReasonCode: blockers[0].ReasonCode,
			}
		}
	}

	now := canonicalNow(req.Now)
	var header wormtradingsqlc.WormTradingWalletSelection
	if configured {
		header, err = queries.UpdateWalletSelection(ctx, wormtradingsqlc.UpdateWalletSelectionParams{
			SelectedWalletCount: int32(len(selected)),
			Now:                 timestampParam(now),
			OwnerAccountID:      ownerUUID,
			ExpectedRevision:    req.ExpectedRevision,
		})
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrWalletSelectionRevision
		}
		if err != nil {
			return nil, fmt.Errorf("update Wallet selection: %w", err)
		}
	} else {
		header, err = queries.CreateWalletSelection(ctx, wormtradingsqlc.CreateWalletSelectionParams{
			OwnerAccountID:      ownerUUID,
			SelectedWalletCount: int32(len(selected)),
			Now:                 timestampParam(now),
		})
		if walletSelectionConstraint(err, "worm_trading_wallet_selections_pkey") {
			return nil, ErrWalletSelectionRevision
		}
		if err != nil {
			return nil, fmt.Errorf("create Wallet selection: %w", err)
		}
	}

	for _, item := range removed {
		if err := persistWalletRetirementIfNeeded(ctx, queries, ownerUUID, item.WalletID, item.Address, item.Ordinal, header.Revision, now); err != nil {
			return nil, err
		}
	}
	for _, item := range retiring {
		existing, err := queries.GetWalletRetirementByWalletID(ctx, wormtradingsqlc.GetWalletRetirementByWalletIDParams{
			OwnerAccountID: ownerUUID,
			WalletID:       item.WalletID,
		})
		if err == nil && existing.Address != item.Address {
			return nil, invalidWalletSelection(fmt.Errorf("a retiring Wallet address cannot change"))
		}
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("get existing Wallet retirement: %w", err)
		}
		if err := persistWalletRetirementIfNeeded(ctx, queries, ownerUUID, item.WalletID, item.Address, item.PriorOrdinal, header.Revision, now); err != nil {
			return nil, err
		}
	}
	for _, item := range selected {
		existing, err := queries.GetWalletRetirementByWalletID(ctx, wormtradingsqlc.GetWalletRetirementByWalletIDParams{
			OwnerAccountID: ownerUUID,
			WalletID:       item.WalletID,
		})
		if err == nil && existing.Address != item.Address {
			return nil, invalidWalletSelection(fmt.Errorf("a reselected Wallet address cannot change"))
		}
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("get reselected Wallet retirement: %w", err)
		}
		blocked, err := queries.WalletRetirementReselectionBlocked(ctx, wormtradingsqlc.WalletRetirementReselectionBlockedParams{
			OwnerAccountID: ownerUUID,
			WalletID:       item.WalletID,
		})
		if err != nil {
			return nil, fmt.Errorf("check reselected Wallet retirement lifecycle: %w", err)
		}
		if blocked {
			return nil, ErrWalletRetirementIncomplete
		}
		if err := queries.DeleteWalletRetirementForReselection(ctx, wormtradingsqlc.DeleteWalletRetirementForReselectionParams{
			OwnerAccountID: ownerUUID,
			WalletID:       item.WalletID,
		}); err != nil {
			return nil, fmt.Errorf("clear reselected Wallet retirement: %w", err)
		}
	}
	if err := queries.DeleteWalletSelectionItems(ctx, ownerUUID); err != nil {
		return nil, fmt.Errorf("delete prior Wallet selection items: %w", err)
	}
	for index, item := range selected {
		if err := queries.CreateWalletSelectionItem(ctx, wormtradingsqlc.CreateWalletSelectionItemParams{
			OwnerAccountID: ownerUUID,
			Ordinal:        int32(index + 1),
			WalletID:       item.WalletID,
			Address:        item.Address,
			Now:            timestampParam(now),
		}); err != nil {
			return nil, fmt.Errorf("create Wallet selection item %d: %w", index+1, err)
		}
	}

	selection, err := loadWalletSelection(ctx, queries, header)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit replace Wallet selection: %w", err)
	}
	selection.OwnerAccountID = ownerAccountID
	return &selection, nil
}

func (s *SQLStore) ListWalletRetirementBlockers(
	ctx context.Context,
	ownerAccountID string,
	walletIDs []int64,
) ([]WalletRetirementBlocker, error) {
	ownerUUID, _, err := walletSelectionOwner(ownerAccountID)
	if err != nil {
		return nil, err
	}
	walletIDs, err = normalizeWalletSelectionIDs(walletIDs)
	if err != nil {
		return nil, err
	}
	if len(walletIDs) == 0 {
		return []WalletRetirementBlocker{}, nil
	}
	if err := s.requireDatabase(); err != nil {
		return nil, err
	}
	return listWalletRetirementBlockers(ctx, s.queries, ownerUUID, walletIDs)
}

func (s *SQLStore) CompleteWalletRetirement(
	ctx context.Context,
	ownerAccountID string,
	walletID int64,
	address string,
) error {
	ownerUUID, _, err := walletSelectionOwner(ownerAccountID)
	if err != nil {
		return err
	}
	if err := validateCanonicalWalletSelectionReference(walletID, address); err != nil {
		return err
	}
	tx, queries, err := s.beginWalletTransaction(ctx, walletID)
	if err != nil {
		return err
	}
	defer rollbackWalletTransaction(tx)
	if _, err := queries.GetWalletRetirement(ctx, wormtradingsqlc.GetWalletRetirementParams{
		OwnerAccountID: ownerUUID,
		WalletID:       walletID,
		Address:        address,
	}); errors.Is(err, pgx.ErrNoRows) {
		return ErrWalletNotRetiring
	} else if err != nil {
		return fmt.Errorf("get Wallet retirement for completion: %w", err)
	}
	deleted, err := queries.DeleteWalletRetirement(ctx, wormtradingsqlc.DeleteWalletRetirementParams{
		OwnerAccountID: ownerUUID,
		WalletID:       walletID,
		Address:        address,
	})
	if err != nil {
		return fmt.Errorf("complete Wallet retirement: %w", err)
	}
	if deleted != 1 {
		return ErrWalletRetirementIncomplete
	}
	return commitWalletTransaction(ctx, tx)
}

func normalizeWalletSelectionInputs(
	selected []WalletSelectionInput,
	retiring []WalletRetirementInput,
) ([]WalletSelectionInput, []WalletRetirementInput, error) {
	if len(selected) > MaximumWalletSelectionItems {
		return nil, nil, invalidWalletSelection(fmt.Errorf("selected Wallets must contain at most %d items", MaximumWalletSelectionItems))
	}
	selectedIDs := make(map[int64]struct{}, len(selected))
	selectedAddresses := make(map[string]struct{}, len(selected))
	normalizedSelected := make([]WalletSelectionInput, len(selected))
	for index, item := range selected {
		if err := validateCanonicalWalletSelectionReference(item.WalletID, item.Address); err != nil {
			return nil, nil, invalidWalletSelection(fmt.Errorf("selected Wallet %d: %v", index+1, err))
		}
		if _, duplicate := selectedIDs[item.WalletID]; duplicate {
			return nil, nil, invalidWalletSelection(fmt.Errorf("selected Wallet IDs must be unique"))
		}
		if _, duplicate := selectedAddresses[item.Address]; duplicate {
			return nil, nil, invalidWalletSelection(fmt.Errorf("selected Wallet addresses must be unique"))
		}
		selectedIDs[item.WalletID] = struct{}{}
		selectedAddresses[item.Address] = struct{}{}
		normalizedSelected[index] = item
	}
	retiringIDs := make(map[int64]struct{}, len(retiring))
	retiringAddresses := make(map[string]struct{}, len(retiring))
	normalizedRetiring := make([]WalletRetirementInput, len(retiring))
	for index, item := range retiring {
		if err := validateCanonicalWalletSelectionReference(item.WalletID, item.Address); err != nil || item.PriorOrdinal <= 0 {
			return nil, nil, invalidWalletSelection(fmt.Errorf("retiring Wallet %d is invalid", index+1))
		}
		if _, duplicate := selectedIDs[item.WalletID]; duplicate {
			return nil, nil, invalidWalletSelection(fmt.Errorf("selected and retiring Wallets must be disjoint"))
		}
		if _, duplicate := selectedAddresses[item.Address]; duplicate {
			return nil, nil, invalidWalletSelection(fmt.Errorf("selected and retiring Wallet addresses must be disjoint"))
		}
		if _, duplicate := retiringIDs[item.WalletID]; duplicate {
			return nil, nil, invalidWalletSelection(fmt.Errorf("retiring Wallet IDs must be unique"))
		}
		if _, duplicate := retiringAddresses[item.Address]; duplicate {
			return nil, nil, invalidWalletSelection(fmt.Errorf("retiring Wallet addresses must be unique"))
		}
		retiringIDs[item.WalletID] = struct{}{}
		retiringAddresses[item.Address] = struct{}{}
		normalizedRetiring[index] = item
	}
	return normalizedSelected, normalizedRetiring, nil
}

func walletSelectionOwner(value string) (pgtype.UUID, string, error) {
	ownerUUID, canonical, err := marketCombinationUUID(value, "owner account ID")
	if err != nil {
		return pgtype.UUID{}, "", invalidWalletSelection(err)
	}
	return ownerUUID, canonical, nil
}

func validateCanonicalWalletSelectionReference(walletID int64, address string) error {
	if err := validateWalletReference(walletID, address); err != nil {
		return err
	}
	publicKey, err := solana.PublicKeyFromBase58(address)
	if err != nil || publicKey.String() != address {
		return fmt.Errorf("Wallet address must be a canonical Solana address")
	}
	return nil
}

func normalizeWalletSelectionIDs(values []int64) ([]int64, error) {
	if len(values) == 0 {
		return []int64{}, nil
	}
	normalized := append([]int64(nil), values...)
	sort.Slice(normalized, func(left, right int) bool { return normalized[left] < normalized[right] })
	for index, value := range normalized {
		if value <= 0 || (index > 0 && normalized[index-1] == value) {
			return nil, invalidWalletSelection(fmt.Errorf("Wallet IDs must be positive and unique"))
		}
	}
	return normalized, nil
}

func walletSelectionOperationIDs(
	prior *WalletSelection,
	selected []WalletSelectionInput,
	retiring []WalletRetirementInput,
) []int64 {
	ids := make(map[int64]struct{}, len(selected)+len(retiring))
	if prior != nil {
		for _, item := range prior.SelectedItems {
			ids[item.WalletID] = struct{}{}
		}
		for _, item := range prior.Retirements {
			ids[item.WalletID] = struct{}{}
		}
	}
	for _, item := range selected {
		ids[item.WalletID] = struct{}{}
	}
	for _, item := range retiring {
		ids[item.WalletID] = struct{}{}
	}
	result := make([]int64, 0, len(ids))
	for walletID := range ids {
		result = append(result, walletID)
	}
	sort.Slice(result, func(left, right int) bool { return result[left] < result[right] })
	return result
}

func persistWalletRetirementIfNeeded(
	ctx context.Context,
	queries *wormtradingsqlc.Queries,
	ownerUUID pgtype.UUID,
	walletID int64,
	address string,
	priorOrdinal int32,
	retiredFromRevision int64,
	now time.Time,
) error {
	needsRetirement, err := queries.WalletNeedsRetirement(ctx, wormtradingsqlc.WalletNeedsRetirementParams{
		WalletID: walletID,
		Address:  address,
	})
	if err != nil {
		return fmt.Errorf("check Wallet retirement requirement: %w", err)
	}
	if !needsRetirement {
		if _, err := queries.DeleteWalletRetirement(ctx, wormtradingsqlc.DeleteWalletRetirementParams{
			OwnerAccountID: ownerUUID,
			WalletID:       walletID,
			Address:        address,
		}); err != nil {
			return fmt.Errorf("clear completed Wallet retirement: %w", err)
		}
		return nil
	}
	if err := queries.UpsertWalletRetirement(ctx, wormtradingsqlc.UpsertWalletRetirementParams{
		OwnerAccountID:      ownerUUID,
		WalletID:            walletID,
		Address:             address,
		PriorOrdinal:        priorOrdinal,
		RetiredFromRevision: retiredFromRevision,
		Now:                 timestampParam(now),
	}); err != nil {
		return fmt.Errorf("persist Wallet retirement: %w", err)
	}
	return nil
}

func listWalletRetirementBlockers(
	ctx context.Context,
	queries *wormtradingsqlc.Queries,
	ownerUUID pgtype.UUID,
	walletIDs []int64,
) ([]WalletRetirementBlocker, error) {
	rows, err := queries.ListWalletRetirementBlockers(ctx, wormtradingsqlc.ListWalletRetirementBlockersParams{
		OwnerAccountID: ownerUUID,
		WalletIds:      walletIDs,
	})
	if err != nil {
		return nil, fmt.Errorf("list Wallet retirement blockers: %w", err)
	}
	result := make([]WalletRetirementBlocker, 0, len(rows))
	for _, row := range rows {
		if row.WalletID <= 0 || strings.TrimSpace(row.ReasonCode) == "" {
			return nil, invalidWalletSelection(fmt.Errorf("stored Wallet retirement blocker is invalid"))
		}
		result = append(result, WalletRetirementBlocker{WalletID: row.WalletID, ReasonCode: row.ReasonCode})
	}
	return result, nil
}

func loadWalletSelection(
	ctx context.Context,
	queries *wormtradingsqlc.Queries,
	row wormtradingsqlc.WormTradingWalletSelection,
) (WalletSelection, error) {
	itemRows, err := queries.ListWalletSelectionItems(ctx, row.OwnerAccountID)
	if err != nil {
		return WalletSelection{}, fmt.Errorf("list Wallet selection items: %w", err)
	}
	if int32(len(itemRows)) != row.SelectedWalletCount || len(itemRows) > MaximumWalletSelectionItems {
		return WalletSelection{}, invalidWalletSelection(fmt.Errorf("stored selected Wallet count is inconsistent"))
	}
	selection := WalletSelection{
		OwnerAccountID: uuidValue(row.OwnerAccountID),
		Configured:     true,
		Revision:       row.Revision,
		SelectedItems:  make([]WalletSelectionItem, 0, len(itemRows)),
		UpdatedAt:      timestampValue(row.UpdatedAt),
	}
	selectedIDs := make(map[int64]struct{}, len(itemRows))
	selectedAddresses := make(map[string]struct{}, len(itemRows))
	for index, item := range itemRows {
		if item.Ordinal != int32(index+1) || validateCanonicalWalletSelectionReference(item.WalletID, item.Address) != nil {
			return WalletSelection{}, invalidWalletSelection(fmt.Errorf("stored Wallet selection item is invalid"))
		}
		selectedIDs[item.WalletID] = struct{}{}
		selectedAddresses[item.Address] = struct{}{}
		selection.SelectedItems = append(selection.SelectedItems, WalletSelectionItem{
			Ordinal: item.Ordinal, WalletID: item.WalletID, Address: item.Address,
		})
	}
	retirementRows, err := queries.ListWalletRetirements(ctx, row.OwnerAccountID)
	if err != nil {
		return WalletSelection{}, fmt.Errorf("list Wallet retirements: %w", err)
	}
	selection.Retirements = make([]WalletRetirement, 0, len(retirementRows))
	for _, retirement := range retirementRows {
		_, selectedID := selectedIDs[retirement.WalletID]
		_, selectedAddress := selectedAddresses[retirement.Address]
		if selectedID || selectedAddress || retirement.PriorOrdinal <= 0 || retirement.RetiredFromRevision <= 0 ||
			validateCanonicalWalletSelectionReference(retirement.WalletID, retirement.Address) != nil {
			return WalletSelection{}, invalidWalletSelection(fmt.Errorf("stored Wallet retirement is invalid"))
		}
		selection.Retirements = append(selection.Retirements, WalletRetirement{
			WalletID: retirement.WalletID, Address: retirement.Address,
			PriorOrdinal: retirement.PriorOrdinal, RetiredFromRevision: retirement.RetiredFromRevision,
			RetiredAt: timestampValue(retirement.RetiredAt),
		})
	}
	return selection, nil
}

func invalidWalletSelection(err error) error {
	return fmt.Errorf("%w: %v", ErrInvalidWalletSelection, err)
}

func walletSelectionConstraint(err error, name string) bool {
	var postgresError *pgconn.PgError
	return errors.As(err, &postgresError) && postgresError.ConstraintName == name
}
