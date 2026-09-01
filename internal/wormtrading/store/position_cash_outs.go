package store

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"math/big"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/gagliardetto/solana-go"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	wormtradingsqlc "github.com/useryege/athena/internal/wormtrading/store/sqlc"
)

const (
	maxPositionCashOutCodeLength    = 100
	maxPositionCashOutDecimalLength = 128
	maxPositionCashOutRecoveryLimit = 100
)

var positionCashOutPositiveDecimalPattern = regexp.MustCompile(`^(?:0|[1-9][0-9]*)(?:\.[0-9]+)?$`)

func (s *SQLStore) GetPositionCashOutCreation(
	ctx context.Context,
	req GetPositionCashOutCreationRequest,
) (*PositionCashOut, bool, error) {
	ownerUUID, ownerAccountID, err := marketCombinationUUID(req.OwnerAccountID, "owner account ID")
	if err != nil {
		return nil, false, invalidPositionCashOut(err)
	}
	commandUUID, commandID, err := marketCombinationUUID(req.CommandID, "command ID")
	if err != nil {
		return nil, false, invalidPositionCashOut(err)
	}
	if err := validateWalletReference(req.WalletID, req.WalletAddress); err != nil {
		return nil, false, invalidPositionCashOut(err)
	}
	if err := validatePositionCashOutPublicKey(req.PositionPubkey, "position pubkey", true); err != nil {
		return nil, false, err
	}
	req.OwnerAccountID = ownerAccountID
	req.CommandID = commandID
	requestDigest, err := positionCashOutCreationDigest(req)
	if err != nil {
		return nil, false, err
	}
	if err := s.requireDatabase(); err != nil {
		return nil, false, err
	}
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if err != nil {
		return nil, false, fmt.Errorf("begin get position Cash Out creation transaction: %w", err)
	}
	defer rollbackWalletTransaction(tx)
	queries := wormtradingsqlc.New(tx)
	command, err := queries.GetPositionCashOutCommand(ctx, commandUUID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, fmt.Errorf("get position Cash Out creation command: %w", err)
	}
	if command.OwnerAccountID != ownerUUID || command.Kind != string(PositionCashOutCommandKindCreate) ||
		!bytes.Equal(command.RequestSha256, requestDigest) {
		return nil, false, ErrPositionCashOutCommandConflict
	}
	cashOut, err := loadPositionCashOutByID(ctx, queries, command.CashOutID)
	if err != nil {
		return nil, false, err
	}
	if cashOut.OwnerAccountID != ownerAccountID || cashOut.WalletID != req.WalletID ||
		cashOut.WalletAddress != req.WalletAddress || cashOut.PositionPubkey != req.PositionPubkey {
		return nil, false, ErrPositionCashOutCommandConflict
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, false, fmt.Errorf("commit get position Cash Out creation: %w", err)
	}
	return &cashOut, true, nil
}

func (s *SQLStore) CreatePositionCashOut(
	ctx context.Context,
	req CreatePositionCashOutRequest,
) (*PositionCashOut, error) {
	ownerUUID, ownerAccountID, err := marketCombinationUUID(req.OwnerAccountID, "owner account ID")
	if err != nil {
		return nil, invalidPositionCashOut(err)
	}
	commandUUID, commandID, err := marketCombinationUUID(req.CommandID, "command ID")
	if err != nil {
		return nil, invalidPositionCashOut(err)
	}
	now := canonicalNow(req.Now)
	if err := normalizePositionCashOutCreateRequest(&req, now); err != nil {
		return nil, err
	}
	requestDigest, err := positionCashOutCreationDigest(GetPositionCashOutCreationRequest{
		OwnerAccountID: ownerAccountID,
		CommandID:      commandID,
		WalletID:       req.WalletID,
		WalletAddress:  req.WalletAddress,
		PositionPubkey: req.PositionPubkey,
	})
	if err != nil {
		return nil, err
	}
	intentDigest, err := positionCashOutDigest(struct {
		OwnerAccountID        string
		WalletID              int64
		WalletAddress         string
		PositionPubkey        string
		MarketConditionID     string
		IsYes                 bool
		PositionCreatedAt     time.Time
		PositionRequestPubkey string
		Shares                string
	}{
		req.OwnerAccountID, req.WalletID, req.WalletAddress, req.PositionPubkey,
		req.MarketConditionID, req.IsYes, req.PositionCreatedAt,
		req.PositionRequestPubkey, req.Shares,
	})
	if err != nil {
		return nil, err
	}

	tx, queries, err := s.beginWalletTransaction(ctx, req.WalletID)
	if err != nil {
		return nil, err
	}
	defer rollbackWalletTransaction(tx)

	if existing, getErr := queries.GetPositionCashOutCommand(ctx, commandUUID); getErr == nil {
		if existing.OwnerAccountID != ownerUUID || existing.Kind != string(PositionCashOutCommandKindCreate) ||
			!bytes.Equal(existing.RequestSha256, requestDigest) {
			return nil, ErrPositionCashOutCommandConflict
		}
		cashOut, loadErr := loadPositionCashOutByID(ctx, queries, existing.CashOutID)
		if loadErr != nil {
			return nil, loadErr
		}
		if err := tx.Commit(ctx); err != nil {
			return nil, fmt.Errorf("commit replayed position Cash Out creation: %w", err)
		}
		return &cashOut, nil
	} else if !errors.Is(getErr, pgx.ErrNoRows) {
		return nil, fmt.Errorf("get position Cash Out creation command: %w", getErr)
	}

	connection, err := queries.GetWalletConnectionForUpdate(ctx, req.WalletID)
	if err != nil || connection.Address != req.WalletAddress || connection.State != string(ConnectionStateConnected) {
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("get position Cash Out Wallet connection: %w", err)
		}
		return nil, ErrPositionCashOutConnectionChanged
	}
	credential, err := queries.GetActiveCredentialForUpdate(ctx, req.WalletID)
	if err != nil || credential.Version != req.CredentialVersion {
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("get position Cash Out active credential: %w", err)
		}
		return nil, ErrPositionCashOutConnectionChanged
	}
	if count, err := queries.CountActiveExecutionWalletLocksForCashOut(ctx, req.WalletID); err != nil {
		return nil, fmt.Errorf("count execution locks for position Cash Out: %w", err)
	} else if count != 0 {
		return nil, ErrPositionCashOutExecutionActive
	}
	if count, err := queries.CountActivePositionCashOutsForWallet(ctx, req.WalletID); err != nil {
		return nil, fmt.Errorf("count active position Cash Outs for Wallet: %w", err)
	} else if count != 0 {
		return nil, ErrPositionCashOutWalletActive
	}
	if count, err := queries.CountActivePositionCashOutBatchWalletLock(ctx, req.WalletID); err != nil {
		return nil, fmt.Errorf("count active position Cash Out batch lock for Wallet: %w", err)
	} else if count != 0 {
		return nil, ErrPositionCashOutBatchActive
	}

	state := PositionCashOutStateAwaitingAuthorization
	reasonCode := ""
	completedAt := time.Time{}
	if req.ProviderIsLiquidated {
		state = PositionCashOutStateFailed
		reasonCode = "POSITION_LIQUIDATED"
		completedAt = now
	} else if req.ProviderIsClosed {
		state = PositionCashOutStateCompleted
		completedAt = now
	}
	cashOutID := pgtype.UUID{Bytes: [16]byte(uuid.New()), Valid: true}
	row, err := queries.CreatePositionCashOut(ctx, wormtradingsqlc.CreatePositionCashOutParams{
		ID: cashOutID, OwnerAccountID: ownerUUID, WalletID: req.WalletID,
		WalletAddress: req.WalletAddress, CredentialVersion: req.CredentialVersion,
		PositionPubkey: req.PositionPubkey, MarketConditionID: req.MarketConditionID,
		IsYes: req.IsYes, PositionCreatedAt: timestampParam(req.PositionCreatedAt),
		PositionRequestPubkey: req.PositionRequestPubkey, Shares: req.Shares,
		IntentDigestSha256: intentDigest, State: string(state), ReasonCode: reasonCode,
		ProviderState: req.ProviderState, ProviderIsClosed: req.ProviderIsClosed,
		ProviderIsLiquidated:   req.ProviderIsLiquidated,
		AuthorizationExpiresAt: timestampParam(req.AuthorizationExpiresAt),
		CompletedAt:            timestampParam(completedAt), Now: timestampParam(now),
	})
	if err != nil {
		if executionConstraint(err, "worm_position_cash_outs_one_active_per_wallet_idx") ||
			executionConstraint(err, "worm_position_cash_outs_one_active_per_position_idx") {
			return nil, ErrPositionCashOutWalletActive
		}
		return nil, fmt.Errorf("create position Cash Out: %w", err)
	}
	if _, err := queries.CreatePositionCashOutCommand(ctx, wormtradingsqlc.CreatePositionCashOutCommandParams{
		ID: commandUUID, CashOutID: cashOutID, OwnerAccountID: ownerUUID,
		Kind: string(PositionCashOutCommandKindCreate), RequestSha256: requestDigest,
		CashOutRevisionAfter: row.Revision, ResultCode: "", Now: timestampParam(now),
	}); err != nil {
		if executionConstraint(err, "worm_position_cash_out_commands_pkey") {
			return nil, ErrPositionCashOutCommandConflict
		}
		return nil, fmt.Errorf("create position Cash Out creation command: %w", err)
	}
	cashOut, err := loadPositionCashOut(ctx, queries, row)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit create position Cash Out: %w", err)
	}
	return &cashOut, nil
}

func (s *SQLStore) GetPositionCashOut(
	ctx context.Context,
	ownerAccountID string,
	cashOutID string,
) (*PositionCashOut, error) {
	ownerUUID, _, id, err := positionCashOutOwnerAndID(ownerAccountID, cashOutID)
	if err != nil {
		return nil, err
	}
	if err := s.requireDatabase(); err != nil {
		return nil, err
	}
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if err != nil {
		return nil, fmt.Errorf("begin get position Cash Out transaction: %w", err)
	}
	defer rollbackWalletTransaction(tx)
	queries := wormtradingsqlc.New(tx)
	row, err := queries.GetPositionCashOut(ctx, wormtradingsqlc.GetPositionCashOutParams{
		ID: id, OwnerAccountID: ownerUUID,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrPositionCashOutNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get position Cash Out: %w", err)
	}
	cashOut, err := loadPositionCashOut(ctx, queries, row)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit get position Cash Out: %w", err)
	}
	return &cashOut, nil
}

func (s *SQLStore) ListActivePositionCashOuts(
	ctx context.Context,
	ownerAccountID string,
) ([]PositionCashOut, error) {
	ownerUUID, _, err := marketCombinationUUID(ownerAccountID, "owner account ID")
	if err != nil {
		return nil, invalidPositionCashOut(err)
	}
	if err := s.requireDatabase(); err != nil {
		return nil, err
	}
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if err != nil {
		return nil, fmt.Errorf("begin list active position Cash Outs transaction: %w", err)
	}
	defer rollbackWalletTransaction(tx)
	queries := wormtradingsqlc.New(tx)
	rows, err := queries.ListActivePositionCashOuts(ctx, ownerUUID)
	if err != nil {
		return nil, fmt.Errorf("list active position Cash Outs: %w", err)
	}
	result := make([]PositionCashOut, 0, len(rows))
	for _, row := range rows {
		cashOut, err := loadPositionCashOut(ctx, queries, row)
		if err != nil {
			return nil, err
		}
		result = append(result, cashOut)
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit list active position Cash Outs: %w", err)
	}
	return result, nil
}

func (s *SQLStore) ListActiveExecutionRunWalletIDs(
	ctx context.Context,
	ownerAccountID string,
	walletIDs []int64,
) ([]int64, error) {
	ownerUUID, _, err := marketCombinationUUID(ownerAccountID, "owner account ID")
	if err != nil {
		return nil, invalidPositionCashOut(err)
	}
	walletIDs, err = normalizePositionCashOutWalletIDs(walletIDs)
	if err != nil {
		return nil, err
	}
	if len(walletIDs) == 0 {
		return []int64{}, nil
	}
	if err := s.requireDatabase(); err != nil {
		return nil, err
	}
	rows, err := s.queries.ListActiveExecutionRunWalletIDs(ctx, wormtradingsqlc.ListActiveExecutionRunWalletIDsParams{
		OwnerAccountID: ownerUUID, WalletIds: walletIDs,
	})
	if err != nil {
		return nil, fmt.Errorf("list active execution Run Wallets: %w", err)
	}
	return append([]int64(nil), rows...), nil
}

func (s *SQLStore) AuthorizePositionCashOut(
	ctx context.Context,
	req AuthorizePositionCashOutRequest,
) (*PositionCashOut, error) {
	ownerUUID, _, cashOutID, err := positionCashOutOwnerAndID(req.OwnerAccountID, req.CashOutID)
	if err != nil {
		return nil, err
	}
	commandUUID, commandID, err := marketCombinationUUID(req.CommandID, "command ID")
	if err != nil {
		return nil, invalidPositionCashOut(err)
	}
	now := canonicalNow(req.Now)
	proofKind, err := normalizePositionCashOutCode(req.ProofKind, true)
	if err != nil {
		return nil, err
	}
	if len(req.SessionJTIDigest) != 32 || req.AccessRevision <= 0 ||
		!req.ExecutionExpiresAt.UTC().After(now) {
		return nil, invalidPositionCashOut(fmt.Errorf("Cash Out authorization binding is invalid"))
	}
	if req.ExpectedRevision <= 0 {
		return nil, invalidPositionCashOut(fmt.Errorf("expected revision must be positive"))
	}
	requestDigest, err := positionCashOutDigest(struct {
		OwnerAccountID   string
		CashOutID        string
		CommandID        string
		ExpectedRevision int64
		ProofKind        string
		SessionJTIDigest []byte
		AccessRevision   int64
	}{
		req.OwnerAccountID, req.CashOutID, commandID, req.ExpectedRevision,
		proofKind, req.SessionJTIDigest, req.AccessRevision,
	})
	if err != nil {
		return nil, err
	}
	walletID, err := s.positionCashOutWalletID(ctx, ownerUUID, cashOutID)
	if err != nil {
		return nil, err
	}
	tx, queries, err := s.beginWalletTransaction(ctx, walletID)
	if err != nil {
		return nil, err
	}
	defer rollbackWalletTransaction(tx)
	if existing, getErr := queries.GetPositionCashOutCommand(ctx, commandUUID); getErr == nil {
		if existing.OwnerAccountID != ownerUUID || existing.CashOutID != cashOutID ||
			existing.Kind != string(PositionCashOutCommandKindAuthorize) ||
			!bytes.Equal(existing.RequestSha256, requestDigest) {
			return nil, ErrPositionCashOutCommandConflict
		}
		cashOut, loadErr := loadPositionCashOutByID(ctx, queries, cashOutID)
		if loadErr != nil {
			return nil, loadErr
		}
		if err := tx.Commit(ctx); err != nil {
			return nil, fmt.Errorf("commit replayed position Cash Out authorization: %w", err)
		}
		return &cashOut, nil
	} else if !errors.Is(getErr, pgx.ErrNoRows) {
		return nil, fmt.Errorf("get position Cash Out authorization command: %w", getErr)
	}
	row, err := queries.GetPositionCashOutForUpdate(ctx, wormtradingsqlc.GetPositionCashOutForUpdateParams{
		ID: cashOutID, OwnerAccountID: ownerUUID,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrPositionCashOutNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("lock position Cash Out for authorization: %w", err)
	}
	if row.Revision != req.ExpectedRevision || row.State != string(PositionCashOutStateAwaitingAuthorization) {
		return nil, ErrPositionCashOutRevision
	}
	if !timestampValue(row.AuthorizationExpiresAt).After(now) {
		return nil, fmt.Errorf("%w: position Cash Out authorization", ErrExpired)
	}
	authorizationID := pgtype.UUID{Bytes: [16]byte(uuid.New()), Valid: true}
	if _, err := queries.CreatePositionCashOutAuthorization(ctx, wormtradingsqlc.CreatePositionCashOutAuthorizationParams{
		ID: authorizationID, CashOutID: cashOutID, OwnerAccountID: ownerUUID,
		ProofKind: proofKind, SessionJtiDigest: append([]byte(nil), req.SessionJTIDigest...),
		AccessRevision: req.AccessRevision, IntentDigestSha256: row.IntentDigestSha256,
		Now: timestampParam(now),
	}); err != nil {
		return nil, fmt.Errorf("create position Cash Out authorization: %w", err)
	}
	queued, err := queries.QueueAuthorizedPositionCashOut(ctx, wormtradingsqlc.QueueAuthorizedPositionCashOutParams{
		ID: cashOutID, OwnerAccountID: ownerUUID, ExpectedRevision: req.ExpectedRevision,
		ExecutionExpiresAt: timestampParam(req.ExecutionExpiresAt), Now: timestampParam(now),
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrPositionCashOutRevision
	}
	if err != nil {
		return nil, fmt.Errorf("queue authorized position Cash Out: %w", err)
	}
	if _, err := queries.CreatePositionCashOutCommand(ctx, wormtradingsqlc.CreatePositionCashOutCommandParams{
		ID: commandUUID, CashOutID: cashOutID, OwnerAccountID: ownerUUID,
		Kind: string(PositionCashOutCommandKindAuthorize), RequestSha256: requestDigest,
		CashOutRevisionAfter: queued.Revision, ResultCode: "", Now: timestampParam(now),
	}); err != nil {
		return nil, fmt.Errorf("create position Cash Out authorization command: %w", err)
	}
	cashOut, err := loadPositionCashOut(ctx, queries, queued)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit position Cash Out authorization: %w", err)
	}
	return &cashOut, nil
}

func (s *SQLStore) ClaimPositionCashOut(
	ctx context.Context,
	req PositionCashOutClaimRequest,
) (*PositionCashOut, error) {
	id, claimID, workerID, now, err := normalizePositionCashOutClaim(req)
	if err != nil {
		return nil, err
	}
	if err := s.requireDatabase(); err != nil {
		return nil, err
	}
	row, err := s.queries.ClaimPositionCashOut(ctx, wormtradingsqlc.ClaimPositionCashOutParams{
		ID: id, ClaimID: claimID, ClaimOwner: workerID,
		ClaimExpiresAt: timestampParam(req.LeaseExpiresAt), Now: timestampParam(now),
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrPositionCashOutClaim
	}
	if err != nil {
		return nil, fmt.Errorf("claim position Cash Out: %w", err)
	}
	cashOut, err := loadPositionCashOut(ctx, s.queries, row)
	if err != nil {
		return nil, err
	}
	return &cashOut, nil
}

func (s *SQLStore) RenewPositionCashOutClaim(
	ctx context.Context,
	req PositionCashOutClaimRequest,
) (*PositionCashOut, error) {
	id, claimID, workerID, now, err := normalizePositionCashOutClaim(req)
	if err != nil {
		return nil, err
	}
	if err := s.requireDatabase(); err != nil {
		return nil, err
	}
	row, err := s.queries.RenewPositionCashOutClaim(ctx, wormtradingsqlc.RenewPositionCashOutClaimParams{
		ID: id, ClaimID: claimID, ClaimOwner: workerID,
		ClaimExpiresAt: timestampParam(req.LeaseExpiresAt), Now: timestampParam(now),
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrPositionCashOutClaim
	}
	if err != nil {
		return nil, fmt.Errorf("renew position Cash Out claim: %w", err)
	}
	cashOut, err := loadPositionCashOut(ctx, s.queries, row)
	if err != nil {
		return nil, err
	}
	return &cashOut, nil
}

func (s *SQLStore) BeginPositionCashOutClosing(
	ctx context.Context,
	req BeginPositionCashOutClosingRequest,
) (*PositionCashOut, error) {
	id, _, err := marketCombinationUUID(req.CashOutID, "position Cash Out ID")
	if err != nil {
		return nil, invalidPositionCashOut(err)
	}
	claimID, _, err := marketCombinationUUID(req.ClaimID, "claim ID")
	if err != nil {
		return nil, invalidPositionCashOut(err)
	}
	providerState, err := normalizePositionCashOutCode(req.ProviderState, false)
	if err != nil {
		return nil, err
	}
	if err := s.requireDatabase(); err != nil {
		return nil, err
	}
	row, err := s.queries.BeginPositionCashOutClosing(ctx, wormtradingsqlc.BeginPositionCashOutClosingParams{
		ID: id, ClaimID: claimID, ProviderState: providerState,
		Now: timestampParam(canonicalNow(req.Now)),
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrPositionCashOutClaim
	}
	if err != nil {
		return nil, fmt.Errorf("begin position Cash Out Close: %w", err)
	}
	cashOut, err := loadPositionCashOut(ctx, s.queries, row)
	if err != nil {
		return nil, err
	}
	return &cashOut, nil
}

func (s *SQLStore) PreparePositionCashOutAttempt(
	ctx context.Context,
	req PreparePositionCashOutAttemptRequest,
) (*PositionCashOutAttempt, error) {
	attemptID, _, err := marketCombinationUUID(req.AttemptID, "position Cash Out attempt ID")
	if err != nil {
		return nil, invalidPositionCashOut(err)
	}
	cashOutID, _, err := marketCombinationUUID(req.CashOutID, "position Cash Out ID")
	if err != nil {
		return nil, invalidPositionCashOut(err)
	}
	claimID, _, err := marketCombinationUUID(req.ClaimID, "claim ID")
	if err != nil {
		return nil, invalidPositionCashOut(err)
	}
	if len(req.RequestSHA256) != 32 {
		return nil, invalidPositionCashOut(fmt.Errorf("Close request digest must contain 32 bytes"))
	}
	if err := s.requireDatabase(); err != nil {
		return nil, err
	}
	row, err := s.queries.CreatePositionCashOutAttempt(ctx, wormtradingsqlc.CreatePositionCashOutAttemptParams{
		ID: attemptID, CashOutID: cashOutID, ClaimID: claimID,
		RequestSha256: append([]byte(nil), req.RequestSHA256...),
		Now:           timestampParam(canonicalNow(req.Now)),
	})
	if err != nil {
		if executionConstraint(err, "worm_position_cash_out_attempts_cash_out_id_key") {
			existing, getErr := s.queries.GetPositionCashOutAttempt(ctx, cashOutID)
			if getErr == nil && existing.ID == attemptID &&
				existing.State == string(PositionCashOutAttemptStatePrepared) &&
				bytes.Equal(existing.RequestSha256, req.RequestSHA256) {
				attempt := mapPositionCashOutAttempt(existing)
				return &attempt, nil
			}
			return nil, ErrPositionCashOutClaim
		}
		return nil, fmt.Errorf("prepare position Cash Out attempt: %w", err)
	}
	attempt := mapPositionCashOutAttempt(row)
	return &attempt, nil
}

func (s *SQLStore) DispatchPositionCashOutAttempt(
	ctx context.Context,
	req DispatchPositionCashOutAttemptRequest,
) (*PositionCashOutAttempt, error) {
	attemptID, _, err := marketCombinationUUID(req.AttemptID, "position Cash Out attempt ID")
	if err != nil {
		return nil, invalidPositionCashOut(err)
	}
	cashOutID, _, err := marketCombinationUUID(req.CashOutID, "position Cash Out ID")
	if err != nil {
		return nil, invalidPositionCashOut(err)
	}
	claimID, _, err := marketCombinationUUID(req.ClaimID, "claim ID")
	if err != nil {
		return nil, invalidPositionCashOut(err)
	}
	if err := s.requireDatabase(); err != nil {
		return nil, err
	}
	now := canonicalNow(req.Now)
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, fmt.Errorf("begin dispatch position Cash Out attempt transaction: %w", err)
	}
	defer rollbackWalletTransaction(tx)
	queries := wormtradingsqlc.New(tx)
	row, err := queries.DispatchPositionCashOutAttempt(ctx, wormtradingsqlc.DispatchPositionCashOutAttemptParams{
		ID: attemptID, CashOutID: cashOutID, ClaimID: claimID,
		Now: timestampParam(now),
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrPositionCashOutClaim
	}
	if err != nil {
		return nil, fmt.Errorf("dispatch position Cash Out attempt: %w", err)
	}
	if count, err := queries.TouchPositionCashOutAfterDispatch(ctx, wormtradingsqlc.TouchPositionCashOutAfterDispatchParams{
		ID: cashOutID, ClaimID: claimID, Now: timestampParam(now),
	}); err != nil || count != 1 {
		if err != nil {
			return nil, fmt.Errorf("checkpoint position Cash Out dispatch: %w", err)
		}
		return nil, ErrPositionCashOutClaim
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit position Cash Out dispatch: %w", err)
	}
	attempt := mapPositionCashOutAttempt(row)
	return &attempt, nil
}

func (s *SQLStore) ResolvePositionCashOutAttempt(
	ctx context.Context,
	req ResolvePositionCashOutAttemptRequest,
) (*PositionCashOutAttempt, error) {
	attemptID, _, err := marketCombinationUUID(req.AttemptID, "position Cash Out attempt ID")
	if err != nil {
		return nil, invalidPositionCashOut(err)
	}
	cashOutID, _, err := marketCombinationUUID(req.CashOutID, "position Cash Out ID")
	if err != nil {
		return nil, invalidPositionCashOut(err)
	}
	if !validPositionCashOutResolvedAttemptState(req.State) || req.HTTPStatus < 0 || req.HTTPStatus > 599 {
		return nil, invalidPositionCashOut(fmt.Errorf("Close attempt result is invalid"))
	}
	providerSlug, err := normalizePositionCashOutCode(req.ProviderSlug, false)
	if err != nil {
		return nil, err
	}
	providerState, err := normalizePositionCashOutCode(req.ProviderState, false)
	if err != nil {
		return nil, err
	}
	errorCode, err := normalizePositionCashOutCode(req.ErrorCode, false)
	if err != nil {
		return nil, err
	}
	if err := s.requireDatabase(); err != nil {
		return nil, err
	}
	row, err := s.queries.ResolvePositionCashOutAttempt(ctx, wormtradingsqlc.ResolvePositionCashOutAttemptParams{
		ID: attemptID, CashOutID: cashOutID, State: string(req.State),
		HttpStatus: nullableIntegerParam(req.HTTPStatus), ProviderCode: nullableIntegerParam(req.ProviderCode),
		ProviderSlug: providerSlug, ProviderState: providerState, ErrorCode: errorCode,
		Now: timestampParam(canonicalNow(req.Now)),
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrPositionCashOutClaim
	}
	if err != nil {
		return nil, fmt.Errorf("resolve position Cash Out attempt: %w", err)
	}
	attempt := mapPositionCashOutAttempt(row)
	return &attempt, nil
}

func (s *SQLStore) CompletePositionCashOutDispatch(
	ctx context.Context,
	req CompletePositionCashOutDispatchRequest,
) (*PositionCashOut, error) {
	attemptID, _, err := marketCombinationUUID(req.AttemptID, "position Cash Out attempt ID")
	if err != nil {
		return nil, invalidPositionCashOut(err)
	}
	cashOutID, _, err := marketCombinationUUID(req.CashOutID, "position Cash Out ID")
	if err != nil {
		return nil, invalidPositionCashOut(err)
	}
	claimID, _, err := marketCombinationUUID(req.ClaimID, "claim ID")
	if err != nil {
		return nil, invalidPositionCashOut(err)
	}
	if req.HTTPStatus < 0 || req.HTTPStatus > 599 {
		return nil, invalidPositionCashOut(fmt.Errorf("Close attempt result is invalid"))
	}
	providerSlug, err := normalizePositionCashOutCode(req.ProviderSlug, false)
	if err != nil {
		return nil, err
	}
	providerState, err := normalizePositionCashOutCode(req.ProviderState, true)
	if err != nil || providerState != "closed" {
		return nil, invalidPositionCashOut(fmt.Errorf("completed Close response state is invalid"))
	}
	errorCode, err := normalizePositionCashOutCode(req.ErrorCode, false)
	if err != nil {
		return nil, err
	}
	if err := s.requireDatabase(); err != nil {
		return nil, err
	}
	now := canonicalNow(req.Now)
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, fmt.Errorf("begin complete position Cash Out dispatch transaction: %w", err)
	}
	defer rollbackWalletTransaction(tx)
	queries := wormtradingsqlc.New(tx)
	if _, err := queries.ResolvePositionCashOutAttempt(ctx, wormtradingsqlc.ResolvePositionCashOutAttemptParams{
		ID: attemptID, CashOutID: cashOutID, State: string(PositionCashOutAttemptStateAcknowledged),
		HttpStatus: nullableIntegerParam(req.HTTPStatus), ProviderCode: nullableIntegerParam(req.ProviderCode),
		ProviderSlug: providerSlug, ProviderState: providerState, ErrorCode: errorCode,
		Now: timestampParam(now),
	}); errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrPositionCashOutClaim
	} else if err != nil {
		return nil, fmt.Errorf("resolve completed position Cash Out attempt: %w", err)
	}
	row, err := queries.RecordPositionCashOutObservation(ctx, wormtradingsqlc.RecordPositionCashOutObservationParams{
		ID: cashOutID, ClaimID: claimID, ExpectedState: string(PositionCashOutStateClosing),
		NextState: string(PositionCashOutStateCompleted), ReasonCode: "",
		ProviderState: providerState, ProviderIsClosed: true, ProviderIsLiquidated: false,
		NextPollAt: timestampParam(time.Time{}), Now: timestampParam(now),
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrPositionCashOutClaim
	}
	if err != nil {
		return nil, fmt.Errorf("complete position Cash Out from response: %w", err)
	}
	if _, err := queries.EndPositionCashOutAuthorization(ctx, wormtradingsqlc.EndPositionCashOutAuthorizationParams{
		State: string(PositionCashOutAuthorizationStateConsumed), Now: timestampParam(now),
		ReasonCode: string(PositionCashOutStateCompleted), CashOutID: cashOutID,
	}); err != nil {
		return nil, fmt.Errorf("consume completed position Cash Out authorization: %w", err)
	}
	cashOut, err := loadPositionCashOut(ctx, queries, row)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit completed position Cash Out dispatch: %w", err)
	}
	return &cashOut, nil
}

func (s *SQLStore) RecordPositionCashOutObservation(
	ctx context.Context,
	req RecordPositionCashOutObservationRequest,
) (*PositionCashOut, error) {
	cashOutID, _, err := marketCombinationUUID(req.CashOutID, "position Cash Out ID")
	if err != nil {
		return nil, invalidPositionCashOut(err)
	}
	claimID, _, err := marketCombinationUUID(req.ClaimID, "claim ID")
	if err != nil {
		return nil, invalidPositionCashOut(err)
	}
	if !validPositionCashOutObservationTransition(req.ExpectedState, req.NextState) {
		return nil, invalidPositionCashOut(fmt.Errorf("Cash Out observation transition is invalid"))
	}
	reasonCode, err := normalizePositionCashOutCode(req.ReasonCode,
		req.NextState == PositionCashOutStateFailed || req.NextState == PositionCashOutStateReconciliationRequired)
	if err != nil {
		return nil, err
	}
	providerState, err := normalizePositionCashOutCode(req.ProviderState, false)
	if err != nil {
		return nil, err
	}
	if req.NextState == PositionCashOutStateCompleted &&
		(!req.ProviderIsClosed || req.ProviderIsLiquidated) {
		return nil, invalidPositionCashOut(fmt.Errorf("completed Cash Out requires non-liquidated closed-position evidence"))
	}
	now := canonicalNow(req.Now)
	if (req.NextState == PositionCashOutStateAwaitingCompletion ||
		req.NextState == PositionCashOutStateReconciliationRequired) &&
		!req.NextPollAt.UTC().After(now) {
		return nil, invalidPositionCashOut(fmt.Errorf("pending Cash Out requires a future poll time"))
	}
	if err := s.requireDatabase(); err != nil {
		return nil, err
	}
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, fmt.Errorf("begin record position Cash Out observation transaction: %w", err)
	}
	defer rollbackWalletTransaction(tx)
	queries := wormtradingsqlc.New(tx)
	row, err := queries.RecordPositionCashOutObservation(ctx, wormtradingsqlc.RecordPositionCashOutObservationParams{
		ID: cashOutID, ClaimID: claimID, ExpectedState: string(req.ExpectedState),
		NextState: string(req.NextState), ReasonCode: reasonCode,
		ProviderState: providerState, ProviderIsClosed: req.ProviderIsClosed,
		ProviderIsLiquidated: req.ProviderIsLiquidated,
		NextPollAt:           timestampParam(req.NextPollAt), Now: timestampParam(now),
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrPositionCashOutClaim
	}
	if err != nil {
		return nil, fmt.Errorf("record position Cash Out observation: %w", err)
	}
	if req.NextState == PositionCashOutStateCompleted || req.NextState == PositionCashOutStateFailed {
		if _, err := queries.EndPositionCashOutAuthorization(ctx, wormtradingsqlc.EndPositionCashOutAuthorizationParams{
			State: string(PositionCashOutAuthorizationStateConsumed), Now: timestampParam(now),
			ReasonCode: string(req.NextState), CashOutID: cashOutID,
		}); err != nil {
			return nil, fmt.Errorf("end position Cash Out authorization: %w", err)
		}
	}
	cashOut, err := loadPositionCashOut(ctx, queries, row)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit position Cash Out observation: %w", err)
	}
	return &cashOut, nil
}

func (s *SQLStore) RequestPositionCashOutReconciliation(
	ctx context.Context,
	req PositionCashOutCommandRequest,
) (*PositionCashOut, error) {
	ownerUUID, _, cashOutID, err := positionCashOutOwnerAndID(req.OwnerAccountID, req.CashOutID)
	if err != nil {
		return nil, err
	}
	commandUUID, commandID, err := marketCombinationUUID(req.CommandID, "command ID")
	if err != nil {
		return nil, invalidPositionCashOut(err)
	}
	if req.ExpectedRevision <= 0 {
		return nil, invalidPositionCashOut(fmt.Errorf("expected revision must be positive"))
	}
	now := canonicalNow(req.Now)
	requestDigest, err := positionCashOutDigest(struct {
		OwnerAccountID   string
		CashOutID        string
		CommandID        string
		ExpectedRevision int64
	}{req.OwnerAccountID, req.CashOutID, commandID, req.ExpectedRevision})
	if err != nil {
		return nil, err
	}
	walletID, err := s.positionCashOutWalletID(ctx, ownerUUID, cashOutID)
	if err != nil {
		return nil, err
	}
	tx, queries, err := s.beginWalletTransaction(ctx, walletID)
	if err != nil {
		return nil, err
	}
	defer rollbackWalletTransaction(tx)
	if existing, getErr := queries.GetPositionCashOutCommand(ctx, commandUUID); getErr == nil {
		if existing.OwnerAccountID != ownerUUID || existing.CashOutID != cashOutID ||
			existing.Kind != string(PositionCashOutCommandKindReconcile) ||
			!bytes.Equal(existing.RequestSha256, requestDigest) {
			return nil, ErrPositionCashOutCommandConflict
		}
		cashOut, loadErr := loadPositionCashOutByID(ctx, queries, cashOutID)
		if loadErr != nil {
			return nil, loadErr
		}
		if err := tx.Commit(ctx); err != nil {
			return nil, fmt.Errorf("commit replayed position Cash Out reconciliation: %w", err)
		}
		return &cashOut, nil
	} else if !errors.Is(getErr, pgx.ErrNoRows) {
		return nil, fmt.Errorf("get position Cash Out reconciliation command: %w", getErr)
	}
	row, err := queries.RequestPositionCashOutReconciliation(ctx, wormtradingsqlc.RequestPositionCashOutReconciliationParams{
		ID: cashOutID, OwnerAccountID: ownerUUID, ExpectedRevision: req.ExpectedRevision,
		Now: timestampParam(now),
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrPositionCashOutRevision
	}
	if err != nil {
		return nil, fmt.Errorf("request position Cash Out reconciliation: %w", err)
	}
	if _, err := queries.CreatePositionCashOutCommand(ctx, wormtradingsqlc.CreatePositionCashOutCommandParams{
		ID: commandUUID, CashOutID: cashOutID, OwnerAccountID: ownerUUID,
		Kind: string(PositionCashOutCommandKindReconcile), RequestSha256: requestDigest,
		CashOutRevisionAfter: row.Revision, ResultCode: "", Now: timestampParam(now),
	}); err != nil {
		return nil, fmt.Errorf("create position Cash Out reconciliation command: %w", err)
	}
	cashOut, err := loadPositionCashOut(ctx, queries, row)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit position Cash Out reconciliation: %w", err)
	}
	return &cashOut, nil
}

func (s *SQLStore) ListRecoverablePositionCashOuts(
	ctx context.Context,
	now time.Time,
	limit int32,
) ([]PositionCashOut, error) {
	if limit <= 0 || limit > maxPositionCashOutRecoveryLimit {
		return nil, invalidPositionCashOut(fmt.Errorf("recovery limit must be between 1 and %d", maxPositionCashOutRecoveryLimit))
	}
	if err := s.requireDatabase(); err != nil {
		return nil, err
	}
	rows, err := s.queries.ListRecoverablePositionCashOutKeys(ctx, wormtradingsqlc.ListRecoverablePositionCashOutKeysParams{
		Now: timestampParam(canonicalNow(now)), RecoveryLimit: limit,
	})
	if err != nil {
		return nil, fmt.Errorf("list recoverable position Cash Outs: %w", err)
	}
	result := make([]PositionCashOut, 0, len(rows))
	for _, id := range rows {
		cashOut, err := loadPositionCashOutByID(ctx, s.queries, id)
		if errors.Is(err, ErrPositionCashOutNotFound) {
			continue
		}
		if err != nil {
			return nil, err
		}
		result = append(result, cashOut)
	}
	return result, nil
}

func (s *SQLStore) ExpirePositionCashOuts(
	ctx context.Context,
	now time.Time,
	limit int32,
) (int64, error) {
	if limit <= 0 || limit > maxPositionCashOutRecoveryLimit {
		return 0, invalidPositionCashOut(fmt.Errorf("expiry limit must be between 1 and %d", maxPositionCashOutRecoveryLimit))
	}
	if err := s.requireDatabase(); err != nil {
		return 0, err
	}
	now = canonicalNow(now)
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return 0, fmt.Errorf("begin expire position Cash Outs transaction: %w", err)
	}
	defer rollbackWalletTransaction(tx)
	queries := wormtradingsqlc.New(tx)
	unauthorized, err := queries.ExpireUnauthorizedPositionCashOuts(ctx, wormtradingsqlc.ExpireUnauthorizedPositionCashOutsParams{
		Now: timestampParam(now), ExpireLimit: limit,
	})
	if err != nil {
		return 0, fmt.Errorf("expire unauthorized position Cash Outs: %w", err)
	}
	remaining := limit - int32(len(unauthorized))
	undispatched := []wormtradingsqlc.WormPositionCashOut{}
	if remaining > 0 {
		undispatched, err = queries.ExpireUndispatchedPositionCashOuts(ctx, wormtradingsqlc.ExpireUndispatchedPositionCashOutsParams{
			Now: timestampParam(now), ExpireLimit: remaining,
		})
		if err != nil {
			return 0, fmt.Errorf("expire undispatched position Cash Outs: %w", err)
		}
		for _, row := range undispatched {
			if _, err := queries.EndPositionCashOutAuthorization(ctx, wormtradingsqlc.EndPositionCashOutAuthorizationParams{
				State: string(PositionCashOutAuthorizationStateRevoked), Now: timestampParam(now),
				ReasonCode: "AUTHORIZATION_EXPIRED", CashOutID: row.ID,
			}); err != nil {
				return 0, fmt.Errorf("revoke expired position Cash Out authorization: %w", err)
			}
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("commit expire position Cash Outs: %w", err)
	}
	return int64(len(unauthorized) + len(undispatched)), nil
}

func (s *SQLStore) positionCashOutWalletID(
	ctx context.Context,
	ownerAccountID pgtype.UUID,
	cashOutID pgtype.UUID,
) (int64, error) {
	if err := s.requireDatabase(); err != nil {
		return 0, err
	}
	row, err := s.queries.GetPositionCashOut(ctx, wormtradingsqlc.GetPositionCashOutParams{
		ID: cashOutID, OwnerAccountID: ownerAccountID,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, ErrPositionCashOutNotFound
	}
	if err != nil {
		return 0, fmt.Errorf("get position Cash Out Wallet: %w", err)
	}
	return row.WalletID, nil
}

func loadPositionCashOutByID(
	ctx context.Context,
	queries *wormtradingsqlc.Queries,
	id pgtype.UUID,
) (PositionCashOut, error) {
	row, err := queries.GetPositionCashOutByID(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return PositionCashOut{}, ErrPositionCashOutNotFound
	}
	if err != nil {
		return PositionCashOut{}, fmt.Errorf("get position Cash Out by ID: %w", err)
	}
	return loadPositionCashOut(ctx, queries, row)
}

func loadPositionCashOut(
	ctx context.Context,
	queries *wormtradingsqlc.Queries,
	row wormtradingsqlc.WormPositionCashOut,
) (PositionCashOut, error) {
	cashOut := mapPositionCashOut(row)
	authorization, err := queries.GetPositionCashOutAuthorization(ctx, row.ID)
	if err == nil {
		mapped := mapPositionCashOutAuthorization(authorization)
		cashOut.Authorization = &mapped
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return PositionCashOut{}, fmt.Errorf("get position Cash Out authorization: %w", err)
	}
	attempt, err := queries.GetPositionCashOutAttempt(ctx, row.ID)
	if err == nil {
		mapped := mapPositionCashOutAttempt(attempt)
		cashOut.Attempt = &mapped
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return PositionCashOut{}, fmt.Errorf("get position Cash Out attempt: %w", err)
	}
	return cashOut, nil
}

func mapPositionCashOut(row wormtradingsqlc.WormPositionCashOut) PositionCashOut {
	return PositionCashOut{
		ID: uuidValue(row.ID), OwnerAccountID: uuidValue(row.OwnerAccountID),
		WalletID: row.WalletID, WalletAddress: row.WalletAddress,
		CredentialVersion: row.CredentialVersion, PositionPubkey: row.PositionPubkey,
		MarketConditionID: row.MarketConditionID, IsYes: row.IsYes,
		PositionCreatedAt:     timestampValue(row.PositionCreatedAt),
		PositionRequestPubkey: row.PositionRequestPubkey, Shares: row.Shares,
		IntentDigestSHA256: append([]byte(nil), row.IntentDigestSha256...),
		State:              PositionCashOutState(row.State), Revision: row.Revision,
		ReasonCode: row.ReasonCode, ProviderState: row.ProviderState,
		ProviderIsClosed: row.ProviderIsClosed, ProviderIsLiquidated: row.ProviderIsLiquidated,
		AuthorizationExpiresAt: timestampValue(row.AuthorizationExpiresAt),
		ExecutionExpiresAt:     timestampValue(row.ExecutionExpiresAt),
		AuthorizedAt:           timestampValue(row.AuthorizedAt), NextPollAt: timestampValue(row.NextPollAt),
		PollCount: row.PollCount, ReconcileRequestedAt: timestampValue(row.ReconcileRequestedAt),
		ClaimID: uuidValue(row.ClaimID), ClaimOwner: row.ClaimOwner,
		ClaimExpiresAt: timestampValue(row.ClaimExpiresAt), CompletedAt: timestampValue(row.CompletedAt),
		CreatedAt: timestampValue(row.CreatedAt), UpdatedAt: timestampValue(row.UpdatedAt),
		BatchID: uuidValue(row.BatchID), BatchItemID: uuidValue(row.BatchItemID),
	}
}

func mapPositionCashOutAuthorization(
	row wormtradingsqlc.WormPositionCashOutAuthorization,
) PositionCashOutAuthorization {
	return PositionCashOutAuthorization{
		ID: uuidValue(row.ID), State: PositionCashOutAuthorizationState(row.State),
		Scope: row.Scope, ProofKind: row.ProofKind,
		SessionJTIDigest:   append([]byte(nil), row.SessionJtiDigest...),
		AccessRevision:     row.AccessRevision,
		IntentDigestSHA256: append([]byte(nil), row.IntentDigestSha256...),
		AuthorizedAt:       timestampValue(row.AuthorizedAt), EndedAt: timestampValue(row.EndedAt),
		EndReasonCode: row.EndReasonCode,
	}
}

func mapPositionCashOutAttempt(
	row wormtradingsqlc.WormPositionCashOutAttempt,
) PositionCashOutAttempt {
	return PositionCashOutAttempt{
		ID: uuidValue(row.ID), State: PositionCashOutAttemptState(row.State),
		RequestSHA256: append([]byte(nil), row.RequestSha256...),
		HTTPStatus:    nullableInt32(row.HttpStatus), ProviderCode: nullableInt32(row.ProviderCode),
		ProviderSlug: row.ProviderSlug, ProviderState: row.ProviderState,
		ErrorCode: row.ErrorCode, PreparedAt: timestampValue(row.PreparedAt),
		DispatchedAt: timestampValue(row.DispatchedAt), CompletedAt: timestampValue(row.CompletedAt),
	}
}

func positionCashOutOwnerAndID(
	ownerAccountID string,
	cashOutID string,
) (pgtype.UUID, string, pgtype.UUID, error) {
	ownerUUID, ownerAccountID, err := marketCombinationUUID(ownerAccountID, "owner account ID")
	if err != nil {
		return pgtype.UUID{}, "", pgtype.UUID{}, invalidPositionCashOut(err)
	}
	id, _, err := marketCombinationUUID(cashOutID, "position Cash Out ID")
	if err != nil {
		return pgtype.UUID{}, "", pgtype.UUID{}, invalidPositionCashOut(err)
	}
	return ownerUUID, ownerAccountID, id, nil
}

func normalizePositionCashOutCreateRequest(req *CreatePositionCashOutRequest, now time.Time) error {
	if req == nil {
		return invalidPositionCashOut(fmt.Errorf("request is required"))
	}
	if err := validateWalletReference(req.WalletID, req.WalletAddress); err != nil {
		return invalidPositionCashOut(err)
	}
	if req.CredentialVersion <= 0 {
		return invalidPositionCashOut(fmt.Errorf("credential version must be positive"))
	}
	if err := validatePositionCashOutPublicKey(req.PositionPubkey, "position pubkey", true); err != nil {
		return err
	}
	if err := validatePositionCashOutPublicKey(req.MarketConditionID, "market condition ID", true); err != nil {
		return err
	}
	if err := validatePositionCashOutPublicKey(req.PositionRequestPubkey, "position request pubkey", false); err != nil {
		return err
	}
	if req.PositionCreatedAt.IsZero() || !req.PositionCreatedAt.UTC().After(time.Unix(0, 0).UTC()) {
		return invalidPositionCashOut(fmt.Errorf("position creation time is invalid"))
	}
	req.PositionCreatedAt = req.PositionCreatedAt.UTC()
	if len(req.Shares) == 0 || len(req.Shares) > maxPositionCashOutDecimalLength ||
		!positionCashOutPositiveDecimalPattern.MatchString(req.Shares) {
		return invalidPositionCashOut(fmt.Errorf("position shares must be a canonical positive decimal"))
	}
	value, ok := new(big.Rat).SetString(req.Shares)
	if !ok || value.Sign() <= 0 {
		return invalidPositionCashOut(fmt.Errorf("position shares must be a canonical positive decimal"))
	}
	providerState, err := normalizePositionCashOutCode(req.ProviderState, false)
	if err != nil {
		return err
	}
	req.ProviderState = providerState
	if !req.AuthorizationExpiresAt.UTC().After(now) {
		return invalidPositionCashOut(fmt.Errorf("authorization expiry must be in the future"))
	}
	req.AuthorizationExpiresAt = req.AuthorizationExpiresAt.UTC()
	return nil
}

func validatePositionCashOutPublicKey(value string, label string, required bool) error {
	if value == "" && !required {
		return nil
	}
	if value == "" || value != strings.TrimSpace(value) {
		return invalidPositionCashOut(fmt.Errorf("%s is invalid", label))
	}
	publicKey, err := solana.PublicKeyFromBase58(value)
	if err != nil || publicKey.IsZero() || publicKey.String() != value {
		return invalidPositionCashOut(fmt.Errorf("%s must be a canonical Solana public key", label))
	}
	return nil
}

func normalizePositionCashOutCode(value string, required bool) (string, error) {
	trimmed := strings.TrimSpace(value)
	if value != trimmed || (required && trimmed == "") || len(trimmed) > maxPositionCashOutCodeLength {
		return "", invalidPositionCashOut(fmt.Errorf("position Cash Out code is invalid"))
	}
	for _, character := range trimmed {
		if character < 0x20 || character == 0x7f {
			return "", invalidPositionCashOut(fmt.Errorf("position Cash Out code is invalid"))
		}
	}
	return trimmed, nil
}

func normalizePositionCashOutClaim(
	req PositionCashOutClaimRequest,
) (pgtype.UUID, pgtype.UUID, string, time.Time, error) {
	id, _, err := marketCombinationUUID(req.CashOutID, "position Cash Out ID")
	if err != nil {
		return pgtype.UUID{}, pgtype.UUID{}, "", time.Time{}, invalidPositionCashOut(err)
	}
	claimID, _, err := marketCombinationUUID(req.ClaimID, "claim ID")
	if err != nil {
		return pgtype.UUID{}, pgtype.UUID{}, "", time.Time{}, invalidPositionCashOut(err)
	}
	workerID, err := normalizePositionCashOutCode(req.WorkerID, true)
	if err != nil {
		return pgtype.UUID{}, pgtype.UUID{}, "", time.Time{}, err
	}
	now := canonicalNow(req.Now)
	if !req.LeaseExpiresAt.UTC().After(now) {
		return pgtype.UUID{}, pgtype.UUID{}, "", time.Time{}, invalidPositionCashOut(fmt.Errorf("claim expiry must be in the future"))
	}
	return id, claimID, workerID, now, nil
}

func normalizePositionCashOutWalletIDs(walletIDs []int64) ([]int64, error) {
	if len(walletIDs) == 0 {
		return []int64{}, nil
	}
	normalized := append([]int64(nil), walletIDs...)
	sort.Slice(normalized, func(left, right int) bool { return normalized[left] < normalized[right] })
	result := normalized[:0]
	for _, walletID := range normalized {
		if walletID <= 0 {
			return nil, invalidPositionCashOut(fmt.Errorf("Wallet IDs must be positive"))
		}
		if len(result) == 0 || result[len(result)-1] != walletID {
			result = append(result, walletID)
		}
	}
	return result, nil
}

func validPositionCashOutResolvedAttemptState(state PositionCashOutAttemptState) bool {
	switch state {
	case PositionCashOutAttemptStateAcknowledged,
		PositionCashOutAttemptStateRejected,
		PositionCashOutAttemptStateOutcomeUnknown:
		return true
	default:
		return false
	}
}

func validPositionCashOutObservationTransition(expected, next PositionCashOutState) bool {
	switch expected {
	case PositionCashOutStatePreflighting:
		return next == PositionCashOutStateCompleted || next == PositionCashOutStateFailed
	case PositionCashOutStateClosing:
		return next == PositionCashOutStateAwaitingCompletion ||
			next == PositionCashOutStateCompleted || next == PositionCashOutStateFailed ||
			next == PositionCashOutStateReconciliationRequired
	case PositionCashOutStateAwaitingCompletion:
		return next == PositionCashOutStateAwaitingCompletion ||
			next == PositionCashOutStateCompleted || next == PositionCashOutStateFailed ||
			next == PositionCashOutStateReconciliationRequired
	case PositionCashOutStateReconciliationRequired:
		return next == PositionCashOutStateCompleted || next == PositionCashOutStateFailed ||
			next == PositionCashOutStateReconciliationRequired
	default:
		return false
	}
}

func positionCashOutDigest(value any) ([]byte, error) {
	digest, err := executionRequestDigest(value)
	if err != nil {
		return nil, fmt.Errorf("digest position Cash Out request: %w", err)
	}
	return digest, nil
}

func positionCashOutCreationDigest(req GetPositionCashOutCreationRequest) ([]byte, error) {
	return positionCashOutDigest(struct {
		OwnerAccountID string
		CommandID      string
		WalletID       int64
		WalletAddress  string
		PositionPubkey string
	}{
		req.OwnerAccountID,
		req.CommandID,
		req.WalletID,
		req.WalletAddress,
		req.PositionPubkey,
	})
}

func invalidPositionCashOut(err error) error {
	return fmt.Errorf("%w: %v", ErrInvalidPositionCashOut, err)
}
