package store

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"math/big"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	wormtradingsqlc "github.com/useryege/athena/internal/wormtrading/store/sqlc"
)

const (
	maxPositionCashOutBatchWallets       = 100
	maxPositionCashOutBatchItems         = 1000
	maxPositionCashOutBatchPageSize      = 100
	maxPositionCashOutBatchRecoveryLimit = 100
	maxPositionCashOutBatchTextLength    = 500
	positionCashOutBatchUSDCMint         = "EPjFWdd5AufqSSqeM2qN1xzybapC8G4wEGGkZwyTDt1v"
	positionCashOutBatchBalanceWait      = 2 * time.Minute
)

func (s *SQLStore) CreatePositionCashOutBatch(
	ctx context.Context,
	req CreatePositionCashOutBatchRequest,
) (*PositionCashOutBatch, error) {
	ownerUUID, ownerAccountID, err := marketCombinationUUID(req.OwnerAccountID, "owner account ID")
	if err != nil {
		return nil, invalidPositionCashOutBatch(err)
	}
	commandUUID, commandID, err := marketCombinationUUID(req.CommandID, "command ID")
	if err != nil {
		return nil, invalidPositionCashOutBatch(err)
	}
	now := canonicalNow(req.Now)
	wallets, walletIDs, err := normalizePositionCashOutBatchWalletInputs(req.Wallets)
	if err != nil {
		return nil, err
	}
	if !req.AuthorizationExpiresAt.UTC().After(now) {
		return nil, invalidPositionCashOutBatch(fmt.Errorf("authorization expiry must be in the future"))
	}
	req.AuthorizationExpiresAt = req.AuthorizationExpiresAt.UTC()
	requestDigest, err := positionCashOutDigest(struct {
		OwnerAccountID string
		CommandID      string
		Wallets        []PositionCashOutBatchWalletInput
	}{ownerAccountID, commandID, wallets})
	if err != nil {
		return nil, err
	}

	tx, queries, err := s.beginWalletsTransaction(ctx, walletIDs)
	if err != nil {
		return nil, err
	}
	defer rollbackWalletTransaction(tx)

	if existing, getErr := queries.GetPositionCashOutBatchCommand(ctx, commandUUID); getErr == nil {
		if existing.OwnerAccountID != ownerUUID || existing.Kind != string(PositionCashOutBatchCommandKindCreate) ||
			!bytes.Equal(existing.RequestSha256, requestDigest) {
			return nil, ErrPositionCashOutBatchCommandConflict
		}
		batch, loadErr := loadPositionCashOutBatchByID(ctx, queries, existing.BatchID)
		if loadErr != nil {
			return nil, loadErr
		}
		if err := tx.Commit(ctx); err != nil {
			return nil, fmt.Errorf("commit replayed position Cash Out batch creation: %w", err)
		}
		return &batch, nil
	} else if !errors.Is(getErr, pgx.ErrNoRows) {
		return nil, fmt.Errorf("get position Cash Out batch creation command: %w", getErr)
	}

	if count, err := queries.CountActiveExecutionWalletLocksForCashOutBatch(ctx, walletIDs); err != nil {
		return nil, fmt.Errorf("count execution locks for position Cash Out batch: %w", err)
	} else if count != 0 {
		return nil, ErrPositionCashOutBatchExecutionActive
	}
	if count, err := queries.CountActivePositionCashOutsForWallets(ctx, walletIDs); err != nil {
		return nil, fmt.Errorf("count active position Cash Outs for batch: %w", err)
	} else if count != 0 {
		return nil, ErrPositionCashOutBatchCashOutActive
	}
	if count, err := queries.CountActivePositionCashOutBatchWalletLocks(ctx, walletIDs); err != nil {
		return nil, fmt.Errorf("count active position Cash Out batch Wallet locks: %w", err)
	} else if count != 0 {
		return nil, ErrPositionCashOutBatchWalletActive
	}

	credentialVersions := make(map[int64]int64, len(wallets))
	for _, wallet := range wallets {
		connection, err := queries.GetWalletConnectionForUpdate(ctx, wallet.WalletID)
		if err != nil || connection.Address != wallet.Address || connection.State != string(ConnectionStateConnected) {
			if err != nil && !errors.Is(err, pgx.ErrNoRows) {
				return nil, fmt.Errorf("get position Cash Out batch Wallet connection: %w", err)
			}
			return nil, ErrPositionCashOutConnectionChanged
		}
		credential, err := queries.GetActiveCredentialForUpdate(ctx, wallet.WalletID)
		if err != nil {
			if !errors.Is(err, pgx.ErrNoRows) {
				return nil, fmt.Errorf("get position Cash Out batch credential: %w", err)
			}
			return nil, ErrPositionCashOutConnectionChanged
		}
		credentialVersions[wallet.WalletID] = credential.Version
	}

	batchID := pgtype.UUID{Bytes: [16]byte(uuid.New()), Valid: true}
	row, err := queries.CreatePositionCashOutBatch(ctx, wormtradingsqlc.CreatePositionCashOutBatchParams{
		ID: batchID, OwnerAccountID: ownerUUID,
		SelectionDigestSha256: requestDigest, WalletCount: int64(len(wallets)),
		AuthorizationExpiresAt: timestampParam(req.AuthorizationExpiresAt), Now: timestampParam(now),
	})
	if err != nil {
		if executionConstraint(err, "worm_position_cash_out_batches_one_active_owner_idx") {
			return nil, ErrPositionCashOutBatchOwnerActive
		}
		return nil, fmt.Errorf("create position Cash Out batch: %w", err)
	}
	for index, wallet := range wallets {
		if _, err := queries.CreatePositionCashOutBatchWallet(ctx, wormtradingsqlc.CreatePositionCashOutBatchWalletParams{
			BatchID: batchID, Ordinal: int32(index + 1), WalletID: wallet.WalletID,
			Address: wallet.Address, CredentialVersion: credentialVersions[wallet.WalletID],
			Remark: wallet.Remark, AvatarKind: wallet.AvatarKind,
			AvatarPresetID: wallet.AvatarPresetID, AvatarUrl: wallet.AvatarURL,
			Now: timestampParam(now),
		}); err != nil {
			return nil, fmt.Errorf("create position Cash Out batch Wallet %d: %w", index+1, err)
		}
		if err := queries.CreatePositionCashOutBatchWalletLock(ctx, wormtradingsqlc.CreatePositionCashOutBatchWalletLockParams{
			WalletID: wallet.WalletID, BatchID: batchID, Address: wallet.Address,
			Now: timestampParam(now),
		}); err != nil {
			if executionConstraint(err, "worm_position_cash_out_batch_wallet_locks_pkey") {
				return nil, ErrPositionCashOutBatchWalletActive
			}
			return nil, fmt.Errorf("lock position Cash Out batch Wallet: %w", err)
		}
	}
	if _, err := queries.CreatePositionCashOutBatchCommand(ctx, wormtradingsqlc.CreatePositionCashOutBatchCommandParams{
		ID: commandUUID, BatchID: batchID, OwnerAccountID: ownerUUID,
		Kind: string(PositionCashOutBatchCommandKindCreate), RequestSha256: requestDigest,
		BatchRevisionAfter: row.Revision, ResultCode: "", Now: timestampParam(now),
	}); err != nil {
		if executionConstraint(err, "worm_position_cash_out_batch_commands_pkey") {
			return nil, ErrPositionCashOutBatchCommandConflict
		}
		return nil, fmt.Errorf("create position Cash Out batch command: %w", err)
	}
	batch, err := loadPositionCashOutBatch(ctx, queries, row)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit position Cash Out batch creation: %w", err)
	}
	return &batch, nil
}

func (s *SQLStore) GetPositionCashOutBatch(
	ctx context.Context,
	ownerAccountID string,
	batchID string,
) (*PositionCashOutBatch, error) {
	ownerUUID, _, err := marketCombinationUUID(ownerAccountID, "owner account ID")
	if err != nil {
		return nil, invalidPositionCashOutBatch(err)
	}
	id, _, err := marketCombinationUUID(batchID, "position Cash Out batch ID")
	if err != nil {
		return nil, invalidPositionCashOutBatch(err)
	}
	if err := s.requireDatabase(); err != nil {
		return nil, err
	}
	row, err := s.queries.GetPositionCashOutBatch(ctx, wormtradingsqlc.GetPositionCashOutBatchParams{
		ID: id, OwnerAccountID: ownerUUID,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrPositionCashOutBatchNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get position Cash Out batch: %w", err)
	}
	batch, err := loadPositionCashOutBatch(ctx, s.queries, row)
	if err != nil {
		return nil, err
	}
	return &batch, nil
}

func (s *SQLStore) GetActivePositionCashOutBatch(
	ctx context.Context,
	ownerAccountID string,
) (*PositionCashOutBatch, error) {
	ownerUUID, _, err := marketCombinationUUID(ownerAccountID, "owner account ID")
	if err != nil {
		return nil, invalidPositionCashOutBatch(err)
	}
	if err := s.requireDatabase(); err != nil {
		return nil, err
	}
	row, err := s.queries.GetActivePositionCashOutBatch(ctx, ownerUUID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrPositionCashOutBatchNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get active position Cash Out batch: %w", err)
	}
	batch, err := loadPositionCashOutBatch(ctx, s.queries, row)
	if err != nil {
		return nil, err
	}
	return &batch, nil
}

func (s *SQLStore) ListPositionCashOutBatchItems(
	ctx context.Context,
	ownerAccountID string,
	batchID string,
	page int32,
	pageSize int32,
) ([]PositionCashOutBatchItem, int64, error) {
	if page <= 0 || pageSize <= 0 || pageSize > maxPositionCashOutBatchPageSize {
		return nil, 0, invalidPositionCashOutBatch(fmt.Errorf("page and page size are invalid"))
	}
	batch, err := s.GetPositionCashOutBatch(ctx, ownerAccountID, batchID)
	if err != nil {
		return nil, 0, err
	}
	id, _, _ := marketCombinationUUID(batch.ID, "position Cash Out batch ID")
	rows, err := s.queries.ListPositionCashOutBatchItems(ctx, wormtradingsqlc.ListPositionCashOutBatchItemsParams{
		BatchID: id, PageLimit: pageSize, PageOffset: (int64(page) - 1) * int64(pageSize),
	})
	if err != nil {
		return nil, 0, fmt.Errorf("list position Cash Out batch items: %w", err)
	}
	total, err := s.queries.CountPositionCashOutBatchItems(ctx, id)
	if err != nil {
		return nil, 0, fmt.Errorf("count position Cash Out batch items: %w", err)
	}
	items := make([]PositionCashOutBatchItem, 0, len(rows))
	for _, row := range rows {
		items = append(items, mapPositionCashOutBatchItem(row))
	}
	return items, total, nil
}

func (s *SQLStore) CompletePositionCashOutBatchBuild(
	ctx context.Context,
	req CompletePositionCashOutBatchBuildRequest,
) (*PositionCashOutBatch, error) {
	batchID, _, err := marketCombinationUUID(req.BatchID, "position Cash Out batch ID")
	if err != nil {
		return nil, invalidPositionCashOutBatch(err)
	}
	claimID, _, err := marketCombinationUUID(req.ClaimID, "claim ID")
	if err != nil {
		return nil, invalidPositionCashOutBatch(err)
	}
	now := canonicalNow(req.Now)
	if !req.AuthorizationExpiresAt.UTC().After(now) {
		return nil, invalidPositionCashOutBatch(fmt.Errorf("authorization expiry must be in the future"))
	}
	if err := s.requireDatabase(); err != nil {
		return nil, err
	}
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, fmt.Errorf("begin complete position Cash Out batch build: %w", err)
	}
	defer rollbackWalletTransaction(tx)
	queries := wormtradingsqlc.New(tx)
	batchRow, err := queries.GetPositionCashOutBatchByIDForUpdate(ctx, batchID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrPositionCashOutBatchNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("lock position Cash Out batch build: %w", err)
	}
	if batchRow.State != string(PositionCashOutBatchStateBuilding) || batchRow.ClaimID != claimID {
		return nil, ErrPositionCashOutBatchClaim
	}
	walletRows, err := queries.ListPositionCashOutBatchWallets(ctx, batchID)
	if err != nil {
		return nil, fmt.Errorf("list position Cash Out batch build Wallets: %w", err)
	}
	items, counts, err := normalizePositionCashOutBatchItems(req.Items, walletRows)
	if err != nil {
		return nil, err
	}
	type intentWallet struct {
		Ordinal           int32
		WalletID          int64
		Address           string
		CredentialVersion int64
		Remark            string
		AvatarKind        string
		AvatarPresetID    string
		AvatarURL         string
	}
	intentWallets := make([]intentWallet, 0, len(walletRows))
	for _, wallet := range walletRows {
		intentWallets = append(intentWallets, intentWallet{
			Ordinal: wallet.Ordinal, WalletID: wallet.WalletID, Address: wallet.Address,
			CredentialVersion: wallet.CredentialVersion, Remark: wallet.Remark,
			AvatarKind: wallet.AvatarKind, AvatarPresetID: wallet.AvatarPresetID,
			AvatarURL: wallet.AvatarUrl,
		})
	}
	intentDigest, err := positionCashOutDigest(struct {
		OwnerAccountID string
		Wallets        []intentWallet
		Items          []PositionCashOutBatchItemInput
	}{uuidValue(batchRow.OwnerAccountID), intentWallets, items})
	if err != nil {
		return nil, err
	}
	for index, item := range items {
		if _, err := queries.CreatePositionCashOutBatchItem(ctx, wormtradingsqlc.CreatePositionCashOutBatchItemParams{
			ID: pgtype.UUID{Bytes: [16]byte(uuid.New()), Valid: true}, BatchID: batchID,
			Ordinal: int64(index + 1), WalletOrdinal: item.WalletOrdinal,
			PositionOrdinal: item.PositionOrdinal, WalletID: item.WalletID,
			WalletAddress: item.WalletAddress, WalletRemark: item.WalletRemark,
			CredentialVersion: item.CredentialVersion, PositionPubkey: item.PositionPubkey,
			PositionRequestPubkey: item.PositionRequestPubkey,
			MarketConditionID:     item.MarketConditionID, MarketTitle: item.MarketTitle,
			IsYes: item.IsYes, Shares: item.Shares,
			PositionCreatedAt: timestampParam(item.PositionCreatedAt), ProviderState: item.ProviderState,
			Now: timestampParam(now),
		}); err != nil {
			return nil, fmt.Errorf("freeze position Cash Out batch item %d: %w", index+1, err)
		}
	}
	for walletOrdinal, count := range counts {
		if affected, err := queries.SetPositionCashOutBatchWalletCounts(ctx, wormtradingsqlc.SetPositionCashOutBatchWalletCountsParams{
			BatchID: batchID, Ordinal: walletOrdinal, PositionCount: count, Now: timestampParam(now),
		}); err != nil || affected != 1 {
			if err != nil {
				return nil, fmt.Errorf("record position Cash Out batch Wallet counts: %w", err)
			}
			return nil, ErrPositionCashOutBatchClaim
		}
	}
	row, err := queries.CompletePositionCashOutBatchBuild(ctx, wormtradingsqlc.CompletePositionCashOutBatchBuildParams{
		ID: batchID, ClaimID: claimID, IntentDigestSha256: intentDigest,
		PositionCount: int64(len(items)), AuthorizationExpiresAt: timestampParam(req.AuthorizationExpiresAt),
		Now: timestampParam(now),
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrPositionCashOutBatchClaim
	}
	if err != nil {
		return nil, fmt.Errorf("complete position Cash Out batch build: %w", err)
	}
	batch, err := loadPositionCashOutBatch(ctx, queries, row)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit position Cash Out batch build: %w", err)
	}
	return &batch, nil
}

func (s *SQLStore) FailPositionCashOutBatchBuild(
	ctx context.Context,
	req FailPositionCashOutBatchBuildRequest,
) (*PositionCashOutBatch, error) {
	batchID, _, err := marketCombinationUUID(req.BatchID, "position Cash Out batch ID")
	if err != nil {
		return nil, invalidPositionCashOutBatch(err)
	}
	claimID, _, err := marketCombinationUUID(req.ClaimID, "claim ID")
	if err != nil {
		return nil, invalidPositionCashOutBatch(err)
	}
	reasonCode, err := normalizePositionCashOutCode(req.ReasonCode, true)
	if err != nil {
		return nil, invalidPositionCashOutBatch(err)
	}
	buildStage, err := normalizePositionCashOutCode(req.BuildStage, true)
	if err != nil {
		return nil, invalidPositionCashOutBatch(err)
	}
	now := canonicalNow(req.Now)
	walletIDs, err := s.positionCashOutBatchWalletIDs(ctx, batchID)
	if err != nil {
		return nil, err
	}
	tx, queries, err := s.beginWalletsTransaction(ctx, walletIDs)
	if err != nil {
		return nil, err
	}
	defer rollbackWalletTransaction(tx)
	row, err := queries.FailPositionCashOutBatchBuild(ctx, wormtradingsqlc.FailPositionCashOutBatchBuildParams{
		ID: batchID, ClaimID: claimID, ReasonCode: reasonCode,
		BuildStage: buildStage, Now: timestampParam(now),
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrPositionCashOutBatchClaim
	}
	if err != nil {
		return nil, fmt.Errorf("fail position Cash Out batch build: %w", err)
	}
	if _, err := queries.ReleasePositionCashOutBatchWalletLocks(ctx, batchID); err != nil {
		return nil, fmt.Errorf("release failed position Cash Out batch Wallet locks: %w", err)
	}
	batch, err := loadPositionCashOutBatch(ctx, queries, row)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit failed position Cash Out batch build: %w", err)
	}
	return &batch, nil
}

func (s *SQLStore) AuthorizePositionCashOutBatch(
	ctx context.Context,
	req AuthorizePositionCashOutBatchRequest,
) (*PositionCashOutBatch, error) {
	ownerUUID, _, batchID, err := positionCashOutBatchOwnerAndID(req.OwnerAccountID, req.BatchID)
	if err != nil {
		return nil, err
	}
	commandUUID, commandID, err := marketCombinationUUID(req.CommandID, "command ID")
	if err != nil {
		return nil, invalidPositionCashOutBatch(err)
	}
	proofKind, err := normalizePositionCashOutCode(req.ProofKind, true)
	if err != nil {
		return nil, invalidPositionCashOutBatch(err)
	}
	if req.ExpectedRevision <= 0 || len(req.SessionJTIDigest) != 32 || req.AccessRevision <= 0 {
		return nil, invalidPositionCashOutBatch(fmt.Errorf("batch authorization binding is invalid"))
	}
	now := canonicalNow(req.Now)
	requestDigest, err := positionCashOutDigest(struct {
		OwnerAccountID   string
		BatchID          string
		CommandID        string
		ExpectedRevision int64
		ProofKind        string
		SessionJTIDigest []byte
		AccessRevision   int64
	}{req.OwnerAccountID, req.BatchID, commandID, req.ExpectedRevision, proofKind,
		req.SessionJTIDigest, req.AccessRevision})
	if err != nil {
		return nil, err
	}
	walletIDs, err := s.positionCashOutBatchWalletIDs(ctx, batchID)
	if err != nil {
		return nil, err
	}
	tx, queries, err := s.beginWalletsTransaction(ctx, walletIDs)
	if err != nil {
		return nil, err
	}
	defer rollbackWalletTransaction(tx)
	if existing, getErr := queries.GetPositionCashOutBatchCommand(ctx, commandUUID); getErr == nil {
		if existing.OwnerAccountID != ownerUUID || existing.BatchID != batchID ||
			existing.Kind != string(PositionCashOutBatchCommandKindAuthorize) ||
			!bytes.Equal(existing.RequestSha256, requestDigest) {
			return nil, ErrPositionCashOutBatchCommandConflict
		}
		batch, loadErr := loadPositionCashOutBatchByID(ctx, queries, batchID)
		if loadErr != nil {
			return nil, loadErr
		}
		if err := tx.Commit(ctx); err != nil {
			return nil, fmt.Errorf("commit replayed position Cash Out batch authorization: %w", err)
		}
		return &batch, nil
	} else if !errors.Is(getErr, pgx.ErrNoRows) {
		return nil, fmt.Errorf("get position Cash Out batch authorization command: %w", getErr)
	}
	row, err := queries.GetPositionCashOutBatchForUpdate(ctx, wormtradingsqlc.GetPositionCashOutBatchForUpdateParams{
		ID: batchID, OwnerAccountID: ownerUUID,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrPositionCashOutBatchNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("lock position Cash Out batch authorization: %w", err)
	}
	if row.Revision != req.ExpectedRevision ||
		(row.State != string(PositionCashOutBatchStateAwaitingAuthorization) &&
			row.State != string(PositionCashOutBatchStatePaused)) {
		return nil, ErrPositionCashOutBatchRevision
	}
	if row.State == string(PositionCashOutBatchStateAwaitingAuthorization) &&
		!timestampValue(row.AuthorizationExpiresAt).After(now) {
		return nil, fmt.Errorf("%w: position Cash Out batch authorization", ErrExpired)
	}
	if len(row.IntentDigestSha256) != 32 {
		return nil, ErrPositionCashOutBatchRevision
	}
	if _, err := queries.SupersedePositionCashOutBatchAuthorization(ctx, wormtradingsqlc.SupersedePositionCashOutBatchAuthorizationParams{
		BatchID: batchID, Now: timestampParam(now),
	}); err != nil {
		return nil, fmt.Errorf("supersede position Cash Out batch authorization: %w", err)
	}
	if _, err := queries.CreatePositionCashOutBatchAuthorization(ctx, wormtradingsqlc.CreatePositionCashOutBatchAuthorizationParams{
		ID: pgtype.UUID{Bytes: [16]byte(uuid.New()), Valid: true}, BatchID: batchID,
		OwnerAccountID: ownerUUID, ProofKind: proofKind,
		SessionJtiDigest: append([]byte(nil), req.SessionJTIDigest...),
		AccessRevision:   req.AccessRevision, IntentDigestSha256: row.IntentDigestSha256,
		Now: timestampParam(now),
	}); err != nil {
		return nil, fmt.Errorf("create position Cash Out batch authorization: %w", err)
	}
	queued, err := queries.QueueAuthorizedPositionCashOutBatch(ctx, wormtradingsqlc.QueueAuthorizedPositionCashOutBatchParams{
		ID: batchID, OwnerAccountID: ownerUUID, ExpectedRevision: req.ExpectedRevision,
		Now: timestampParam(now),
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrPositionCashOutBatchRevision
	}
	if err != nil {
		return nil, fmt.Errorf("queue authorized position Cash Out batch: %w", err)
	}
	if _, err := queries.CreatePositionCashOutBatchCommand(ctx, wormtradingsqlc.CreatePositionCashOutBatchCommandParams{
		ID: commandUUID, BatchID: batchID, OwnerAccountID: ownerUUID,
		Kind: string(PositionCashOutBatchCommandKindAuthorize), RequestSha256: requestDigest,
		BatchRevisionAfter: queued.Revision, ResultCode: "", Now: timestampParam(now),
	}); err != nil {
		return nil, fmt.Errorf("create position Cash Out batch authorization command: %w", err)
	}
	batch, err := loadPositionCashOutBatch(ctx, queries, queued)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit position Cash Out batch authorization: %w", err)
	}
	return &batch, nil
}

func (s *SQLStore) ControlPositionCashOutBatch(
	ctx context.Context,
	req PositionCashOutBatchCommandRequest,
) (*PositionCashOutBatch, error) {
	ownerUUID, _, batchID, err := positionCashOutBatchOwnerAndID(req.OwnerAccountID, req.BatchID)
	if err != nil {
		return nil, err
	}
	commandUUID, commandID, err := marketCombinationUUID(req.CommandID, "command ID")
	if err != nil {
		return nil, invalidPositionCashOutBatch(err)
	}
	if req.ExpectedRevision <= 0 || !validPositionCashOutBatchControlKind(req.Kind) {
		return nil, invalidPositionCashOutBatch(fmt.Errorf("batch control is invalid"))
	}
	now := canonicalNow(req.Now)
	requestDigest, err := positionCashOutDigest(struct {
		OwnerAccountID   string
		BatchID          string
		CommandID        string
		ExpectedRevision int64
		Kind             PositionCashOutBatchCommandKind
	}{req.OwnerAccountID, req.BatchID, commandID, req.ExpectedRevision, req.Kind})
	if err != nil {
		return nil, err
	}
	walletIDs, err := s.positionCashOutBatchWalletIDs(ctx, batchID)
	if err != nil {
		return nil, err
	}
	tx, queries, err := s.beginWalletsTransaction(ctx, walletIDs)
	if err != nil {
		return nil, err
	}
	defer rollbackWalletTransaction(tx)
	if existing, getErr := queries.GetPositionCashOutBatchCommand(ctx, commandUUID); getErr == nil {
		if existing.OwnerAccountID != ownerUUID || existing.BatchID != batchID ||
			existing.Kind != string(req.Kind) || !bytes.Equal(existing.RequestSha256, requestDigest) {
			return nil, ErrPositionCashOutBatchCommandConflict
		}
		batch, loadErr := loadPositionCashOutBatchByID(ctx, queries, batchID)
		if loadErr != nil {
			return nil, loadErr
		}
		if err := tx.Commit(ctx); err != nil {
			return nil, fmt.Errorf("commit replayed position Cash Out batch control: %w", err)
		}
		return &batch, nil
	} else if !errors.Is(getErr, pgx.ErrNoRows) {
		return nil, fmt.Errorf("get position Cash Out batch control command: %w", getErr)
	}
	row, err := queries.GetPositionCashOutBatchForUpdate(ctx, wormtradingsqlc.GetPositionCashOutBatchForUpdateParams{
		ID: batchID, OwnerAccountID: ownerUUID,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrPositionCashOutBatchNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("lock position Cash Out batch control: %w", err)
	}
	if row.Revision != req.ExpectedRevision || row.ClaimID.Valid {
		return nil, ErrPositionCashOutBatchRevision
	}
	nextState, reasonCode, nextPollAt, checkRequestedAt, terminal, err :=
		positionCashOutBatchControlTransition(row, req.Kind, now)
	if err != nil {
		return nil, err
	}
	var notExecuted int64
	if req.Kind == PositionCashOutBatchCommandKindTerminate && !row.CurrentItemOrdinal.Valid {
		marked, err := queries.MarkRemainingPositionCashOutBatchItemsNotExecuted(ctx, wormtradingsqlc.MarkRemainingPositionCashOutBatchItemsNotExecutedParams{
			BatchID: batchID, ReasonCode: "BATCH_TERMINATED", Now: timestampParam(now),
		})
		if err != nil {
			return nil, fmt.Errorf("mark terminated position Cash Out batch items: %w", err)
		}
		notExecuted = int64(len(marked))
	}
	updated, err := queries.ApplyPositionCashOutBatchControl(ctx, wormtradingsqlc.ApplyPositionCashOutBatchControlParams{
		ID: batchID, OwnerAccountID: ownerUUID, ExpectedRevision: req.ExpectedRevision,
		NextState: string(nextState), ReasonCode: reasonCode,
		NextPollAt: timestampParam(nextPollAt), CheckRequestedAt: timestampParam(checkRequestedAt),
		Now: timestampParam(now),
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrPositionCashOutBatchRevision
	}
	if err != nil {
		return nil, fmt.Errorf("apply position Cash Out batch control: %w", err)
	}
	if notExecuted != 0 {
		terminalRow, err := queries.CompletePositionCashOutBatch(ctx, wormtradingsqlc.CompletePositionCashOutBatchParams{
			ID: batchID, ExpectedState: string(PositionCashOutBatchStateTerminated),
			TerminalState:    string(PositionCashOutBatchStateTerminated),
			NotExecutedCount: notExecuted, Now: timestampParam(now),
		})
		if err == nil {
			updated = terminalRow
		} else if !errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("record terminated position Cash Out batch count: %w", err)
		}
	}
	if terminal {
		if _, err := queries.ReleasePositionCashOutBatchWalletLocks(ctx, batchID); err != nil {
			return nil, fmt.Errorf("release terminal position Cash Out batch Wallet locks: %w", err)
		}
		if _, err := queries.EndPositionCashOutBatchAuthorization(ctx, wormtradingsqlc.EndPositionCashOutBatchAuthorizationParams{
			BatchID: batchID, State: string(PositionCashOutBatchAuthorizationStateConsumed),
			ReasonCode: string(nextState), Now: timestampParam(now),
		}); err != nil {
			return nil, fmt.Errorf("end terminal position Cash Out batch authorization: %w", err)
		}
	}
	if _, err := queries.CreatePositionCashOutBatchCommand(ctx, wormtradingsqlc.CreatePositionCashOutBatchCommandParams{
		ID: commandUUID, BatchID: batchID, OwnerAccountID: ownerUUID,
		Kind: string(req.Kind), RequestSha256: requestDigest,
		BatchRevisionAfter: updated.Revision, ResultCode: "", Now: timestampParam(now),
	}); err != nil {
		return nil, fmt.Errorf("create position Cash Out batch control command: %w", err)
	}
	batch, err := loadPositionCashOutBatch(ctx, queries, updated)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit position Cash Out batch control: %w", err)
	}
	return &batch, nil
}

func (s *SQLStore) ClaimPositionCashOutBatch(
	ctx context.Context,
	req PositionCashOutBatchClaimRequest,
) (*PositionCashOutBatch, error) {
	batchID, claimID, workerID, now, err := normalizePositionCashOutBatchClaim(req)
	if err != nil {
		return nil, err
	}
	if err := s.requireDatabase(); err != nil {
		return nil, err
	}
	row, err := s.queries.ClaimPositionCashOutBatch(ctx, wormtradingsqlc.ClaimPositionCashOutBatchParams{
		ID: batchID, ClaimID: claimID, ClaimOwner: workerID,
		ClaimExpiresAt: timestampParam(req.LeaseExpiresAt), Now: timestampParam(now),
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrPositionCashOutBatchClaim
	}
	if err != nil {
		return nil, fmt.Errorf("claim position Cash Out batch: %w", err)
	}
	batch, err := loadPositionCashOutBatch(ctx, s.queries, row)
	if err != nil {
		return nil, err
	}
	return &batch, nil
}

func (s *SQLStore) RenewPositionCashOutBatchClaim(
	ctx context.Context,
	req PositionCashOutBatchClaimRequest,
) (*PositionCashOutBatch, error) {
	batchID, claimID, workerID, now, err := normalizePositionCashOutBatchClaim(req)
	if err != nil {
		return nil, err
	}
	if err := s.requireDatabase(); err != nil {
		return nil, err
	}
	row, err := s.queries.RenewPositionCashOutBatchClaim(ctx, wormtradingsqlc.RenewPositionCashOutBatchClaimParams{
		ID: batchID, ClaimID: claimID, ClaimOwner: workerID,
		ClaimExpiresAt: timestampParam(req.LeaseExpiresAt), Now: timestampParam(now),
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrPositionCashOutBatchClaim
	}
	if err != nil {
		return nil, fmt.Errorf("renew position Cash Out batch claim: %w", err)
	}
	batch, err := loadPositionCashOutBatch(ctx, s.queries, row)
	if err != nil {
		return nil, err
	}
	return &batch, nil
}

func (s *SQLStore) DeferPositionCashOutBatchPreflight(
	ctx context.Context,
	req DeferPositionCashOutBatchPreflightRequest,
) (*PositionCashOutBatch, error) {
	batchID, _, err := marketCombinationUUID(req.BatchID, "position Cash Out batch ID")
	if err != nil {
		return nil, invalidPositionCashOutBatch(err)
	}
	itemID, _, err := marketCombinationUUID(req.ItemID, "position Cash Out batch item ID")
	if err != nil {
		return nil, invalidPositionCashOutBatch(err)
	}
	claimID, _, err := marketCombinationUUID(req.ClaimID, "claim ID")
	if err != nil {
		return nil, invalidPositionCashOutBatch(err)
	}
	now := canonicalNow(req.Now)
	if err := s.requireDatabase(); err != nil {
		return nil, err
	}
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, fmt.Errorf("begin defer position Cash Out batch preflight: %w", err)
	}
	defer rollbackWalletTransaction(tx)
	queries := wormtradingsqlc.New(tx)
	row, err := queries.DeferPositionCashOutBatchPreflight(ctx, wormtradingsqlc.DeferPositionCashOutBatchPreflightParams{
		BatchID: batchID, ItemID: itemID, ClaimID: claimID,
		NextPollAt: timestampParam(now.Add(time.Second)), Now: timestampParam(now),
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrPositionCashOutBatchClaim
	}
	if err != nil {
		return nil, fmt.Errorf("defer position Cash Out batch preflight: %w", err)
	}
	batch, err := loadPositionCashOutBatch(ctx, queries, row)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit deferred position Cash Out batch preflight: %w", err)
	}
	return &batch, nil
}

func (s *SQLStore) DeferPositionCashOutBatchCheck(
	ctx context.Context,
	req DeferPositionCashOutBatchCheckRequest,
) (*PositionCashOutBatch, error) {
	batchID, _, err := marketCombinationUUID(req.BatchID, "position Cash Out batch ID")
	if err != nil {
		return nil, invalidPositionCashOutBatch(err)
	}
	claimID, _, err := marketCombinationUUID(req.ClaimID, "claim ID")
	if err != nil {
		return nil, invalidPositionCashOutBatch(err)
	}
	now := canonicalNow(req.Now)
	if err := s.requireDatabase(); err != nil {
		return nil, err
	}
	row, err := s.queries.DeferPositionCashOutBatchCheck(ctx, wormtradingsqlc.DeferPositionCashOutBatchCheckParams{
		BatchID: batchID, ClaimID: claimID,
		NextPollAt: timestampParam(now.Add(time.Second)), Now: timestampParam(now),
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrPositionCashOutBatchClaim
	}
	if err != nil {
		return nil, fmt.Errorf("defer position Cash Out batch check: %w", err)
	}
	batch, err := loadPositionCashOutBatch(ctx, s.queries, row)
	if err != nil {
		return nil, err
	}
	return &batch, nil
}

func (s *SQLStore) ActivatePositionCashOutBatchItem(
	ctx context.Context,
	req ActivatePositionCashOutBatchItemRequest,
) (*PositionCashOutBatch, error) {
	batchID, _, err := marketCombinationUUID(req.BatchID, "position Cash Out batch ID")
	if err != nil {
		return nil, invalidPositionCashOutBatch(err)
	}
	itemID, _, err := marketCombinationUUID(req.ItemID, "position Cash Out batch item ID")
	if err != nil {
		return nil, invalidPositionCashOutBatch(err)
	}
	claimID, _, err := marketCombinationUUID(req.ClaimID, "claim ID")
	if err != nil {
		return nil, invalidPositionCashOutBatch(err)
	}
	cashOutID, _, err := marketCombinationUUID(req.CashOutID, "position Cash Out ID")
	if err != nil {
		return nil, invalidPositionCashOutBatch(err)
	}
	providerState, err := normalizePositionCashOutCode(req.ProviderState, false)
	if err != nil {
		return nil, invalidPositionCashOutBatch(err)
	}
	now := canonicalNow(req.Now)
	if !req.NextPollAt.UTC().After(now) {
		return nil, invalidPositionCashOutBatch(fmt.Errorf("next poll time must be in the future"))
	}
	if err := s.requireDatabase(); err != nil {
		return nil, err
	}
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, fmt.Errorf("begin activate position Cash Out batch item: %w", err)
	}
	defer rollbackWalletTransaction(tx)
	queries := wormtradingsqlc.New(tx)
	batchRow, err := queries.GetPositionCashOutBatchByIDForUpdate(ctx, batchID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrPositionCashOutBatchNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("lock position Cash Out batch item activation: %w", err)
	}
	if batchRow.ClaimID != claimID || batchRow.CurrentItemOrdinal.Valid ||
		batchRow.State != string(PositionCashOutBatchStateRunning) {
		return nil, ErrPositionCashOutBatchClaim
	}
	itemRow, err := queries.GetPositionCashOutBatchItemForUpdate(ctx, wormtradingsqlc.GetPositionCashOutBatchItemForUpdateParams{
		ID: itemID, BatchID: batchID,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrPositionCashOutBatchClaim
	}
	if err != nil {
		return nil, fmt.Errorf("lock next position Cash Out batch item: %w", err)
	}
	if itemRow.State != string(PositionCashOutBatchItemStatePending) || itemRow.Ordinal != nullableInt64(batchRow.NextItemOrdinal) {
		return nil, ErrPositionCashOutBatchClaim
	}
	connection, err := queries.GetWalletConnectionForUpdate(ctx, itemRow.WalletID)
	if err != nil || connection.Address != itemRow.WalletAddress || connection.State != string(ConnectionStateConnected) {
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("get batch item Wallet connection: %w", err)
		}
		return nil, ErrPositionCashOutConnectionChanged
	}
	credential, err := queries.GetActiveCredentialForUpdate(ctx, itemRow.WalletID)
	if err != nil || credential.Version != itemRow.CredentialVersion {
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("get batch item active credential: %w", err)
		}
		return nil, ErrPositionCashOutConnectionChanged
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
	}{uuidValue(batchRow.OwnerAccountID), itemRow.WalletID, itemRow.WalletAddress,
		itemRow.PositionPubkey, itemRow.MarketConditionID, itemRow.IsYes,
		timestampValue(itemRow.PositionCreatedAt), itemRow.PositionRequestPubkey, itemRow.Shares})
	if err != nil {
		return nil, err
	}
	if _, err := queries.CreateBatchChildPositionCashOut(ctx, wormtradingsqlc.CreateBatchChildPositionCashOutParams{
		CashOutID: cashOutID, BatchID: batchID, ItemID: itemID, ClaimID: claimID,
		IntentDigestSha256: intentDigest, ProviderState: providerState, Now: timestampParam(now),
	}); err != nil {
		if executionConstraint(err, "worm_position_cash_outs_one_active_per_wallet_idx") ||
			executionConstraint(err, "worm_position_cash_outs_batch_item_unique") {
			return nil, ErrPositionCashOutBatchClaim
		}
		return nil, fmt.Errorf("create batch child position Cash Out: %w", err)
	}
	if _, err := queries.ActivatePositionCashOutBatchItem(ctx, wormtradingsqlc.ActivatePositionCashOutBatchItemParams{
		ItemID: itemID, BatchID: batchID, CashOutID: cashOutID,
		ProviderState: providerState, Now: timestampParam(now),
	}); err != nil {
		return nil, fmt.Errorf("activate position Cash Out batch item: %w", err)
	}
	updated, err := queries.SetPositionCashOutBatchCurrentItem(ctx, wormtradingsqlc.SetPositionCashOutBatchCurrentItemParams{
		ID: batchID, ClaimID: claimID, ItemOrdinal: itemRow.Ordinal,
		NextPollAt: timestampParam(req.NextPollAt), Now: timestampParam(now),
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrPositionCashOutBatchClaim
	}
	if err != nil {
		return nil, fmt.Errorf("set current position Cash Out batch item: %w", err)
	}
	batch, err := loadPositionCashOutBatch(ctx, queries, updated)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit position Cash Out batch item activation: %w", err)
	}
	return &batch, nil
}

func (s *SQLStore) DispatchPositionCashOutBatchAttempt(
	ctx context.Context,
	req DispatchPositionCashOutBatchAttemptRequest,
) (*PositionCashOutAttempt, error) {
	batchID, _, err := marketCombinationUUID(req.BatchID, "position Cash Out batch ID")
	if err != nil {
		return nil, invalidPositionCashOutBatch(err)
	}
	itemID, _, err := marketCombinationUUID(req.ItemID, "position Cash Out batch item ID")
	if err != nil {
		return nil, invalidPositionCashOutBatch(err)
	}
	attemptID, _, err := marketCombinationUUID(req.AttemptID, "position Cash Out attempt ID")
	if err != nil {
		return nil, invalidPositionCashOutBatch(err)
	}
	cashOutID, _, err := marketCombinationUUID(req.CashOutID, "position Cash Out ID")
	if err != nil {
		return nil, invalidPositionCashOutBatch(err)
	}
	claimID, _, err := marketCombinationUUID(req.ClaimID, "position Cash Out claim ID")
	if err != nil {
		return nil, invalidPositionCashOutBatch(err)
	}
	baseline, err := normalizePositionCashOutBatchBalance(req.Baseline)
	if err != nil {
		return nil, err
	}
	now := canonicalNow(req.Now)
	nextPollAt := req.NextPollAt.UTC()
	if nextPollAt.IsZero() || nextPollAt.Before(now) {
		nextPollAt = now
	}
	if err := s.requireDatabase(); err != nil {
		return nil, err
	}
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, fmt.Errorf("begin dispatch batch position Cash Out: %w", err)
	}
	defer rollbackWalletTransaction(tx)
	queries := wormtradingsqlc.New(tx)
	batchRow, err := queries.GetPositionCashOutBatchByIDForUpdate(ctx, batchID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrPositionCashOutBatchClaim
	}
	if err != nil {
		return nil, fmt.Errorf("lock batch before position Cash Out dispatch: %w", err)
	}
	if (batchRow.State != string(PositionCashOutBatchStateRunning) &&
		batchRow.State != string(PositionCashOutBatchStatePauseRequested)) ||
		!batchRow.CurrentItemOrdinal.Valid {
		return nil, ErrPositionCashOutBatchClaim
	}
	itemRow, err := queries.GetPositionCashOutBatchItemForUpdate(ctx, wormtradingsqlc.GetPositionCashOutBatchItemForUpdateParams{
		ID: itemID, BatchID: batchID,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrPositionCashOutBatchClaim
	}
	if err != nil {
		return nil, fmt.Errorf("lock batch item dispatch baseline: %w", err)
	}
	if itemRow.ChildCashOutID != cashOutID || batchRow.CurrentItemOrdinal.Int64 != itemRow.Ordinal {
		return nil, ErrPositionCashOutBatchClaim
	}
	attemptRow, err := queries.DispatchPositionCashOutAttempt(ctx, wormtradingsqlc.DispatchPositionCashOutAttemptParams{
		ID: attemptID, CashOutID: cashOutID, ClaimID: claimID, Now: timestampParam(now),
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrPositionCashOutClaim
	}
	if err != nil {
		return nil, fmt.Errorf("dispatch batch position Cash Out attempt: %w", err)
	}
	if _, err := queries.SetPositionCashOutBatchItemBalanceBaseline(ctx, wormtradingsqlc.SetPositionCashOutBatchItemBalanceBaselineParams{
		ItemID: itemID, BatchID: batchID, CashOutID: cashOutID,
		Mint: baseline.Mint, Decimals: baseline.Decimals,
		AtomicAmount: baseline.AtomicAmount, ObservedSlot: int64(baseline.ObservedSlot),
		Now: timestampParam(now),
	}); err != nil {
		return nil, fmt.Errorf("record batch position Cash Out balance baseline: %w", err)
	}
	if affected, err := queries.TouchPositionCashOutAfterDispatch(ctx, wormtradingsqlc.TouchPositionCashOutAfterDispatchParams{
		ID: cashOutID, ClaimID: claimID, Now: timestampParam(now),
	}); err != nil || affected != 1 {
		if err != nil {
			return nil, fmt.Errorf("checkpoint batch child Cash Out dispatch: %w", err)
		}
		return nil, ErrPositionCashOutClaim
	}
	if affected, err := queries.TouchPositionCashOutBatchAfterDispatch(ctx, wormtradingsqlc.TouchPositionCashOutBatchAfterDispatchParams{
		ID: batchID, ItemOrdinal: itemRow.Ordinal,
		NextPollAt: timestampParam(nextPollAt), Now: timestampParam(now),
	}); err != nil || affected != 1 {
		if err != nil {
			return nil, fmt.Errorf("checkpoint position Cash Out batch dispatch: %w", err)
		}
		return nil, ErrPositionCashOutBatchClaim
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit batch position Cash Out dispatch: %w", err)
	}
	attempt := mapPositionCashOutAttempt(attemptRow)
	return &attempt, nil
}

func (s *SQLStore) RecordPositionCashOutBatchItemState(
	ctx context.Context,
	req RecordPositionCashOutBatchItemStateRequest,
) (*PositionCashOutBatch, error) {
	batchID, itemID, claimID, now, err := normalizePositionCashOutBatchItemClaim(
		req.BatchID, req.ItemID, req.ClaimID, req.Now,
	)
	if err != nil {
		return nil, err
	}
	if !validPositionCashOutBatchItemTransition(req.ExpectedState, req.NextState) {
		return nil, invalidPositionCashOutBatch(fmt.Errorf("batch item transition is invalid"))
	}
	reasonCode, err := normalizePositionCashOutCode(req.ReasonCode,
		req.NextState == PositionCashOutBatchItemStateFailed ||
			req.NextState == PositionCashOutBatchItemStateReconciliationRequired)
	if err != nil {
		return nil, invalidPositionCashOutBatch(err)
	}
	providerState, err := normalizePositionCashOutCode(req.ProviderState, false)
	if err != nil {
		return nil, invalidPositionCashOutBatch(err)
	}
	if req.NextState == PositionCashOutBatchItemStateAwaitingBalance {
		deadline := req.BalanceDeadlineAt.UTC()
		if deadline.IsZero() || deadline.After(now.Add(positionCashOutBatchBalanceWait)) {
			return nil, invalidPositionCashOutBatch(fmt.Errorf("balance deadline is invalid"))
		}
	}
	if !req.NextPollAt.UTC().After(now) {
		return nil, invalidPositionCashOutBatch(fmt.Errorf("next poll time must be in the future"))
	}
	if err := s.requireDatabase(); err != nil {
		return nil, err
	}
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, fmt.Errorf("begin record position Cash Out batch item state: %w", err)
	}
	defer rollbackWalletTransaction(tx)
	queries := wormtradingsqlc.New(tx)
	batchRow, err := queries.GetPositionCashOutBatchByIDForUpdate(ctx, batchID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrPositionCashOutBatchNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("lock position Cash Out batch item state: %w", err)
	}
	itemRow, err := queries.GetPositionCashOutBatchItemForUpdate(ctx, wormtradingsqlc.GetPositionCashOutBatchItemForUpdateParams{
		ID: itemID, BatchID: batchID,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrPositionCashOutBatchClaim
	}
	if err != nil {
		return nil, fmt.Errorf("lock position Cash Out batch item: %w", err)
	}
	if batchRow.ClaimID != claimID || nullableInt64(batchRow.CurrentItemOrdinal) != itemRow.Ordinal {
		return nil, ErrPositionCashOutBatchClaim
	}
	if _, err := queries.RecordPositionCashOutBatchItemState(ctx, wormtradingsqlc.RecordPositionCashOutBatchItemStateParams{
		ItemID: itemID, BatchID: batchID, ExpectedState: string(req.ExpectedState),
		NextState: string(req.NextState), ReasonCode: reasonCode,
		ProviderState: providerState, BalanceDeadlineAt: timestampParam(req.BalanceDeadlineAt),
		Now: timestampParam(now),
	}); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrPositionCashOutBatchClaim
		}
		return nil, fmt.Errorf("record position Cash Out batch item state: %w", err)
	}
	var updated wormtradingsqlc.WormPositionCashOutBatch
	if req.ExpectedState == PositionCashOutBatchItemStateReconciliationRequired &&
		req.NextState == PositionCashOutBatchItemStateAwaitingBalance {
		updated, err = queries.ResumePositionCashOutBatchBalanceAfterReconciliation(ctx,
			wormtradingsqlc.ResumePositionCashOutBatchBalanceAfterReconciliationParams{
				ID: batchID, ClaimID: claimID, CurrentItemOrdinal: itemRow.Ordinal,
				NextPollAt: timestampParam(req.NextPollAt), Now: timestampParam(now),
			})
	} else {
		updated, err = queries.SetPositionCashOutBatchNextPoll(ctx, wormtradingsqlc.SetPositionCashOutBatchNextPollParams{
			ID: batchID, ClaimID: claimID, NextPollAt: timestampParam(req.NextPollAt), Now: timestampParam(now),
		})
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrPositionCashOutBatchClaim
	}
	if err != nil {
		return nil, fmt.Errorf("schedule position Cash Out batch item observation: %w", err)
	}
	batch, err := loadPositionCashOutBatch(ctx, queries, updated)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit position Cash Out batch item state: %w", err)
	}
	return &batch, nil
}

func (s *SQLStore) RecordPositionCashOutBatchBalance(
	ctx context.Context,
	req RecordPositionCashOutBatchBalanceRequest,
) (*PositionCashOutBatch, error) {
	batchID, itemID, claimID, now, err := normalizePositionCashOutBatchItemClaim(
		req.BatchID, req.ItemID, req.ClaimID, req.Now,
	)
	if err != nil {
		return nil, err
	}
	observed, err := normalizePositionCashOutBatchBalance(req.Observed)
	if err != nil {
		return nil, err
	}
	if err := s.requireDatabase(); err != nil {
		return nil, err
	}
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, fmt.Errorf("begin record position Cash Out batch balance: %w", err)
	}
	defer rollbackWalletTransaction(tx)
	queries := wormtradingsqlc.New(tx)
	batchRow, err := queries.GetPositionCashOutBatchByIDForUpdate(ctx, batchID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrPositionCashOutBatchNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("lock position Cash Out batch balance: %w", err)
	}
	itemRow, err := queries.GetPositionCashOutBatchItemForUpdate(ctx, wormtradingsqlc.GetPositionCashOutBatchItemForUpdateParams{
		ID: itemID, BatchID: batchID,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrPositionCashOutBatchClaim
	}
	if err != nil {
		return nil, fmt.Errorf("lock position Cash Out batch balance item: %w", err)
	}
	if batchRow.ClaimID != claimID || nullableInt64(batchRow.CurrentItemOrdinal) != itemRow.Ordinal ||
		itemRow.State != string(PositionCashOutBatchItemStateAwaitingBalance) {
		return nil, ErrPositionCashOutBatchClaim
	}
	baseline := positionCashOutBatchBaseline(itemRow)
	deadline := timestampValue(itemRow.BalanceDeadlineAt)
	if baseline == nil || deadline.IsZero() {
		return nil, ErrPositionCashOutBatchClaim
	}
	baselineAmount, _ := new(big.Int).SetString(baseline.AtomicAmount, 10)
	observedAmount, _ := new(big.Int).SetString(observed.AtomicAmount, 10)
	passed := observed.Mint == baseline.Mint && observed.Decimals == baseline.Decimals &&
		observed.ObservedSlot > baseline.ObservedSlot && observedAmount.Cmp(baselineAmount) > 0
	if passed {
		delta := new(big.Int).Sub(new(big.Int).Set(observedAmount), baselineAmount).String()
		if _, err := queries.CompletePositionCashOutBatchItemBalance(ctx, wormtradingsqlc.CompletePositionCashOutBatchItemBalanceParams{
			ItemID: itemID, BatchID: batchID, Mint: observed.Mint,
			Decimals: observed.Decimals, AtomicAmount: observed.AtomicAmount,
			ObservedSlot: int64(observed.ObservedSlot), DeltaAtomicAmount: delta,
			Now: timestampParam(now),
		}); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return nil, ErrPositionCashOutBatchClaim
			}
			return nil, fmt.Errorf("complete position Cash Out batch balance gate: %w", err)
		}
		if affected, err := queries.IncrementPositionCashOutBatchWalletCompleted(ctx, wormtradingsqlc.IncrementPositionCashOutBatchWalletCompletedParams{
			BatchID: batchID, WalletOrdinal: itemRow.WalletOrdinal, Now: timestampParam(now),
		}); err != nil || affected != 1 {
			if err != nil {
				return nil, fmt.Errorf("increment position Cash Out batch Wallet completion: %w", err)
			}
			return nil, ErrPositionCashOutBatchClaim
		}
		nextOrdinal := itemRow.Ordinal + 1
		updated, err := queries.AdvancePositionCashOutBatchAfterBalance(ctx, wormtradingsqlc.AdvancePositionCashOutBatchAfterBalanceParams{
			ID: batchID, ClaimID: claimID, CurrentItemOrdinal: itemRow.Ordinal,
			NextItemOrdinal: nextOrdinal, Now: timestampParam(now),
		})
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrPositionCashOutBatchClaim
		}
		if err != nil {
			return nil, fmt.Errorf("advance position Cash Out batch after balance: %w", err)
		}
		batch, err := loadPositionCashOutBatch(ctx, queries, updated)
		if err != nil {
			return nil, err
		}
		if err := tx.Commit(ctx); err != nil {
			return nil, fmt.Errorf("commit position Cash Out batch balance completion: %w", err)
		}
		return &batch, nil
	}

	if _, err := queries.RecordPositionCashOutBatchObservedBalance(ctx, wormtradingsqlc.RecordPositionCashOutBatchObservedBalanceParams{
		ItemID: itemID, BatchID: batchID, Mint: observed.Mint,
		Decimals: observed.Decimals, AtomicAmount: observed.AtomicAmount,
		ObservedSlot: int64(observed.ObservedSlot), Now: timestampParam(now),
	}); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrPositionCashOutBatchClaim
		}
		return nil, fmt.Errorf("record position Cash Out batch observed balance: %w", err)
	}
	var updated wormtradingsqlc.WormPositionCashOutBatch
	timedOut := req.TimedOut || !now.Before(deadline)
	if timedOut {
		updated, err = queries.PausePositionCashOutBatchForBalance(ctx, wormtradingsqlc.PausePositionCashOutBatchForBalanceParams{
			ID: batchID, ClaimID: claimID, CurrentItemOrdinal: itemRow.Ordinal,
			ReasonCode: "USDC_CREDIT_NOT_OBSERVED", Now: timestampParam(now),
		})
	} else {
		updated, err = queries.SetPositionCashOutBatchNextPoll(ctx, wormtradingsqlc.SetPositionCashOutBatchNextPollParams{
			ID: batchID, ClaimID: claimID, NextPollAt: timestampParam(now.Add(2 * time.Second)),
			Now: timestampParam(now),
		})
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrPositionCashOutBatchClaim
	}
	if err != nil {
		return nil, fmt.Errorf("schedule position Cash Out batch balance observation: %w", err)
	}
	batch, err := loadPositionCashOutBatch(ctx, queries, updated)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit position Cash Out batch balance observation: %w", err)
	}
	return &batch, nil
}

func (s *SQLStore) DeferPositionCashOutBatchBalance(
	ctx context.Context,
	req DeferPositionCashOutBatchBalanceRequest,
) (*PositionCashOutBatch, error) {
	batchID, itemID, claimID, now, err := normalizePositionCashOutBatchItemClaim(
		req.BatchID, req.ItemID, req.ClaimID, req.Now,
	)
	if err != nil {
		return nil, err
	}
	if err := s.requireDatabase(); err != nil {
		return nil, err
	}
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, fmt.Errorf("begin defer position Cash Out batch balance: %w", err)
	}
	defer rollbackWalletTransaction(tx)
	queries := wormtradingsqlc.New(tx)
	batchRow, err := queries.GetPositionCashOutBatchByIDForUpdate(ctx, batchID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrPositionCashOutBatchNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("lock deferred position Cash Out batch balance: %w", err)
	}
	itemRow, err := queries.GetPositionCashOutBatchItemForUpdate(ctx, wormtradingsqlc.GetPositionCashOutBatchItemForUpdateParams{
		ID: itemID, BatchID: batchID,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrPositionCashOutBatchClaim
	}
	if err != nil {
		return nil, fmt.Errorf("lock deferred position Cash Out batch balance item: %w", err)
	}
	if batchRow.ClaimID != claimID || nullableInt64(batchRow.CurrentItemOrdinal) != itemRow.Ordinal ||
		itemRow.State != string(PositionCashOutBatchItemStateAwaitingBalance) {
		return nil, ErrPositionCashOutBatchClaim
	}
	deadline := timestampValue(itemRow.BalanceDeadlineAt)
	if deadline.IsZero() || positionCashOutBatchBaseline(itemRow) == nil {
		return nil, ErrPositionCashOutBatchClaim
	}
	var updated wormtradingsqlc.WormPositionCashOutBatch
	timedOut := req.TimedOut || !now.Before(deadline)
	if timedOut {
		updated, err = queries.PausePositionCashOutBatchForBalance(ctx, wormtradingsqlc.PausePositionCashOutBatchForBalanceParams{
			ID: batchID, ClaimID: claimID, CurrentItemOrdinal: itemRow.Ordinal,
			ReasonCode: "USDC_CREDIT_NOT_OBSERVED", Now: timestampParam(now),
		})
	} else {
		nextPollAt := time.Time{}
		if (batchRow.State == string(PositionCashOutBatchStateRunning) ||
			batchRow.State == string(PositionCashOutBatchStatePauseRequested) ||
			batchRow.State == string(PositionCashOutBatchStateTerminateRequested)) && batchRow.ReasonCode == "" {
			nextPollAt = now.Add(2 * time.Second)
		}
		updated, err = queries.SetPositionCashOutBatchNextPoll(ctx, wormtradingsqlc.SetPositionCashOutBatchNextPollParams{
			ID: batchID, ClaimID: claimID, NextPollAt: timestampParam(nextPollAt),
			Now: timestampParam(now),
		})
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrPositionCashOutBatchClaim
	}
	if err != nil {
		return nil, fmt.Errorf("defer position Cash Out batch balance: %w", err)
	}
	batch, err := loadPositionCashOutBatch(ctx, queries, updated)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit deferred position Cash Out batch balance: %w", err)
	}
	return &batch, nil
}

func (s *SQLStore) RecordPositionCashOutBatchBlocking(
	ctx context.Context,
	req RecordPositionCashOutBatchBlockingRequest,
) (*PositionCashOutBatch, error) {
	batchID, itemID, claimID, now, err := normalizePositionCashOutBatchItemClaim(
		req.BatchID, req.ItemID, req.ClaimID, req.Now,
	)
	if err != nil {
		return nil, err
	}
	if (req.ItemState != PositionCashOutBatchItemStateFailed &&
		req.ItemState != PositionCashOutBatchItemStateReconciliationRequired) ||
		(req.BatchState != PositionCashOutBatchStatePaused &&
			req.BatchState != PositionCashOutBatchStateFailed &&
			req.BatchState != PositionCashOutBatchStateReconciliationRequired &&
			req.BatchState != PositionCashOutBatchStateTerminateRequested) {
		return nil, invalidPositionCashOutBatch(fmt.Errorf("blocking state is invalid"))
	}
	if req.BatchState == PositionCashOutBatchStateTerminateRequested &&
		req.ItemState != PositionCashOutBatchItemStateReconciliationRequired {
		return nil, invalidPositionCashOutBatch(fmt.Errorf("terminate-requested blocking item must require reconciliation"))
	}
	reasonCode, err := normalizePositionCashOutCode(req.ReasonCode, true)
	if err != nil {
		return nil, invalidPositionCashOutBatch(err)
	}
	walletIDs, err := s.positionCashOutBatchWalletIDs(ctx, batchID)
	if err != nil {
		return nil, err
	}
	tx, queries, err := s.beginWalletsTransaction(ctx, walletIDs)
	if err != nil {
		return nil, err
	}
	defer rollbackWalletTransaction(tx)
	batchRow, err := queries.GetPositionCashOutBatchByIDForUpdate(ctx, batchID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrPositionCashOutBatchClaim
	}
	if err != nil {
		return nil, fmt.Errorf("lock blocking position Cash Out batch: %w", err)
	}
	if batchRow.ClaimID != claimID {
		return nil, ErrPositionCashOutBatchClaim
	}
	itemRow, err := queries.GetPositionCashOutBatchItemForUpdate(ctx, wormtradingsqlc.GetPositionCashOutBatchItemForUpdateParams{
		ID: itemID, BatchID: batchID,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrPositionCashOutBatchClaim
	}
	if err != nil {
		return nil, fmt.Errorf("lock blocking position Cash Out batch item: %w", err)
	}
	matchesCurrent := batchRow.CurrentItemOrdinal.Valid &&
		batchRow.CurrentItemOrdinal.Int64 == itemRow.Ordinal
	matchesPendingNext := !batchRow.CurrentItemOrdinal.Valid &&
		nullableInt64(batchRow.NextItemOrdinal) == itemRow.Ordinal &&
		itemRow.State == string(PositionCashOutBatchItemStatePending)
	if !matchesCurrent && !matchesPendingNext {
		return nil, ErrPositionCashOutBatchClaim
	}
	if _, err := queries.MarkPositionCashOutBatchItemBlocking(ctx, wormtradingsqlc.MarkPositionCashOutBatchItemBlockingParams{
		ItemID: itemID, BatchID: batchID, NextState: string(req.ItemState),
		ReasonCode: reasonCode, Now: timestampParam(now),
	}); err != nil {
		return nil, fmt.Errorf("mark position Cash Out batch item blocking: %w", err)
	}
	notExecutedCount := batchRow.NotExecutedCount
	if req.BatchState == PositionCashOutBatchStateFailed {
		marked, markErr := queries.MarkRemainingPositionCashOutBatchItemsNotExecuted(ctx, wormtradingsqlc.MarkRemainingPositionCashOutBatchItemsNotExecutedParams{
			BatchID: batchID, ReasonCode: reasonCode, Now: timestampParam(now),
		})
		if markErr != nil {
			return nil, fmt.Errorf("mark failed batch remaining items not executed: %w", markErr)
		}
		notExecutedCount += int64(len(marked))
	}
	updated, err := queries.MarkPositionCashOutBatchBlocking(ctx, wormtradingsqlc.MarkPositionCashOutBatchBlockingParams{
		ID: batchID, ClaimID: claimID, ItemOrdinal: itemRow.Ordinal, NextState: string(req.BatchState),
		ReasonCode: reasonCode, NotExecutedCount: notExecutedCount, Now: timestampParam(now),
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrPositionCashOutBatchClaim
	}
	if err != nil {
		return nil, fmt.Errorf("mark position Cash Out batch blocking: %w", err)
	}
	if req.BatchState == PositionCashOutBatchStateFailed {
		if _, err := queries.ReleasePositionCashOutBatchWalletLocks(ctx, batchID); err != nil {
			return nil, fmt.Errorf("release failed position Cash Out batch Wallet locks: %w", err)
		}
		if _, err := queries.EndPositionCashOutBatchAuthorization(ctx, wormtradingsqlc.EndPositionCashOutBatchAuthorizationParams{
			BatchID: batchID, State: string(PositionCashOutBatchAuthorizationStateConsumed),
			ReasonCode: reasonCode, Now: timestampParam(now),
		}); err != nil {
			return nil, fmt.Errorf("end failed position Cash Out batch authorization: %w", err)
		}
	}
	batch, err := loadPositionCashOutBatch(ctx, queries, updated)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit blocking position Cash Out batch: %w", err)
	}
	return &batch, nil
}

func (s *SQLStore) CompletePositionCashOutBatchBoundary(
	ctx context.Context,
	req CompletePositionCashOutBatchBoundaryRequest,
) (*PositionCashOutBatch, error) {
	batchID, _, err := marketCombinationUUID(req.BatchID, "position Cash Out batch ID")
	if err != nil {
		return nil, invalidPositionCashOutBatch(err)
	}
	claimID, _, err := marketCombinationUUID(req.ClaimID, "claim ID")
	if err != nil {
		return nil, invalidPositionCashOutBatch(err)
	}
	now := canonicalNow(req.Now)
	walletIDs, err := s.positionCashOutBatchWalletIDs(ctx, batchID)
	if err != nil {
		return nil, err
	}
	tx, queries, err := s.beginWalletsTransaction(ctx, walletIDs)
	if err != nil {
		return nil, err
	}
	defer rollbackWalletTransaction(tx)
	row, err := queries.GetPositionCashOutBatchByIDForUpdate(ctx, batchID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrPositionCashOutBatchClaim
	}
	if err != nil {
		return nil, fmt.Errorf("lock position Cash Out batch safe boundary: %w", err)
	}
	if row.ClaimID != claimID {
		return nil, ErrPositionCashOutBatchClaim
	}
	var updated wormtradingsqlc.WormPositionCashOutBatch
	terminal := false
	switch PositionCashOutBatchState(row.State) {
	case PositionCashOutBatchStatePauseRequested:
		if row.CurrentItemOrdinal.Valid {
			return nil, ErrPositionCashOutBatchClaim
		}
		updated, err = queries.PausePositionCashOutBatchAtBoundary(ctx, wormtradingsqlc.PausePositionCashOutBatchAtBoundaryParams{
			ID: batchID, ClaimID: claimID, Now: timestampParam(now),
		})
	case PositionCashOutBatchStateTerminateRequested:
		currentNotExecuted, terminateErr := terminateUndispatchedPositionCashOutBatchCurrent(
			ctx, queries, row, now,
		)
		if terminateErr != nil {
			return nil, terminateErr
		}
		marked, markErr := queries.MarkRemainingPositionCashOutBatchItemsNotExecuted(ctx, wormtradingsqlc.MarkRemainingPositionCashOutBatchItemsNotExecutedParams{
			BatchID: batchID, ReasonCode: "BATCH_TERMINATED", Now: timestampParam(now),
		})
		if markErr != nil {
			return nil, fmt.Errorf("mark remaining batch items not executed: %w", markErr)
		}
		updated, err = queries.CompletePositionCashOutBatch(ctx, wormtradingsqlc.CompletePositionCashOutBatchParams{
			ID: batchID, ExpectedState: row.State,
			TerminalState:    string(PositionCashOutBatchStateTerminated),
			NotExecutedCount: int64(len(marked)) + currentNotExecuted, Now: timestampParam(now),
		})
		terminal = true
	case PositionCashOutBatchStateRunning:
		if row.CurrentItemOrdinal.Valid {
			return nil, ErrPositionCashOutBatchClaim
		}
		if row.CompletedCount != row.PositionCount || row.PositionCount == 0 {
			return nil, ErrPositionCashOutBatchClaim
		}
		updated, err = queries.CompletePositionCashOutBatch(ctx, wormtradingsqlc.CompletePositionCashOutBatchParams{
			ID: batchID, ExpectedState: row.State,
			TerminalState:    string(PositionCashOutBatchStateCompleted),
			NotExecutedCount: 0, Now: timestampParam(now),
		})
		terminal = true
	default:
		return nil, ErrPositionCashOutBatchClaim
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrPositionCashOutBatchClaim
	}
	if err != nil {
		return nil, fmt.Errorf("complete position Cash Out batch safe boundary: %w", err)
	}
	if terminal {
		if _, err := queries.ReleasePositionCashOutBatchWalletLocks(ctx, batchID); err != nil {
			return nil, fmt.Errorf("release terminal position Cash Out batch Wallet locks: %w", err)
		}
		if _, err := queries.EndPositionCashOutBatchAuthorization(ctx, wormtradingsqlc.EndPositionCashOutBatchAuthorizationParams{
			BatchID: batchID, State: string(PositionCashOutBatchAuthorizationStateConsumed),
			ReasonCode: updated.State, Now: timestampParam(now),
		}); err != nil {
			return nil, fmt.Errorf("end terminal position Cash Out batch authorization: %w", err)
		}
	}
	batch, err := loadPositionCashOutBatch(ctx, queries, updated)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit position Cash Out batch safe boundary: %w", err)
	}
	return &batch, nil
}

func terminateUndispatchedPositionCashOutBatchCurrent(
	ctx context.Context,
	queries *wormtradingsqlc.Queries,
	batchRow wormtradingsqlc.WormPositionCashOutBatch,
	now time.Time,
) (int64, error) {
	if !batchRow.CurrentItemOrdinal.Valid {
		return 0, nil
	}
	itemRow, err := queries.GetPositionCashOutBatchItemByOrdinalForUpdate(ctx, wormtradingsqlc.GetPositionCashOutBatchItemByOrdinalForUpdateParams{
		BatchID: batchRow.ID, Ordinal: batchRow.CurrentItemOrdinal.Int64,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, ErrPositionCashOutBatchClaim
	}
	if err != nil {
		return 0, fmt.Errorf("lock current position Cash Out batch item for termination: %w", err)
	}
	if itemRow.State != string(PositionCashOutBatchItemStateFailed) &&
		itemRow.State != string(PositionCashOutBatchItemStateReconciliationRequired) {
		return 0, ErrPositionCashOutBatchClaim
	}
	if itemRow.ChildCashOutID.Valid {
		childRow, err := queries.GetPositionCashOutByIDForUpdate(ctx, itemRow.ChildCashOutID)
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, ErrPositionCashOutBatchClaim
		}
		if err != nil {
			return 0, fmt.Errorf("lock undispatched batch child for termination: %w", err)
		}
		if childRow.BatchID != batchRow.ID || childRow.BatchItemID != itemRow.ID ||
			(childRow.State != string(PositionCashOutStateFailed) &&
				childRow.State != string(PositionCashOutStateReconciliationRequired)) {
			return 0, ErrPositionCashOutBatchClaim
		}
		attemptRow, err := queries.GetPositionCashOutAttempt(ctx, itemRow.ChildCashOutID)
		if err == nil && attemptRow.State != string(PositionCashOutAttemptStatePrepared) {
			return 0, ErrPositionCashOutBatchClaim
		}
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			return 0, fmt.Errorf("get batch child attempt for termination: %w", err)
		}
		if _, err := queries.FailUndispatchedPositionCashOutBatchChild(ctx, wormtradingsqlc.FailUndispatchedPositionCashOutBatchChildParams{
			CashOutID: itemRow.ChildCashOutID, BatchID: batchRow.ID,
			ItemID: itemRow.ID, Now: timestampParam(now),
		}); errors.Is(err, pgx.ErrNoRows) {
			return 0, ErrPositionCashOutBatchClaim
		} else if err != nil {
			return 0, fmt.Errorf("fail undispatched batch child for termination: %w", err)
		}
	}
	if _, err := queries.MarkCurrentPositionCashOutBatchItemNotExecuted(ctx, wormtradingsqlc.MarkCurrentPositionCashOutBatchItemNotExecutedParams{
		ItemID: itemRow.ID, BatchID: batchRow.ID,
		ReasonCode: "BATCH_TERMINATED", Now: timestampParam(now),
	}); errors.Is(err, pgx.ErrNoRows) {
		return 0, ErrPositionCashOutBatchClaim
	} else if err != nil {
		return 0, fmt.Errorf("mark current batch item not executed: %w", err)
	}
	return 1, nil
}

func (s *SQLStore) ListRecoverablePositionCashOutBatches(
	ctx context.Context,
	now time.Time,
	limit int32,
) ([]PositionCashOutBatch, error) {
	if limit <= 0 || limit > maxPositionCashOutBatchRecoveryLimit {
		return nil, invalidPositionCashOutBatch(fmt.Errorf("recovery limit must be between 1 and %d", maxPositionCashOutBatchRecoveryLimit))
	}
	if err := s.requireDatabase(); err != nil {
		return nil, err
	}
	ids, err := s.queries.ListRecoverablePositionCashOutBatchKeys(ctx, wormtradingsqlc.ListRecoverablePositionCashOutBatchKeysParams{
		Now: timestampParam(canonicalNow(now)), RecoveryLimit: limit,
	})
	if err != nil {
		return nil, fmt.Errorf("list recoverable position Cash Out batches: %w", err)
	}
	result := make([]PositionCashOutBatch, 0, len(ids))
	for _, id := range ids {
		batch, err := loadPositionCashOutBatchByID(ctx, s.queries, id)
		if errors.Is(err, ErrPositionCashOutBatchNotFound) {
			continue
		}
		if err != nil {
			return nil, err
		}
		result = append(result, batch)
	}
	return result, nil
}

func (s *SQLStore) ExpirePositionCashOutBatches(
	ctx context.Context,
	now time.Time,
	limit int32,
) (int64, error) {
	if limit <= 0 || limit > maxPositionCashOutBatchRecoveryLimit {
		return 0, invalidPositionCashOutBatch(fmt.Errorf("expiry limit must be between 1 and %d", maxPositionCashOutBatchRecoveryLimit))
	}
	if err := s.requireDatabase(); err != nil {
		return 0, err
	}
	now = canonicalNow(now)
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return 0, fmt.Errorf("begin expire position Cash Out batches: %w", err)
	}
	defer rollbackWalletTransaction(tx)
	queries := wormtradingsqlc.New(tx)
	rows, err := queries.ExpirePositionCashOutBatches(ctx, wormtradingsqlc.ExpirePositionCashOutBatchesParams{
		Now: timestampParam(now), ExpireLimit: limit,
	})
	if err != nil {
		return 0, fmt.Errorf("expire position Cash Out batches: %w", err)
	}
	for _, row := range rows {
		if _, err := queries.ReleasePositionCashOutBatchWalletLocks(ctx, row.ID); err != nil {
			return 0, fmt.Errorf("release expired position Cash Out batch Wallet locks: %w", err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("commit expired position Cash Out batches: %w", err)
	}
	return int64(len(rows)), nil
}

func loadPositionCashOutBatchByID(
	ctx context.Context,
	queries *wormtradingsqlc.Queries,
	id pgtype.UUID,
) (PositionCashOutBatch, error) {
	row, err := queries.GetPositionCashOutBatchByID(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return PositionCashOutBatch{}, ErrPositionCashOutBatchNotFound
	}
	if err != nil {
		return PositionCashOutBatch{}, fmt.Errorf("get position Cash Out batch by ID: %w", err)
	}
	return loadPositionCashOutBatch(ctx, queries, row)
}

func loadPositionCashOutBatch(
	ctx context.Context,
	queries *wormtradingsqlc.Queries,
	row wormtradingsqlc.WormPositionCashOutBatch,
) (PositionCashOutBatch, error) {
	batch := mapPositionCashOutBatch(row)
	walletRows, err := queries.ListPositionCashOutBatchWallets(ctx, row.ID)
	if err != nil {
		return PositionCashOutBatch{}, fmt.Errorf("list position Cash Out batch Wallets: %w", err)
	}
	batch.Wallets = make([]PositionCashOutBatchWallet, 0, len(walletRows))
	for _, walletRow := range walletRows {
		batch.Wallets = append(batch.Wallets, mapPositionCashOutBatchWallet(walletRow))
	}
	if row.CurrentItemOrdinal.Valid {
		itemRow, err := queries.GetPositionCashOutBatchItemByOrdinal(ctx, wormtradingsqlc.GetPositionCashOutBatchItemByOrdinalParams{
			BatchID: row.ID, Ordinal: row.CurrentItemOrdinal.Int64,
		})
		if err != nil {
			return PositionCashOutBatch{}, fmt.Errorf("get current position Cash Out batch item: %w", err)
		}
		item := mapPositionCashOutBatchItem(itemRow)
		batch.CurrentItem = &item
	}
	authorization, err := queries.GetPositionCashOutBatchAuthorization(ctx, row.ID)
	if err == nil {
		mapped := mapPositionCashOutBatchAuthorization(authorization)
		batch.Authorization = &mapped
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return PositionCashOutBatch{}, fmt.Errorf("get position Cash Out batch authorization: %w", err)
	}
	return batch, nil
}

func mapPositionCashOutBatch(row wormtradingsqlc.WormPositionCashOutBatch) PositionCashOutBatch {
	return PositionCashOutBatch{
		ID: uuidValue(row.ID), OwnerAccountID: uuidValue(row.OwnerAccountID),
		SelectionDigestSHA256: append([]byte(nil), row.SelectionDigestSha256...),
		IntentDigestSHA256:    append([]byte(nil), row.IntentDigestSha256...),
		State:                 PositionCashOutBatchState(row.State), Revision: row.Revision,
		ReasonCode: row.ReasonCode, BuildStage: row.BuildStage,
		WalletCount: row.WalletCount, PositionCount: row.PositionCount,
		CompletedCount: row.CompletedCount, NotExecutedCount: row.NotExecutedCount,
		NextItemOrdinal:        nullableInt64(row.NextItemOrdinal),
		CurrentItemOrdinal:     nullableInt64(row.CurrentItemOrdinal),
		AuthorizationExpiresAt: timestampValue(row.AuthorizationExpiresAt),
		AuthorizedAt:           timestampValue(row.AuthorizedAt),
		ExecutionStartedAt:     timestampValue(row.ExecutionStartedAt),
		NextPollAt:             timestampValue(row.NextPollAt),
		CheckRequestedAt:       timestampValue(row.CheckRequestedAt),
		ClaimID:                uuidValue(row.ClaimID), ClaimOwner: row.ClaimOwner,
		ClaimExpiresAt: timestampValue(row.ClaimExpiresAt),
		CompletedAt:    timestampValue(row.CompletedAt), CreatedAt: timestampValue(row.CreatedAt),
		UpdatedAt: timestampValue(row.UpdatedAt),
	}
}

func mapPositionCashOutBatchWallet(
	row wormtradingsqlc.WormPositionCashOutBatchWallet,
) PositionCashOutBatchWallet {
	return PositionCashOutBatchWallet{
		Ordinal: row.Ordinal, WalletID: row.WalletID, Address: row.Address,
		CredentialVersion: row.CredentialVersion, Remark: row.Remark,
		AvatarKind: row.AvatarKind, AvatarPresetID: row.AvatarPresetID,
		AvatarURL: row.AvatarUrl, PositionCount: row.PositionCount,
		CompletedCount: row.CompletedCount,
	}
}

func mapPositionCashOutBatchItem(
	row wormtradingsqlc.WormPositionCashOutBatchItem,
) PositionCashOutBatchItem {
	item := PositionCashOutBatchItem{
		ID: uuidValue(row.ID), BatchID: uuidValue(row.BatchID), Ordinal: row.Ordinal,
		WalletOrdinal: row.WalletOrdinal, PositionOrdinal: row.PositionOrdinal,
		WalletID: row.WalletID, WalletAddress: row.WalletAddress,
		WalletRemark: row.WalletRemark, CredentialVersion: row.CredentialVersion,
		PositionPubkey: row.PositionPubkey, PositionRequestPubkey: row.PositionRequestPubkey,
		MarketConditionID: row.MarketConditionID, MarketTitle: row.MarketTitle,
		IsYes: row.IsYes, Shares: row.Shares,
		PositionCreatedAt: timestampValue(row.PositionCreatedAt), ProviderState: row.ProviderState,
		State: PositionCashOutBatchItemState(row.State), ReasonCode: row.ReasonCode,
		ChildCashOutID:        uuidValue(row.ChildCashOutID),
		DeltaUSDCAtomicAmount: row.DeltaUsdcAtomicAmount,
		BalanceStartedAt:      timestampValue(row.BalanceStartedAt),
		BalanceDeadlineAt:     timestampValue(row.BalanceDeadlineAt),
		BalanceConfirmedAt:    timestampValue(row.BalanceConfirmedAt),
		CompletedAt:           timestampValue(row.CompletedAt), CreatedAt: timestampValue(row.CreatedAt),
		UpdatedAt: timestampValue(row.UpdatedAt),
	}
	item.Baseline = positionCashOutBatchBaseline(row)
	item.Observed = positionCashOutBatchObserved(row)
	return item
}

func mapPositionCashOutBatchAuthorization(
	row wormtradingsqlc.WormPositionCashOutBatchAuthorization,
) PositionCashOutBatchAuthorization {
	return PositionCashOutBatchAuthorization{
		ID: uuidValue(row.ID), State: PositionCashOutBatchAuthorizationState(row.State),
		Scope: row.Scope, ProofKind: row.ProofKind,
		SessionJTIDigest:   append([]byte(nil), row.SessionJtiDigest...),
		AccessRevision:     row.AccessRevision,
		IntentDigestSHA256: append([]byte(nil), row.IntentDigestSha256...),
		AuthorizedAt:       timestampValue(row.AuthorizedAt), EndedAt: timestampValue(row.EndedAt),
		EndReasonCode: row.EndReasonCode,
	}
}

func positionCashOutBatchBaseline(
	row wormtradingsqlc.WormPositionCashOutBatchItem,
) *PositionCashOutBatchBalanceEvidence {
	if row.BaselineUsdcMint == "" || !row.BaselineUsdcDecimals.Valid ||
		row.BaselineUsdcAtomicAmount == "" || !row.BaselineUsdcObservedSlot.Valid {
		return nil
	}
	return &PositionCashOutBatchBalanceEvidence{
		Mint: row.BaselineUsdcMint, Decimals: row.BaselineUsdcDecimals.Int32,
		AtomicAmount: row.BaselineUsdcAtomicAmount,
		ObservedSlot: uint64(row.BaselineUsdcObservedSlot.Int64),
	}
}

func positionCashOutBatchObserved(
	row wormtradingsqlc.WormPositionCashOutBatchItem,
) *PositionCashOutBatchBalanceEvidence {
	if row.ObservedUsdcMint == "" || !row.ObservedUsdcDecimals.Valid ||
		row.ObservedUsdcAtomicAmount == "" || !row.ObservedUsdcSlot.Valid {
		return nil
	}
	return &PositionCashOutBatchBalanceEvidence{
		Mint: row.ObservedUsdcMint, Decimals: row.ObservedUsdcDecimals.Int32,
		AtomicAmount: row.ObservedUsdcAtomicAmount,
		ObservedSlot: uint64(row.ObservedUsdcSlot.Int64),
	}
}

func positionCashOutBatchOwnerAndID(
	ownerAccountID string,
	batchID string,
) (pgtype.UUID, string, pgtype.UUID, error) {
	ownerUUID, ownerAccountID, err := marketCombinationUUID(ownerAccountID, "owner account ID")
	if err != nil {
		return pgtype.UUID{}, "", pgtype.UUID{}, invalidPositionCashOutBatch(err)
	}
	id, _, err := marketCombinationUUID(batchID, "position Cash Out batch ID")
	if err != nil {
		return pgtype.UUID{}, "", pgtype.UUID{}, invalidPositionCashOutBatch(err)
	}
	return ownerUUID, ownerAccountID, id, nil
}

func (s *SQLStore) positionCashOutBatchWalletIDs(
	ctx context.Context,
	batchID pgtype.UUID,
) ([]int64, error) {
	if err := s.requireDatabase(); err != nil {
		return nil, err
	}
	walletIDs, err := s.queries.ListPositionCashOutBatchWalletIDs(ctx, batchID)
	if err != nil {
		return nil, fmt.Errorf("list position Cash Out batch Wallet IDs: %w", err)
	}
	if len(walletIDs) == 0 {
		return nil, ErrPositionCashOutBatchNotFound
	}
	return normalizeWalletOperationIDs(walletIDs)
}

func normalizePositionCashOutBatchWalletInputs(
	inputs []PositionCashOutBatchWalletInput,
) ([]PositionCashOutBatchWalletInput, []int64, error) {
	if len(inputs) == 0 || len(inputs) > maxPositionCashOutBatchWallets {
		return nil, nil, invalidPositionCashOutBatch(fmt.Errorf("batch must contain between 1 and %d Wallets", maxPositionCashOutBatchWallets))
	}
	result := make([]PositionCashOutBatchWalletInput, len(inputs))
	walletIDs := make([]int64, 0, len(inputs))
	seen := make(map[int64]struct{}, len(inputs))
	seenAddresses := make(map[string]struct{}, len(inputs))
	for index, input := range inputs {
		if err := validateWalletReference(input.WalletID, input.Address); err != nil {
			return nil, nil, invalidPositionCashOutBatch(err)
		}
		if _, exists := seen[input.WalletID]; exists {
			return nil, nil, invalidPositionCashOutBatch(fmt.Errorf("Wallet IDs must be unique"))
		}
		seen[input.WalletID] = struct{}{}
		if _, exists := seenAddresses[input.Address]; exists {
			return nil, nil, invalidPositionCashOutBatch(fmt.Errorf("Wallet addresses must be unique"))
		}
		seenAddresses[input.Address] = struct{}{}
		input.Remark = strings.TrimSpace(input.Remark)
		input.AvatarKind = strings.TrimSpace(input.AvatarKind)
		input.AvatarPresetID = strings.TrimSpace(input.AvatarPresetID)
		input.AvatarURL = strings.TrimSpace(input.AvatarURL)
		if utf8.RuneCountInString(input.Remark) > maxPositionCashOutBatchTextLength ||
			len(input.AvatarKind) > maxPositionCashOutCodeLength ||
			len(input.AvatarPresetID) > maxPositionCashOutCodeLength ||
			utf8.RuneCountInString(input.AvatarURL) > maxPositionCashOutBatchTextLength {
			return nil, nil, invalidPositionCashOutBatch(fmt.Errorf("Wallet display snapshot is invalid"))
		}
		result[index] = input
		walletIDs = append(walletIDs, input.WalletID)
	}
	sortedIDs := append([]int64(nil), walletIDs...)
	sort.Slice(sortedIDs, func(left, right int) bool { return sortedIDs[left] < sortedIDs[right] })
	return result, sortedIDs, nil
}

func normalizePositionCashOutBatchItems(
	inputs []PositionCashOutBatchItemInput,
	walletRows []wormtradingsqlc.WormPositionCashOutBatchWallet,
) ([]PositionCashOutBatchItemInput, map[int32]int64, error) {
	if len(inputs) == 0 || len(inputs) > maxPositionCashOutBatchItems {
		return nil, nil, invalidPositionCashOutBatch(fmt.Errorf("batch must contain between 1 and %d positions", maxPositionCashOutBatchItems))
	}
	if len(walletRows) == 0 || len(walletRows) > maxPositionCashOutBatchWallets {
		return nil, nil, ErrPositionCashOutBatchClaim
	}
	wallets := make(map[int32]wormtradingsqlc.WormPositionCashOutBatchWallet, len(walletRows))
	for _, row := range walletRows {
		wallets[row.Ordinal] = row
	}
	result := make([]PositionCashOutBatchItemInput, len(inputs))
	counts := make(map[int32]int64, len(walletRows))
	seenPubkeys := make(map[string]struct{}, len(inputs))
	var lastWalletOrdinal int32
	var lastPositionOrdinal int32
	var lastCreatedAt time.Time
	for index, input := range inputs {
		wallet, exists := wallets[input.WalletOrdinal]
		if !exists || input.WalletID != wallet.WalletID || input.WalletAddress != wallet.Address ||
			input.CredentialVersion != wallet.CredentialVersion || input.WalletRemark != wallet.Remark {
			return nil, nil, invalidPositionCashOutBatch(fmt.Errorf("batch item Wallet snapshot is invalid"))
		}
		if input.WalletOrdinal < lastWalletOrdinal ||
			(input.WalletOrdinal == lastWalletOrdinal && input.PositionOrdinal != lastPositionOrdinal+1) ||
			(input.WalletOrdinal != lastWalletOrdinal && input.PositionOrdinal != 1) {
			return nil, nil, invalidPositionCashOutBatch(fmt.Errorf("batch item order is invalid"))
		}
		if input.WalletOrdinal == lastWalletOrdinal && !lastCreatedAt.IsZero() &&
			input.PositionCreatedAt.UTC().After(lastCreatedAt) {
			return nil, nil, invalidPositionCashOutBatch(fmt.Errorf("positions must be ordered newest first"))
		}
		if err := validatePositionCashOutPublicKey(input.PositionPubkey, "position pubkey", true); err != nil {
			return nil, nil, invalidPositionCashOutBatch(err)
		}
		if _, exists := seenPubkeys[input.PositionPubkey]; exists {
			return nil, nil, invalidPositionCashOutBatch(fmt.Errorf("position pubkeys must be unique"))
		}
		seenPubkeys[input.PositionPubkey] = struct{}{}
		if err := validatePositionCashOutPublicKey(input.PositionRequestPubkey, "position request pubkey", false); err != nil {
			return nil, nil, invalidPositionCashOutBatch(err)
		}
		if err := validatePositionCashOutPublicKey(input.MarketConditionID, "market condition ID", true); err != nil {
			return nil, nil, invalidPositionCashOutBatch(err)
		}
		input.MarketTitle = strings.TrimSpace(input.MarketTitle)
		if input.MarketTitle == "" || utf8.RuneCountInString(input.MarketTitle) > maxPositionCashOutBatchTextLength {
			return nil, nil, invalidPositionCashOutBatch(fmt.Errorf("market title is invalid"))
		}
		if input.PositionCreatedAt.IsZero() || !input.PositionCreatedAt.UTC().After(time.Unix(0, 0).UTC()) {
			return nil, nil, invalidPositionCashOutBatch(fmt.Errorf("position creation time is invalid"))
		}
		input.PositionCreatedAt = input.PositionCreatedAt.UTC()
		if len(input.Shares) == 0 || len(input.Shares) > maxPositionCashOutDecimalLength ||
			!positionCashOutPositiveDecimalPattern.MatchString(input.Shares) {
			return nil, nil, invalidPositionCashOutBatch(fmt.Errorf("position shares are invalid"))
		}
		shares, ok := new(big.Rat).SetString(input.Shares)
		if !ok || shares.Sign() <= 0 {
			return nil, nil, invalidPositionCashOutBatch(fmt.Errorf("position shares are invalid"))
		}
		providerState, err := normalizePositionCashOutCode(input.ProviderState, false)
		if err != nil {
			return nil, nil, invalidPositionCashOutBatch(err)
		}
		input.ProviderState = providerState
		result[index] = input
		counts[input.WalletOrdinal]++
		lastWalletOrdinal = input.WalletOrdinal
		lastPositionOrdinal = input.PositionOrdinal
		lastCreatedAt = input.PositionCreatedAt
	}
	if len(counts) != len(walletRows) {
		return nil, nil, invalidPositionCashOutBatch(fmt.Errorf("every selected Wallet must contain an Open Position"))
	}
	return result, counts, nil
}

func normalizePositionCashOutBatchClaim(
	req PositionCashOutBatchClaimRequest,
) (pgtype.UUID, pgtype.UUID, string, time.Time, error) {
	batchID, _, err := marketCombinationUUID(req.BatchID, "position Cash Out batch ID")
	if err != nil {
		return pgtype.UUID{}, pgtype.UUID{}, "", time.Time{}, invalidPositionCashOutBatch(err)
	}
	claimID, _, err := marketCombinationUUID(req.ClaimID, "claim ID")
	if err != nil {
		return pgtype.UUID{}, pgtype.UUID{}, "", time.Time{}, invalidPositionCashOutBatch(err)
	}
	workerID, err := normalizePositionCashOutCode(req.WorkerID, true)
	if err != nil {
		return pgtype.UUID{}, pgtype.UUID{}, "", time.Time{}, invalidPositionCashOutBatch(err)
	}
	now := canonicalNow(req.Now)
	if !req.LeaseExpiresAt.UTC().After(now) {
		return pgtype.UUID{}, pgtype.UUID{}, "", time.Time{}, invalidPositionCashOutBatch(fmt.Errorf("claim expiry must be in the future"))
	}
	return batchID, claimID, workerID, now, nil
}

func normalizePositionCashOutBatchItemClaim(
	batchID string,
	itemID string,
	claimID string,
	now time.Time,
) (pgtype.UUID, pgtype.UUID, pgtype.UUID, time.Time, error) {
	batchUUID, _, err := marketCombinationUUID(batchID, "position Cash Out batch ID")
	if err != nil {
		return pgtype.UUID{}, pgtype.UUID{}, pgtype.UUID{}, time.Time{}, invalidPositionCashOutBatch(err)
	}
	itemUUID, _, err := marketCombinationUUID(itemID, "position Cash Out batch item ID")
	if err != nil {
		return pgtype.UUID{}, pgtype.UUID{}, pgtype.UUID{}, time.Time{}, invalidPositionCashOutBatch(err)
	}
	claimUUID, _, err := marketCombinationUUID(claimID, "claim ID")
	if err != nil {
		return pgtype.UUID{}, pgtype.UUID{}, pgtype.UUID{}, time.Time{}, invalidPositionCashOutBatch(err)
	}
	return batchUUID, itemUUID, claimUUID, canonicalNow(now), nil
}

func normalizePositionCashOutBatchBalance(
	evidence PositionCashOutBatchBalanceEvidence,
) (PositionCashOutBatchBalanceEvidence, error) {
	if err := validatePositionCashOutPublicKey(evidence.Mint, "USDC mint", true); err != nil {
		return PositionCashOutBatchBalanceEvidence{}, invalidPositionCashOutBatch(err)
	}
	if evidence.Mint != positionCashOutBatchUSDCMint || evidence.Decimals != 6 ||
		evidence.ObservedSlot == 0 || evidence.ObservedSlot > uint64(^uint64(0)>>1) ||
		len(evidence.AtomicAmount) == 0 || len(evidence.AtomicAmount) > maxPositionCashOutDecimalLength ||
		strings.TrimLeft(evidence.AtomicAmount, "0") != evidence.AtomicAmount && evidence.AtomicAmount != "0" {
		return PositionCashOutBatchBalanceEvidence{}, invalidPositionCashOutBatch(fmt.Errorf("USDC balance evidence is invalid"))
	}
	amount, ok := new(big.Int).SetString(evidence.AtomicAmount, 10)
	if !ok || amount.Sign() < 0 {
		return PositionCashOutBatchBalanceEvidence{}, invalidPositionCashOutBatch(fmt.Errorf("USDC balance evidence is invalid"))
	}
	return evidence, nil
}

func validPositionCashOutBatchControlKind(kind PositionCashOutBatchCommandKind) bool {
	switch kind {
	case PositionCashOutBatchCommandKindCancel,
		PositionCashOutBatchCommandKindPause,
		PositionCashOutBatchCommandKindContinue,
		PositionCashOutBatchCommandKindTerminate,
		PositionCashOutBatchCommandKindCheckStatus:
		return true
	default:
		return false
	}
}

func positionCashOutBatchControlTransition(
	row wormtradingsqlc.WormPositionCashOutBatch,
	kind PositionCashOutBatchCommandKind,
	now time.Time,
) (PositionCashOutBatchState, string, time.Time, time.Time, bool, error) {
	state := PositionCashOutBatchState(row.State)
	switch kind {
	case PositionCashOutBatchCommandKindCancel:
		if state != PositionCashOutBatchStateBuilding && state != PositionCashOutBatchStateAwaitingAuthorization {
			return "", "", time.Time{}, time.Time{}, false, ErrPositionCashOutBatchRevision
		}
		return PositionCashOutBatchStateCancelled, "", time.Time{}, time.Time{}, true, nil
	case PositionCashOutBatchCommandKindPause:
		if state != PositionCashOutBatchStateRunning && state != PositionCashOutBatchStateQueued {
			return "", "", time.Time{}, time.Time{}, false, ErrPositionCashOutBatchRevision
		}
		if row.CurrentItemOrdinal.Valid {
			return PositionCashOutBatchStatePauseRequested, "", timestampValue(row.NextPollAt), time.Time{}, false, nil
		}
		return PositionCashOutBatchStatePaused, "", time.Time{}, time.Time{}, false, nil
	case PositionCashOutBatchCommandKindContinue:
		if state != PositionCashOutBatchStatePaused || row.CurrentItemOrdinal.Valid {
			return "", "", time.Time{}, time.Time{}, false, ErrPositionCashOutBatchRevision
		}
		return PositionCashOutBatchStateQueued, "", now, time.Time{}, false, nil
	case PositionCashOutBatchCommandKindTerminate:
		if state != PositionCashOutBatchStateQueued && state != PositionCashOutBatchStateRunning &&
			state != PositionCashOutBatchStatePauseRequested && state != PositionCashOutBatchStatePaused &&
			state != PositionCashOutBatchStateReconciliationRequired && state != PositionCashOutBatchStateTerminateRequested {
			return "", "", time.Time{}, time.Time{}, false, ErrPositionCashOutBatchRevision
		}
		if row.CurrentItemOrdinal.Valid {
			return PositionCashOutBatchStateTerminateRequested, row.ReasonCode,
				timestampValue(row.NextPollAt), time.Time{}, false, nil
		}
		return PositionCashOutBatchStateTerminated, "", time.Time{}, time.Time{}, true, nil
	case PositionCashOutBatchCommandKindCheckStatus:
		if (state != PositionCashOutBatchStatePaused &&
			state != PositionCashOutBatchStateReconciliationRequired &&
			state != PositionCashOutBatchStateTerminateRequested) || !row.CurrentItemOrdinal.Valid {
			return "", "", time.Time{}, time.Time{}, false, ErrPositionCashOutBatchRevision
		}
		return state, row.ReasonCode, now, now, false, nil
	default:
		return "", "", time.Time{}, time.Time{}, false, ErrPositionCashOutBatchRevision
	}
}

func validPositionCashOutBatchItemTransition(
	expected PositionCashOutBatchItemState,
	next PositionCashOutBatchItemState,
) bool {
	switch expected {
	case PositionCashOutBatchItemStatePreflighting:
		return next == PositionCashOutBatchItemStateClosing ||
			next == PositionCashOutBatchItemStateAwaitingPosition
	case PositionCashOutBatchItemStateClosing:
		return next == PositionCashOutBatchItemStateClosing ||
			next == PositionCashOutBatchItemStateAwaitingPosition ||
			next == PositionCashOutBatchItemStateAwaitingBalance
	case PositionCashOutBatchItemStateAwaitingPosition:
		return next == PositionCashOutBatchItemStateAwaitingPosition ||
			next == PositionCashOutBatchItemStateAwaitingBalance
	case PositionCashOutBatchItemStateReconciliationRequired:
		return next == PositionCashOutBatchItemStateAwaitingBalance
	default:
		return false
	}
}

func invalidPositionCashOutBatch(err error) error {
	return fmt.Errorf("%w: %v", ErrInvalidPositionCashOutBatch, err)
}
