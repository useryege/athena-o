package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	tokensqlc "github.com/useryege/athena/internal/token/adapters/postgres/sqlc"
	"github.com/useryege/athena/internal/token/collection"
	"github.com/useryege/athena/internal/token/discovery"
	discoveryapp "github.com/useryege/athena/internal/token/discovery/application"
)

const (
	projectCandidateStatusValidated = "validated"
	projectCandidateStatusRejected  = "rejected"
)

type preparedCandidateInspection struct {
	inspection discoveryapp.CandidateInspection
	params     tokensqlc.UpsertProjectCandidateParams
}

func (s *ChainRepository) CommitProcessedBlock(ctx context.Context, command discoveryapp.CommitProcessedBlockCommand) (*discoveryapp.CommitProcessedBlockResult, error) {
	if s == nil || s.pool == nil {
		return nil, fmt.Errorf("token chain repository is not configured")
	}
	checkpoint := command.Checkpoint
	if command.Block.Number == 0 {
		return nil, fmt.Errorf("processed block number must be positive")
	}
	if command.Block.Number != checkpoint.CursorBlockNumber {
		return nil, fmt.Errorf("processed block %d does not match checkpoint block %d", command.Block.Number, checkpoint.CursorBlockNumber)
	}
	cursorBlockNumber, err := uint64ToInt64("cursor_block_number", checkpoint.CursorBlockNumber)
	if err != nil {
		return nil, err
	}
	prepared, err := prepareCandidateInspections(checkpoint, command.Block, command.Inspections)
	if err != nil {
		return nil, err
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin commit processed token block transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	queries := tokensqlc.New(tx)
	lockedCheckpoint, err := queries.LockChainProcessingCheckpoint(ctx, checkpoint.ChainID)
	if err != nil {
		return nil, fmt.Errorf("lock chain %d processing checkpoint for block %d: %w", checkpoint.ChainID, command.Block.Number, err)
	}
	currentCursor, err := int64ToUint64("current_cursor_block_number", lockedCheckpoint.CursorBlockNumber)
	if err != nil {
		return nil, err
	}
	if currentCursor >= command.Block.Number {
		mapped, mapErr := mapChainCheckpoint(lockedCheckpoint)
		if mapErr != nil {
			return nil, fmt.Errorf("map already-advanced chain %d checkpoint: %w", checkpoint.ChainID, mapErr)
		}
		mapped.ChainName = checkpoint.ChainName
		mapped.Enabled = checkpoint.Enabled
		return &discoveryapp.CommitProcessedBlockResult{Checkpoint: *mapped, AlreadyProcessed: true}, nil
	}
	expectedCursor := command.Block.Number - 1
	if currentCursor != expectedCursor {
		return nil, fmt.Errorf("chain %d checkpoint cursor %d cannot advance directly to block %d", checkpoint.ChainID, currentCursor, command.Block.Number)
	}
	if lockedCheckpoint.Status != string(discovery.ChainProcessingStatusRunning) {
		return nil, fmt.Errorf("chain %d checkpoint is %s while committing block %d", checkpoint.ChainID, lockedCheckpoint.Status, command.Block.Number)
	}
	for _, item := range prepared {
		if err := queries.UpsertProjectCandidate(ctx, item.params); err != nil {
			return nil, fmt.Errorf("upsert project candidate %s: %w", item.inspection.Candidate.Contract, err)
		}
	}
	for _, item := range prepared {
		if !item.inspection.Accepted {
			continue
		}
		if err := initializeInspectedProject(ctx, queries, item.inspection); err != nil {
			return nil, err
		}
	}
	expectedCursorValue, err := uint64ToInt64("expected_cursor_block_number", expectedCursor)
	if err != nil {
		return nil, err
	}
	row, err := queries.AdvanceChainProcessingCheckpoint(ctx, tokensqlc.AdvanceChainProcessingCheckpointParams{
		ChainID:                   checkpoint.ChainID,
		CursorBlockNumber:         cursorBlockNumber,
		ExpectedCursorBlockNumber: expectedCursorValue,
	})
	if err != nil {
		return nil, fmt.Errorf("advance chain processing checkpoint: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit processed token block transaction: %w", err)
	}
	mapped, err := mapChainCheckpoint(row)
	if err != nil {
		return nil, fmt.Errorf("map chain processing checkpoint: %w", err)
	}
	mapped.ChainName = checkpoint.ChainName
	mapped.Enabled = checkpoint.Enabled
	return &discoveryapp.CommitProcessedBlockResult{Checkpoint: *mapped}, nil
}

func prepareCandidateInspections(checkpoint discovery.ChainProcessingCheckpoint, block discovery.BlockHeader, inspections []discoveryapp.CandidateInspection) ([]preparedCandidateInspection, error) {
	prepared := make([]preparedCandidateInspection, 0, len(inspections))
	for _, inspection := range inspections {
		candidate := inspection.Candidate
		if candidate.ChainID != checkpoint.ChainID {
			return nil, fmt.Errorf("project candidate chain %d does not match checkpoint chain %d", candidate.ChainID, checkpoint.ChainID)
		}
		if candidate.BlockNumber != checkpoint.CursorBlockNumber {
			return nil, fmt.Errorf("project candidate block %d does not match checkpoint block %d", candidate.BlockNumber, checkpoint.CursorBlockNumber)
		}
		if candidate.BlockNumber != block.Number || candidate.BlockTime != block.Timestamp {
			return nil, fmt.Errorf("project candidate block position %d@%d does not match processed block %d@%d", candidate.BlockNumber, candidate.BlockTime, block.Number, block.Timestamp)
		}
		txIndex, err := uint64ToInt64("tx_index", candidate.TxIndex)
		if err != nil {
			return nil, err
		}
		deploymentNonce, err := uint64ToInt64("deployment_nonce", candidate.DeploymentNonce)
		if err != nil {
			return nil, err
		}
		blockNumber, err := uint64ToInt64("block_number", candidate.BlockNumber)
		if err != nil {
			return nil, err
		}
		blockTime, err := uint64ToInt64("block_time", candidate.BlockTime)
		if err != nil {
			return nil, err
		}
		status := projectCandidateStatusRejected
		if inspection.Accepted {
			status = projectCandidateStatusValidated
		}
		prepared = append(prepared, preparedCandidateInspection{
			inspection: inspection,
			params: tokensqlc.UpsertProjectCandidateParams{
				ChainID:         candidate.ChainID,
				Contract:        candidate.Contract.Bytes(),
				TxSender:        candidate.TxSender.Bytes(),
				TxHash:          candidate.TxHash.Bytes(),
				TxIndex:         txIndex,
				DeploymentNonce: deploymentNonce,
				BlockNumber:     blockNumber,
				BlockTime:       blockTime,
				Status:          status,
			},
		})
	}
	return prepared, nil
}

func initializeInspectedProject(ctx context.Context, queries *tokensqlc.Queries, inspection discoveryapp.CandidateInspection) error {
	candidate := inspection.Candidate
	existing, err := queries.GetProjectByContract(ctx, tokensqlc.GetProjectByContractParams{
		ChainID: candidate.ChainID, Contract: candidate.Contract.Bytes(),
	})
	if err == nil {
		taskTypeCount, countErr := queries.CountProjectDataCollectionTaskTypes(ctx, existing.ID)
		if countErr != nil {
			return fmt.Errorf("count existing project %d one-time collection tasks: %w", existing.ID, countErr)
		}
		expectedTaskTypes := int64(len(collection.AllDataTypes()))
		if taskTypeCount != expectedTaskTypes {
			return fmt.Errorf("existing project %d has %d collection task types, expected %d", existing.ID, taskTypeCount, expectedTaskTypes)
		}
		return nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return fmt.Errorf("get existing project candidate %s: %w", candidate.Contract, err)
	}
	if inspection.WethPair.IsZero() {
		return fmt.Errorf("validated project candidate %s has no WETH pair", candidate.Contract)
	}
	if inspection.UsdtPair.IsZero() {
		return fmt.Errorf("validated project candidate %s has no USDT pair", candidate.Contract)
	}
	txIndex, err := uint64ToInt64("tx_index", candidate.TxIndex)
	if err != nil {
		return err
	}
	deploymentNonce, err := uint64ToInt64("deployment_nonce", candidate.DeploymentNonce)
	if err != nil {
		return err
	}
	blockNumber, err := uint64ToInt64("block_number", candidate.BlockNumber)
	if err != nil {
		return err
	}
	blockTime, err := uint64ToInt64("block_time", candidate.BlockTime)
	if err != nil {
		return err
	}
	if err := queries.UpsertContractCode(ctx, inspection.CodeHash.Bytes()); err != nil {
		return fmt.Errorf("upsert contract code for project candidate %s: %w", candidate.Contract, err)
	}
	project, err := queries.UpsertProject(ctx, tokensqlc.UpsertProjectParams{
		ChainID:         candidate.ChainID,
		Contract:        candidate.Contract.Bytes(),
		TxSender:        candidate.TxSender.Bytes(),
		TxHash:          candidate.TxHash.Bytes(),
		TxIndex:         txIndex,
		DeploymentNonce: deploymentNonce,
		BlockNumber:     blockNumber,
		BlockTime:       blockTime,
		CodeHash:        inspection.CodeHash.Bytes(),
		Name:            inspection.Name,
		Symbol:          inspection.Symbol,
		Decimals:        int16(inspection.Decimals),
		TotalSupply:     numericFromBigInt(inspection.TotalSupply),
		WethPair:        optionalAddressBytes(inspection.WethPair),
		UsdtPair:        optionalAddressBytes(inspection.UsdtPair),
	})
	if err != nil {
		return fmt.Errorf("upsert project for candidate %s: %w", candidate.Contract, err)
	}
	for _, wallet := range inspection.RelatedWallets {
		if wallet.Wallet.IsZero() || wallet.Role == "" {
			continue
		}
		if _, err := queries.UpsertProjectRelatedWallet(ctx, tokensqlc.UpsertProjectRelatedWalletParams{
			ProjectID: project.ID,
			Wallet:    wallet.Wallet.Bytes(),
			Role:      wallet.Role,
		}); err != nil {
			return fmt.Errorf("upsert related wallet for project %d: %w", project.ID, err)
		}
	}
	if _, err := queries.DeleteProjectInitialRecipientsByProject(ctx, project.ID); err != nil {
		return fmt.Errorf("replace initial recipients for project %d: %w", project.ID, err)
	}
	for _, recipient := range inspection.InitialRecipients {
		if recipient.Wallet.IsZero() {
			continue
		}
		sourceBlockNumber, err := uint64ToInt64("source_block_number", recipient.SourceBlockNumber)
		if err != nil {
			return err
		}
		if _, err := queries.UpsertProjectInitialRecipient(ctx, tokensqlc.UpsertProjectInitialRecipientParams{
			ProjectID:         project.ID,
			Wallet:            recipient.Wallet.Bytes(),
			RatioBps:          recipient.RatioBPS,
			RankIndex:         recipient.RankIndex,
			SourceTxHash:      recipient.SourceTxHash.Bytes(),
			SourceBlockNumber: sourceBlockNumber,
		}); err != nil {
			return fmt.Errorf("upsert initial recipient for project %d: %w", project.ID, err)
		}
	}
	if _, err := queries.CreateProjectDataCollectionTasks(ctx, project.ID); err != nil {
		return fmt.Errorf("create one-time collection tasks for project %d: %w", project.ID, err)
	}
	taskTypeCount, err := queries.CountProjectDataCollectionTaskTypes(ctx, project.ID)
	if err != nil {
		return fmt.Errorf("count one-time collection tasks for project %d: %w", project.ID, err)
	}
	expectedTaskTypes := int64(len(collection.AllDataTypes()))
	if taskTypeCount != expectedTaskTypes {
		return fmt.Errorf("project %d has %d collection task types, expected %d", project.ID, taskTypeCount, expectedTaskTypes)
	}
	return nil
}
