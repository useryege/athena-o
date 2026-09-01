package wormtrading

import (
	"context"
	"sync"

	"github.com/useryege/athena/internal/wormtrading/apiclient"
	wormstore "github.com/useryege/athena/internal/wormtrading/store"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	maxWalletSelectionInspectionReferences      = 100
	walletSelectionRemovalOpenPositionActive    = "OPEN_POSITION_ACTIVE"
	walletSelectionRemovalRequestInFlight       = "IN_FLIGHT_REQUEST_ACTIVE"
	walletSelectionRemovalCheckUnavailable      = "REMOVAL_CHECK_UNAVAILABLE"
	walletSelectionRemovalCredentialUnavailable = "CREDENTIAL_UNAVAILABLE"
)

// InspectWalletSelectionCandidates is the explicit, provider-backed removal
// safety boundary. Ordinary GetWalletSelection calls stay local and cheap;
// callers invoke this RPC only before offering or committing a removal.
func (s *Service) InspectWalletSelectionCandidates(
	ctx context.Context,
	req *apiclient.InspectWalletSelectionCandidatesRequest,
) (*apiclient.InspectWalletSelectionCandidatesResponse, error) {
	if req == nil || len(req.GetRefs()) == 0 {
		return nil, status.Error(codes.InvalidArgument, "refs must contain at least one Wallet")
	}
	if len(req.GetRefs()) > maxWalletSelectionInspectionReferences {
		return nil, status.Errorf(codes.InvalidArgument, "refs must contain at most %d Wallets", maxWalletSelectionInspectionReferences)
	}
	ownerAccountID, err := normalizeMarketCombinationAccountID(req.GetOwnerAccountId())
	if err != nil {
		return nil, err
	}
	refs, err := walletSelectionCandidateReferences(req.GetRefs())
	if err != nil {
		return nil, err
	}
	if err := s.requireCredentialCapability(); err != nil {
		return nil, err
	}

	inspectionCtx, cancel := context.WithTimeout(ctx, s.wormPositionBudget)
	defer cancel()
	selection, err := s.credentialStore.GetWalletSelection(inspectionCtx, ownerAccountID)
	s.recordCredentialStoreResult(err)
	if err != nil {
		return nil, status.Error(codes.Unavailable, "Worm Trading Wallet selection is unavailable")
	}
	walletIDs := make([]int64, len(refs))
	for index := range refs {
		walletIDs[index] = refs[index].WalletID
	}
	blockers, err := s.credentialStore.ListWalletRetirementBlockers(inspectionCtx, ownerAccountID, walletIDs)
	s.recordCredentialStoreResult(err)
	if err != nil {
		return nil, status.Error(codes.Unavailable, "Wallet removal safety check is unavailable")
	}
	blockerByWalletID := make(map[int64]string, len(blockers))
	for _, blocker := range blockers {
		if _, exists := blockerByWalletID[blocker.WalletID]; !exists {
			blockerByWalletID[blocker.WalletID] = blocker.ReasonCode
		}
	}

	snapshots, err := s.credentialStore.ListWalletConnectionSnapshots(inspectionCtx, refs)
	s.recordCredentialStoreResult(err)
	if err != nil {
		return nil, status.Error(codes.Unavailable, "Worm Wallet connection inventory is unavailable")
	}
	if err := validateStoredConnectionSnapshots(refs, snapshots); err != nil {
		return nil, status.Error(codes.Internal, "Worm credential store returned mismatched Wallet connections")
	}

	items := make([]*apiclient.WalletSelectionCandidateInspection, len(refs))
	clients := make([]WormAPIClient, len(refs))
	for index, ref := range refs {
		items[index] = &apiclient.WalletSelectionCandidateInspection{
			WalletId: ref.WalletID,
			Address:  ref.Address,
		}
		if reasonCode := blockerByWalletID[ref.WalletID]; reasonCode != "" {
			items[index].ReasonCode = reasonCode
			continue
		}
		snapshot := snapshots[index]
		if snapshot.ActiveCredential == nil {
			// Terminal local connection failures can leave a diagnostic warning on
			// an otherwise clean NOT_CONNECTED row. With no active credential and
			// no durable blocker above, there is no provider authority or exposure
			// to inspect; retirement may safely clear that local residue. Unknown
			// create outcomes are handled by blockerByWalletID before this branch.
			if snapshot.State == wormstore.ConnectionStateNotConnected {
				items[index].RemovalAllowed = true
			} else {
				items[index].ReasonCode = walletSelectionRemovalCredentialUnavailable
			}
			continue
		}
		apiKey, apiSecret, decryptErr := s.credentialCipher.decrypt(
			snapshot.WalletID,
			snapshot.ActiveCredential.APIKeyCiphertext,
			snapshot.ActiveCredential.APISecretCiphertext,
		)
		if decryptErr != nil {
			items[index].ReasonCode = walletSelectionRemovalCredentialUnavailable
			continue
		}
		client, clientErr := s.wormClientFactory.NewAuthenticatedClient(apiKey, apiSecret)
		if clientErr != nil {
			items[index].ReasonCode = walletSelectionRemovalCredentialUnavailable
			continue
		}
		clients[index] = client
	}

	var workers sync.WaitGroup
	for index, client := range clients {
		if client == nil {
			continue
		}
		workers.Add(1)
		go func(resultIndex int, readClient WormAPIClient) {
			defer workers.Done()
			reasonCode, inspectErr := s.inspectWalletSelectionProviderExposure(inspectionCtx, readClient)
			if inspectErr != nil {
				items[resultIndex].ReasonCode = walletSelectionRemovalCheckUnavailable
				if isWormAuthenticationError(inspectErr) {
					items[resultIndex].ReasonCode = wormErrorReconnectRequired
					snapshot := snapshots[resultIndex]
					s.markWalletReconnectRequired(
						inspectionCtx,
						snapshot.WalletID,
						effectiveConnectionAddress(snapshot),
						snapshot.ActiveCredential.ID,
					)
				}
				return
			}
			items[resultIndex].ReasonCode = reasonCode
			items[resultIndex].RemovalAllowed = reasonCode == ""
		}(index, client)
	}
	workers.Wait()

	return &apiclient.InspectWalletSelectionCandidatesResponse{
		Items:             items,
		SelectionRevision: selection.Revision,
		InspectedAt:       timeNowUTC().Unix(),
	}, nil
}

func walletSelectionCandidateReferences(values []*apiclient.WalletSelectionInput) ([]wormstore.WalletReference, error) {
	refs := make([]wormstore.WalletReference, 0, len(values))
	walletIDs := make(map[int64]struct{}, len(values))
	addresses := make(map[string]struct{}, len(values))
	for index, value := range values {
		if value == nil {
			return nil, status.Errorf(codes.InvalidArgument, "ref %d is required", index+1)
		}
		walletID, address, err := normalizeWormWalletReference(value.GetWalletId(), value.GetAddress())
		if err != nil {
			return nil, err
		}
		if _, duplicate := walletIDs[walletID]; duplicate {
			return nil, status.Error(codes.InvalidArgument, "Wallet IDs must be unique")
		}
		if _, duplicate := addresses[address]; duplicate {
			return nil, status.Error(codes.InvalidArgument, "Wallet addresses must be unique")
		}
		walletIDs[walletID] = struct{}{}
		addresses[address] = struct{}{}
		refs = append(refs, wormstore.WalletReference{WalletID: walletID, Address: address})
	}
	return refs, nil
}

func (s *Service) inspectWalletSelectionProviderExposure(
	ctx context.Context,
	client ExecutionPreviewReadClient,
) (string, error) {
	if err := acquireWormPositionSlot(ctx, s.wormPositionSemaphore); err != nil {
		return "", err
	}
	defer releaseWormPositionSlot(s.wormPositionSemaphore)
	exposure, err := fetchExecutionPreviewWalletExposure(ctx, client)
	s.wormCapabilities.recordWormResult(err)
	if err != nil {
		return "", err
	}
	for _, market := range exposure.markets {
		for _, positions := range market.positions {
			if len(positions) > 0 {
				return walletSelectionRemovalOpenPositionActive, nil
			}
		}
	}
	if len(exposure.requests) > 0 {
		return walletSelectionRemovalRequestInFlight, nil
	}
	return "", nil
}
