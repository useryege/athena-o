package store

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	wormtradingsqlc "github.com/useryege/athena/internal/wormtrading/store/sqlc"
)

const (
	expiredChallengeWarning     = "CHALLENGE_EXPIRED"
	unknownCreateOutcomeWarning = "CONNECT_OUTCOME_UNKNOWN"
)

func (s *SQLStore) PrepareConnectionAttempt(ctx context.Context, req PrepareConnectionAttemptRequest) (*ConnectionAttempt, error) {
	if err := validatePrepareConnectionAttemptRequest(req); err != nil {
		return nil, err
	}
	now := canonicalNow(req.Now)
	if !req.ExpiresAt.After(now) {
		return nil, ErrConnectionAttemptExpired
	}
	attemptID := strings.TrimSpace(req.AttemptID)
	if attemptID == "" {
		attemptID = uuid.NewString()
	}
	attemptUUID, attemptID, err := uuidParam(attemptID)
	if err != nil {
		return nil, err
	}

	tx, queries, err := s.beginWalletTransaction(ctx, req.WalletID)
	if err != nil {
		return nil, err
	}
	defer rollbackWalletTransaction(tx)

	if err := queries.CreateWalletConnectionIfMissing(ctx, wormtradingsqlc.CreateWalletConnectionIfMissingParams{
		WalletID: req.WalletID,
		Address:  req.Address,
		Now:      timestampParam(now),
	}); err != nil {
		return nil, fmt.Errorf("ensure Worm wallet connection: %w", err)
	}
	connection, err := queries.GetWalletConnectionForUpdate(ctx, req.WalletID)
	if err != nil {
		return nil, fmt.Errorf("lock Worm wallet connection: %w", err)
	}
	if connection.Address != req.Address {
		return nil, ErrWalletAddressMismatch
	}

	active, activeErr := queries.GetActiveConnectionAttemptForWallet(ctx, req.WalletID)
	recoveredUnknownOutcome := false
	if activeErr == nil {
		if timestampValue(active.ExpiresAt).After(now) {
			return nil, ErrConnectionOperationActive
		}
		switch ConnectionAttemptState(active.State) {
		case ConnectionAttemptStatePrepared:
			if err := failConnectionAttemptInTransaction(ctx, queries, active, expiredChallengeWarning, now); err != nil {
				return nil, err
			}
			connection.State = active.PreviousConnectionState
			connection.ConnectedAt = active.PreviousConnectedAt
		case ConnectionAttemptStateCompleting:
			if _, err := queries.MarkConnectionAttemptTerminal(ctx, wormtradingsqlc.MarkConnectionAttemptTerminalParams{
				State:       string(ConnectionAttemptStateOutcomeUnknown),
				FailureCode: unknownCreateOutcomeWarning,
				Now:         timestampParam(now),
				ID:          active.ID,
			}); err != nil {
				return nil, fmt.Errorf("expire indeterminate Worm connection attempt: %w", err)
			}
			if _, err := queries.UpdateWalletConnectionState(ctx, wormtradingsqlc.UpdateWalletConnectionStateParams{
				State:       string(ConnectionStateReconnectRequired),
				WarningCode: unknownCreateOutcomeWarning,
				ConnectedAt: active.PreviousConnectedAt,
				Now:         timestampParam(now),
				WalletID:    req.WalletID,
				Address:     req.Address,
			}); err != nil {
				return nil, fmt.Errorf("recover indeterminate Worm connection attempt: %w", err)
			}
			connection.State = string(ConnectionStateReconnectRequired)
			connection.WarningCode = unknownCreateOutcomeWarning
			connection.ConnectedAt = active.PreviousConnectedAt
			recoveredUnknownOutcome = true
		default:
			return nil, ErrConnectionAttemptState
		}
	} else if !isNoRows(activeErr) {
		return nil, fmt.Errorf("inspect active Worm connection attempt: %w", activeErr)
	}
	if recoveredUnknownOutcome {
		if err := commitWalletTransaction(ctx, tx); err != nil {
			return nil, err
		}
		return nil, ErrCredentialOutcomeUnknown
	}
	if connection.WarningCode == unknownCreateOutcomeWarning {
		return nil, ErrCredentialOutcomeUnknown
	}

	if err := validateConnectionAttemptKind(req.Kind, ConnectionState(connection.State)); err != nil {
		return nil, err
	}
	created, err := queries.CreateConnectionAttempt(ctx, wormtradingsqlc.CreateConnectionAttemptParams{
		ID:                      attemptUUID,
		WalletID:                req.WalletID,
		Address:                 req.Address,
		Kind:                    string(req.Kind),
		PreviousConnectionState: connection.State,
		PreviousConnectedAt:     connection.ConnectedAt,
		Nonce:                   req.Nonce,
		ChallengeMessage:        req.ChallengeMessage,
		MessageDigest:           append([]byte(nil), req.MessageDigest...),
		ExpiresAt:               timestampParam(req.ExpiresAt),
		Now:                     timestampParam(now),
	})
	if err != nil {
		return nil, fmt.Errorf("create Worm connection attempt %s: %w", attemptID, err)
	}
	if _, err := queries.UpdateWalletConnectionState(ctx, wormtradingsqlc.UpdateWalletConnectionStateParams{
		State:       string(ConnectionStateConnecting),
		WarningCode: "",
		ConnectedAt: timestampParam(time.Time{}),
		Now:         timestampParam(now),
		WalletID:    req.WalletID,
		Address:     req.Address,
	}); err != nil {
		return nil, fmt.Errorf("mark Worm wallet connection connecting: %w", err)
	}
	if err := commitWalletTransaction(ctx, tx); err != nil {
		return nil, err
	}
	result := mapConnectionAttempt(created)
	return &result, nil
}

func (s *SQLStore) GetConnectionAttempt(ctx context.Context, attemptID string) (*ConnectionAttempt, error) {
	if err := s.requireDatabase(); err != nil {
		return nil, err
	}
	attemptUUID, _, err := uuidParam(attemptID)
	if err != nil {
		return nil, err
	}
	row, err := s.queries.GetConnectionAttempt(ctx, attemptUUID)
	if err != nil {
		if isNoRows(err) {
			return nil, ErrConnectionAttemptNotFound
		}
		return nil, fmt.Errorf("get Worm connection attempt: %w", err)
	}
	result := mapConnectionAttempt(row)
	return &result, nil
}

func (s *SQLStore) BeginConnectionAttemptCompletion(ctx context.Context, attemptID string, at time.Time) (*ConnectionAttempt, error) {
	attempt, err := s.GetConnectionAttempt(ctx, attemptID)
	if err != nil {
		return nil, err
	}
	now := canonicalNow(at)
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
	if locked.State != string(ConnectionAttemptStatePrepared) {
		return nil, ErrConnectionAttemptState
	}
	if !timestampValue(locked.ExpiresAt).After(now) {
		if err := failConnectionAttemptInTransaction(ctx, queries, locked, expiredChallengeWarning, now); err != nil {
			return nil, err
		}
		if err := commitWalletTransaction(ctx, tx); err != nil {
			return nil, err
		}
		return nil, ErrConnectionAttemptExpired
	}
	updated, err := queries.MarkConnectionAttemptCompleting(ctx, wormtradingsqlc.MarkConnectionAttemptCompletingParams{
		Now: timestampParam(now),
		ID:  locked.ID,
	})
	if err != nil {
		if isNoRows(err) {
			return nil, ErrConnectionAttemptState
		}
		return nil, fmt.Errorf("begin Worm connection attempt completion: %w", err)
	}
	if err := commitWalletTransaction(ctx, tx); err != nil {
		return nil, err
	}
	result := mapConnectionAttempt(updated)
	return &result, nil
}

func (s *SQLStore) FailConnectionAttempt(ctx context.Context, attemptID, failureCode string, at time.Time) error {
	return s.finishConnectionAttempt(ctx, attemptID, ConnectionAttemptStateFailed, failureCode, at)
}

func (s *SQLStore) MarkConnectionAttemptOutcomeUnknown(ctx context.Context, attemptID, warningCode string, at time.Time) error {
	return s.finishConnectionAttempt(ctx, attemptID, ConnectionAttemptStateOutcomeUnknown, warningCode, at)
}

func (s *SQLStore) finishConnectionAttempt(ctx context.Context, attemptID string, state ConnectionAttemptState, failureCode string, at time.Time) error {
	if err := validateWarningCode(failureCode); err != nil {
		return err
	}
	attempt, err := s.GetConnectionAttempt(ctx, attemptID)
	if err != nil {
		return err
	}
	now := canonicalNow(at)
	tx, queries, err := s.beginWalletTransaction(ctx, attempt.WalletID)
	if err != nil {
		return err
	}
	defer rollbackWalletTransaction(tx)
	locked, err := queries.GetConnectionAttemptForUpdate(ctx, mustUUIDParam(attempt.ID))
	if err != nil {
		if isNoRows(err) {
			return ErrConnectionAttemptNotFound
		}
		return fmt.Errorf("lock Worm connection attempt: %w", err)
	}
	if locked.State != string(ConnectionAttemptStatePrepared) && locked.State != string(ConnectionAttemptStateCompleting) {
		return ErrConnectionAttemptState
	}
	if state == ConnectionAttemptStateOutcomeUnknown {
		if _, err := queries.MarkConnectionAttemptTerminal(ctx, wormtradingsqlc.MarkConnectionAttemptTerminalParams{
			State:       string(state),
			FailureCode: failureCode,
			Now:         timestampParam(now),
			ID:          locked.ID,
		}); err != nil {
			return fmt.Errorf("mark Worm connection attempt outcome unknown: %w", err)
		}
		if _, err := queries.UpdateWalletConnectionState(ctx, wormtradingsqlc.UpdateWalletConnectionStateParams{
			State:       string(ConnectionStateReconnectRequired),
			WarningCode: failureCode,
			ConnectedAt: locked.PreviousConnectedAt,
			Now:         timestampParam(now),
			WalletID:    locked.WalletID,
			Address:     locked.Address,
		}); err != nil {
			return fmt.Errorf("mark Worm wallet reconnect required: %w", err)
		}
	} else if err := failConnectionAttemptInTransaction(ctx, queries, locked, failureCode, now); err != nil {
		return err
	}
	return commitWalletTransaction(ctx, tx)
}

func failConnectionAttemptInTransaction(ctx context.Context, queries *wormtradingsqlc.Queries, attempt wormtradingsqlc.WormWalletConnectionAttempt, failureCode string, now time.Time) error {
	if _, err := queries.MarkConnectionAttemptTerminal(ctx, wormtradingsqlc.MarkConnectionAttemptTerminalParams{
		State:       string(ConnectionAttemptStateFailed),
		FailureCode: failureCode,
		Now:         timestampParam(now),
		ID:          attempt.ID,
	}); err != nil {
		return fmt.Errorf("fail Worm connection attempt: %w", err)
	}
	warningCode, err := credentialCleanupWarning(ctx, queries, attempt.WalletID, failureCode)
	if err != nil {
		return err
	}
	if _, err := queries.UpdateWalletConnectionState(ctx, wormtradingsqlc.UpdateWalletConnectionStateParams{
		State:       attempt.PreviousConnectionState,
		WarningCode: warningCode,
		ConnectedAt: attempt.PreviousConnectedAt,
		Now:         timestampParam(now),
		WalletID:    attempt.WalletID,
		Address:     attempt.Address,
	}); err != nil {
		return fmt.Errorf("restore Worm wallet connection state: %w", err)
	}
	return nil
}

func validatePrepareConnectionAttemptRequest(req PrepareConnectionAttemptRequest) error {
	if err := validateWalletReference(req.WalletID, req.Address); err != nil {
		return err
	}
	if req.Kind != ConnectionAttemptKindConnect && req.Kind != ConnectionAttemptKindReconnect {
		return fmt.Errorf("connection attempt kind is invalid")
	}
	if req.Nonce == "" || req.Nonce != strings.TrimSpace(req.Nonce) || len(req.Nonce) > maxNonceLength {
		return fmt.Errorf("connection attempt nonce is invalid")
	}
	if req.ChallengeMessage == "" || len(req.ChallengeMessage) > maxChallengeLength {
		return fmt.Errorf("connection attempt challenge is invalid")
	}
	if len(req.MessageDigest) != 32 {
		return fmt.Errorf("connection attempt message digest must contain 32 bytes")
	}
	return nil
}

func validateConnectionAttemptKind(kind ConnectionAttemptKind, current ConnectionState) error {
	switch kind {
	case ConnectionAttemptKindConnect:
		if current != ConnectionStateNotConnected {
			return ErrConnectionAlreadyConnected
		}
	case ConnectionAttemptKindReconnect:
		if current == ConnectionStateNotConnected {
			return ErrConnectionNotConnected
		}
		if current == ConnectionStateConnecting || current == ConnectionStateDisconnecting {
			return ErrConnectionOperationActive
		}
	default:
		return fmt.Errorf("connection attempt kind is invalid")
	}
	return nil
}

func mustUUIDParam(value string) pgtype.UUID {
	parsed, _, err := uuidParam(value)
	if err != nil {
		panic(err)
	}
	return parsed
}
