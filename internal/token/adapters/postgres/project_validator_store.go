package postgres

import (
	"context"
	"fmt"

	tokensqlc "github.com/useryege/athena/internal/token/adapters/postgres/sqlc"
	"github.com/useryege/athena/internal/token/discovery"
	discoveryapp "github.com/useryege/athena/internal/token/discovery/application"
	"github.com/useryege/athena/internal/token/shared"
)

func (repository *CandidateRepository) PromoteCandidateAndInitializeResearch(ctx context.Context, command discoveryapp.PromoteCandidateCommand) (shared.ProjectID, error) {
	if repository == nil || repository.pool == nil {
		return 0, fmt.Errorf("token candidate repository is not configured")
	}
	inspection := command.Inspection
	candidate := inspection.Candidate
	txIndex, err := uint64ToInt64("tx_index", candidate.TxIndex)
	if err != nil {
		return 0, err
	}
	blockNumber, err := uint64ToInt64("block_number", candidate.BlockNumber)
	if err != nil {
		return 0, err
	}
	blockTime, err := uint64ToInt64("block_time", candidate.BlockTime)
	if err != nil {
		return 0, err
	}
	tx, err := repository.pool.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("begin promote project candidate transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	queries := tokensqlc.New(tx)
	if err = queries.UpsertContractCode(ctx, inspection.CodeHash.Bytes()); err != nil {
		return 0, fmt.Errorf("upsert contract code: %w", err)
	}
	row, err := queries.UpsertProject(ctx, tokensqlc.UpsertProjectParams{
		ChainID: candidate.ChainID, Contract: candidate.Contract.Bytes(), TxSender: candidate.TxSender.Bytes(), TxHash: candidate.TxHash.Bytes(),
		TxIndex: txIndex, BlockNumber: blockNumber, BlockTime: blockTime, CodeHash: inspection.CodeHash.Bytes(), Name: inspection.Name,
		Symbol: inspection.Symbol, Decimals: int16(inspection.Decimals), TotalSupply: numericFromBigInt(inspection.TotalSupply),
		WethPair: optionalAddressBytes(inspection.WethPair), UsdtPair: optionalAddressBytes(inspection.UsdtPair),
	})
	if err != nil {
		return 0, fmt.Errorf("upsert project: %w", err)
	}
	if _, err = queries.CompleteProjectCandidateValidation(ctx, tokensqlc.CompleteProjectCandidateValidationParams{ID: candidate.ID, Status: string(discovery.ProjectCandidateStatusValidated), ValidationLockToken: uuidParam(candidate.ValidationLockToken)}); err != nil {
		return 0, fmt.Errorf("mark project candidate validated: %w", err)
	}
	if _, err = queries.CreateProjectResearchState(ctx, row.ID); err != nil {
		return 0, fmt.Errorf("create project research state: %w", err)
	}
	for _, schedule := range command.Schedules {
		if _, err = queries.UpsertProjectDataCollectionSchedule(ctx, tokensqlc.UpsertProjectDataCollectionScheduleParams{ProjectID: row.ID, DataType: schedule.DataType, RetryIntervalSeconds: int64(schedule.RetryInterval.Seconds()), NextRunAt: nullableTime(schedule.NextRunAt)}); err != nil {
			return 0, fmt.Errorf("create %s collection schedule: %w", schedule.DataType, err)
		}
	}
	for _, wallet := range inspection.RelatedWallets {
		if wallet.Wallet.IsZero() || wallet.Role == "" {
			continue
		}
		if _, err = queries.UpsertProjectRelatedWallet(ctx, tokensqlc.UpsertProjectRelatedWalletParams{ProjectID: row.ID, Wallet: wallet.Wallet.Bytes(), Role: wallet.Role}); err != nil {
			return 0, fmt.Errorf("upsert related wallet: %w", err)
		}
	}
	if _, err = queries.DeleteProjectInitialRecipientsByProject(ctx, row.ID); err != nil {
		return 0, fmt.Errorf("delete project initial recipients: %w", err)
	}
	for _, recipient := range inspection.InitialRecipients {
		if recipient.Wallet.IsZero() {
			continue
		}
		sourceBlock, conversionErr := uint64ToInt64("source_block_number", recipient.SourceBlockNumber)
		if conversionErr != nil {
			return 0, conversionErr
		}
		if _, err = queries.UpsertProjectInitialRecipient(ctx, tokensqlc.UpsertProjectInitialRecipientParams{ProjectID: row.ID, Wallet: recipient.Wallet.Bytes(), RatioBps: recipient.RatioBPS, RankIndex: recipient.RankIndex, SourceTxHash: recipient.SourceTxHash.Bytes(), SourceBlockNumber: sourceBlock}); err != nil {
			return 0, fmt.Errorf("upsert initial recipient: %w", err)
		}
	}
	if err = tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("commit promote project candidate: %w", err)
	}
	return shared.ProjectID(row.ID), nil
}
