package store

import (
	"context"
	"fmt"
	"time"

	wormtradingsqlc "github.com/useryege/athena/internal/wormtrading/store/sqlc"
)

const connectionOutcomeUnknownWarning = "CONNECT_OUTCOME_UNKNOWN"

// ExpireConnectionAttempts reconciles abandoned challenge operations. A
// PREPARED attempt cannot have created a remote credential and safely restores
// its previous connection state. A COMPLETING attempt may have reached Worm,
// so it is locked as outcome-unknown and requires an explicit reconnect.
func (s *SQLStore) ExpireConnectionAttempts(ctx context.Context, at time.Time, limit int32) error {
	if err := s.requireDatabase(); err != nil {
		return err
	}
	if limit <= 0 {
		limit = 100
	}
	if limit > 1000 {
		limit = 1000
	}
	now := canonicalNow(at)
	walletIDs, err := s.queries.ListExpiredConnectionAttemptWalletIDs(ctx, wormtradingsqlc.ListExpiredConnectionAttemptWalletIDsParams{
		Now:         timestampParam(now),
		ResultLimit: limit,
	})
	if err != nil {
		return fmt.Errorf("list expired Worm connection attempts: %w", err)
	}
	for _, walletID := range walletIDs {
		if err := s.expireConnectionAttemptForWallet(ctx, walletID, now); err != nil {
			return err
		}
	}
	return nil
}

func (s *SQLStore) expireConnectionAttemptForWallet(ctx context.Context, walletID int64, now time.Time) error {
	tx, queries, err := s.beginWalletTransaction(ctx, walletID)
	if err != nil {
		return err
	}
	defer rollbackWalletTransaction(tx)
	attempt, err := queries.GetActiveConnectionAttemptForWallet(ctx, walletID)
	if err != nil {
		if isNoRows(err) {
			return commitWalletTransaction(ctx, tx)
		}
		return fmt.Errorf("lock expired Worm connection attempt: %w", err)
	}
	if timestampValue(attempt.ExpiresAt).After(now) {
		return commitWalletTransaction(ctx, tx)
	}

	switch ConnectionAttemptState(attempt.State) {
	case ConnectionAttemptStatePrepared:
		if err := failConnectionAttemptInTransaction(ctx, queries, attempt, expiredChallengeWarning, now); err != nil {
			return err
		}
	case ConnectionAttemptStateCompleting:
		if _, err := queries.MarkConnectionAttemptTerminal(ctx, wormtradingsqlc.MarkConnectionAttemptTerminalParams{
			State:       string(ConnectionAttemptStateOutcomeUnknown),
			FailureCode: connectionOutcomeUnknownWarning,
			Now:         timestampParam(now),
			ID:          attempt.ID,
		}); err != nil {
			return fmt.Errorf("mark expired Worm connection outcome unknown: %w", err)
		}
		if _, err := queries.UpdateWalletConnectionState(ctx, wormtradingsqlc.UpdateWalletConnectionStateParams{
			State:       string(ConnectionStateReconnectRequired),
			WarningCode: connectionOutcomeUnknownWarning,
			ConnectedAt: attempt.PreviousConnectedAt,
			Now:         timestampParam(now),
			WalletID:    attempt.WalletID,
			Address:     attempt.Address,
		}); err != nil {
			return fmt.Errorf("mark expired Worm connection reconnect required: %w", err)
		}
	default:
		return fmt.Errorf("%w: active connection attempt %q", ErrInvalidState, attempt.State)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit expired Worm connection attempt: %w", err)
	}
	return nil
}
