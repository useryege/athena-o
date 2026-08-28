package store

import (
	"context"
	"fmt"
	"time"

	wormtradingsqlc "github.com/useryege/athena/internal/wormtrading/store/sqlc"
)

func (s *SQLStore) ListWalletConnectionSnapshots(ctx context.Context, refs []WalletReference) ([]WalletConnectionSnapshot, error) {
	if err := s.requireDatabase(); err != nil {
		return nil, err
	}
	if len(refs) == 0 {
		return []WalletConnectionSnapshot{}, nil
	}
	walletIDs := make([]int64, 0, len(refs))
	addresses := make([]string, 0, len(refs))
	seen := make(map[int64]struct{}, len(refs))
	for _, ref := range refs {
		if err := validateWalletReference(ref.WalletID, ref.Address); err != nil {
			return nil, err
		}
		if _, duplicate := seen[ref.WalletID]; duplicate {
			return nil, fmt.Errorf("wallet ID %d is duplicated", ref.WalletID)
		}
		seen[ref.WalletID] = struct{}{}
		walletIDs = append(walletIDs, ref.WalletID)
		addresses = append(addresses, ref.Address)
	}
	rows, err := s.queries.ListWalletConnectionSnapshots(ctx, wormtradingsqlc.ListWalletConnectionSnapshotsParams{
		WalletIds: walletIDs,
		Addresses: addresses,
	})
	if err != nil {
		return nil, fmt.Errorf("list Worm wallet connection snapshots: %w", err)
	}
	if len(rows) != len(refs) {
		return nil, fmt.Errorf("Worm wallet connection snapshot count mismatch")
	}
	result := make([]WalletConnectionSnapshot, 0, len(rows))
	for index, row := range rows {
		if row.InputOrdinality != int64(index+1) || row.WalletID != refs[index].WalletID || row.RequestedAddress != refs[index].Address {
			return nil, fmt.Errorf("Worm wallet connection snapshot order mismatch")
		}
		if row.StoredAddress != "" && row.StoredAddress != row.RequestedAddress {
			return nil, ErrWalletAddressMismatch
		}
		snapshot := WalletConnectionSnapshot{
			WalletID:          row.WalletID,
			RequestedAddress:  row.RequestedAddress,
			StoredAddress:     row.StoredAddress,
			State:             ConnectionState(row.ConnectionState),
			WarningCode:       row.WarningCode,
			ConnectedAt:       timestampValue(row.ConnectedAt),
			ConnectionCreated: timestampValue(row.ConnectionCreatedAt),
			ConnectionUpdated: timestampValue(row.ConnectionUpdatedAt),
		}
		if !validConnectionState(snapshot.State) {
			return nil, fmt.Errorf("%w: persisted connection state %q", ErrInvalidState, snapshot.State)
		}
		if row.CredentialID > 0 {
			credential := StoredCredential{
				ID:                  row.CredentialID,
				WalletID:            row.WalletID,
				Version:             row.CredentialVersion,
				State:               CredentialState(row.CredentialState),
				APIKeyCiphertext:    append([]byte(nil), row.ApiKeyCiphertext...),
				APISecretCiphertext: append([]byte(nil), row.ApiSecretCiphertext...),
				CreatedAt:           timestampValue(row.CredentialCreatedAt),
				UpdatedAt:           timestampValue(row.CredentialUpdatedAt),
			}
			if credential.State != CredentialStateActive || len(credential.APIKeyCiphertext) == 0 || len(credential.APISecretCiphertext) == 0 {
				return nil, fmt.Errorf("%w: active credential projection", ErrInvalidState)
			}
			snapshot.ActiveCredential = &credential
		}
		result = append(result, snapshot)
	}
	return result, nil
}

func (s *SQLStore) GetWalletConnectionSnapshot(ctx context.Context, walletID int64, address string) (*WalletConnectionSnapshot, error) {
	items, err := s.ListWalletConnectionSnapshots(ctx, []WalletReference{{WalletID: walletID, Address: address}})
	if err != nil {
		return nil, err
	}
	if len(items) != 1 {
		return nil, ErrWalletConnectionNotFound
	}
	return &items[0], nil
}

func (s *SQLStore) MarkReconnectRequired(ctx context.Context, walletID int64, address string, activeCredentialID int64, warningCode string, at time.Time) error {
	if err := validateWalletReference(walletID, address); err != nil {
		return err
	}
	if activeCredentialID <= 0 {
		return fmt.Errorf("active credential ID must be positive")
	}
	if err := validateWarningCode(warningCode); err != nil {
		return err
	}
	now := canonicalNow(at)
	tx, queries, err := s.beginWalletTransaction(ctx, walletID)
	if err != nil {
		return err
	}
	defer rollbackWalletTransaction(tx)
	connection, err := queries.GetWalletConnectionForUpdate(ctx, walletID)
	if err != nil {
		if isNoRows(err) {
			return ErrWalletConnectionNotFound
		}
		return fmt.Errorf("lock Worm wallet connection: %w", err)
	}
	if connection.Address != address {
		return ErrWalletAddressMismatch
	}
	// A read that started before a reconnect or disconnect must never
	// downgrade the newer lifecycle state. Only the credential that is still
	// ACTIVE may force its own connection into reconnect-required.
	if connection.State != string(ConnectionStateConnected) {
		return commitWalletTransaction(ctx, tx)
	}
	activeCredential, err := queries.GetActiveCredentialForUpdate(ctx, walletID)
	if err != nil {
		if isNoRows(err) {
			return commitWalletTransaction(ctx, tx)
		}
		return fmt.Errorf("lock active Worm credential for reconnect: %w", err)
	}
	if activeCredential.ID != activeCredentialID {
		return commitWalletTransaction(ctx, tx)
	}
	warningCode, err = credentialCleanupWarning(ctx, queries, walletID, warningCode)
	if err != nil {
		return err
	}
	if _, err := queries.UpdateWalletConnectionState(ctx, wormtradingsqlc.UpdateWalletConnectionStateParams{
		State:       string(ConnectionStateReconnectRequired),
		WarningCode: warningCode,
		ConnectedAt: connection.ConnectedAt,
		Now:         timestampParam(now),
		WalletID:    walletID,
		Address:     address,
	}); err != nil {
		return fmt.Errorf("mark Worm wallet reconnect required: %w", err)
	}
	return commitWalletTransaction(ctx, tx)
}
