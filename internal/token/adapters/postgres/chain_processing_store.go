package postgres

import (
	"context"
	"fmt"
	"time"

	tokensqlc "github.com/useryege/athena/internal/token/adapters/postgres/sqlc"
	"github.com/useryege/athena/internal/token/discovery"
	discoveryapp "github.com/useryege/athena/internal/token/discovery/application"
	"github.com/useryege/athena/internal/token/swap"
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
	if command.Block.Number != checkpoint.CursorBlockNumber {
		return nil, fmt.Errorf("processed block %d does not match checkpoint block %d", command.Block.Number, checkpoint.CursorBlockNumber)
	}
	cursorBlockNumber, err := uint64ToInt64("cursor_block_number", checkpoint.CursorBlockNumber)
	if err != nil {
		return nil, err
	}
	blockTime, err := uint64ToInt64("block_time", command.Block.Timestamp)
	if err != nil {
		return nil, err
	}
	prepared, err := prepareCandidateInspections(checkpoint, command.Block, command.Inspections)
	if err != nil {
		return nil, err
	}
	status := checkpoint.Status
	if status == "" {
		status = discovery.ChainProcessingStatusRunning
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin commit processed token block transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	queries := tokensqlc.New(tx)
	for _, item := range prepared {
		if err := queries.UpsertProjectCandidate(ctx, item.params); err != nil {
			return nil, fmt.Errorf("upsert project candidate %s: %w", item.inspection.Candidate.Contract, err)
		}
	}
	for _, item := range prepared {
		if !item.inspection.Accepted {
			continue
		}
		if err := initializeInspectedProject(ctx, queries, item.inspection, command.Schedules, command.ResearchTTL); err != nil {
			return nil, err
		}
	}
	expiredResearchStates, err := queries.ExpireProjectResearchStatesForBlock(ctx, tokensqlc.ExpireProjectResearchStatesForBlockParams{
		BlockNumber: cursorBlockNumber,
		BlockTime:   blockTime,
		ChainID:     checkpoint.ChainID,
	})
	if err != nil {
		return nil, fmt.Errorf("expire project research states for chain %d block %d: %w", checkpoint.ChainID, checkpoint.CursorBlockNumber, err)
	}
	row, err := queries.UpsertChainProcessingCheckpointCursor(ctx, tokensqlc.UpsertChainProcessingCheckpointCursorParams{
		ChainID:           checkpoint.ChainID,
		CursorBlockNumber: cursorBlockNumber,
		Status:            string(status),
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
	return &discoveryapp.CommitProcessedBlockResult{Checkpoint: *mapped, ExpiredResearchStates: expiredResearchStates}, nil
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

func initializeInspectedProject(ctx context.Context, queries *tokensqlc.Queries, inspection discoveryapp.CandidateInspection, schedules []discoveryapp.CollectionScheduleSeed, researchTTL time.Duration) error {
	candidate := inspection.Candidate
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
	attentionExpiryBlockTime, err := addProjectAttentionDuration(candidate.BlockTime, researchTTL)
	if err != nil {
		return fmt.Errorf("calculate research attention expiry for project %d: %w", project.ID, err)
	}
	absoluteExpiryBlockTime, err := addSwapObservationDuration(candidate.BlockTime, swap.AbsoluteObservationTimeout)
	if err != nil {
		return fmt.Errorf("calculate absolute Swap expiry for project %d: %w", project.ID, err)
	}
	nextExpiryBlockTime, err := addSwapObservationDuration(candidate.BlockTime, swap.FirstSwapTimeout)
	if err != nil {
		return fmt.Errorf("calculate initial Swap expiry for project %d: %w", project.ID, err)
	}
	swapPairs := []struct {
		kind    swap.PairKind
		address []byte
	}{
		{kind: swap.PairKindWETH, address: inspection.WethPair.Bytes()},
		{kind: swap.PairKindUSDT, address: inspection.UsdtPair.Bytes()},
	}
	for _, pair := range swapPairs {
		if _, err := queries.CreateProjectSwapPair(ctx, tokensqlc.CreateProjectSwapPairParams{
			ProjectID:               project.ID,
			ChainID:                 candidate.ChainID,
			PairKind:                string(pair.kind),
			PairAddress:             pair.address,
			StartBlockNumber:        blockNumber,
			StartBlockTime:          blockTime,
			AbsoluteExpiryBlockTime: absoluteExpiryBlockTime,
			NextExpiryBlockTime:     nextExpiryBlockTime,
		}); err != nil {
			return fmt.Errorf("create %s Swap pair for project %d: %w", pair.kind, project.ID, err)
		}
	}
	if _, err := queries.CreateProjectResearchState(ctx, tokensqlc.CreateProjectResearchStateParams{
		ProjectID:                project.ID,
		AttentionExpiryBlockTime: attentionExpiryBlockTime,
	}); err != nil {
		return fmt.Errorf("create research state for project %d: %w", project.ID, err)
	}
	for _, schedule := range schedules {
		if _, err := queries.UpsertProjectDataCollectionSchedule(ctx, tokensqlc.UpsertProjectDataCollectionScheduleParams{
			ProjectID:            project.ID,
			DataType:             schedule.DataType,
			RetryIntervalSeconds: int64(schedule.RetryInterval.Seconds()),
			NextRunAt:            nullableTime(schedule.NextRunAt),
		}); err != nil {
			return fmt.Errorf("upsert %s collection schedule for project %d: %w", schedule.DataType, project.ID, err)
		}
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
	return nil
}

func addProjectAttentionDuration(blockTime uint64, duration time.Duration) (int64, error) {
	if duration <= 0 || duration%time.Second != 0 {
		return 0, fmt.Errorf("research attention duration must be a positive whole number of seconds")
	}
	seconds := uint64(duration / time.Second)
	result := blockTime + seconds
	if result < blockTime {
		return 0, fmt.Errorf("research attention block time overflow")
	}
	return uint64ToInt64("attention_expiry_block_time", result)
}
