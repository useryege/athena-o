package wormtrading

import (
	"context"
	"errors"
	"strings"

	"github.com/useryege/athena/internal/wormtrading/apiclient"
	wormstore "github.com/useryege/athena/internal/wormtrading/store"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	maxWalletConnectionReferences    = 100
	maxWalletConnectionWarningLength = 100
)

// BatchGetWalletConnections returns only the durable, non-secret connection
// projection for caller-resolved Wallet references. It deliberately avoids the
// Worm provider and never decrypts an active credential.
func (s *Service) BatchGetWalletConnections(
	ctx context.Context,
	req *apiclient.BatchGetWalletConnectionsRequest,
) (*apiclient.BatchGetWalletConnectionsResponse, error) {
	if req == nil || len(req.GetRefs()) == 0 {
		return nil, status.Error(codes.InvalidArgument, "refs must contain at least one wallet")
	}
	if len(req.GetRefs()) > maxWalletConnectionReferences {
		return nil, status.Errorf(codes.InvalidArgument, "refs must contain at most %d wallets", maxWalletConnectionReferences)
	}
	if !s.isStarted() {
		return nil, status.Error(codes.FailedPrecondition, "Worm Trading service is not started")
	}
	if s.credentialStore == nil {
		return nil, status.Error(codes.FailedPrecondition, "Worm credential store is not configured")
	}

	refs := make([]wormstore.WalletReference, 0, len(req.GetRefs()))
	walletIDs := make(map[int64]struct{}, len(req.GetRefs()))
	addresses := make(map[string]struct{}, len(req.GetRefs()))
	for _, ref := range req.GetRefs() {
		if ref == nil {
			return nil, status.Error(codes.InvalidArgument, "every ref must identify a wallet")
		}
		walletID, address, err := normalizeWormWalletReference(ref.GetWalletId(), ref.GetAddress())
		if err != nil {
			return nil, err
		}
		if _, exists := walletIDs[walletID]; exists {
			return nil, status.Error(codes.InvalidArgument, "wallet_id values must be unique")
		}
		if _, exists := addresses[address]; exists {
			return nil, status.Error(codes.InvalidArgument, "address values must be unique")
		}
		walletIDs[walletID] = struct{}{}
		addresses[address] = struct{}{}
		refs = append(refs, wormstore.WalletReference{WalletID: walletID, Address: address})
	}

	snapshots, err := s.credentialStore.ListWalletConnectionSnapshots(ctx, refs)
	s.recordCredentialStoreResult(err)
	if err != nil {
		if ctx.Err() != nil {
			return nil, status.FromContextError(ctx.Err()).Err()
		}
		return nil, status.Error(codes.Unavailable, "Worm credential store is unavailable")
	}
	if err := validateStoredConnectionSnapshots(refs, snapshots); err != nil {
		return nil, status.Error(codes.Internal, "Worm credential store returned mismatched wallet connections")
	}

	items := make([]*apiclient.WormWalletConnection, len(snapshots))
	for index, snapshot := range snapshots {
		items[index] = wormWalletConnectionFromStore(snapshot)
	}
	return &apiclient.BatchGetWalletConnectionsResponse{
		Items:     items,
		FetchedAt: timeNowUTC().Unix(),
	}, nil
}

func validateStoredConnectionSnapshots(refs []wormstore.WalletReference, snapshots []wormstore.WalletConnectionSnapshot) error {
	if len(refs) != len(snapshots) {
		return errors.New("wallet connection count mismatch")
	}
	for index, ref := range refs {
		snapshot := snapshots[index]
		if snapshot.WalletID != ref.WalletID || snapshot.RequestedAddress != ref.Address {
			return errors.New("wallet connection order mismatch")
		}
		if snapshot.StoredAddress != "" && snapshot.StoredAddress != ref.Address {
			return errors.New("wallet connection address mismatch")
		}
		if !validConnectionProjectionState(snapshot.State) {
			return errors.New("wallet connection state is invalid")
		}
		if snapshot.WarningCode != strings.TrimSpace(snapshot.WarningCode) || len(snapshot.WarningCode) > maxWalletConnectionWarningLength {
			return errors.New("wallet connection warning is invalid")
		}
		if snapshot.ActiveCredential != nil && snapshot.ActiveCredential.WalletID != snapshot.WalletID {
			return errors.New("wallet credential mismatch")
		}
	}
	return nil
}

func validConnectionProjectionState(state wormstore.ConnectionState) bool {
	switch state {
	case wormstore.ConnectionStateNotConnected,
		wormstore.ConnectionStateConnecting,
		wormstore.ConnectionStateConnected,
		wormstore.ConnectionStateReconnectRequired,
		wormstore.ConnectionStateDisconnecting,
		wormstore.ConnectionStateRevocationRequired:
		return true
	default:
		return false
	}
}
