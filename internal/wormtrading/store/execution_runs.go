package store

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	wormtradingsqlc "github.com/useryege/athena/internal/wormtrading/store/sqlc"
)

const (
	executionPlanDigestVersion     = int64(4)
	executionCoordinatorTokenBytes = 32
	executionCoordinatorLease      = 30 * time.Second
	maxExecutionRunPageSize        = 100
	maxExecutionRunCodeLength      = 100
)

func (s *SQLStore) CreateExecutionRun(
	ctx context.Context,
	req CreateExecutionRunRequest,
) (*ExecutionRun, error) {
	ownerUUID, _, err := marketCombinationUUID(req.OwnerAccountID, "owner account ID")
	if err != nil {
		return nil, invalidExecutionRun(err)
	}
	planID, _, err := marketCombinationUUID(req.PlanID, "execution plan ID")
	if err != nil {
		return nil, invalidExecutionRun(err)
	}
	_, commandIDText, err := marketCombinationUUID(req.CommandID, "command ID")
	if err != nil {
		return nil, invalidExecutionRun(err)
	}
	if req.ExpectedRevision <= 0 {
		return nil, invalidExecutionRun(fmt.Errorf("expected revision must be positive"))
	}
	if err := s.requireDatabase(); err != nil {
		return nil, err
	}

	now := canonicalNow(req.Now)
	idempotencyDigest := sha256.Sum256([]byte(commandIDText))
	requestDigest, err := executionRequestDigest(struct {
		OwnerAccountID   string
		PlanID           string
		CommandID        string
		ExpectedRevision int64
	}{req.OwnerAccountID, req.PlanID, commandIDText, req.ExpectedRevision})
	if err != nil {
		return nil, err
	}

	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, fmt.Errorf("begin create execution run transaction: %w", err)
	}
	defer rollbackWalletTransaction(tx)
	queries := wormtradingsqlc.New(tx)

	existing, err := queries.GetExecutionRunByCreationKey(ctx, wormtradingsqlc.GetExecutionRunByCreationKeyParams{
		OwnerAccountID:       ownerUUID,
		IdempotencyKeySha256: idempotencyDigest[:],
	})
	if err == nil {
		if !bytes.Equal(existing.RequestSha256, requestDigest) {
			return nil, ErrExecutionRunCommandConflict
		}
		run, err := loadExecutionRun(ctx, queries, existing)
		if err != nil {
			return nil, err
		}
		if err := tx.Commit(ctx); err != nil {
			return nil, fmt.Errorf("commit replayed execution run creation: %w", err)
		}
		return &run, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("get execution run creation key: %w", err)
	}

	planRow, err := queries.GetExecutionRunSourcePlanForUpdate(ctx, wormtradingsqlc.GetExecutionRunSourcePlanForUpdateParams{
		PlanID:           planID,
		OwnerAccountID:   ownerUUID,
		Now:              timestampParam(now),
		ExpectedRevision: req.ExpectedRevision,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrExecutionRunConflict
	}
	if err != nil {
		return nil, fmt.Errorf("lock execution run source plan: %w", err)
	}
	plan, err := loadExecutionPlan(ctx, queries, planRow, now)
	if err != nil {
		return nil, err
	}
	walletIDs, err := queries.ListExecutionRunSourcePlanWalletIDs(ctx, planID)
	if err != nil {
		return nil, fmt.Errorf("list execution Run source Wallet IDs: %w", err)
	}
	if int64(len(walletIDs)) != planRow.WalletCount {
		return nil, ErrExecutionRunConflict
	}
	sort.Slice(walletIDs, func(left, right int) bool { return walletIDs[left] < walletIDs[right] })
	for index, walletID := range walletIDs {
		if walletID <= 0 || (index > 0 && walletIDs[index-1] == walletID) {
			return nil, ErrExecutionRunConflict
		}
		if _, err := tx.Exec(ctx, "SELECT pg_advisory_xact_lock($1::bigint)", walletID); err != nil {
			return nil, fmt.Errorf("lock execution Run Wallet %d: %w", walletID, err)
		}
	}
	if count, err := queries.CountActivePositionCashOutsForWallets(ctx, walletIDs); err != nil {
		return nil, fmt.Errorf("count active position Cash Outs for execution Run: %w", err)
	} else if count != 0 {
		return nil, ErrExecutionRunWalletCashOutActive
	}
	stepRows, err := queries.ListExecutionRunSourcePlanSteps(ctx, planID)
	if err != nil {
		return nil, fmt.Errorf("list execution run source steps: %w", err)
	}
	steps := make([]ExecutionPlanStep, 0, len(stepRows))
	var nextStepOrdinal int64
	var skippedStepCount int64
	for _, stepRow := range stepRows {
		step := mapExecutionPlanStep(stepRow)
		steps = append(steps, step)
		if nextStepOrdinal == 0 && step.Disposition == ExecutionPlanStepDispositionReady {
			nextStepOrdinal = step.Ordinal
		}
		if step.Disposition == ExecutionPlanStepDispositionSkipped {
			skippedStepCount++
		}
	}
	if nextStepOrdinal == 0 || int64(len(steps)) != plan.TotalStepCount {
		return nil, ErrExecutionRunConflict
	}
	planDigest, err := executionPlanDigest(plan, steps)
	if err != nil {
		return nil, err
	}

	runUUID := uuid.New()
	runID := pgtype.UUID{Bytes: [16]byte(runUUID), Valid: true}
	row, err := queries.CreateExecutionRun(ctx, wormtradingsqlc.CreateExecutionRunParams{
		ID:                   runID,
		OwnerAccountID:       ownerUUID,
		PlanID:               planID,
		PlanVersion:          executionPlanDigestVersion,
		PlanDigestSha256:     planDigest,
		IdempotencyKeySha256: idempotencyDigest[:],
		RequestSha256:        requestDigest,
		CombinationID:        planRow.CombinationID,
		CombinationName:      planRow.CombinationName,
		CombinationRevision:  planRow.CombinationRevision,
		NextStepOrdinal:      nextStepOrdinal,
		WalletCount:          planRow.WalletCount,
		ItemCount:            planRow.ItemCount,
		TotalStepCount:       planRow.TotalStepCount,
		ActionableStepCount:  planRow.ReadyStepCount,
		SatisfiedStepCount:   0,
		SkippedStepCount:     skippedStepCount,
		Now:                  timestampParam(now),
	})
	if err != nil {
		if executionConstraint(err, "worm_execution_runs_one_nonterminal_per_owner_idx") ||
			executionConstraint(err, "worm_execution_runs_plan_unique") {
			return nil, ErrExecutionRunConflict
		}
		return nil, fmt.Errorf("create execution run: %w", err)
	}
	if count, err := queries.SnapshotExecutionRunWallets(ctx, wormtradingsqlc.SnapshotExecutionRunWalletsParams{
		RunID:  runID,
		PlanID: planID,
	}); err != nil || count != planRow.WalletCount {
		if err != nil {
			return nil, fmt.Errorf("snapshot execution run wallets: %w", err)
		}
		return nil, ErrExecutionRunConflict
	}
	if count, err := queries.SnapshotExecutionRunItems(ctx, wormtradingsqlc.SnapshotExecutionRunItemsParams{
		RunID:  runID,
		PlanID: planID,
	}); err != nil || count != planRow.ItemCount {
		if err != nil {
			return nil, fmt.Errorf("snapshot execution run items: %w", err)
		}
		return nil, ErrExecutionRunConflict
	}
	if count, err := queries.SnapshotExecutionRunSteps(ctx, wormtradingsqlc.SnapshotExecutionRunStepsParams{
		RunID:  runID,
		PlanID: planID,
		Now:    timestampParam(now),
	}); err != nil || count != planRow.TotalStepCount {
		if err != nil {
			return nil, fmt.Errorf("snapshot execution run steps: %w", err)
		}
		return nil, ErrExecutionRunConflict
	}
	invalidWallets, err := queries.CountInvalidExecutionRunWalletSnapshots(ctx, runID)
	if err != nil {
		return nil, fmt.Errorf("validate execution run wallet snapshots: %w", err)
	}
	if invalidWallets != 0 {
		return nil, ErrExecutionPlanWalletConnectionChanged
	}
	blockedPairs, err := queries.CountBlockedExecutionRunStepPairs(ctx, runID)
	if err != nil {
		return nil, fmt.Errorf("validate execution run isolated pairs: %w", err)
	}
	if blockedPairs != 0 {
		return nil, ErrExecutionRunIsolation
	}
	if err := queries.CreateExecutionCombinationLock(ctx, wormtradingsqlc.CreateExecutionCombinationLockParams{
		CombinationID:       planRow.CombinationID,
		RunID:               runID,
		CombinationRevision: planRow.CombinationRevision,
		Now:                 timestampParam(now),
	}); err != nil {
		if executionConstraint(err, "worm_execution_combination_locks_pkey") {
			return nil, ErrExecutionRunConflict
		}
		return nil, fmt.Errorf("lock execution combination: %w", err)
	}
	if count, err := queries.CreateExecutionWalletLocks(ctx, wormtradingsqlc.CreateExecutionWalletLocksParams{
		RunID: runID,
		Now:   timestampParam(now),
	}); err != nil || count != planRow.WalletCount {
		if err != nil {
			if executionConstraint(err, "worm_execution_wallet_locks_pkey") {
				return nil, ErrExecutionRunConflict
			}
			return nil, fmt.Errorf("lock execution wallets: %w", err)
		}
		return nil, ErrExecutionRunConflict
	}

	run, err := loadExecutionRun(ctx, queries, row)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit create execution run: %w", err)
	}
	return &run, nil
}

func (s *SQLStore) GetExecutionRun(
	ctx context.Context,
	ownerAccountID string,
	runID string,
) (*ExecutionRun, error) {
	ownerUUID, _, runUUID, err := executionRunIDs(ownerAccountID, runID)
	if err != nil {
		return nil, err
	}
	if err := s.requireDatabase(); err != nil {
		return nil, err
	}
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if err != nil {
		return nil, fmt.Errorf("begin get execution run transaction: %w", err)
	}
	defer rollbackWalletTransaction(tx)
	queries := wormtradingsqlc.New(tx)
	row, err := queries.GetExecutionRun(ctx, wormtradingsqlc.GetExecutionRunParams{ID: runUUID, OwnerAccountID: ownerUUID})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrExecutionRunNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get execution run: %w", err)
	}
	run, err := loadExecutionRun(ctx, queries, row)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit get execution run: %w", err)
	}
	return &run, nil
}

func (s *SQLStore) ListExecutionRuns(
	ctx context.Context,
	ownerAccountID string,
	page int32,
	pageSize int32,
) ([]ExecutionRun, int64, error) {
	ownerUUID, _, err := marketCombinationUUID(ownerAccountID, "owner account ID")
	if err != nil {
		return nil, 0, invalidExecutionRun(err)
	}
	pageOffset, err := executionPageOffset(page, pageSize)
	if err != nil {
		return nil, 0, err
	}
	if err := s.requireDatabase(); err != nil {
		return nil, 0, err
	}
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if err != nil {
		return nil, 0, fmt.Errorf("begin list execution runs transaction: %w", err)
	}
	defer rollbackWalletTransaction(tx)
	queries := wormtradingsqlc.New(tx)
	total, err := queries.CountExecutionRuns(ctx, ownerUUID)
	if err != nil {
		return nil, 0, fmt.Errorf("count execution runs: %w", err)
	}
	rows, err := queries.ListExecutionRuns(ctx, wormtradingsqlc.ListExecutionRunsParams{
		OwnerAccountID: ownerUUID,
		PageSize:       pageSize,
		PageOffset:     pageOffset,
	})
	if err != nil {
		return nil, 0, fmt.Errorf("list execution runs: %w", err)
	}
	runs := make([]ExecutionRun, 0, len(rows))
	for _, row := range rows {
		run, err := loadExecutionRun(ctx, queries, row)
		if err != nil {
			return nil, 0, err
		}
		runs = append(runs, run)
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, 0, fmt.Errorf("commit list execution runs: %w", err)
	}
	return runs, total, nil
}

func (s *SQLStore) ListExecutionRunSteps(
	ctx context.Context,
	ownerAccountID string,
	runID string,
	page int32,
	pageSize int32,
) ([]ExecutionRunStep, int64, error) {
	ownerUUID, _, runUUID, err := executionRunIDs(ownerAccountID, runID)
	if err != nil {
		return nil, 0, err
	}
	pageOffset, err := executionPageOffset(page, pageSize)
	if err != nil {
		return nil, 0, err
	}
	if err := s.requireDatabase(); err != nil {
		return nil, 0, err
	}
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if err != nil {
		return nil, 0, fmt.Errorf("begin list execution run steps transaction: %w", err)
	}
	defer rollbackWalletTransaction(tx)
	queries := wormtradingsqlc.New(tx)
	total, err := queries.CountExecutionRunSteps(ctx, wormtradingsqlc.CountExecutionRunStepsParams{
		RunID: runUUID, OwnerAccountID: ownerUUID,
	})
	if err != nil {
		return nil, 0, fmt.Errorf("count execution run steps: %w", err)
	}
	if total == 0 {
		if _, err := queries.GetExecutionRun(ctx, wormtradingsqlc.GetExecutionRunParams{ID: runUUID, OwnerAccountID: ownerUUID}); errors.Is(err, pgx.ErrNoRows) {
			return nil, 0, ErrExecutionRunNotFound
		} else if err != nil {
			return nil, 0, fmt.Errorf("get execution run for empty step list: %w", err)
		}
	}
	rows, err := queries.ListExecutionRunSteps(ctx, wormtradingsqlc.ListExecutionRunStepsParams{
		RunID: runUUID, OwnerAccountID: ownerUUID, PageSize: pageSize, PageOffset: pageOffset,
	})
	if err != nil {
		return nil, 0, fmt.Errorf("list execution run steps: %w", err)
	}
	steps := make([]ExecutionRunStep, 0, len(rows))
	for _, row := range rows {
		step, err := loadExecutionRunStep(ctx, queries, row)
		if err != nil {
			return nil, 0, err
		}
		steps = append(steps, step)
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, 0, fmt.Errorf("commit list execution run steps: %w", err)
	}
	return steps, total, nil
}

func (s *SQLStore) AuthorizeExecutionRun(
	ctx context.Context,
	req AuthorizeExecutionRunRequest,
) (*ExecutionRun, error) {
	ownerUUID, _, runUUID, commandUUID, err := executionCommandIDs(req.ExecutionRunCommandRequest)
	if err != nil {
		return nil, err
	}
	proofKind, err := normalizeExecutionCode(req.ProofKind, true)
	if err != nil {
		return nil, err
	}
	if len(req.SessionJTIDigest) != sha256.Size || req.AccessRevision <= 0 {
		return nil, invalidExecutionRun(fmt.Errorf("authorization binding is invalid"))
	}
	now := canonicalNow(req.Now)
	requestDigest, err := executionRequestDigest(struct {
		RunID            string
		CommandID        string
		ExpectedRevision int64
		ProofKind        string
		SessionJTIDigest []byte
		AccessRevision   int64
	}{req.RunID, req.CommandID, req.ExpectedRevision, proofKind, req.SessionJTIDigest, req.AccessRevision})
	if err != nil {
		return nil, err
	}
	tx, queries, row, replay, err := s.beginExecutionRunCommand(ctx, ownerUUID, runUUID, commandUUID,
		req.ExpectedRevision, ExecutionCommandKindAuthorize, requestDigest, nil, now)
	if err != nil {
		return nil, err
	}
	defer rollbackWalletTransaction(tx)
	if replay {
		run, err := loadExecutionRun(ctx, queries, row)
		if err != nil {
			return nil, err
		}
		if err := tx.Commit(ctx); err != nil {
			return nil, fmt.Errorf("commit replayed authorize execution run: %w", err)
		}
		return &run, nil
	}
	if _, err := queries.SupersedeExecutionAuthorization(ctx, wormtradingsqlc.SupersedeExecutionAuthorizationParams{
		RunID: runUUID, Now: timestampParam(now),
	}); err != nil {
		return nil, fmt.Errorf("supersede execution authorization: %w", err)
	}
	authorizationID := uuid.New()
	if _, err := queries.CreateExecutionAuthorization(ctx, wormtradingsqlc.CreateExecutionAuthorizationParams{
		ID: pgtype.UUID{Bytes: [16]byte(authorizationID), Valid: true}, RunID: runUUID,
		ProofKind: proofKind, SessionJtiDigest: append([]byte(nil), req.SessionJTIDigest...),
		AccessRevision: req.AccessRevision, Now: timestampParam(now),
	}); err != nil {
		return nil, fmt.Errorf("create execution authorization: %w", err)
	}
	updated, err := queries.AuthorizeExecutionRun(ctx, wormtradingsqlc.AuthorizeExecutionRunParams{
		Now: timestampParam(now), ID: runUUID, OwnerAccountID: ownerUUID,
		ExpectedRevision: req.ExpectedRevision,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrExecutionRunRevision
	}
	if err != nil {
		return nil, fmt.Errorf("authorize execution run: %w", err)
	}
	if err := completeExecutionCommand(ctx, queries, commandUUID, updated.Revision, "AUTHORIZED", now); err != nil {
		return nil, err
	}
	run, err := loadExecutionRun(ctx, queries, updated)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit authorize execution run: %w", err)
	}
	return &run, nil
}

func (s *SQLStore) StartExecutionRun(
	ctx context.Context,
	req StartExecutionRunRequest,
) (*ExecutionCoordinatorLease, error) {
	return s.startOrResumeExecutionRun(ctx, req, false)
}

func (s *SQLStore) ResumeExecutionRun(
	ctx context.Context,
	req ResumeExecutionRunRequest,
) (*ExecutionCoordinatorLease, error) {
	return s.startOrResumeExecutionRun(ctx, StartExecutionRunRequest(req), true)
}

func (s *SQLStore) startOrResumeExecutionRun(
	ctx context.Context,
	req StartExecutionRunRequest,
	resume bool,
) (*ExecutionCoordinatorLease, error) {
	ownerUUID, _, runUUID, commandUUID, err := executionCommandIDs(req.ExecutionRunCommandRequest)
	if err != nil {
		return nil, err
	}
	if len(req.SessionJTIDigest) != sha256.Size || req.AccessRevision <= 0 {
		return nil, invalidExecutionRun(fmt.Errorf("coordinator binding is invalid"))
	}
	kind := ExecutionCommandKindStart
	if resume {
		kind = ExecutionCommandKindResume
	}
	now := canonicalNow(req.Now)
	requestDigest, err := executionRequestDigest(struct {
		RunID            string
		CommandID        string
		ExpectedRevision int64
		SessionJTIDigest []byte
		AccessRevision   int64
		Kind             ExecutionCommandKind
	}{req.RunID, req.CommandID, req.ExpectedRevision, req.SessionJTIDigest, req.AccessRevision, kind})
	if err != nil {
		return nil, err
	}
	tx, queries, current, replay, err := s.beginExecutionRunCommand(ctx, ownerUUID, runUUID, commandUUID,
		req.ExpectedRevision, kind, requestDigest, nil, now)
	if err != nil {
		return nil, err
	}
	defer rollbackWalletTransaction(tx)
	var updated wormtradingsqlc.WormExecutionRun
	if replay {
		if current.State != string(ExecutionRunStateRunning) {
			return nil, ErrExecutionRunCoordinator
		}
		authorization, authErr := queries.GetActiveExecutionAuthorization(ctx, runUUID)
		if authErr != nil || !bytes.Equal(authorization.SessionJtiDigest, req.SessionJTIDigest) ||
			authorization.AccessRevision != req.AccessRevision || authorization.PlanVersion != current.PlanVersion ||
			!bytes.Equal(authorization.PlanDigestSha256, current.PlanDigestSha256) {
			return nil, ErrExecutionRunAuthorization
		}
		updated = current
	} else if resume {
		updated, err = queries.ResumeExecutionRun(ctx, wormtradingsqlc.ResumeExecutionRunParams{
			Now: timestampParam(now), ID: runUUID, OwnerAccountID: ownerUUID,
			ExpectedRevision: req.ExpectedRevision, SessionJtiDigest: req.SessionJTIDigest,
			AccessRevision: req.AccessRevision,
		})
	} else {
		updated, err = queries.StartExecutionRun(ctx, wormtradingsqlc.StartExecutionRunParams{
			Now: timestampParam(now), ID: runUUID, OwnerAccountID: ownerUUID,
			ExpectedRevision: req.ExpectedRevision, SessionJtiDigest: req.SessionJTIDigest,
			AccessRevision: req.AccessRevision,
		})
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrExecutionRunRevision
	}
	if err != nil {
		return nil, fmt.Errorf("%s execution run: %w", strings.ToLower(string(kind)), err)
	}
	if _, err := queries.ReleaseExecutionCoordinator(ctx, wormtradingsqlc.ReleaseExecutionCoordinatorParams{
		RunID: runUUID, Now: timestampParam(now),
	}); err != nil {
		return nil, fmt.Errorf("release prior execution coordinator: %w", err)
	}
	token, tokenDigest, err := newExecutionCoordinatorToken()
	if err != nil {
		return nil, err
	}
	coordinatorID := uuid.New()
	createdCoordinator, err := queries.CreateExecutionCoordinator(ctx, wormtradingsqlc.CreateExecutionCoordinatorParams{
		ID: pgtype.UUID{Bytes: [16]byte(coordinatorID), Valid: true}, RunID: runUUID,
		TokenSha256: tokenDigest, SessionJtiDigest: append([]byte(nil), req.SessionJTIDigest...),
		AccessRevision: req.AccessRevision, Now: timestampParam(now),
		LeaseExpiresAt: timestampParam(now.Add(executionCoordinatorLease)),
	})
	if err != nil {
		return nil, fmt.Errorf("create execution coordinator: %w", err)
	}
	if !replay {
		if _, err := queries.BindExecutionCommandCoordinator(ctx, wormtradingsqlc.BindExecutionCommandCoordinatorParams{
			CoordinatorID: createdCoordinator.ID, CoordinatorGeneration: createdCoordinator.Generation,
			Now: timestampParam(now), ID: commandUUID, RunID: runUUID,
		}); err != nil {
			return nil, fmt.Errorf("bind start execution coordinator: %w", err)
		}
		if err := completeExecutionCommand(ctx, queries, commandUUID, updated.Revision, string(kind), now); err != nil {
			return nil, err
		}
	}
	run, err := loadExecutionRun(ctx, queries, updated)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit %s execution run: %w", strings.ToLower(string(kind)), err)
	}
	return &ExecutionCoordinatorLease{Run: run, Token: token}, nil
}

func (s *SQLStore) PauseExecutionRun(
	ctx context.Context,
	req ExecutionRunCommandRequest,
) (*ExecutionRun, error) {
	ownerUUID, _, runUUID, commandUUID, err := executionCommandIDs(req)
	if err != nil {
		return nil, err
	}
	now := canonicalNow(req.Now)
	requestDigest, err := executionRequestDigest(struct {
		OwnerAccountID   string
		RunID            string
		CommandID        string
		ExpectedRevision int64
	}{req.OwnerAccountID, req.RunID, req.CommandID, req.ExpectedRevision})
	if err != nil {
		return nil, err
	}
	tx, queries, row, replay, err := s.beginExecutionRunCommand(ctx, ownerUUID, runUUID, commandUUID,
		req.ExpectedRevision, ExecutionCommandKindPause, requestDigest, nil, now)
	if err != nil {
		return nil, err
	}
	defer rollbackWalletTransaction(tx)
	if !replay {
		row, err = queries.RequestExecutionRunPause(ctx, wormtradingsqlc.RequestExecutionRunPauseParams{
			PauseCode: "USER_REQUESTED", Now: timestampParam(now), ID: runUUID,
			OwnerAccountID: ownerUUID, ExpectedRevision: req.ExpectedRevision,
		})
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrExecutionRunRevision
		}
		if err != nil {
			return nil, fmt.Errorf("pause execution run: %w", err)
		}
		if row.State == string(ExecutionRunStatePaused) {
			if _, err := queries.ReleaseExecutionCoordinator(ctx, wormtradingsqlc.ReleaseExecutionCoordinatorParams{
				RunID: runUUID, Now: timestampParam(now),
			}); err != nil {
				return nil, fmt.Errorf("release paused execution coordinator: %w", err)
			}
		}
		if err := completeExecutionCommand(ctx, queries, commandUUID, row.Revision, "PAUSE_REQUESTED", now); err != nil {
			return nil, err
		}
	}
	run, err := loadExecutionRun(ctx, queries, row)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit pause execution run: %w", err)
	}
	return &run, nil
}

func (s *SQLStore) TerminateExecutionRun(
	ctx context.Context,
	req ExecutionRunCommandRequest,
) (*ExecutionRun, error) {
	ownerUUID, _, runUUID, commandUUID, err := executionCommandIDs(req)
	if err != nil {
		return nil, err
	}
	now := canonicalNow(req.Now)
	requestDigest, err := executionRequestDigest(struct {
		OwnerAccountID   string
		RunID            string
		CommandID        string
		ExpectedRevision int64
	}{req.OwnerAccountID, req.RunID, req.CommandID, req.ExpectedRevision})
	if err != nil {
		return nil, err
	}
	tx, queries, row, replay, err := s.beginExecutionRunCommand(ctx, ownerUUID, runUUID, commandUUID,
		req.ExpectedRevision, ExecutionCommandKindTerminate, requestDigest, nil, now)
	if err != nil {
		return nil, err
	}
	defer rollbackWalletTransaction(tx)
	if !replay {
		row, err = queries.RequestExecutionRunTermination(ctx, wormtradingsqlc.RequestExecutionRunTerminationParams{
			Now: timestampParam(now), ID: runUUID, OwnerAccountID: ownerUUID,
			ExpectedRevision: req.ExpectedRevision,
		})
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrExecutionRunRevision
		}
		if err != nil {
			return nil, fmt.Errorf("terminate execution run: %w", err)
		}
		if row.State == string(ExecutionRunStateTerminated) {
			if _, err := queries.MarkPendingExecutionRunStepsNotExecuted(ctx, wormtradingsqlc.MarkPendingExecutionRunStepsNotExecutedParams{
				ReasonCode: "RUN_TERMINATED", Now: timestampParam(now), RunID: runUUID,
			}); err != nil {
				return nil, fmt.Errorf("mark terminated execution steps: %w", err)
			}
			row, err = queries.RefreshExecutionRunProgress(ctx, wormtradingsqlc.RefreshExecutionRunProgressParams{
				ID: runUUID, Now: timestampParam(now),
			})
			if err != nil {
				return nil, fmt.Errorf("refresh terminated execution run: %w", err)
			}
			if err := closeTerminalExecutionRun(ctx, queries, runUUID, "TERMINATED", now); err != nil {
				return nil, err
			}
		}
		if err := completeExecutionCommand(ctx, queries, commandUUID, row.Revision, string(row.State), now); err != nil {
			return nil, err
		}
	}
	run, err := loadExecutionRun(ctx, queries, row)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit terminate execution run: %w", err)
	}
	return &run, nil
}

func (s *SQLStore) HeartbeatExecutionCoordinator(
	ctx context.Context,
	req HeartbeatExecutionCoordinatorRequest,
) (*ExecutionRun, error) {
	ownerUUID, _, runUUID, commandUUID, err := executionCommandIDs(req.ExecutionRunCommandRequest)
	if err != nil {
		return nil, err
	}
	if len(req.CoordinatorToken) != executionCoordinatorTokenBytes ||
		len(req.SessionJTIDigest) != sha256.Size || req.AccessRevision <= 0 {
		return nil, invalidExecutionRun(fmt.Errorf("coordinator heartbeat binding is invalid"))
	}
	now := canonicalNow(req.Now)
	tokenDigest := sha256.Sum256(req.CoordinatorToken)
	requestDigest, err := executionRequestDigest(struct {
		OwnerAccountID         string
		RunID                  string
		CommandID              string
		ExpectedRevision       int64
		CoordinatorTokenSHA256 []byte
		SessionJTIDigest       []byte
		AccessRevision         int64
	}{req.OwnerAccountID, req.RunID, req.CommandID, req.ExpectedRevision,
		tokenDigest[:], req.SessionJTIDigest, req.AccessRevision})
	if err != nil {
		return nil, err
	}
	tx, queries, row, replay, err := s.beginExecutionRunCommand(ctx, ownerUUID, runUUID, commandUUID,
		req.ExpectedRevision, ExecutionCommandKindHeartbeat, requestDigest, nil, now)
	if err != nil {
		return nil, err
	}
	defer rollbackWalletTransaction(tx)
	if !replay {
		coordinator, err := queries.GetActiveExecutionCoordinator(ctx, runUUID)
		if err != nil || !bytes.Equal(coordinator.TokenSha256, tokenDigest[:]) ||
			!timestampValue(coordinator.LeaseExpiresAt).After(now) {
			return nil, ErrExecutionRunCoordinator
		}
		if _, err := queries.BindExecutionCommandCoordinator(ctx, wormtradingsqlc.BindExecutionCommandCoordinatorParams{
			CoordinatorID: coordinator.ID, CoordinatorGeneration: coordinator.Generation,
			Now: timestampParam(now), ID: commandUUID, RunID: runUUID,
		}); err != nil {
			return nil, fmt.Errorf("bind heartbeat execution coordinator: %w", err)
		}
		authorization, authorizationErr := queries.GetActiveExecutionAuthorization(ctx, runUUID)
		bindingChanged := !bytes.Equal(coordinator.SessionJtiDigest, req.SessionJTIDigest) ||
			coordinator.AccessRevision != req.AccessRevision || errors.Is(authorizationErr, pgx.ErrNoRows)
		if authorizationErr == nil {
			bindingChanged = bindingChanged ||
				!bytes.Equal(authorization.SessionJtiDigest, req.SessionJTIDigest) ||
				authorization.AccessRevision != req.AccessRevision ||
				authorization.PlanVersion != row.PlanVersion ||
				!bytes.Equal(authorization.PlanDigestSha256, row.PlanDigestSha256)
		} else if !errors.Is(authorizationErr, pgx.ErrNoRows) {
			return nil, fmt.Errorf("get heartbeat execution authorization: %w", authorizationErr)
		}
		if bindingChanged {
			row, err = pauseExecutionRunForReauthorization(ctx, queries, runUUID, now)
			if err != nil {
				return nil, err
			}
			if err := completeExecutionCommand(ctx, queries, commandUUID, row.Revision,
				"AUTHORIZATION_REQUIRED", now); err != nil {
				return nil, err
			}
			run, err := loadExecutionRun(ctx, queries, row)
			if err != nil {
				return nil, err
			}
			if err := tx.Commit(ctx); err != nil {
				return nil, fmt.Errorf("commit authorization-required heartbeat: %w", err)
			}
			return &run, nil
		}
		if _, err := queries.HeartbeatExecutionCoordinator(ctx, wormtradingsqlc.HeartbeatExecutionCoordinatorParams{
			Now: timestampParam(now), LeaseExpiresAt: timestampParam(now.Add(executionCoordinatorLease)),
			RunID: runUUID, TokenSha256: tokenDigest[:],
			SessionJtiDigest: append([]byte(nil), req.SessionJTIDigest...),
			AccessRevision:   req.AccessRevision,
		}); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return nil, ErrExecutionRunCoordinator
			}
			return nil, fmt.Errorf("heartbeat execution coordinator: %w", err)
		}
		row, err = queries.TouchExecutionRunRevision(ctx, wormtradingsqlc.TouchExecutionRunRevisionParams{
			Now: timestampParam(now), ID: runUUID,
		})
		if err != nil {
			return nil, fmt.Errorf("advance execution run heartbeat revision: %w", err)
		}
		if err := completeExecutionCommand(ctx, queries, commandUUID, row.Revision, "HEARTBEAT", now); err != nil {
			return nil, err
		}
	}
	run, err := loadExecutionRun(ctx, queries, row)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit heartbeat execution coordinator: %w", err)
	}
	return &run, nil
}

func (s *SQLStore) beginExecutionRunCommand(
	ctx context.Context,
	ownerUUID pgtype.UUID,
	runUUID pgtype.UUID,
	commandUUID pgtype.UUID,
	expectedRevision int64,
	kind ExecutionCommandKind,
	requestDigest []byte,
	stepOrdinal *int64,
	now time.Time,
) (pgx.Tx, *wormtradingsqlc.Queries, wormtradingsqlc.WormExecutionRun, bool, error) {
	if expectedRevision <= 0 || len(requestDigest) != sha256.Size {
		return nil, nil, wormtradingsqlc.WormExecutionRun{}, false,
			invalidExecutionRun(fmt.Errorf("execution command revision or digest is invalid"))
	}
	if err := s.requireDatabase(); err != nil {
		return nil, nil, wormtradingsqlc.WormExecutionRun{}, false, err
	}
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, nil, wormtradingsqlc.WormExecutionRun{}, false,
			fmt.Errorf("begin execution command transaction: %w", err)
	}
	queries := wormtradingsqlc.New(tx)
	row, err := queries.GetExecutionRunForOwnerUpdate(ctx, wormtradingsqlc.GetExecutionRunForOwnerUpdateParams{
		ID: runUUID, OwnerAccountID: ownerUUID,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		rollbackWalletTransaction(tx)
		return nil, nil, wormtradingsqlc.WormExecutionRun{}, false, ErrExecutionRunNotFound
	}
	if err != nil {
		rollbackWalletTransaction(tx)
		return nil, nil, wormtradingsqlc.WormExecutionRun{}, false,
			fmt.Errorf("lock execution run command: %w", err)
	}
	existing, err := queries.GetExecutionCommand(ctx, commandUUID)
	if err == nil {
		if existing.RunID != runUUID || existing.Kind != string(kind) ||
			!bytes.Equal(existing.RequestSha256, requestDigest) ||
			(existing.State != string(ExecutionCommandStateApplied) &&
				existing.State != string(ExecutionCommandStateInProgress)) {
			rollbackWalletTransaction(tx)
			return nil, nil, wormtradingsqlc.WormExecutionRun{}, false, ErrExecutionRunCommandConflict
		}
		return tx, queries, row, true, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		rollbackWalletTransaction(tx)
		return nil, nil, wormtradingsqlc.WormExecutionRun{}, false,
			fmt.Errorf("get execution command: %w", err)
	}
	stepParam := pgtype.Int8{}
	if stepOrdinal != nil {
		stepParam = pgtype.Int8{Int64: *stepOrdinal, Valid: true}
	}
	if _, err := queries.CreateExecutionCommand(ctx, wormtradingsqlc.CreateExecutionCommandParams{
		ID: commandUUID, RunID: runUUID, CoordinatorID: pgtype.UUID{},
		CoordinatorGeneration: pgtype.Int8{}, Kind: string(kind), RequestSha256: requestDigest,
		RunRevisionBefore: row.Revision, StepOrdinal: stepParam, Now: timestampParam(now),
	}); err != nil {
		rollbackWalletTransaction(tx)
		return nil, nil, wormtradingsqlc.WormExecutionRun{}, false,
			fmt.Errorf("create execution command: %w", err)
	}
	if _, err := queries.BeginExecutionCommand(ctx, wormtradingsqlc.BeginExecutionCommandParams{
		Now: timestampParam(now), ID: commandUUID,
	}); err != nil {
		rollbackWalletTransaction(tx)
		return nil, nil, wormtradingsqlc.WormExecutionRun{}, false,
			fmt.Errorf("begin execution command: %w", err)
	}
	return tx, queries, row, false, nil
}

func completeExecutionCommand(
	ctx context.Context,
	queries *wormtradingsqlc.Queries,
	commandID pgtype.UUID,
	runRevision int64,
	resultCode string,
	now time.Time,
) error {
	if _, err := queries.CompleteExecutionCommand(ctx, wormtradingsqlc.CompleteExecutionCommandParams{
		State: string(ExecutionCommandStateApplied), RunRevisionAfter: runRevision,
		ResultCode: resultCode, Now: timestampParam(now), ID: commandID,
	}); err != nil {
		return fmt.Errorf("complete execution command: %w", err)
	}
	return nil
}

func pauseExecutionRunForReauthorization(
	ctx context.Context,
	queries *wormtradingsqlc.Queries,
	runID pgtype.UUID,
	now time.Time,
) (wormtradingsqlc.WormExecutionRun, error) {
	if _, err := queries.InvalidateExecutionAuthorization(ctx,
		wormtradingsqlc.InvalidateExecutionAuthorizationParams{
			Now: timestampParam(now), RunID: runID,
		}); err != nil {
		return wormtradingsqlc.WormExecutionRun{}, fmt.Errorf("invalidate execution authorization: %w", err)
	}
	row, err := queries.PauseExecutionRunForReauthorization(ctx,
		wormtradingsqlc.PauseExecutionRunForReauthorizationParams{
			Now: timestampParam(now), ID: runID,
		})
	if errors.Is(err, pgx.ErrNoRows) {
		return wormtradingsqlc.WormExecutionRun{}, ErrExecutionRunAuthorization
	}
	if err != nil {
		return wormtradingsqlc.WormExecutionRun{}, fmt.Errorf("pause execution run for reauthorization: %w", err)
	}
	if _, err := queries.ReleaseExecutionCoordinator(ctx, wormtradingsqlc.ReleaseExecutionCoordinatorParams{
		RunID: runID, Now: timestampParam(now),
	}); err != nil {
		return wormtradingsqlc.WormExecutionRun{}, fmt.Errorf("release reauthorization coordinator: %w", err)
	}
	return row, nil
}

func executionCommandIDs(req ExecutionRunCommandRequest) (pgtype.UUID, string, pgtype.UUID, pgtype.UUID, error) {
	ownerUUID, ownerID, runUUID, err := executionRunIDs(req.OwnerAccountID, req.RunID)
	if err != nil {
		return pgtype.UUID{}, "", pgtype.UUID{}, pgtype.UUID{}, err
	}
	commandUUID, _, err := marketCombinationUUID(req.CommandID, "execution command ID")
	if err != nil {
		return pgtype.UUID{}, "", pgtype.UUID{}, pgtype.UUID{}, invalidExecutionRun(err)
	}
	if req.ExpectedRevision <= 0 {
		return pgtype.UUID{}, "", pgtype.UUID{}, pgtype.UUID{}, invalidExecutionRun(fmt.Errorf("expected revision must be positive"))
	}
	return ownerUUID, ownerID, runUUID, commandUUID, nil
}

func closeTerminalExecutionRun(
	ctx context.Context,
	queries *wormtradingsqlc.Queries,
	runID pgtype.UUID,
	reasonCode string,
	now time.Time,
) error {
	if err := queries.ReleaseExecutionRunLocks(ctx, runID); err != nil {
		return fmt.Errorf("release terminal execution locks: %w", err)
	}
	if _, err := queries.ReleaseExecutionCoordinator(ctx, wormtradingsqlc.ReleaseExecutionCoordinatorParams{
		RunID: runID, Now: timestampParam(now),
	}); err != nil {
		return fmt.Errorf("release terminal execution coordinator: %w", err)
	}
	if _, err := queries.EndExecutionRunAuthorization(ctx, wormtradingsqlc.EndExecutionRunAuthorizationParams{
		Now: timestampParam(now), ReasonCode: reasonCode, RunID: runID,
	}); err != nil {
		return fmt.Errorf("end terminal execution authorization: %w", err)
	}
	return nil
}

func (s *SQLStore) BeginExecutionStep(
	ctx context.Context,
	req BeginExecutionStepRequest,
) (*ExecutionRunStep, error) {
	return s.ClaimExecutionStep(ctx, ExecutionStepClaimRequest{
		OwnerAccountID: req.OwnerAccountID, RunID: req.RunID, CommandID: req.CommandID,
		ExpectedRunRevision: req.ExpectedRevision, ExpectedStepOrdinal: req.ExpectedStepOrdinal,
		CoordinatorToken: req.CoordinatorToken, SessionJTIDigest: req.SessionJTIDigest,
		AccessRevision: req.AccessRevision, LeaseExpiresAt: canonicalNow(req.Now).Add(executionCoordinatorLease),
		Now: req.Now,
	})
}

func (s *SQLStore) ClaimExecutionStep(
	ctx context.Context,
	req ExecutionStepClaimRequest,
) (*ExecutionRunStep, error) {
	return s.claimExecutionStep(ctx, req)
}

func (s *SQLStore) RecoverExecutionStep(
	ctx context.Context,
	req RecoverExecutionStepRequest,
) (*ExecutionRunStep, error) {
	runID, _, err := marketCombinationUUID(req.RunID, "execution run ID")
	if err != nil {
		return nil, invalidExecutionRun(err)
	}
	claimID, _, err := marketCombinationUUID(req.ClaimID, "execution recovery claim ID")
	if err != nil || req.StepOrdinal <= 0 {
		return nil, invalidExecutionRun(fmt.Errorf("execution recovery claim identity is invalid"))
	}
	workerID := strings.TrimSpace(req.WorkerID)
	if workerID == "" || len(workerID) > 200 {
		return nil, invalidExecutionRun(fmt.Errorf("execution recovery worker ID is invalid"))
	}
	now := canonicalNow(req.Now)
	if !req.LeaseExpiresAt.After(now) {
		return nil, invalidExecutionRun(fmt.Errorf("execution recovery lease must be in the future"))
	}
	if err := s.requireDatabase(); err != nil {
		return nil, err
	}
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, fmt.Errorf("begin recover execution step transaction: %w", err)
	}
	defer rollbackWalletTransaction(tx)
	queries := wormtradingsqlc.New(tx)
	row, err := queries.ClaimExecutionRunStepForRecovery(ctx,
		wormtradingsqlc.ClaimExecutionRunStepForRecoveryParams{
			ClaimID: claimID, ClaimOwner: workerID, ClaimExpiresAt: timestampParam(req.LeaseExpiresAt),
			Now: timestampParam(now), RunID: runID, StepOrdinal: req.StepOrdinal,
		})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrExecutionRunConflict
	}
	if err != nil {
		return nil, fmt.Errorf("recover execution step: %w", err)
	}
	step, err := loadExecutionRunStep(ctx, queries, row)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit recover execution step: %w", err)
	}
	return &step, nil
}

func (s *SQLStore) RenewExecutionStepClaim(
	ctx context.Context,
	req RenewExecutionStepClaimRequest,
) (*ExecutionRunStep, error) {
	runID, _, err := marketCombinationUUID(req.RunID, "execution run ID")
	claimID, _, claimErr := marketCombinationUUID(req.ClaimID, "execution claim ID")
	workerID := strings.TrimSpace(req.WorkerID)
	now := canonicalNow(req.Now)
	if err != nil || claimErr != nil || req.StepOrdinal <= 0 || workerID == "" ||
		len(workerID) > 200 || !req.LeaseExpiresAt.After(now) {
		return nil, invalidExecutionRun(fmt.Errorf("execution claim renewal is invalid"))
	}
	if err := s.requireDatabase(); err != nil {
		return nil, err
	}
	row, err := s.queries.RenewExecutionRunStepClaim(ctx, wormtradingsqlc.RenewExecutionRunStepClaimParams{
		ClaimExpiresAt: timestampParam(req.LeaseExpiresAt), Now: timestampParam(now),
		RunID: runID, StepOrdinal: req.StepOrdinal, ClaimID: claimID, ClaimOwner: workerID,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrExecutionRunConflict
	}
	if err != nil {
		return nil, fmt.Errorf("renew execution step claim: %w", err)
	}
	step, err := loadExecutionRunStep(ctx, s.queries, row)
	if err != nil {
		return nil, err
	}
	return &step, nil
}

func (s *SQLStore) claimExecutionStep(
	ctx context.Context,
	req ExecutionStepClaimRequest,
) (*ExecutionRunStep, error) {
	commandReq := ExecutionRunCommandRequest{
		OwnerAccountID: req.OwnerAccountID, RunID: req.RunID, CommandID: req.CommandID,
		ExpectedRevision: req.ExpectedRunRevision, Now: req.Now,
	}
	ownerUUID, _, runUUID, commandUUID, err := executionCommandIDs(commandReq)
	if err != nil {
		return nil, err
	}
	if req.ExpectedStepOrdinal <= 0 || len(req.CoordinatorToken) != executionCoordinatorTokenBytes ||
		len(req.SessionJTIDigest) != sha256.Size || req.AccessRevision <= 0 {
		return nil, invalidExecutionRun(fmt.Errorf("execution step claim binding is invalid"))
	}
	now := canonicalNow(req.Now)
	leaseExpiresAt := req.LeaseExpiresAt.UTC()
	if !leaseExpiresAt.After(now) {
		return nil, invalidExecutionRun(fmt.Errorf("execution step claim lease must be in the future"))
	}
	tokenDigest := sha256.Sum256(req.CoordinatorToken)
	requestDigest, err := executionRequestDigest(struct {
		RunID                  string
		CommandID              string
		ExpectedRunRevision    int64
		ExpectedStepOrdinal    int64
		CoordinatorTokenSHA256 []byte
		SessionJTIDigest       []byte
		AccessRevision         int64
	}{req.RunID, req.CommandID, req.ExpectedRunRevision, req.ExpectedStepOrdinal,
		tokenDigest[:], req.SessionJTIDigest, req.AccessRevision})
	if err != nil {
		return nil, err
	}
	stepOrdinal := req.ExpectedStepOrdinal
	tx, queries, runRow, replay, err := s.beginExecutionRunCommand(ctx, ownerUUID, runUUID, commandUUID,
		req.ExpectedRunRevision, ExecutionCommandKindExecuteNext, requestDigest, &stepOrdinal, now)
	if err != nil {
		return nil, err
	}
	defer rollbackWalletTransaction(tx)
	if replay {
		command, commandErr := queries.GetExecutionCommand(ctx, commandUUID)
		if commandErr != nil {
			return nil, fmt.Errorf("get replayed execution step command: %w", commandErr)
		}
		if command.ResultCode == "AUTHORIZATION_REQUIRED" {
			if err := tx.Commit(ctx); err != nil {
				return nil, fmt.Errorf("commit replayed authorization-required step: %w", err)
			}
			return nil, ErrExecutionRunAuthorization
		}
		row, err := queries.GetExecutionRunStep(ctx, wormtradingsqlc.GetExecutionRunStepParams{
			RunID: runUUID, Ordinal: stepOrdinal,
		})
		if err != nil {
			return nil, fmt.Errorf("get replayed execution step claim: %w", err)
		}
		step, err := loadExecutionRunStep(ctx, queries, row)
		if err != nil {
			return nil, err
		}
		if err := tx.Commit(ctx); err != nil {
			return nil, fmt.Errorf("commit replayed execution step claim: %w", err)
		}
		return &step, nil
	}
	coordinator, err := queries.GetActiveExecutionCoordinator(ctx, runUUID)
	if err != nil || !bytes.Equal(coordinator.TokenSha256, tokenDigest[:]) ||
		!timestampValue(coordinator.LeaseExpiresAt).After(now) {
		return nil, ErrExecutionRunCoordinator
	}
	if _, err := queries.BindExecutionCommandCoordinator(ctx, wormtradingsqlc.BindExecutionCommandCoordinatorParams{
		CoordinatorID: coordinator.ID, CoordinatorGeneration: coordinator.Generation,
		Now: timestampParam(now), ID: commandUUID, RunID: runUUID,
	}); err != nil {
		return nil, fmt.Errorf("bind execution command coordinator: %w", err)
	}
	authorization, authorizationErr := queries.GetActiveExecutionAuthorization(ctx, runUUID)
	bindingChanged := !bytes.Equal(coordinator.SessionJtiDigest, req.SessionJTIDigest) ||
		coordinator.AccessRevision != req.AccessRevision || errors.Is(authorizationErr, pgx.ErrNoRows)
	if authorizationErr == nil {
		bindingChanged = bindingChanged ||
			!bytes.Equal(authorization.SessionJtiDigest, req.SessionJTIDigest) ||
			authorization.AccessRevision != req.AccessRevision ||
			authorization.PlanVersion != runRow.PlanVersion ||
			!bytes.Equal(authorization.PlanDigestSha256, runRow.PlanDigestSha256)
	} else if !errors.Is(authorizationErr, pgx.ErrNoRows) {
		return nil, fmt.Errorf("get execution step authorization: %w", authorizationErr)
	}
	if bindingChanged {
		runRow, err = pauseExecutionRunForReauthorization(ctx, queries, runUUID, now)
		if err != nil {
			return nil, err
		}
		if err := completeExecutionCommand(ctx, queries, commandUUID, runRow.Revision,
			"AUTHORIZATION_REQUIRED", now); err != nil {
			return nil, err
		}
		if err := tx.Commit(ctx); err != nil {
			return nil, fmt.Errorf("commit authorization-required execution step: %w", err)
		}
		return nil, ErrExecutionRunAuthorization
	}
	claimParams := wormtradingsqlc.ClaimFreshExecutionRunStepParams{
		CommandID: commandUUID, ClaimExpiresAt: timestampParam(leaseExpiresAt), Now: timestampParam(now),
		RunID: runUUID, StepOrdinal: stepOrdinal, ExpectedRunRevision: req.ExpectedRunRevision,
		CoordinatorTokenSha256: tokenDigest[:], SessionJtiDigest: req.SessionJTIDigest,
		AccessRevision: req.AccessRevision,
	}
	row, err := queries.ClaimFreshExecutionRunStep(ctx, claimParams)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrExecutionRunRevision
	}
	if err != nil {
		return nil, fmt.Errorf("claim execution step: %w", err)
	}
	if _, err := queries.SetExecutionRunCurrentStep(ctx, wormtradingsqlc.SetExecutionRunCurrentStepParams{
		CurrentStepOrdinal: pgtype.Int8{Int64: stepOrdinal, Valid: true},
		Now:                timestampParam(now), ID: runUUID,
	}); err != nil {
		return nil, fmt.Errorf("set execution run current step: %w", err)
	}
	step, err := loadExecutionRunStep(ctx, queries, row)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit execution step claim: %w", err)
	}
	return &step, nil
}

func (s *SQLStore) BeginExecutionStepReconciliation(
	ctx context.Context,
	req ReconcileExecutionStepRequest,
) (*ExecutionRunStep, error) {
	ownerUUID, _, runUUID, commandUUID, err := executionCommandIDs(req.ExecutionRunCommandRequest)
	if err != nil {
		return nil, err
	}
	stepID, _, err := marketCombinationUUID(req.StepID, "execution step ID")
	if err != nil || req.ExpectedStepOrdinal <= 0 {
		return nil, invalidExecutionRun(fmt.Errorf("execution reconciliation step is invalid"))
	}
	now := canonicalNow(req.Now)
	requestDigest, err := executionRequestDigest(struct {
		RunID               string
		CommandID           string
		ExpectedRevision    int64
		StepID              string
		ExpectedStepOrdinal int64
	}{req.RunID, req.CommandID, req.ExpectedRevision, req.StepID, req.ExpectedStepOrdinal})
	if err != nil {
		return nil, err
	}
	ordinal := req.ExpectedStepOrdinal
	tx, queries, runRow, replay, err := s.beginExecutionRunCommand(ctx, ownerUUID, runUUID, commandUUID,
		req.ExpectedRevision, ExecutionCommandKindReconcile, requestDigest, &ordinal, now)
	if err != nil {
		return nil, err
	}
	defer rollbackWalletTransaction(tx)
	if !replay && runRow.State != string(ExecutionRunStateReconciliationRequired) &&
		!(runRow.State == string(ExecutionRunStateTerminated) && runRow.BlockCode != "") {
		return nil, ErrExecutionRunConflict
	}
	var row wormtradingsqlc.WormExecutionRunStep
	if replay {
		row, err = queries.GetExecutionRunStepByID(ctx, wormtradingsqlc.GetExecutionRunStepByIDParams{
			RunID: runUUID, ID: stepID,
		})
	} else {
		row, err = queries.RequestExecutionRunStepReconciliation(ctx,
			wormtradingsqlc.RequestExecutionRunStepReconciliationParams{
				CommandID: commandUUID, Now: timestampParam(now), RunID: runUUID,
				StepID: stepID, StepOrdinal: ordinal,
			})
	}
	if errors.Is(err, pgx.ErrNoRows) || row.Ordinal != ordinal {
		return nil, ErrExecutionRunConflict
	}
	if err != nil {
		return nil, fmt.Errorf("request execution step reconciliation: %w", err)
	}
	step, err := loadExecutionRunStep(ctx, queries, row)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit begin execution reconciliation: %w", err)
	}
	return &step, nil
}

func (s *SQLStore) CompleteExecutionPreflight(
	ctx context.Context,
	req CompleteExecutionPreflightRequest,
) (*ExecutionRunStep, error) {
	runID, _, err := marketCombinationUUID(req.RunID, "execution run ID")
	if err != nil {
		return nil, invalidExecutionRun(err)
	}
	commandID, _, err := marketCombinationUUID(req.CommandID, "execution command ID")
	claimID, _, claimErr := marketCombinationUUID(req.ClaimID, "execution claim ID")
	if err != nil || claimErr != nil || req.StepOrdinal <= 0 {
		return nil, invalidExecutionRun(fmt.Errorf("execution preflight identity is invalid"))
	}
	switch req.NextState {
	case ExecutionStepStateOpening:
		if req.SkipScope != "" && req.SkipScope != ExecutionStepScopeCurrent {
			return nil, invalidExecutionRun(fmt.Errorf("opening preflight cannot skip a scope"))
		}
	case ExecutionStepStateSatisfied, ExecutionStepStateSkipped, ExecutionStepStateFailed:
	default:
		return nil, invalidExecutionRun(fmt.Errorf("execution preflight next state is invalid"))
	}
	reasonCode, err := normalizeExecutionCode(req.ReasonCode, req.NextState != ExecutionStepStateOpening)
	if err != nil {
		return nil, err
	}
	now := canonicalNow(req.Now)
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, fmt.Errorf("begin complete execution preflight transaction: %w", err)
	}
	defer rollbackWalletTransaction(tx)
	queries := wormtradingsqlc.New(tx)
	prior, err := queries.GetExecutionRunStepForUpdate(ctx, wormtradingsqlc.GetExecutionRunStepForUpdateParams{
		RunID: runID, Ordinal: req.StepOrdinal,
	})
	if errors.Is(err, pgx.ErrNoRows) || prior.ClaimID != claimID || prior.ActiveCommandID != commandID {
		return nil, ErrExecutionRunConflict
	}
	if err != nil {
		return nil, fmt.Errorf("lock execution preflight step: %w", err)
	}
	row, err := queries.CompleteExecutionStepPreflight(ctx, wormtradingsqlc.CompleteExecutionStepPreflightParams{
		NextState: string(req.NextState), ReasonCode: reasonCode, Now: timestampParam(now),
		RunID: runID, StepOrdinal: req.StepOrdinal, ClaimID: claimID,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrExecutionRunConflict
	}
	if err != nil {
		return nil, fmt.Errorf("complete execution step preflight: %w", err)
	}
	if req.NextState != ExecutionStepStateOpening {
		if req.SkipScope == ExecutionStepScopeRemainingWallet || req.SkipScope == ExecutionStepScopeRemainingMarket {
			if _, err := queries.SkipScopedPendingExecutionSteps(ctx, wormtradingsqlc.SkipScopedPendingExecutionStepsParams{
				ReasonCode: reasonCode, Now: timestampParam(now), RunID: runID,
				SourceStepOrdinal: req.StepOrdinal, Scope: string(req.SkipScope),
			}); err != nil {
				return nil, fmt.Errorf("skip scoped execution steps: %w", err)
			}
		}
		runRow, err := finalizeExecutionStepProgress(ctx, queries, runID, now)
		if err != nil {
			return nil, err
		}
		if err := completeExecutionCommand(ctx, queries, commandID, runRow.Revision, string(req.NextState), now); err != nil {
			return nil, err
		}
	}
	step, err := loadExecutionRunStep(ctx, queries, row)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit complete execution preflight: %w", err)
	}
	return &step, nil
}

func (s *SQLStore) PrepareExecutionMutation(
	ctx context.Context,
	req PrepareExecutionMutationRequest,
) (*ExecutionMutationAttempt, error) {
	attemptID, _, err := marketCombinationUUID(req.AttemptID, "execution attempt ID")
	if err != nil {
		return nil, invalidExecutionRun(err)
	}
	runID, _, err := marketCombinationUUID(req.RunID, "execution run ID")
	if err != nil {
		return nil, invalidExecutionRun(err)
	}
	commandID, _, err := marketCombinationUUID(req.CommandID, "execution command ID")
	if err != nil {
		return nil, invalidExecutionRun(err)
	}
	claimID, _, err := marketCombinationUUID(req.ClaimID, "execution claim ID")
	if err != nil {
		return nil, invalidExecutionRun(err)
	}
	if req.StepOrdinal <= 0 || len(req.RequestSHA256) != sha256.Size {
		return nil, invalidExecutionRun(fmt.Errorf("execution mutation binding is invalid"))
	}
	if req.Kind != ExecutionMutationKindOpen && req.Kind != ExecutionMutationKindFinalize {
		return nil, invalidExecutionRun(fmt.Errorf("execution mutation kind is invalid"))
	}
	if req.Kind == ExecutionMutationKindFinalize && req.PositionRequestID <= 0 {
		return nil, invalidExecutionRun(fmt.Errorf("finalize requires a numeric position request ID"))
	}
	if err := s.requireDatabase(); err != nil {
		return nil, err
	}
	now := canonicalNow(req.Now)
	row, err := s.queries.CreateExecutionMutationAttempt(ctx, wormtradingsqlc.CreateExecutionMutationAttemptParams{
		ID: attemptID, RunID: runID, StepOrdinal: req.StepOrdinal,
		CommandID: commandID, ClaimID: claimID, Kind: string(req.Kind),
		RequestSha256:     append([]byte(nil), req.RequestSHA256...),
		PositionRequestID: nullableBigintParam(req.PositionRequestID), Now: timestampParam(now),
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrExecutionRunConflict
	}
	if err != nil {
		if executionConstraint(err, "worm_execution_mutation_attempts_one_kind_per_step") {
			existing, getErr := s.queries.GetExecutionMutationAttempt(ctx, wormtradingsqlc.GetExecutionMutationAttemptParams{
				RunID: runID, StepOrdinal: req.StepOrdinal, Kind: string(req.Kind),
			})
			if getErr == nil && bytes.Equal(existing.RequestSha256, req.RequestSHA256) {
				attempt := mapExecutionMutationAttempt(existing)
				return &attempt, nil
			}
			return nil, ErrExecutionRunCommandConflict
		}
		return nil, fmt.Errorf("prepare execution mutation: %w", err)
	}
	attempt := mapExecutionMutationAttempt(row)
	return &attempt, nil
}

func (s *SQLStore) DispatchExecutionMutation(
	ctx context.Context,
	req DispatchExecutionMutationRequest,
) (*ExecutionMutationAttempt, error) {
	id, _, err := marketCombinationUUID(req.AttemptID, "execution attempt ID")
	runID, _, runErr := marketCombinationUUID(req.RunID, "execution run ID")
	claimID, _, claimErr := marketCombinationUUID(req.ClaimID, "execution claim ID")
	if err != nil || runErr != nil || claimErr != nil || req.StepOrdinal <= 0 {
		return nil, invalidExecutionRun(fmt.Errorf("execution mutation dispatch binding is invalid"))
	}
	if err := s.requireDatabase(); err != nil {
		return nil, err
	}
	row, err := s.queries.DispatchExecutionMutationAttempt(ctx, wormtradingsqlc.DispatchExecutionMutationAttemptParams{
		Now: timestampParam(canonicalNow(req.Now)), ID: id, RunID: runID,
		StepOrdinal: req.StepOrdinal, ClaimID: claimID,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrExecutionRunConflict
	}
	if err != nil {
		return nil, fmt.Errorf("dispatch execution mutation: %w", err)
	}
	attempt := mapExecutionMutationAttempt(row)
	return &attempt, nil
}

func (s *SQLStore) ResolveExecutionMutation(
	ctx context.Context,
	req ResolveExecutionMutationRequest,
) (*ExecutionMutationAttempt, error) {
	id, _, err := marketCombinationUUID(req.AttemptID, "execution attempt ID")
	if err != nil {
		return nil, invalidExecutionRun(err)
	}
	switch req.State {
	case ExecutionMutationStateSucceeded,
		ExecutionMutationStateDefiniteFailure,
		ExecutionMutationStateOutcomeUnknown:
	default:
		return nil, invalidExecutionRun(fmt.Errorf("execution mutation result state is invalid"))
	}
	providerSlug, err := normalizeExecutionCode(req.ProviderSlug, false)
	if err != nil {
		return nil, err
	}
	errorCode, err := normalizeExecutionCode(req.ErrorCode, req.State != ExecutionMutationStateSucceeded)
	if err != nil {
		return nil, err
	}
	if req.HTTPStatus < 0 || req.HTTPStatus > 599 {
		return nil, invalidExecutionRun(fmt.Errorf("execution mutation HTTP status is invalid"))
	}
	if err := s.requireDatabase(); err != nil {
		return nil, err
	}
	attemptRow, err := s.queries.GetExecutionMutationAttemptByID(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrExecutionRunConflict
	}
	if err != nil {
		return nil, fmt.Errorf("get execution mutation result target: %w", err)
	}
	if attemptRow.Kind == string(ExecutionMutationKindOpen) && req.State == ExecutionMutationStateSucceeded {
		return nil, invalidExecutionRun(fmt.Errorf("successful open must be recorded atomically with its durable transaction digest"))
	}
	row, err := s.queries.CompleteExecutionMutationAttempt(ctx, wormtradingsqlc.CompleteExecutionMutationAttemptParams{
		State: string(req.State), PositionRequestID: nullableBigintParam(req.PositionRequestID),
		HttpStatus: nullableIntegerParam(req.HTTPStatus), ProviderCode: nullableIntegerParam(req.ProviderCode),
		ProviderSlug: providerSlug, ErrorCode: errorCode, Now: timestampParam(canonicalNow(req.Now)), ID: id,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrExecutionRunConflict
	}
	if err != nil {
		return nil, fmt.Errorf("resolve execution mutation: %w", err)
	}
	attempt := mapExecutionMutationAttempt(row)
	return &attempt, nil
}

func (s *SQLStore) RecordExecutionStepOpened(
	ctx context.Context,
	req RecordExecutionStepOpenedRequest,
) (*ExecutionRunStep, error) {
	attemptID, _, err := marketCombinationUUID(req.AttemptID, "execution attempt ID")
	if err != nil {
		return nil, invalidExecutionRun(err)
	}
	runID, _, err := marketCombinationUUID(req.RunID, "execution run ID")
	if err != nil {
		return nil, invalidExecutionRun(err)
	}
	claimID, _, err := marketCombinationUUID(req.ClaimID, "execution claim ID")
	if err != nil || req.StepOrdinal <= 0 || req.PositionRequestID <= 0 ||
		len(req.TransactionMessageSHA256) != sha256.Size {
		return nil, invalidExecutionRun(fmt.Errorf("opened execution step metadata is invalid"))
	}
	providerState, err := normalizeExecutionCode(req.ProviderState, false)
	if err != nil {
		return nil, err
	}
	providerOrderState, err := normalizeExecutionCode(req.ProviderOrderState, false)
	if err != nil {
		return nil, err
	}
	providerSlug, err := normalizeExecutionCode(req.ProviderSlug, false)
	if err != nil {
		return nil, err
	}
	if req.HTTPStatus < 200 || req.HTTPStatus > 299 {
		return nil, invalidExecutionRun(fmt.Errorf("successful open HTTP status is invalid"))
	}
	if err := s.requireDatabase(); err != nil {
		return nil, err
	}
	now := canonicalNow(req.Now)
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, fmt.Errorf("begin record opened execution step transaction: %w", err)
	}
	defer rollbackWalletTransaction(tx)
	queries := wormtradingsqlc.New(tx)
	prior, err := queries.GetExecutionRunStepForUpdate(ctx, wormtradingsqlc.GetExecutionRunStepForUpdateParams{
		RunID: runID, Ordinal: req.StepOrdinal,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrExecutionRunConflict
	}
	if err != nil {
		return nil, fmt.Errorf("lock opened execution step: %w", err)
	}
	attempt, err := queries.GetExecutionMutationAttemptByID(ctx, attemptID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrExecutionRunConflict
	}
	if err != nil {
		return nil, fmt.Errorf("get opened execution attempt: %w", err)
	}
	if attempt.RunID != runID || attempt.StepOrdinal != req.StepOrdinal ||
		attempt.Kind != string(ExecutionMutationKindOpen) {
		return nil, ErrExecutionRunConflict
	}
	if prior.State == string(ExecutionStepStateOpened) {
		if nullableInt64(prior.PositionRequestID) != req.PositionRequestID ||
			!bytes.Equal(prior.TransactionMessageSha256, req.TransactionMessageSHA256) ||
			prior.ProviderState != providerState || prior.ProviderOrderState != providerOrderState ||
			prior.ClaimID != claimID ||
			attempt.State != string(ExecutionMutationStateSucceeded) ||
			nullableInt64(attempt.PositionRequestID) != req.PositionRequestID {
			return nil, ErrExecutionRunCommandConflict
		}
		step, err := loadExecutionRunStep(ctx, queries, prior)
		if err != nil {
			return nil, err
		}
		if err := tx.Commit(ctx); err != nil {
			return nil, fmt.Errorf("commit replayed opened execution step: %w", err)
		}
		return &step, nil
	}
	if prior.State != string(ExecutionStepStateOpening) || prior.ClaimID != claimID {
		return nil, ErrExecutionRunConflict
	}
	if attempt.State == string(ExecutionMutationStateDispatched) {
		attempt, err = queries.CompleteExecutionMutationAttempt(ctx,
			wormtradingsqlc.CompleteExecutionMutationAttemptParams{
				State:             string(ExecutionMutationStateSucceeded),
				PositionRequestID: nullableBigintParam(req.PositionRequestID),
				HttpStatus:        nullableIntegerParam(req.HTTPStatus), ProviderCode: nullableIntegerParam(req.ProviderCode),
				ProviderSlug: providerSlug, ErrorCode: "", Now: timestampParam(now), ID: attemptID,
			})
		if err != nil {
			return nil, fmt.Errorf("resolve successful open attempt: %w", err)
		}
	}
	if attempt.State != string(ExecutionMutationStateSucceeded) ||
		nullableInt64(attempt.PositionRequestID) != req.PositionRequestID {
		return nil, ErrExecutionRunConflict
	}
	row, err := queries.RecordExecutionRunStepOpened(ctx, wormtradingsqlc.RecordExecutionRunStepOpenedParams{
		PositionRequestID:        req.PositionRequestID,
		TransactionMessageSha256: append([]byte(nil), req.TransactionMessageSHA256...),
		ProviderState:            providerState, ProviderOrderState: providerOrderState,
		Now: timestampParam(now), RunID: runID,
		StepOrdinal: req.StepOrdinal, ClaimID: claimID,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrExecutionRunConflict
	}
	if err != nil {
		return nil, fmt.Errorf("record opened execution step: %w", err)
	}
	step, err := loadExecutionRunStep(ctx, queries, row)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit opened execution step: %w", err)
	}
	return &step, nil
}

func (s *SQLStore) MarkExecutionStepSigning(
	ctx context.Context,
	req AdvanceExecutionStepRequest,
) (*ExecutionRunStep, error) {
	runID, _, err := marketCombinationUUID(req.RunID, "execution run ID")
	claimID, _, claimErr := marketCombinationUUID(req.ClaimID, "execution claim ID")
	if err != nil || claimErr != nil || req.StepOrdinal <= 0 {
		return nil, invalidExecutionRun(fmt.Errorf("execution step identity is invalid"))
	}
	if err := s.requireDatabase(); err != nil {
		return nil, err
	}
	row, err := s.queries.MarkExecutionRunStepSigning(ctx, wormtradingsqlc.MarkExecutionRunStepSigningParams{
		Now: timestampParam(canonicalNow(req.Now)), RunID: runID,
		StepOrdinal: req.StepOrdinal, ClaimID: claimID,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrExecutionRunConflict
	}
	if err != nil {
		return nil, fmt.Errorf("advance execution signing phase: %w", err)
	}
	step := mapExecutionRunStep(row)
	return &step, nil
}

func (s *SQLStore) RecordExecutionStepSigned(
	ctx context.Context,
	req RecordExecutionStepSignedRequest,
) (*ExecutionRunStep, error) {
	runID, _, err := marketCombinationUUID(req.RunID, "execution run ID")
	claimID, _, claimErr := marketCombinationUUID(req.ClaimID, "execution claim ID")
	if err != nil || claimErr != nil || req.StepOrdinal <= 0 ||
		req.RequiredSignatureCount <= 0 || req.WalletSignerIndex < 0 ||
		req.WalletSignerIndex >= req.RequiredSignatureCount {
		return nil, invalidExecutionRun(fmt.Errorf("signed execution step metadata is invalid"))
	}
	if req.FinalizeMode != "signature" && req.FinalizeMode != "signed_transaction" {
		return nil, invalidExecutionRun(fmt.Errorf("execution finalize mode is invalid"))
	}
	version, err := normalizeExecutionCode(req.TransactionVersion, true)
	if err != nil || len(version) > 20 {
		return nil, invalidExecutionRun(fmt.Errorf("execution transaction version is invalid"))
	}
	if err := s.requireDatabase(); err != nil {
		return nil, err
	}
	row, err := s.queries.RecordExecutionRunStepSigned(ctx, wormtradingsqlc.RecordExecutionRunStepSignedParams{
		FinalizeMode: req.FinalizeMode, TransactionVersion: version,
		RequiredSignatureCount: req.RequiredSignatureCount, WalletSignerIndex: req.WalletSignerIndex,
		Now: timestampParam(canonicalNow(req.Now)), RunID: runID,
		StepOrdinal: req.StepOrdinal, ClaimID: claimID,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrExecutionRunConflict
	}
	if err != nil {
		return nil, fmt.Errorf("record signed execution step: %w", err)
	}
	step := mapExecutionRunStep(row)
	return &step, nil
}

func (s *SQLStore) RecordExecutionProviderObservation(
	ctx context.Context,
	req RecordExecutionProviderObservationRequest,
) (*ExecutionRunStep, error) {
	runID, _, err := marketCombinationUUID(req.RunID, "execution run ID")
	if err != nil || req.StepOrdinal <= 0 {
		return nil, invalidExecutionRun(fmt.Errorf("provider observation step is invalid"))
	}
	commandID, _, err := marketCombinationUUID(req.CommandID, "execution command ID")
	if err != nil {
		return nil, invalidExecutionRun(err)
	}
	claimID, _, err := marketCombinationUUID(req.ClaimID, "execution claim ID")
	if err != nil {
		return nil, invalidExecutionRun(err)
	}
	if req.ExpectedState != ExecutionStepStateOpening &&
		req.ExpectedState != ExecutionStepStateOpened &&
		req.ExpectedState != ExecutionStepStateSigning &&
		req.ExpectedState != ExecutionStepStateFinalizing &&
		req.ExpectedState != ExecutionStepStateAwaitingCompletion &&
		req.ExpectedState != ExecutionStepStateOutcomeUnknown {
		return nil, invalidExecutionRun(fmt.Errorf("provider observation expected state is invalid"))
	}
	switch req.NextState {
	case ExecutionStepStateAwaitingCompletion, ExecutionStepStateCompleted,
		ExecutionStepStateSatisfied, ExecutionStepStateSkipped, ExecutionStepStateFailed,
		ExecutionStepStateOutcomeUnknown:
	default:
		return nil, invalidExecutionRun(fmt.Errorf("provider observation next state is invalid"))
	}
	if req.ExpectedState == ExecutionStepStateOpening &&
		req.NextState != ExecutionStepStateAwaitingCompletion &&
		req.NextState != ExecutionStepStateCompleted && req.NextState != ExecutionStepStateSkipped &&
		req.NextState != ExecutionStepStateFailed && req.NextState != ExecutionStepStateOutcomeUnknown {
		return nil, invalidExecutionRun(fmt.Errorf("opening observation transition is invalid"))
	}
	if (req.ExpectedState == ExecutionStepStateOpened || req.ExpectedState == ExecutionStepStateSigning) &&
		req.NextState != ExecutionStepStateAwaitingCompletion &&
		req.NextState != ExecutionStepStateCompleted && req.NextState != ExecutionStepStateFailed &&
		req.NextState != ExecutionStepStateOutcomeUnknown {
		return nil, invalidExecutionRun(fmt.Errorf("opened or signing observation transition is invalid"))
	}
	if req.ExpectedState == ExecutionStepStateOutcomeUnknown &&
		req.NextState == ExecutionStepStateAwaitingCompletion {
		return nil, invalidExecutionRun(fmt.Errorf("reconciliation cannot return to awaiting completion"))
	}
	if req.NextState == ExecutionStepStateSatisfied &&
		req.ExpectedState != ExecutionStepStateOutcomeUnknown {
		return nil, invalidExecutionRun(fmt.Errorf("only reconciliation can satisfy an execution step"))
	}
	providerState, err := normalizeExecutionCode(req.ProviderState,
		req.NextState == ExecutionStepStateAwaitingCompletion)
	if err != nil {
		return nil, err
	}
	providerOrderState, err := normalizeExecutionCode(req.ProviderOrderState, false)
	if err != nil {
		return nil, err
	}
	reasonCode, err := normalizeExecutionCode(req.ReasonCode,
		req.NextState == ExecutionStepStateSkipped || req.NextState == ExecutionStepStateFailed ||
			req.NextState == ExecutionStepStateOutcomeUnknown)
	if err != nil {
		return nil, err
	}
	completionPositionPubkey, err := normalizeExecutionCompletionPubkey(
		req.CompletionPositionPubkey,
		req.NextState == ExecutionStepStateCompleted,
	)
	if err != nil {
		return nil, err
	}
	completionPositionRequestPubkey, err := normalizeExecutionCompletionPubkey(
		req.CompletionPositionRequestPubkey,
		false,
	)
	if err != nil {
		return nil, err
	}
	completionPositionCreatedAt := req.CompletionPositionCreatedAt.UTC()
	if req.NextState == ExecutionStepStateCompleted {
		if req.CompletionSource != ExecutionCompletionSourceOpenPosition ||
			completionPositionCreatedAt.IsZero() || completionPositionCreatedAt.Unix() <= 0 {
			return nil, invalidExecutionRun(fmt.Errorf("open-position completion evidence is invalid"))
		}
	} else if req.CompletionSource != "" || completionPositionPubkey != "" ||
		completionPositionRequestPubkey != "" || !req.CompletionPositionCreatedAt.IsZero() {
		return nil, invalidExecutionRun(fmt.Errorf("only a completed step can record completion evidence"))
	}
	if req.PositionRequestID < 0 {
		return nil, invalidExecutionRun(fmt.Errorf("provider observation position request ID is invalid"))
	}
	if req.PositionRequestID > 0 &&
		(req.ExpectedState != ExecutionStepStateOpening ||
			(req.NextState != ExecutionStepStateAwaitingCompletion &&
				req.NextState != ExecutionStepStateCompleted && req.NextState != ExecutionStepStateFailed &&
				req.NextState != ExecutionStepStateOutcomeUnknown)) {
		return nil, invalidExecutionRun(fmt.Errorf("only an authoritative or unknown Open outcome can attach a provider request ID"))
	}
	if req.ExpectedState == ExecutionStepStateOpening && req.NextState == ExecutionStepStateFailed &&
		req.PositionRequestID > 0 && !executionStoreProviderTerminalFailure(providerState) {
		return nil, invalidExecutionRun(fmt.Errorf("successful Open can fail only from an authoritative terminal provider state"))
	}
	now := canonicalNow(req.Now)
	if req.NextState == ExecutionStepStateAwaitingCompletion && !req.NextPollAt.After(now) {
		return nil, invalidExecutionRun(fmt.Errorf("awaiting completion requires a future poll time"))
	}
	if req.NextState == ExecutionStepStateOutcomeUnknown &&
		req.ExpectedState != ExecutionStepStateOutcomeUnknown &&
		(req.IsolationID == "" || req.AttemptID == "") {
		return nil, invalidExecutionRun(fmt.Errorf("outcome unknown requires durable isolation identity"))
	}
	if req.SkipScope != "" && req.SkipScope != ExecutionStepScopeCurrent &&
		req.SkipScope != ExecutionStepScopeRemainingWallet &&
		req.SkipScope != ExecutionStepScopeRemainingMarket {
		return nil, invalidExecutionRun(fmt.Errorf("provider observation skip scope is invalid"))
	}
	if req.NextState == ExecutionStepStateSkipped {
		if req.ExpectedState != ExecutionStepStateOpening ||
			req.SkipScope != ExecutionStepScopeRemainingWallet || req.PositionRequestID != 0 ||
			(reasonCode != "MARKET_POSITION_EXISTS" && reasonCode != "WALLET_REQUEST_IN_FLIGHT") {
			return nil, invalidExecutionRun(fmt.Errorf("an Open guard can only skip the remaining wallet"))
		}
	} else if req.NextState != ExecutionStepStateFailed && req.SkipScope != "" {
		return nil, invalidExecutionRun(fmt.Errorf("only a definite failure or Open guard can skip a scope"))
	}
	fundingTxID := strings.TrimSpace(req.FundingTxID)
	refundTxID := strings.TrimSpace(req.RefundTxID)
	if len(fundingTxID) > 200 || len(refundTxID) > 200 {
		return nil, invalidExecutionRun(fmt.Errorf("provider transaction identity is invalid"))
	}
	var isolationID pgtype.UUID
	var attemptID pgtype.UUID
	if req.NextState == ExecutionStepStateOutcomeUnknown &&
		req.ExpectedState != ExecutionStepStateOutcomeUnknown {
		isolationID, _, err = marketCombinationUUID(req.IsolationID, "execution isolation ID")
		if err != nil {
			return nil, invalidExecutionRun(err)
		}
		attemptID, _, err = marketCombinationUUID(req.AttemptID, "execution attempt ID")
		if err != nil {
			return nil, invalidExecutionRun(err)
		}
	}
	var resolveIsolationID pgtype.UUID
	resolutionCode := ""
	if req.ExpectedState == ExecutionStepStateOutcomeUnknown &&
		req.NextState != ExecutionStepStateOutcomeUnknown {
		resolveIsolationID, _, err = marketCombinationUUID(req.ResolveIsolationID, "execution isolation ID")
		if err != nil {
			return nil, invalidExecutionRun(err)
		}
		resolutionCode, err = normalizeExecutionCode(req.IsolationResolutionCode, true)
		if err != nil {
			return nil, err
		}
	}
	if err := s.requireDatabase(); err != nil {
		return nil, err
	}
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, fmt.Errorf("begin provider observation transaction: %w", err)
	}
	defer rollbackWalletTransaction(tx)
	queries := wormtradingsqlc.New(tx)
	runBefore, err := queries.GetExecutionRunByIDForUpdate(ctx, runID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrExecutionRunNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("lock provider observation run: %w", err)
	}
	prior, err := queries.GetExecutionRunStepForUpdate(ctx, wormtradingsqlc.GetExecutionRunStepForUpdateParams{
		RunID: runID, Ordinal: req.StepOrdinal,
	})
	if errors.Is(err, pgx.ErrNoRows) || prior.State != string(req.ExpectedState) ||
		prior.ActiveCommandID != commandID || prior.ClaimID != claimID {
		return nil, ErrExecutionRunConflict
	}
	if err != nil {
		return nil, fmt.Errorf("lock provider observation step: %w", err)
	}
	if req.PositionRequestID > 0 && nullableInt64(prior.PositionRequestID) > 0 &&
		nullableInt64(prior.PositionRequestID) != req.PositionRequestID {
		return nil, ErrExecutionRunConflict
	}
	if req.ExpectedState == ExecutionStepStateOutcomeUnknown &&
		runBefore.State != string(ExecutionRunStateReconciliationRequired) &&
		runBefore.State != string(ExecutionRunStateTerminated) {
		return nil, ErrExecutionRunConflict
	}
	row, err := queries.RecordExecutionRunProviderObservation(ctx, wormtradingsqlc.RecordExecutionRunProviderObservationParams{
		NextState: string(req.NextState), ReasonCode: reasonCode, ProviderState: providerState,
		PositionRequestID: req.PositionRequestID, ProviderOrderState: providerOrderState, FundingTxid: fundingTxID,
		RefundTxid: refundTxID, CompletionSource: string(req.CompletionSource),
		CompletionPositionPubkey:        completionPositionPubkey,
		CompletionPositionRequestPubkey: completionPositionRequestPubkey,
		CompletionPositionCreatedAt:     timestampParam(completionPositionCreatedAt), Now: timestampParam(now),
		NextPollAt: timestampParam(req.NextPollAt), RunID: runID, StepOrdinal: req.StepOrdinal,
		ExpectedState: string(req.ExpectedState), ClaimID: claimID,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrExecutionRunConflict
	}
	if err != nil {
		if executionConstraint(err, "worm_execution_run_steps_completion_position_pubkey_unique") ||
			executionConstraint(err, "worm_execution_run_steps_completion_request_pubkey_unique") {
			return nil, ErrExecutionRunConflict
		}
		return nil, fmt.Errorf("record execution provider observation: %w", err)
	}
	if req.NextState == ExecutionStepStateAwaitingCompletion {
		// The original execute-next command intentionally remains IN_PROGRESS.
		// A recovered worker can poll by using the durable command and claim binding.
	} else if req.NextState == ExecutionStepStateOutcomeUnknown {
		if req.ExpectedState != ExecutionStepStateOutcomeUnknown {
			if _, err := queries.CreateExecutionStepIsolation(ctx, wormtradingsqlc.CreateExecutionStepIsolationParams{
				ID: isolationID, AttemptID: attemptID, ReasonCode: reasonCode,
				Now: timestampParam(now), RunID: runID, StepOrdinal: req.StepOrdinal,
			}); err != nil {
				return nil, fmt.Errorf("isolate unknown execution step: %w", err)
			}
			runBefore, err = queries.MarkExecutionRunReconciliationRequired(ctx,
				wormtradingsqlc.MarkExecutionRunReconciliationRequiredParams{
					BlockCode: reasonCode, Now: timestampParam(now), ID: runID,
				})
			if err != nil {
				return nil, fmt.Errorf("block unknown execution run: %w", err)
			}
			if _, err := queries.ReleaseExecutionCoordinator(ctx, wormtradingsqlc.ReleaseExecutionCoordinatorParams{
				RunID: runID, Now: timestampParam(now),
			}); err != nil {
				return nil, fmt.Errorf("release unknown execution coordinator: %w", err)
			}
			if runBefore.State == string(ExecutionRunStateTerminated) {
				if _, err := queries.MarkPendingExecutionRunStepsNotExecuted(ctx,
					wormtradingsqlc.MarkPendingExecutionRunStepsNotExecutedParams{
						ReasonCode: "RUN_TERMINATED", Now: timestampParam(now), RunID: runID,
					}); err != nil {
					return nil, fmt.Errorf("mark unknown-terminated execution steps: %w", err)
				}
				runBefore, err = queries.RefreshExecutionRunProgress(ctx,
					wormtradingsqlc.RefreshExecutionRunProgressParams{ID: runID, Now: timestampParam(now)})
				if err != nil {
					return nil, fmt.Errorf("refresh unknown-terminated execution run: %w", err)
				}
				if err := closeTerminalExecutionRun(ctx, queries, runID, "TERMINATED", now); err != nil {
					return nil, err
				}
			}
		} else {
			runBefore, err = queries.TouchExecutionRunRevision(ctx,
				wormtradingsqlc.TouchExecutionRunRevisionParams{Now: timestampParam(now), ID: runID})
			if err != nil {
				return nil, fmt.Errorf("advance unresolved reconciliation revision: %w", err)
			}
		}
		if err := completeExecutionCommand(ctx, queries, commandID, runBefore.Revision, reasonCode, now); err != nil {
			return nil, err
		}
	} else if req.ExpectedState == ExecutionStepStateOutcomeUnknown {
		if _, err := queries.ResolveExecutionStepIsolation(ctx, wormtradingsqlc.ResolveExecutionStepIsolationParams{
			ID: resolveIsolationID, RunID: runID, StepOrdinal: req.StepOrdinal,
			ResolutionCode: resolutionCode, Now: timestampParam(now),
		}); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return nil, ErrExecutionRunIsolation
			}
			return nil, fmt.Errorf("resolve reconciled execution isolation: %w", err)
		}
		if err := skipExecutionTerminalScope(ctx, queries, runID, req.StepOrdinal,
			req.NextState, req.SkipScope, reasonCode, now); err != nil {
			return nil, err
		}
		runBefore, err = queries.RefreshExecutionRunProgress(ctx,
			wormtradingsqlc.RefreshExecutionRunProgressParams{ID: runID, Now: timestampParam(now)})
		if err != nil {
			return nil, fmt.Errorf("refresh reconciled execution progress: %w", err)
		}
		if runBefore.State == string(ExecutionRunStateTerminated) {
			runBefore, err = queries.TouchExecutionRunRevision(ctx,
				wormtradingsqlc.TouchExecutionRunRevisionParams{Now: timestampParam(now), ID: runID})
		} else if runBefore.TerminalStepCount == runBefore.TotalStepCount {
			runBefore, err = queries.CompleteReconciledExecutionRun(ctx,
				wormtradingsqlc.CompleteReconciledExecutionRunParams{Now: timestampParam(now), ID: runID})
			if err == nil {
				err = closeTerminalExecutionRun(ctx, queries, runID, "COMPLETED", now)
			}
		} else {
			runBefore, err = queries.ResolveExecutionRunReconciliation(ctx,
				wormtradingsqlc.ResolveExecutionRunReconciliationParams{Now: timestampParam(now), ID: runID})
		}
		if err != nil {
			return nil, fmt.Errorf("resolve execution run reconciliation: %w", err)
		}
		if err := completeExecutionCommand(ctx, queries, commandID, runBefore.Revision,
			string(req.NextState), now); err != nil {
			return nil, err
		}
	} else {
		if err := skipExecutionTerminalScope(ctx, queries, runID, req.StepOrdinal,
			req.NextState, req.SkipScope, reasonCode, now); err != nil {
			return nil, err
		}
		runBefore, err = finalizeExecutionStepProgress(ctx, queries, runID, now)
		if err != nil {
			return nil, err
		}
		if err := completeExecutionCommand(ctx, queries, commandID, runBefore.Revision,
			string(req.NextState), now); err != nil {
			return nil, err
		}
	}
	step, err := loadExecutionRunStep(ctx, queries, row)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit provider observation: %w", err)
	}
	return &step, nil
}

func skipExecutionTerminalScope(
	ctx context.Context,
	queries *wormtradingsqlc.Queries,
	runID pgtype.UUID,
	stepOrdinal int64,
	nextState ExecutionStepState,
	scope ExecutionStepScope,
	reasonCode string,
	now time.Time,
) error {
	failureScope := nextState == ExecutionStepStateFailed &&
		(scope == ExecutionStepScopeRemainingWallet || scope == ExecutionStepScopeRemainingMarket)
	guardScope := nextState == ExecutionStepStateSkipped && scope == ExecutionStepScopeRemainingWallet
	if !failureScope && !guardScope {
		return nil
	}
	if _, err := queries.SkipScopedPendingExecutionSteps(ctx, wormtradingsqlc.SkipScopedPendingExecutionStepsParams{
		ReasonCode: reasonCode, Now: timestampParam(now), RunID: runID,
		SourceStepOrdinal: stepOrdinal, Scope: string(scope),
	}); err != nil {
		return fmt.Errorf("skip execution scope: %w", err)
	}
	return nil
}

func finalizeExecutionStepProgress(
	ctx context.Context,
	queries *wormtradingsqlc.Queries,
	runID pgtype.UUID,
	now time.Time,
) (wormtradingsqlc.WormExecutionRun, error) {
	runRow, err := queries.RefreshExecutionRunProgress(ctx, wormtradingsqlc.RefreshExecutionRunProgressParams{
		ID: runID, Now: timestampParam(now),
	})
	if err != nil {
		return wormtradingsqlc.WormExecutionRun{}, fmt.Errorf("refresh execution step progress: %w", err)
	}
	switch ExecutionRunState(runRow.State) {
	case ExecutionRunStatePauseRequested:
		runRow, err = queries.CheckpointExecutionRunPaused(ctx, wormtradingsqlc.CheckpointExecutionRunPausedParams{
			Now: timestampParam(now), ID: runID,
		})
		if err == nil {
			_, err = queries.ReleaseExecutionCoordinator(ctx, wormtradingsqlc.ReleaseExecutionCoordinatorParams{
				RunID: runID, Now: timestampParam(now),
			})
		}
	case ExecutionRunStateTerminateRequested:
		_, err = queries.MarkPendingExecutionRunStepsNotExecuted(ctx, wormtradingsqlc.MarkPendingExecutionRunStepsNotExecutedParams{
			ReasonCode: "RUN_TERMINATED", Now: timestampParam(now), RunID: runID,
		})
		if err == nil {
			runRow, err = queries.RefreshExecutionRunProgress(ctx, wormtradingsqlc.RefreshExecutionRunProgressParams{
				ID: runID, Now: timestampParam(now),
			})
		}
		if err == nil {
			runRow, err = queries.CompleteExecutionRunTermination(ctx, wormtradingsqlc.CompleteExecutionRunTerminationParams{
				Now: timestampParam(now), ID: runID,
			})
		}
		if err == nil {
			err = closeTerminalExecutionRun(ctx, queries, runID, "TERMINATED", now)
		}
	case ExecutionRunStateRunning:
		if runRow.TerminalStepCount == runRow.TotalStepCount {
			runRow, err = queries.CompleteExecutionRun(ctx, wormtradingsqlc.CompleteExecutionRunParams{
				Now: timestampParam(now), ID: runID,
			})
			if err == nil {
				err = closeTerminalExecutionRun(ctx, queries, runID, "COMPLETED", now)
			}
		} else {
			runRow, err = queries.TouchExecutionRunRevision(ctx, wormtradingsqlc.TouchExecutionRunRevisionParams{
				Now: timestampParam(now), ID: runID,
			})
		}
	default:
		runRow, err = queries.TouchExecutionRunRevision(ctx, wormtradingsqlc.TouchExecutionRunRevisionParams{
			Now: timestampParam(now), ID: runID,
		})
	}
	if err != nil {
		return wormtradingsqlc.WormExecutionRun{}, fmt.Errorf("finalize execution step progress: %w", err)
	}
	return runRow, nil
}

func (s *SQLStore) PauseExecutionRunForFailure(
	ctx context.Context,
	req PauseExecutionRunForFailureRequest,
) (*ExecutionRun, error) {
	runID, _, err := marketCombinationUUID(req.RunID, "execution run ID")
	if err != nil {
		return nil, invalidExecutionRun(err)
	}
	pauseCode, err := normalizeExecutionCode(req.PauseCode, true)
	if err != nil {
		return nil, err
	}
	if err := s.requireDatabase(); err != nil {
		return nil, err
	}
	now := canonicalNow(req.Now)
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, fmt.Errorf("begin failure pause transaction: %w", err)
	}
	defer rollbackWalletTransaction(tx)
	queries := wormtradingsqlc.New(tx)
	row, err := queries.PauseExecutionRunForFailure(ctx, wormtradingsqlc.PauseExecutionRunForFailureParams{
		PauseCode: pauseCode, Now: timestampParam(now), RunID: runID,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrExecutionRunConflict
	}
	if err != nil {
		return nil, fmt.Errorf("pause execution run for failure: %w", err)
	}
	if _, err := queries.ReleaseExecutionCoordinator(ctx, wormtradingsqlc.ReleaseExecutionCoordinatorParams{
		RunID: runID, Now: timestampParam(now),
	}); err != nil {
		return nil, fmt.Errorf("release failure-paused coordinator: %w", err)
	}
	run, err := loadExecutionRun(ctx, queries, row)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit failure pause: %w", err)
	}
	return &run, nil
}

func (s *SQLStore) CreateExecutionStepIsolation(
	ctx context.Context,
	req CreateExecutionStepIsolationRequest,
) (*ExecutionStepIsolation, error) {
	isolationID, _, err := marketCombinationUUID(req.IsolationID, "execution isolation ID")
	if err != nil {
		return nil, invalidExecutionRun(err)
	}
	runID, _, err := marketCombinationUUID(req.RunID, "execution run ID")
	if err != nil {
		return nil, invalidExecutionRun(err)
	}
	attemptID, _, err := marketCombinationUUID(req.AttemptID, "execution attempt ID")
	if err != nil {
		return nil, invalidExecutionRun(err)
	}
	reasonCode, err := normalizeExecutionCode(req.ReasonCode, true)
	if err != nil || req.StepOrdinal <= 0 {
		return nil, invalidExecutionRun(fmt.Errorf("execution isolation is invalid"))
	}
	if err := s.requireDatabase(); err != nil {
		return nil, err
	}
	row, err := s.queries.CreateExecutionStepIsolation(ctx, wormtradingsqlc.CreateExecutionStepIsolationParams{
		ID: isolationID, AttemptID: attemptID, ReasonCode: reasonCode,
		Now: timestampParam(canonicalNow(req.Now)), RunID: runID, StepOrdinal: req.StepOrdinal,
	})
	if err != nil {
		if executionConstraint(err, "worm_execution_step_isolations_active_idx") {
			return nil, ErrExecutionRunIsolation
		}
		return nil, fmt.Errorf("create execution step isolation: %w", err)
	}
	isolation := mapExecutionStepIsolation(row)
	return &isolation, nil
}

func (s *SQLStore) ResolveExecutionStepIsolation(
	ctx context.Context,
	req ResolveExecutionStepIsolationRequest,
) (*ExecutionStepIsolation, error) {
	id, _, err := marketCombinationUUID(req.IsolationID, "execution isolation ID")
	if err != nil {
		return nil, invalidExecutionRun(err)
	}
	runID, _, err := marketCombinationUUID(req.RunID, "execution run ID")
	if err != nil || req.StepOrdinal <= 0 {
		return nil, invalidExecutionRun(fmt.Errorf("execution isolation step is invalid"))
	}
	resolutionCode, err := normalizeExecutionCode(req.ResolutionCode, true)
	if err != nil {
		return nil, err
	}
	if err := s.requireDatabase(); err != nil {
		return nil, err
	}
	row, err := s.queries.ResolveExecutionStepIsolation(ctx, wormtradingsqlc.ResolveExecutionStepIsolationParams{
		Now: timestampParam(canonicalNow(req.Now)), ResolutionCode: resolutionCode, ID: id,
		RunID: runID, StepOrdinal: req.StepOrdinal,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrExecutionRunIsolation
	}
	if err != nil {
		return nil, fmt.Errorf("resolve execution step isolation: %w", err)
	}
	isolation := mapExecutionStepIsolation(row)
	return &isolation, nil
}

func (s *SQLStore) ListRecoverableExecutionSteps(
	ctx context.Context,
	now time.Time,
	limit int32,
) ([]RecoverableExecutionStep, error) {
	if limit <= 0 || limit > 100 {
		return nil, invalidExecutionRun(fmt.Errorf("recovery limit must be between 1 and 100"))
	}
	if err := s.requireDatabase(); err != nil {
		return nil, err
	}
	rows, err := s.queries.ListRecoverableExecutionRunStepKeys(ctx,
		wormtradingsqlc.ListRecoverableExecutionRunStepKeysParams{
			Now: timestampParam(canonicalNow(now)), RecoveryLimit: limit,
		})
	if err != nil {
		return nil, fmt.Errorf("list recoverable execution steps: %w", err)
	}
	result := make([]RecoverableExecutionStep, 0, len(rows))
	for _, key := range rows {
		row, err := s.queries.GetExecutionRunStep(ctx, wormtradingsqlc.GetExecutionRunStepParams{
			RunID: key.RunID, Ordinal: key.StepOrdinal,
		})
		if err != nil {
			return nil, fmt.Errorf("get recoverable execution step: %w", err)
		}
		step, err := loadExecutionRunStep(ctx, s.queries, row)
		if err != nil {
			return nil, err
		}
		result = append(result, RecoverableExecutionStep{
			OwnerAccountID: uuidValue(key.OwnerAccountID), RunID: uuidValue(key.RunID),
			RunState: ExecutionRunState(key.RunState), RunRevision: key.RunRevision,
			CommandID: uuidValue(key.CommandID), CoordinatorID: uuidValue(key.CoordinatorID),
			CoordinatorGeneration: nullableInt64(key.CoordinatorGeneration), Step: step,
		})
	}
	return result, nil
}

func nullableBigintParam(value int64) pgtype.Int8 {
	return pgtype.Int8{Int64: value, Valid: value > 0}
}

func nullableIntegerParam(value int32) pgtype.Int4 {
	return pgtype.Int4{Int32: value, Valid: value != 0}
}

func executionPlanDigest(plan ExecutionPlan, steps []ExecutionPlanStep) ([]byte, error) {
	plan.UsabilityCode = ""
	plan.WorkerID = ""
	plan.LockedAt = time.Time{}
	plan.LeaseExpiresAt = time.Time{}
	payload := struct {
		Version int64
		Plan    ExecutionPlan
		Steps   []ExecutionPlanStep
	}{executionPlanDigestVersion, plan, steps}
	encoded, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal execution plan digest v%d: %w", executionPlanDigestVersion, err)
	}
	digest := sha256.Sum256(encoded)
	return digest[:], nil
}

func executionRequestDigest(value any) ([]byte, error) {
	encoded, err := json.Marshal(value)
	if err != nil {
		return nil, fmt.Errorf("marshal execution command request: %w", err)
	}
	digest := sha256.Sum256(encoded)
	return digest[:], nil
}

func executionRunIDs(ownerAccountID string, runID string) (pgtype.UUID, string, pgtype.UUID, error) {
	ownerUUID, ownerAccountID, err := marketCombinationUUID(ownerAccountID, "owner account ID")
	if err != nil {
		return pgtype.UUID{}, "", pgtype.UUID{}, invalidExecutionRun(err)
	}
	runUUID, _, err := marketCombinationUUID(runID, "execution run ID")
	if err != nil {
		return pgtype.UUID{}, "", pgtype.UUID{}, invalidExecutionRun(err)
	}
	return ownerUUID, ownerAccountID, runUUID, nil
}

func executionPageOffset(page int32, pageSize int32) (int64, error) {
	if page <= 0 || pageSize <= 0 || pageSize > maxExecutionRunPageSize {
		return 0, invalidExecutionRun(fmt.Errorf("page must be positive and page size must be between 1 and %d", maxExecutionRunPageSize))
	}
	offset := int64(page-1) * int64(pageSize)
	if offset < 0 {
		return 0, invalidExecutionRun(fmt.Errorf("page offset is invalid"))
	}
	return offset, nil
}

func invalidExecutionRun(err error) error {
	return fmt.Errorf("%w: %v", ErrInvalidExecutionRun, err)
}

func executionConstraint(err error, name string) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.ConstraintName == name
}

func normalizeExecutionCode(value string, required bool) (string, error) {
	value = strings.TrimSpace(value)
	if (required && value == "") || len(value) > maxExecutionRunCodeLength {
		return "", invalidExecutionRun(fmt.Errorf("execution code is invalid"))
	}
	return value, nil
}

func normalizeExecutionCompletionPubkey(value string, required bool) (string, error) {
	trimmed := strings.TrimSpace(value)
	if value != trimmed || (required && trimmed == "") ||
		(trimmed != "" && (len(trimmed) < 32 || len(trimmed) > 64)) {
		return "", invalidExecutionRun(fmt.Errorf("execution completion pubkey is invalid"))
	}
	for _, character := range trimmed {
		if character < 0x21 || character == 0x7f {
			return "", invalidExecutionRun(fmt.Errorf("execution completion pubkey is invalid"))
		}
	}
	return trimmed, nil
}

func newExecutionCoordinatorToken() ([]byte, []byte, error) {
	token := make([]byte, executionCoordinatorTokenBytes)
	if _, err := rand.Read(token); err != nil {
		return nil, nil, fmt.Errorf("generate execution coordinator token: %w", err)
	}
	digest := sha256.Sum256(token)
	return token, digest[:], nil
}

func loadExecutionRun(
	ctx context.Context,
	queries *wormtradingsqlc.Queries,
	row wormtradingsqlc.WormExecutionRun,
) (ExecutionRun, error) {
	walletRows, err := queries.ListExecutionRunWallets(ctx, row.ID)
	if err != nil {
		return ExecutionRun{}, fmt.Errorf("list execution run wallets: %w", err)
	}
	itemRows, err := queries.ListExecutionRunItems(ctx, row.ID)
	if err != nil {
		return ExecutionRun{}, fmt.Errorf("list execution run items: %w", err)
	}
	run := mapExecutionRun(row)
	run.Wallets = make([]ExecutionPlanWallet, 0, len(walletRows))
	for _, walletRow := range walletRows {
		run.Wallets = append(run.Wallets, mapExecutionRunWallet(walletRow))
	}
	run.Items = make([]ExecutionPlanItem, 0, len(itemRows))
	for _, itemRow := range itemRows {
		run.Items = append(run.Items, mapExecutionRunItem(itemRow))
	}
	authorizationRow, err := queries.GetActiveExecutionAuthorization(ctx, row.ID)
	if err == nil {
		authorization := mapExecutionAuthorization(authorizationRow)
		run.Authorization = &authorization
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return ExecutionRun{}, fmt.Errorf("get active execution authorization: %w", err)
	}
	coordinatorRow, err := queries.GetActiveExecutionCoordinator(ctx, row.ID)
	if err == nil {
		coordinator := mapExecutionCoordinator(coordinatorRow)
		if coordinator.State == ExecutionCoordinatorStateActive && !coordinator.LeaseExpiresAt.After(time.Now().UTC()) {
			// Expiry is authoritative even before a later write checkpoints the
			// database row. Never project an elapsed lease as an active driver.
			coordinator.State = ExecutionCoordinatorStateExpired
			coordinator.ReleasedAt = coordinator.LeaseExpiresAt
		}
		run.Coordinator = &coordinator
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return ExecutionRun{}, fmt.Errorf("get active execution coordinator: %w", err)
	}
	if run.CurrentStepOrdinal > 0 {
		stepRow, err := queries.GetExecutionRunStep(ctx, wormtradingsqlc.GetExecutionRunStepParams{
			RunID: row.ID, Ordinal: run.CurrentStepOrdinal,
		})
		if err != nil {
			return ExecutionRun{}, fmt.Errorf("get current execution run step: %w", err)
		}
		step, err := loadExecutionRunStep(ctx, queries, stepRow)
		if err != nil {
			return ExecutionRun{}, err
		}
		run.CurrentStep = &step
	}
	run.AllowedActions = executionRunAllowedActions(run, canonicalNow(time.Now()))
	return run, nil
}

func loadExecutionRunStep(
	ctx context.Context,
	queries *wormtradingsqlc.Queries,
	row wormtradingsqlc.WormExecutionRunStep,
) (ExecutionRunStep, error) {
	step := mapExecutionRunStep(row)
	attemptRows, err := queries.ListExecutionStepMutationAttempts(ctx, wormtradingsqlc.ListExecutionStepMutationAttemptsParams{
		RunID: row.RunID, StepOrdinal: row.Ordinal,
	})
	if err != nil {
		return ExecutionRunStep{}, fmt.Errorf("list execution mutation attempts: %w", err)
	}
	step.Attempts = make([]ExecutionMutationAttempt, 0, len(attemptRows))
	for _, attemptRow := range attemptRows {
		step.Attempts = append(step.Attempts, mapExecutionMutationAttempt(attemptRow))
	}
	isolationRow, err := queries.GetActiveExecutionStepIsolation(ctx, wormtradingsqlc.GetActiveExecutionStepIsolationParams{
		RunID: row.RunID, WalletOrdinal: row.WalletOrdinal, ItemOrdinal: row.ItemOrdinal,
	})
	if err == nil {
		isolation := mapExecutionStepIsolation(isolationRow)
		step.Isolation = &isolation
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return ExecutionRunStep{}, fmt.Errorf("get execution step isolation: %w", err)
	}
	return step, nil
}

func mapExecutionRun(row wormtradingsqlc.WormExecutionRun) ExecutionRun {
	return ExecutionRun{
		ID:                   uuidValue(row.ID),
		OwnerAccountID:       uuidValue(row.OwnerAccountID),
		PlanID:               uuidValue(row.PlanID),
		PlanVersion:          row.PlanVersion,
		PlanDigestSHA256:     append([]byte(nil), row.PlanDigestSha256...),
		CombinationID:        uuidValue(row.CombinationID),
		CombinationName:      row.CombinationName,
		CombinationRevision:  row.CombinationRevision,
		State:                ExecutionRunState(row.State),
		Revision:             row.Revision,
		CurrentStepOrdinal:   nullableInt64(row.CurrentStepOrdinal),
		NextStepOrdinal:      nullableInt64(row.NextStepOrdinal),
		WalletCount:          row.WalletCount,
		ItemCount:            row.ItemCount,
		TotalStepCount:       row.TotalStepCount,
		ActionableStepCount:  row.ActionableStepCount,
		TerminalStepCount:    row.TerminalStepCount,
		CompletedStepCount:   row.CompletedStepCount,
		SatisfiedStepCount:   row.SatisfiedStepCount,
		SkippedStepCount:     row.SkippedStepCount,
		FailedStepCount:      row.FailedStepCount,
		NotExecutedStepCount: row.NotExecutedStepCount,
		PauseCode:            row.PauseCode,
		FailureCode:          row.FailureCode,
		BlockCode:            row.BlockCode,
		RequestedAt:          timestampValue(row.RequestedAt),
		AuthorizedAt:         timestampValue(row.AuthorizedAt),
		StartedAt:            timestampValue(row.StartedAt),
		PausedAt:             timestampValue(row.PausedAt),
		CompletedAt:          timestampValue(row.CompletedAt),
		CreatedAt:            timestampValue(row.CreatedAt),
		UpdatedAt:            timestampValue(row.UpdatedAt),
	}
}

func mapExecutionRunWallet(row wormtradingsqlc.WormExecutionRunWallet) ExecutionPlanWallet {
	return ExecutionPlanWallet{
		Ordinal: row.Ordinal,
		ExecutionPlanWalletInput: ExecutionPlanWalletInput{
			WalletID: row.WalletID, Address: row.Address, Remark: row.Remark,
			AvatarKind: row.AvatarKind, AvatarPresetID: row.AvatarPresetID, AvatarURL: row.AvatarUrl,
		},
		ConnectionState: ConnectionState(row.ConnectionState), ConnectionWarningCode: row.ConnectionWarningCode,
		ConnectedAt: timestampValue(row.ConnectedAt), CredentialVersion: row.CredentialVersion,
		SOLAtomicAmount: row.SolAtomicAmount, SOLAmount: row.SolAmount, SOLDecimals: row.SolDecimals,
		SOLObservedSlot: uint64(row.SolObservedSlot), SOLAvailability: row.SolAvailability, SOLErrorCode: row.SolErrorCode,
		USDCMint: row.UsdcMint, USDCAtomicAmount: row.UsdcAtomicAmount, USDCAmount: row.UsdcAmount,
		USDCDecimals: row.UsdcDecimals, USDCObservedSlot: uint64(row.UsdcObservedSlot),
		USDCAvailability: row.UsdcAvailability, USDCErrorCode: row.UsdcErrorCode,
		USDCTokenAccountCount: row.UsdcTokenAccountCount, Status: row.Status, ReasonCode: row.ReasonCode,
	}
}

func mapExecutionRunItem(row wormtradingsqlc.WormExecutionRunItem) ExecutionPlanItem {
	return ExecutionPlanItem{
		Ordinal: row.Ordinal,
		MarketCombinationItemInput: MarketCombinationItemInput{
			EventConditionID: row.EventConditionID, EventTitle: row.EventTitle, EventLogo: row.EventLogo,
			MarketConditionID: row.MarketConditionID, MarketTitle: row.MarketTitle, MarketLogo: row.MarketLogo,
			IsYes: row.IsYes, OutcomeLabel: row.OutcomeLabel,
		},
		Backend: row.Backend, Funds: row.Funds, Leverage: row.Leverage,
		State: row.PreviewState, ReasonCode: row.PreviewReasonCode,
		Estimate: ExecutionPlanEstimate{
			AveragePrice: row.EstimateAveragePrice, TotalShares: row.EstimateTotalShares,
			TotalCost: row.EstimateTotalCost, BestAsk: row.EstimateBestAsk,
			WorstFillPrice: row.EstimateWorstFillPrice, IsFullyFilled: row.EstimateIsFullyFilled,
			FeeAmount: row.EstimateFeeAmount, UserFundsNeeded: row.EstimateUserFundsNeeded,
			LiquidationPrice: row.EstimateLiquidationPrice,
		},
	}
}

func mapExecutionRunStep(row wormtradingsqlc.WormExecutionRunStep) ExecutionRunStep {
	return ExecutionRunStep{
		ID: uuidValue(row.ID), Ordinal: row.Ordinal, PlanStepOrdinal: row.PlanStepOrdinal,
		WalletOrdinal: row.WalletOrdinal, ItemOrdinal: row.ItemOrdinal,
		SourceDisposition: ExecutionPlanStepDisposition(row.SourceDisposition),
		SourceReasonCode:  row.SourceReasonCode, ProjectedUSDCBefore: row.ProjectedUsdcBefore,
		ProjectedUSDCAfter: row.ProjectedUsdcAfter, State: ExecutionStepState(row.State),
		ReasonCode:        row.ReasonCode,
		PositionRequestID: nullableInt64(row.PositionRequestID),
		FinalizeMode:      row.FinalizeMode, TransactionMessageSHA256: append([]byte(nil), row.TransactionMessageSha256...),
		TransactionVersion: row.TransactionVersion, RequiredSignatureCount: row.RequiredSignatureCount,
		WalletSignerIndex: row.WalletSignerIndex, ProviderState: row.ProviderState,
		ProviderOrderState: row.ProviderOrderState, FundingTxID: row.FundingTxid, RefundTxID: row.RefundTxid,
		CompletionSource:                ExecutionCompletionSource(row.CompletionSource),
		CompletionPositionPubkey:        row.CompletionPositionPubkey,
		CompletionPositionRequestPubkey: row.CompletionPositionRequestPubkey,
		CompletionPositionCreatedAt:     timestampValue(row.CompletionPositionCreatedAt),
		StartedAt:                       timestampValue(row.StartedAt), OpenedAt: timestampValue(row.OpenedAt),
		FinalizedAt: timestampValue(row.FinalizedAt), LastObservedAt: timestampValue(row.LastObservedAt),
		CompletedAt: timestampValue(row.CompletedAt), CreatedAt: timestampValue(row.CreatedAt),
		UpdatedAt: timestampValue(row.UpdatedAt), NextPollAt: timestampValue(row.NextPollAt),
		PollCount: row.PollCount, ClaimCommandID: uuidValue(row.ActiveCommandID),
		ClaimID: uuidValue(row.ClaimID), ClaimOwner: row.ClaimOwner,
		ClaimExpiresAt:       timestampValue(row.ClaimExpiresAt),
		ReconcileRequestedAt: timestampValue(row.ReconcileRequestedAt),
	}
}

func mapExecutionAuthorization(row wormtradingsqlc.WormExecutionAuthorization) ExecutionAuthorization {
	return ExecutionAuthorization{
		ID: uuidValue(row.ID), State: ExecutionAuthorizationState(row.State), Scope: row.Scope,
		ProofKind: row.ProofKind, SessionJTIDigest: append([]byte(nil), row.SessionJtiDigest...),
		AccessRevision: row.AccessRevision, PlanVersion: row.PlanVersion,
		PlanDigestSHA256: append([]byte(nil), row.PlanDigestSha256...),
		AuthorizedAt:     timestampValue(row.AuthorizedAt), EndedAt: timestampValue(row.EndedAt),
		EndReasonCode: row.EndReasonCode,
	}
}

func mapExecutionCoordinator(row wormtradingsqlc.WormExecutionCoordinator) ExecutionCoordinator {
	return ExecutionCoordinator{
		ID: uuidValue(row.ID), Generation: row.Generation, State: ExecutionCoordinatorState(row.State),
		AccessRevision: row.AccessRevision, AcquiredAt: timestampValue(row.AcquiredAt),
		HeartbeatAt: timestampValue(row.HeartbeatAt), LeaseExpiresAt: timestampValue(row.LeaseExpiresAt),
		ReleasedAt: timestampValue(row.ReleasedAt),
	}
}

func mapExecutionMutationAttempt(row wormtradingsqlc.WormExecutionMutationAttempt) ExecutionMutationAttempt {
	return ExecutionMutationAttempt{
		ID: uuidValue(row.ID), CommandID: uuidValue(row.CommandID), Kind: ExecutionMutationKind(row.Kind),
		State: ExecutionMutationState(row.State), RequestSHA256: append([]byte(nil), row.RequestSha256...),
		PositionRequestID: nullableInt64(row.PositionRequestID),
		HTTPStatus:        nullableInt32(row.HttpStatus), ProviderCode: nullableInt32(row.ProviderCode),
		ProviderSlug: row.ProviderSlug, ErrorCode: row.ErrorCode,
		PreparedAt: timestampValue(row.PreparedAt), DispatchedAt: timestampValue(row.DispatchedAt),
		CompletedAt: timestampValue(row.CompletedAt),
	}
}

func mapExecutionStepIsolation(row wormtradingsqlc.WormExecutionStepIsolation) ExecutionStepIsolation {
	return ExecutionStepIsolation{
		ID: uuidValue(row.ID), WalletID: row.WalletID, MarketConditionID: row.MarketConditionID,
		ReasonCode: row.ReasonCode, CreatedAt: timestampValue(row.CreatedAt),
		ResolvedAt: timestampValue(row.ResolvedAt), ResolutionCode: row.ResolutionCode,
	}
}

func nullableInt64(value pgtype.Int8) int64 {
	if !value.Valid {
		return 0
	}
	return value.Int64
}

func nullableInt32(value pgtype.Int4) int32 {
	if !value.Valid {
		return 0
	}
	return value.Int32
}

func executionStoreProviderTerminalFailure(state string) bool {
	switch strings.ToLower(strings.TrimSpace(state)) {
	case "failed", "cancelled", "canceled":
		return true
	default:
		return false
	}
}

func executionRunAllowedActions(run ExecutionRun, now time.Time) []ExecutionRunAction {
	coordinatorActive := run.Coordinator != nil &&
		run.Coordinator.State == ExecutionCoordinatorStateActive &&
		run.Coordinator.LeaseExpiresAt.After(now)
	switch run.State {
	case ExecutionRunStateAwaitingAuthorization:
		return []ExecutionRunAction{ExecutionRunActionAuthorize, ExecutionRunActionTerminate}
	case ExecutionRunStateAuthorized:
		return []ExecutionRunAction{ExecutionRunActionStart, ExecutionRunActionTerminate}
	case ExecutionRunStateRunning:
		actions := []ExecutionRunAction{ExecutionRunActionPause, ExecutionRunActionTerminate}
		if coordinatorActive {
			actions = append(actions, ExecutionRunActionHeartbeat)
			if run.NextStepOrdinal > 0 {
				actions = append(actions, ExecutionRunActionExecuteNext)
			}
		}
		return actions
	case ExecutionRunStatePauseRequested:
		actions := []ExecutionRunAction{ExecutionRunActionTerminate}
		if coordinatorActive {
			actions = append(actions, ExecutionRunActionHeartbeat)
		}
		return actions
	case ExecutionRunStateTerminateRequested:
		if coordinatorActive {
			return []ExecutionRunAction{ExecutionRunActionHeartbeat}
		}
		return []ExecutionRunAction{}
	case ExecutionRunStatePaused:
		if run.Authorization == nil {
			return []ExecutionRunAction{ExecutionRunActionAuthorize, ExecutionRunActionTerminate}
		}
		return []ExecutionRunAction{ExecutionRunActionContinue, ExecutionRunActionTerminate}
	case ExecutionRunStateReconciliationRequired:
		actions := []ExecutionRunAction{ExecutionRunActionTerminate, ExecutionRunActionReconcile}
		if run.Authorization == nil {
			actions = append([]ExecutionRunAction{ExecutionRunActionAuthorize}, actions...)
		}
		return actions
	case ExecutionRunStateTerminated:
		if run.CurrentStep != nil && run.CurrentStep.Isolation != nil {
			return []ExecutionRunAction{ExecutionRunActionReconcile}
		}
		return []ExecutionRunAction{}
	default:
		return []ExecutionRunAction{}
	}
}
