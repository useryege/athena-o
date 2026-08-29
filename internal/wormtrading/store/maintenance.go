package store

import (
	"context"
	"fmt"
	"time"

	wormtradingsqlc "github.com/useryege/athena/internal/wormtrading/store/sqlc"
)

const (
	connectionOutcomeUnknownWarning   = "CONNECT_OUTCOME_UNKNOWN"
	connectionServiceRestartedWarning = "SERVICE_RESTARTED"
)

// RecoverConnectionAttempts resolves every credential attempt inherited from a
// previous service process. PREPARED means Worm credential creation was never
// dispatched and can fail deterministically. COMPLETING may have reached Worm,
// so it remains outcome-unknown. The caller repeats this bounded operation until
// it returns fewer rows than requested.
func (s *SQLStore) RecoverConnectionAttempts(ctx context.Context, at time.Time, limit int32) (int64, error) {
	if err := s.requireDatabase(); err != nil {
		return 0, err
	}
	limit = connectionAttemptMaintenanceLimit(limit)
	walletIDs, err := s.queries.ListActiveConnectionAttemptWalletIDs(ctx, limit)
	if err != nil {
		return 0, fmt.Errorf("list active Worm connection attempts: %w", err)
	}
	for _, walletID := range walletIDs {
		_, recoverErr := s.recoverConnectionAttemptForWallet(ctx, walletID, canonicalNow(at), false)
		if recoverErr != nil {
			return 0, recoverErr
		}
	}
	return int64(len(walletIDs)), nil
}

// ExpireConnectionAttempts reconciles abandoned challenge operations. A
// PREPARED attempt cannot have created a remote credential and safely restores
// its previous connection state. A COMPLETING attempt may have reached Worm,
// so it is locked as outcome-unknown and requires explicit credential
// regeneration. Failed or expired regeneration attempts preserve that lock.
func (s *SQLStore) ExpireConnectionAttempts(ctx context.Context, at time.Time, limit int32) error {
	if err := s.requireDatabase(); err != nil {
		return err
	}
	limit = connectionAttemptMaintenanceLimit(limit)
	now := canonicalNow(at)
	walletIDs, err := s.queries.ListExpiredConnectionAttemptWalletIDs(ctx, wormtradingsqlc.ListExpiredConnectionAttemptWalletIDsParams{
		Now:         timestampParam(now),
		ResultLimit: limit,
	})
	if err != nil {
		return fmt.Errorf("list expired Worm connection attempts: %w", err)
	}
	for _, walletID := range walletIDs {
		if _, err := s.recoverConnectionAttemptForWallet(ctx, walletID, now, true); err != nil {
			return err
		}
	}
	return nil
}

func connectionAttemptMaintenanceLimit(limit int32) int32 {
	if limit <= 0 {
		return 100
	}
	if limit > 1000 {
		return 1000
	}
	return limit
}

func (s *SQLStore) recoverConnectionAttemptForWallet(ctx context.Context, walletID int64, now time.Time, expiredOnly bool) (bool, error) {
	tx, queries, err := s.beginWalletTransaction(ctx, walletID)
	if err != nil {
		return false, err
	}
	defer rollbackWalletTransaction(tx)
	attempt, err := queries.GetActiveConnectionAttemptForWallet(ctx, walletID)
	if err != nil {
		if isNoRows(err) {
			return false, commitWalletTransaction(ctx, tx)
		}
		return false, fmt.Errorf("lock active Worm connection attempt: %w", err)
	}
	if expiredOnly && timestampValue(attempt.ExpiresAt).After(now) {
		return false, commitWalletTransaction(ctx, tx)
	}

	switch ConnectionAttemptState(attempt.State) {
	case ConnectionAttemptStatePrepared:
		failureCode := connectionServiceRestartedWarning
		if expiredOnly {
			failureCode = expiredChallengeWarning
		}
		if err := failConnectionAttemptInTransaction(ctx, queries, attempt, failureCode, now); err != nil {
			return false, err
		}
	case ConnectionAttemptStateCompleting:
		if _, err := queries.MarkConnectionAttemptTerminal(ctx, wormtradingsqlc.MarkConnectionAttemptTerminalParams{
			State:       string(ConnectionAttemptStateOutcomeUnknown),
			FailureCode: connectionOutcomeUnknownWarning,
			Now:         timestampParam(now),
			ID:          attempt.ID,
		}); err != nil {
			return false, fmt.Errorf("mark interrupted Worm connection outcome unknown: %w", err)
		}
		if _, err := queries.UpdateWalletConnectionState(ctx, wormtradingsqlc.UpdateWalletConnectionStateParams{
			State:       string(ConnectionStateReconnectRequired),
			WarningCode: connectionOutcomeUnknownWarning,
			ConnectedAt: attempt.PreviousConnectedAt,
			Now:         timestampParam(now),
			WalletID:    attempt.WalletID,
			Address:     attempt.Address,
		}); err != nil {
			return false, fmt.Errorf("mark interrupted Worm connection reconnect required: %w", err)
		}
	default:
		return false, fmt.Errorf("%w: active connection attempt %q", ErrInvalidState, attempt.State)
	}
	if err := tx.Commit(ctx); err != nil {
		return false, fmt.Errorf("commit Worm connection attempt recovery: %w", err)
	}
	return true, nil
}
