package store

import (
	"context"
	"fmt"
	"time"

	wormtradingsqlc "github.com/useryege/athena/internal/wormtrading/store/sqlc"
)

func (s *SQLStore) ActivateCredential(ctx context.Context, req ActivateCredentialRequest) (*WalletConnectionSnapshot, error) {
	if len(req.APIKeyCiphertext) == 0 || len(req.APISecretCiphertext) == 0 {
		return nil, fmt.Errorf("encrypted Worm credential fields must not be empty")
	}
	attempt, err := s.GetConnectionAttempt(ctx, req.AttemptID)
	if err != nil {
		return nil, err
	}
	now := canonicalNow(req.Now)
	tx, queries, err := s.beginWalletTransaction(ctx, attempt.WalletID)
	if err != nil {
		return nil, err
	}
	defer rollbackWalletTransaction(tx)
	locked, err := queries.GetConnectionAttemptForUpdate(ctx, mustUUIDParam(attempt.ID))
	if err != nil {
		if isNoRows(err) {
			return nil, ErrConnectionAttemptNotFound
		}
		return nil, fmt.Errorf("lock Worm connection attempt: %w", err)
	}
	if locked.State != string(ConnectionAttemptStateCompleting) {
		return nil, ErrConnectionAttemptState
	}
	if _, err := queries.MarkActiveCredentialPendingRevocation(ctx, wormtradingsqlc.MarkActiveCredentialPendingRevocationParams{
		Now:      timestampParam(now),
		WalletID: locked.WalletID,
	}); err != nil {
		return nil, fmt.Errorf("retire prior Worm credential: %w", err)
	}
	createdCredential, err := queries.CreateActiveCredential(ctx, wormtradingsqlc.CreateActiveCredentialParams{
		WalletID:            locked.WalletID,
		ApiKeyCiphertext:    append([]byte(nil), req.APIKeyCiphertext...),
		ApiSecretCiphertext: append([]byte(nil), req.APISecretCiphertext...),
		Now:                 timestampParam(now),
	})
	if err != nil {
		return nil, fmt.Errorf("persist active Worm credential: %w", err)
	}
	if _, err := queries.MarkConnectionAttemptTerminal(ctx, wormtradingsqlc.MarkConnectionAttemptTerminalParams{
		State:       string(ConnectionAttemptStateCompleted),
		FailureCode: "",
		Now:         timestampParam(now),
		ID:          locked.ID,
	}); err != nil {
		return nil, fmt.Errorf("complete Worm connection attempt: %w", err)
	}
	warningCode, err := credentialCleanupWarning(ctx, queries, locked.WalletID, "")
	if err != nil {
		return nil, err
	}
	updatedConnection, err := queries.UpdateWalletConnectionState(ctx, wormtradingsqlc.UpdateWalletConnectionStateParams{
		State:       string(ConnectionStateConnected),
		WarningCode: warningCode,
		ConnectedAt: timestampParam(now),
		Now:         timestampParam(now),
		WalletID:    locked.WalletID,
		Address:     locked.Address,
	})
	if err != nil {
		return nil, fmt.Errorf("mark Worm wallet connected: %w", err)
	}
	snapshot := mapConnection(updatedConnection)
	activeCredential := mapCredential(createdCredential)
	snapshot.ActiveCredential = &activeCredential
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrTransactionOutcomeUnknown, err)
	}
	return &snapshot, nil
}

func (s *SQLStore) BeginDisconnect(ctx context.Context, walletID int64, address string, at time.Time) (*StoredCredential, error) {
	if err := validateWalletReference(walletID, address); err != nil {
		return nil, err
	}
	now := canonicalNow(at)
	tx, queries, err := s.beginWalletTransaction(ctx, walletID)
	if err != nil {
		return nil, err
	}
	defer rollbackWalletTransaction(tx)
	connection, err := queries.GetWalletConnectionForUpdate(ctx, walletID)
	if err != nil {
		if isNoRows(err) {
			return nil, ErrWalletConnectionNotFound
		}
		return nil, fmt.Errorf("lock Worm wallet connection: %w", err)
	}
	if connection.Address != address {
		return nil, ErrWalletAddressMismatch
	}
	if connection.WarningCode == unknownCreateOutcomeWarning {
		return nil, ErrCredentialOutcomeUnknown
	}
	if connection.State == string(ConnectionStateConnecting) {
		return nil, ErrConnectionOperationActive
	}
	credential, err := queries.GetNextCredentialForRevocation(ctx, walletID)
	if err != nil {
		if isNoRows(err) {
			if _, updateErr := queries.UpdateWalletConnectionState(ctx, wormtradingsqlc.UpdateWalletConnectionStateParams{
				State:       string(ConnectionStateNotConnected),
				WarningCode: "",
				ConnectedAt: timestampParam(time.Time{}),
				Now:         timestampParam(now),
				WalletID:    walletID,
				Address:     address,
			}); updateErr != nil {
				return nil, fmt.Errorf("clear empty Worm wallet connection: %w", updateErr)
			}
			if commitErr := commitWalletTransaction(ctx, tx); commitErr != nil {
				return nil, commitErr
			}
			return nil, ErrCredentialNotFound
		}
		return nil, fmt.Errorf("get Worm credential for disconnect: %w", err)
	}
	updated, err := queries.UpdateCredentialState(ctx, wormtradingsqlc.UpdateCredentialStateParams{
		State:        string(CredentialStateRevoking),
		Now:          timestampParam(now),
		CredentialID: credential.ID,
		WalletID:     walletID,
	})
	if err != nil {
		return nil, fmt.Errorf("mark Worm credential revoking: %w", err)
	}
	if _, err := queries.UpdateWalletConnectionState(ctx, wormtradingsqlc.UpdateWalletConnectionStateParams{
		State:       string(ConnectionStateDisconnecting),
		WarningCode: "",
		ConnectedAt: connection.ConnectedAt,
		Now:         timestampParam(now),
		WalletID:    walletID,
		Address:     address,
	}); err != nil {
		return nil, fmt.Errorf("mark Worm wallet disconnecting: %w", err)
	}
	if err := commitWalletTransaction(ctx, tx); err != nil {
		return nil, err
	}
	result := mapCredential(updated)
	return &result, nil
}

func (s *SQLStore) BeginCredentialRevocation(ctx context.Context, walletID, credentialID int64, at time.Time) (*StoredCredential, error) {
	if walletID <= 0 || credentialID <= 0 {
		return nil, fmt.Errorf("wallet ID and credential ID must be positive")
	}
	now := canonicalNow(at)
	tx, queries, err := s.beginWalletTransaction(ctx, walletID)
	if err != nil {
		return nil, err
	}
	defer rollbackWalletTransaction(tx)
	credential, err := queries.GetCredentialForUpdate(ctx, wormtradingsqlc.GetCredentialForUpdateParams{
		CredentialID: credentialID,
		WalletID:     walletID,
	})
	if err != nil {
		if isNoRows(err) {
			return nil, ErrCredentialNotFound
		}
		return nil, fmt.Errorf("lock Worm credential: %w", err)
	}
	if credential.State != string(CredentialStatePendingRevocation) && credential.State != string(CredentialStateRevoking) {
		return nil, ErrConnectionAttemptState
	}
	updated, err := queries.UpdateCredentialState(ctx, wormtradingsqlc.UpdateCredentialStateParams{
		State:        string(CredentialStateRevoking),
		Now:          timestampParam(now),
		CredentialID: credentialID,
		WalletID:     walletID,
	})
	if err != nil {
		return nil, fmt.Errorf("mark Worm credential revoking: %w", err)
	}
	if err := commitWalletTransaction(ctx, tx); err != nil {
		return nil, err
	}
	result := mapCredential(updated)
	return &result, nil
}

func (s *SQLStore) ListCredentialsNeedingRevocation(ctx context.Context, retryBefore time.Time, limit int32) ([]StoredCredential, error) {
	if err := s.requireDatabase(); err != nil {
		return nil, err
	}
	if limit <= 0 {
		limit = 100
	}
	if limit > 1000 {
		limit = 1000
	}
	rows, err := s.queries.ListCredentialsNeedingRevocation(ctx, wormtradingsqlc.ListCredentialsNeedingRevocationParams{
		RetryBefore: timestampParam(canonicalNow(retryBefore)),
		Limit:       limit,
	})
	if err != nil {
		return nil, fmt.Errorf("list Worm credentials needing revocation: %w", err)
	}
	result := make([]StoredCredential, 0, len(rows))
	for _, row := range rows {
		result = append(result, mapCredential(row))
	}
	return result, nil
}

func (s *SQLStore) MarkCredentialRevoked(ctx context.Context, walletID, credentialID int64, at time.Time) error {
	if walletID <= 0 || credentialID <= 0 {
		return fmt.Errorf("wallet ID and credential ID must be positive")
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
	if _, err := queries.GetCredentialForUpdate(ctx, wormtradingsqlc.GetCredentialForUpdateParams{
		CredentialID: credentialID,
		WalletID:     walletID,
	}); err != nil {
		if isNoRows(err) {
			return ErrCredentialNotFound
		}
		return fmt.Errorf("lock revoked Worm credential: %w", err)
	}
	deleted, err := queries.DeleteCredential(ctx, wormtradingsqlc.DeleteCredentialParams{
		CredentialID: credentialID,
		WalletID:     walletID,
	})
	if err != nil {
		return fmt.Errorf("delete revoked Worm credential: %w", err)
	}
	if deleted != 1 {
		return ErrCredentialNotFound
	}
	remaining, err := queries.CountCredentialsForWallet(ctx, walletID)
	if err != nil {
		return fmt.Errorf("count remaining Worm credentials: %w", err)
	}
	if remaining == 0 {
		nextState := ConnectionStateReconnectRequired
		warningCode := connection.WarningCode
		connectedAt := connection.ConnectedAt
		if connection.State == string(ConnectionStateDisconnecting) {
			nextState = ConnectionStateNotConnected
			warningCode = ""
			connectedAt = timestampParam(time.Time{})
		}
		if _, err := queries.UpdateWalletConnectionState(ctx, wormtradingsqlc.UpdateWalletConnectionStateParams{
			State:       string(nextState),
			WarningCode: warningCode,
			ConnectedAt: connectedAt,
			Now:         timestampParam(now),
			WalletID:    walletID,
			Address:     connection.Address,
		}); err != nil {
			return fmt.Errorf("finalize Worm credential revocation state: %w", err)
		}
	} else if connection.State != string(ConnectionStateDisconnecting) {
		active, err := queries.CountActiveCredentialsForWallet(ctx, walletID)
		if err != nil {
			return fmt.Errorf("count active Worm credentials: %w", err)
		}
		if active > 0 && (connection.State == string(ConnectionStateConnected) || connection.State == string(ConnectionStateRevocationRequired)) {
			warningCode, err := credentialCleanupWarning(ctx, queries, walletID, "")
			if err != nil {
				return err
			}
			if _, err := queries.UpdateWalletConnectionState(ctx, wormtradingsqlc.UpdateWalletConnectionStateParams{
				State:       string(ConnectionStateConnected),
				WarningCode: warningCode,
				ConnectedAt: connection.ConnectedAt,
				Now:         timestampParam(now),
				WalletID:    walletID,
				Address:     connection.Address,
			}); err != nil {
				return fmt.Errorf("clear Worm credential revocation warning: %w", err)
			}
		}
	}
	return commitWalletTransaction(ctx, tx)
}

func (s *SQLStore) MarkCredentialRevocationFailed(
	ctx context.Context,
	walletID, credentialID int64,
	connectionState ConnectionState,
	credentialState CredentialState,
	warningCode string,
	at time.Time,
) error {
	if walletID <= 0 || credentialID <= 0 {
		return fmt.Errorf("wallet ID and credential ID must be positive")
	}
	if connectionState != ConnectionStateConnected && connectionState != ConnectionStateDisconnecting && connectionState != ConnectionStateRevocationRequired {
		return fmt.Errorf("revocation failure connection state is invalid")
	}
	if credentialState != CredentialStatePendingRevocation && credentialState != CredentialStateRevocationRequired {
		return fmt.Errorf("revocation failure credential state is invalid")
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
	if _, err := queries.UpdateCredentialState(ctx, wormtradingsqlc.UpdateCredentialStateParams{
		State:        string(credentialState),
		Now:          timestampParam(now),
		CredentialID: credentialID,
		WalletID:     walletID,
	}); err != nil {
		if isNoRows(err) {
			return ErrCredentialNotFound
		}
		return fmt.Errorf("mark Worm credential revocation required: %w", err)
	}
	if connectionState == ConnectionStateConnected && connection.State != string(ConnectionStateConnected) {
		// A concurrent authenticated read or explicit connection mutation may
		// have moved the connection to a stricter state while the old-key
		// revoke call was in flight. Persist the credential failure without
		// overwriting that newer connection decision.
		return commitWalletTransaction(ctx, tx)
	}
	if connectionState == ConnectionStateConnected {
		warningCode, err = credentialCleanupWarning(ctx, queries, walletID, warningCode)
		if err != nil {
			return err
		}
	}
	if _, err := queries.UpdateWalletConnectionState(ctx, wormtradingsqlc.UpdateWalletConnectionStateParams{
		State:       string(connectionState),
		WarningCode: warningCode,
		ConnectedAt: connection.ConnectedAt,
		Now:         timestampParam(now),
		WalletID:    walletID,
		Address:     connection.Address,
	}); err != nil {
		return fmt.Errorf("mark Worm wallet credential revocation failure: %w", err)
	}
	return commitWalletTransaction(ctx, tx)
}
