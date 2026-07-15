package store

import (
	"context"
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/common"
	tokensqlc "github.com/useryege/athena/internal/token/store/sqlc"
)

var defaultCollectionIntervals = map[string]time.Duration{
	ProjectDataCollectionTypeChainState:         15 * time.Second,
	ProjectDataCollectionTypeWalletAssetState:   60 * time.Second,
	ProjectDataCollectionTypeSimulationResult:   60 * time.Second,
	ProjectDataCollectionTypeAve:                5 * time.Minute,
	ProjectDataCollectionTypeContractCodeSource: 10 * time.Minute,
}

func (s *SQLStore) ValidateProjectCandidate(ctx context.Context, candidate ProjectCandidate, codeHash common.Hash, token ProjectTokenMetadata, wethPair, usdtPair common.Address, relatedWallets []ProjectRelatedWallet, initialRecipients []ProjectInitialRecipient) (*Project, error) {
	if s == nil || s.pool == nil {
		return nil, fmt.Errorf("token postgres database is not configured")
	}
	txIndex, err := uint64ToInt64("tx_index", candidate.TxIndex)
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
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin validate project candidate transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	q := tokensqlc.New(tx)
	if err := q.UpsertContractCode(ctx, codeHash.Bytes()); err != nil {
		return nil, fmt.Errorf("upsert contract code: %w", err)
	}
	row, err := q.UpsertProject(ctx, tokensqlc.UpsertProjectParams{ChainID: candidate.ChainID, Contract: candidate.Contract.Bytes(), TxSender: candidate.TxSender.Bytes(), TxHash: candidate.TxHash.Bytes(), TxIndex: txIndex, BlockNumber: blockNumber, BlockTime: blockTime, CodeHash: codeHash.Bytes(), Name: token.Name, Symbol: token.Symbol, Decimals: int16(token.Decimals), TotalSupply: numericFromBigInt(token.TotalSupply), WethPair: optionalAddressBytes(wethPair), UsdtPair: optionalAddressBytes(usdtPair)})
	if err != nil {
		return nil, fmt.Errorf("upsert project: %w", err)
	}
	if _, err = q.MarkProjectCandidateStatus(ctx, tokensqlc.MarkProjectCandidateStatusParams{ID: candidate.ID, Status: ProjectCandidateStatusValidated}); err != nil {
		return nil, fmt.Errorf("mark project candidate validated: %w", err)
	}
	if _, err = q.CreateProjectResearchState(ctx, row.ID); err != nil {
		return nil, fmt.Errorf("create project research state: %w", err)
	}
	now := time.Now().UTC()
	for dataType, interval := range defaultCollectionIntervals {
		if _, err = q.UpsertProjectDataCollectionSchedule(ctx, tokensqlc.UpsertProjectDataCollectionScheduleParams{ProjectID: row.ID, DataType: dataType, RefreshIntervalSeconds: int64(interval / time.Second), NextRunAt: nullableTime(now)}); err != nil {
			return nil, fmt.Errorf("create %s collection schedule: %w", dataType, err)
		}
	}
	for _, wallet := range relatedWallets {
		if wallet.Wallet == (common.Address{}) || wallet.Role == "" {
			continue
		}
		if _, err = q.UpsertProjectRelatedWallet(ctx, tokensqlc.UpsertProjectRelatedWalletParams{ProjectID: row.ID, Wallet: wallet.Wallet.Bytes(), Role: wallet.Role}); err != nil {
			return nil, fmt.Errorf("upsert related wallet: %w", err)
		}
	}
	if _, err = q.DeleteProjectInitialRecipientsByProject(ctx, row.ID); err != nil {
		return nil, fmt.Errorf("delete project initial recipients: %w", err)
	}
	for _, recipient := range initialRecipients {
		if recipient.Wallet == (common.Address{}) {
			continue
		}
		sourceBlock, err := uint64ToInt64("source_block_number", recipient.SourceBlockNumber)
		if err != nil {
			return nil, err
		}
		if _, err = q.UpsertProjectInitialRecipient(ctx, tokensqlc.UpsertProjectInitialRecipientParams{ProjectID: row.ID, Wallet: recipient.Wallet.Bytes(), RatioBps: recipient.RatioBPS, RankIndex: recipient.RankIndex, SourceTxHash: recipient.SourceTxHash.Bytes(), SourceBlockNumber: sourceBlock}); err != nil {
			return nil, fmt.Errorf("upsert initial recipient: %w", err)
		}
	}
	if err = tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit validate project candidate: %w", err)
	}
	return mapProject(row)
}
