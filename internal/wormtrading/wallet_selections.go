package wormtrading

import (
	"context"
	"errors"
	"math"
	"sort"
	"strings"

	"github.com/useryege/athena/internal/wormtrading/apiclient"
	wormstore "github.com/useryege/athena/internal/wormtrading/store"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (s *Service) GetWalletSelection(
	ctx context.Context,
	req *apiclient.GetWalletSelectionRequest,
) (*apiclient.GetWalletSelectionResponse, error) {
	ownerAccountID, err := normalizeMarketCombinationAccountID(req.GetOwnerAccountId())
	if err != nil {
		return nil, err
	}
	if err := s.requireMarketCombinationStore(); err != nil {
		return nil, err
	}
	selection, err := s.credentialStore.GetWalletSelection(ctx, ownerAccountID)
	s.recordCredentialStoreResult(err)
	if err != nil {
		return nil, walletSelectionRPCError(err)
	}
	return &apiclient.GetWalletSelectionResponse{Selection: walletSelectionToProto(selection)}, nil
}

func (s *Service) ReplaceWalletSelection(
	ctx context.Context,
	req *apiclient.ReplaceWalletSelectionRequest,
) (*apiclient.ReplaceWalletSelectionResponse, error) {
	ownerAccountID, err := normalizeMarketCombinationAccountID(req.GetOwnerAccountId())
	if err != nil {
		return nil, err
	}
	if req.GetExpectedRevision() < 0 || req.GetExpectedRevision() == math.MaxInt64 {
		return nil, status.Error(codes.InvalidArgument, "expected_revision must be an incrementable non-negative integer")
	}
	selected, retiring, err := walletSelectionInputsFromProto(req.GetSelectedItems(), req.GetRetiringWallets())
	if err != nil {
		return nil, err
	}
	if err := s.requireMarketCombinationStore(); err != nil {
		return nil, err
	}
	current, err := s.credentialStore.GetWalletSelection(ctx, ownerAccountID)
	s.recordCredentialStoreResult(err)
	if err != nil {
		return nil, walletSelectionRPCError(err)
	}
	if (!current.Configured && req.GetExpectedRevision() != 0) ||
		(current.Configured && current.Revision != req.GetExpectedRevision()) {
		return nil, status.Error(codes.Aborted, "Wallet selection revision conflict")
	}
	removals, err := walletSelectionRemovalReferences(current, selected, retiring)
	if err != nil {
		return nil, err
	}
	if err := s.ensureWalletSelectionRemovalAllowed(ctx, ownerAccountID, removals, current.Revision); err != nil {
		return nil, err
	}
	selection, err := s.credentialStore.ReplaceWalletSelection(ctx, wormstore.ReplaceWalletSelectionRequest{
		OwnerAccountID:   ownerAccountID,
		ExpectedRevision: req.GetExpectedRevision(),
		SelectedItems:    selected,
		RetiringWallets:  retiring,
		Now:              timeNowUTC(),
	})
	s.recordCredentialStoreResult(err)
	if err != nil {
		return nil, walletSelectionRPCError(err)
	}
	return &apiclient.ReplaceWalletSelectionResponse{Selection: walletSelectionToProto(selection)}, nil
}

func walletSelectionRemovalReferences(
	current *wormstore.WalletSelection,
	selected []wormstore.WalletSelectionInput,
	retiring []wormstore.WalletRetirementInput,
) ([]wormstore.WalletReference, error) {
	selectedByWalletID := make(map[int64]string, len(selected))
	for _, item := range selected {
		selectedByWalletID[item.WalletID] = item.Address
	}
	removalByWalletID := make(map[int64]wormstore.WalletReference, len(retiring))
	if current != nil {
		for _, item := range current.SelectedItems {
			selectedAddress, remainsSelected := selectedByWalletID[item.WalletID]
			if remainsSelected && selectedAddress == item.Address {
				continue
			}
			if remainsSelected {
				return nil, status.Error(codes.InvalidArgument, "a selected Wallet address cannot change")
			}
			removalByWalletID[item.WalletID] = wormstore.WalletReference{WalletID: item.WalletID, Address: item.Address}
		}
	}
	for _, item := range retiring {
		if existing, duplicate := removalByWalletID[item.WalletID]; duplicate && existing.Address != item.Address {
			return nil, status.Error(codes.InvalidArgument, "a retiring Wallet ID cannot identify multiple addresses")
		}
		removalByWalletID[item.WalletID] = wormstore.WalletReference{WalletID: item.WalletID, Address: item.Address}
	}
	refs := make([]wormstore.WalletReference, 0, len(removalByWalletID))
	for _, ref := range removalByWalletID {
		refs = append(refs, ref)
	}
	sort.Slice(refs, func(left, right int) bool { return refs[left].WalletID < refs[right].WalletID })
	return refs, nil
}

func (s *Service) ensureWalletSelectionRemovalAllowed(
	ctx context.Context,
	ownerAccountID string,
	refs []wormstore.WalletReference,
	expectedSelectionRevision int64,
) error {
	for first := 0; first < len(refs); first += maxWalletSelectionInspectionReferences {
		last := first + maxWalletSelectionInspectionReferences
		if last > len(refs) {
			last = len(refs)
		}
		protoRefs := make([]*apiclient.WalletSelectionInput, 0, last-first)
		for _, ref := range refs[first:last] {
			protoRefs = append(protoRefs, &apiclient.WalletSelectionInput{WalletId: ref.WalletID, Address: ref.Address})
		}
		inspection, err := s.InspectWalletSelectionCandidates(ctx, &apiclient.InspectWalletSelectionCandidatesRequest{
			OwnerAccountId: ownerAccountID,
			Refs:           protoRefs,
		})
		if err != nil {
			return err
		}
		if inspection.GetSelectionRevision() != expectedSelectionRevision {
			return status.Error(codes.Aborted, "Wallet selection revision conflict")
		}
		if len(inspection.GetItems()) != len(protoRefs) {
			return status.Error(codes.Internal, "Wallet removal safety check returned mismatched results")
		}
		for index, item := range inspection.GetItems() {
			ref := refs[first+index]
			if item == nil || item.GetWalletId() != ref.WalletID || item.GetAddress() != ref.Address {
				return status.Error(codes.Internal, "Wallet removal safety check returned mismatched results")
			}
			if item.GetRemovalAllowed() {
				continue
			}
			reasonCode := strings.TrimSpace(item.GetReasonCode())
			if reasonCode == "" {
				reasonCode = walletSelectionRemovalCheckUnavailable
			}
			return status.Error(codes.FailedPrecondition, reasonCode)
		}
	}
	return nil
}

func walletSelectionInputsFromProto(
	selectedValues []*apiclient.WalletSelectionInput,
	retiringValues []*apiclient.WalletRetirementInput,
) ([]wormstore.WalletSelectionInput, []wormstore.WalletRetirementInput, error) {
	if len(selectedValues) > wormstore.MaximumWalletSelectionItems {
		return nil, nil, status.Errorf(codes.InvalidArgument, "selected_items must contain at most %d Wallets", wormstore.MaximumWalletSelectionItems)
	}
	selected := make([]wormstore.WalletSelectionInput, 0, len(selectedValues))
	selectedIDs := make(map[int64]struct{}, len(selectedValues))
	selectedAddresses := make(map[string]struct{}, len(selectedValues))
	for index, value := range selectedValues {
		if value == nil {
			return nil, nil, status.Errorf(codes.InvalidArgument, "selected item %d is required", index+1)
		}
		walletID, address, err := normalizeWormWalletReference(value.GetWalletId(), value.GetAddress())
		if err != nil {
			return nil, nil, err
		}
		if _, duplicate := selectedIDs[walletID]; duplicate {
			return nil, nil, status.Error(codes.InvalidArgument, "selected Wallet IDs must be unique")
		}
		if _, duplicate := selectedAddresses[address]; duplicate {
			return nil, nil, status.Error(codes.InvalidArgument, "selected Wallet addresses must be unique")
		}
		selectedIDs[walletID] = struct{}{}
		selectedAddresses[address] = struct{}{}
		selected = append(selected, wormstore.WalletSelectionInput{WalletID: walletID, Address: address})
	}
	retiring := make([]wormstore.WalletRetirementInput, 0, len(retiringValues))
	retiringIDs := make(map[int64]struct{}, len(retiringValues))
	retiringAddresses := make(map[string]struct{}, len(retiringValues))
	for index, value := range retiringValues {
		if value == nil || value.GetPriorOrdinal() <= 0 {
			return nil, nil, status.Errorf(codes.InvalidArgument, "retiring Wallet %d is invalid", index+1)
		}
		walletID, address, err := normalizeWormWalletReference(value.GetWalletId(), value.GetAddress())
		if err != nil {
			return nil, nil, err
		}
		if _, duplicate := selectedIDs[walletID]; duplicate {
			return nil, nil, status.Error(codes.InvalidArgument, "selected and retiring Wallets must be disjoint")
		}
		if _, duplicate := selectedAddresses[address]; duplicate {
			return nil, nil, status.Error(codes.InvalidArgument, "selected and retiring Wallet addresses must be disjoint")
		}
		if _, duplicate := retiringIDs[walletID]; duplicate {
			return nil, nil, status.Error(codes.InvalidArgument, "retiring Wallet IDs must be unique")
		}
		if _, duplicate := retiringAddresses[address]; duplicate {
			return nil, nil, status.Error(codes.InvalidArgument, "retiring Wallet addresses must be unique")
		}
		retiringIDs[walletID] = struct{}{}
		retiringAddresses[address] = struct{}{}
		retiring = append(retiring, wormstore.WalletRetirementInput{
			WalletID: walletID, Address: address, PriorOrdinal: value.GetPriorOrdinal(),
		})
	}
	return selected, retiring, nil
}

func walletSelectionToProto(selection *wormstore.WalletSelection) *apiclient.WalletSelection {
	if selection == nil {
		return nil
	}
	selected := make([]*apiclient.WalletSelectionItem, 0, len(selection.SelectedItems))
	for _, item := range selection.SelectedItems {
		selected = append(selected, &apiclient.WalletSelectionItem{
			Ordinal: item.Ordinal, WalletId: item.WalletID, Address: item.Address,
		})
	}
	retirements := make([]*apiclient.WalletRetirement, 0, len(selection.Retirements))
	for _, retirement := range selection.Retirements {
		retiredAt := int64(0)
		if !retirement.RetiredAt.IsZero() {
			retiredAt = retirement.RetiredAt.Unix()
		}
		retirements = append(retirements, &apiclient.WalletRetirement{
			WalletId: retirement.WalletID, Address: retirement.Address,
			PriorOrdinal: retirement.PriorOrdinal, RetiredFromRevision: retirement.RetiredFromRevision,
			RetiredAt: retiredAt,
		})
	}
	updatedAt := int64(0)
	if !selection.UpdatedAt.IsZero() {
		updatedAt = selection.UpdatedAt.Unix()
	}
	return &apiclient.WalletSelection{
		Configured: selection.Configured, Revision: selection.Revision,
		SelectedItems: selected, Retirements: retirements, UpdatedAt: updatedAt,
		MaximumWallets: wormstore.MaximumWalletSelectionItems,
	}
}

func walletSelectionContains(selection *wormstore.WalletSelection, walletID int64, address string) bool {
	if selection == nil || !selection.Configured {
		return false
	}
	for _, item := range selection.SelectedItems {
		if item.WalletID == walletID && item.Address == address {
			return true
		}
	}
	return false
}

func walletSelectionContainsRetirement(selection *wormstore.WalletSelection, walletID int64, address string) bool {
	if selection == nil || !selection.Configured {
		return false
	}
	for _, item := range selection.Retirements {
		if item.WalletID == walletID && item.Address == address {
			return true
		}
	}
	return false
}

func walletSelectionRPCError(err error) error {
	var blocked *wormstore.WalletSelectionRetirementBlockedError
	switch {
	case errors.Is(err, wormstore.ErrWalletSelectionRevision):
		return status.Error(codes.Aborted, "Wallet selection revision conflict")
	case errors.As(err, &blocked):
		reasonCode := strings.TrimSpace(blocked.ReasonCode)
		if reasonCode == "" {
			reasonCode = walletSelectionRemovalCheckUnavailable
		}
		return status.Error(codes.FailedPrecondition, reasonCode)
	case errors.Is(err, wormstore.ErrWalletConnectionCapacityPending):
		return status.Error(codes.FailedPrecondition, "WALLET_CONNECTION_CAPACITY_PENDING")
	case errors.Is(err, wormstore.ErrWalletNotSelected):
		return status.Error(codes.FailedPrecondition, "WALLET_NOT_SELECTED")
	case errors.Is(err, wormstore.ErrWalletNotRetiring):
		return status.Error(codes.FailedPrecondition, "WALLET_NOT_RETIRING")
	case errors.Is(err, wormstore.ErrWalletRetirementIncomplete):
		return status.Error(codes.FailedPrecondition, "WALLET_RETIREMENT_INCOMPLETE")
	case errors.Is(err, wormstore.ErrInvalidWalletSelection):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, context.Canceled), errors.Is(err, context.DeadlineExceeded):
		return status.FromContextError(err).Err()
	default:
		return status.Error(codes.Internal, "Wallet selection store operation failed")
	}
}
