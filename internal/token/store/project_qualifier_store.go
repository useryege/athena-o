package store

import (
	"context"
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/common"
	tokensqlc "github.com/useryege/athena/internal/token/store/sqlc"
)

func (s *SQLStore) QualifyProjectCandidate(ctx context.Context, candidate ProjectCandidate, codeHash common.Hash, token ProjectTokenMetadata, wethPair, usdtPair common.Address, relatedWallets []ProjectRelatedWallet, walletAssetStates []WalletAssetState, initialRecipients []ProjectInitialRecipient) (*Project, error) {
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
		return nil, fmt.Errorf("begin qualify project candidate transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	q := tokensqlc.New(tx)
	if err := q.UpsertContractCode(ctx, codeHash.Bytes()); err != nil {
		return nil, fmt.Errorf("upsert contract code: %w", err)
	}
	row, err := q.UpsertProject(ctx, tokensqlc.UpsertProjectParams{
		ChainID:     candidate.ChainID,
		Contract:    candidate.Contract.Bytes(),
		Creator:     candidate.Creator.Bytes(),
		TxHash:      candidate.TxHash.Bytes(),
		TxIndex:     txIndex,
		BlockNumber: blockNumber,
		BlockTime:   blockTime,
		CodeHash:    codeHash.Bytes(),
		Name:        token.Name,
		Symbol:      token.Symbol,
		Decimals:    int16(token.Decimals),
		TotalSupply: numericFromBigInt(token.TotalSupply),
		WethPair:    optionalAddressBytes(wethPair),
		UsdtPair:    optionalAddressBytes(usdtPair),
	})
	if err != nil {
		return nil, fmt.Errorf("upsert project: %w", err)
	}
	if _, err := q.MarkProjectCandidateStatus(ctx, tokensqlc.MarkProjectCandidateStatusParams{
		ID:     candidate.ID,
		Status: ProjectCandidateStatusQualified,
	}); err != nil {
		return nil, fmt.Errorf("mark project candidate qualified: %w", err)
	}
	for _, dataType := range []string{ProjectDataCollectionTypeAve, ProjectDataCollectionTypeContractCodeSource, ProjectDataCollectionTypeChainState, ProjectDataCollectionTypeWalletAssetState, ProjectDataCollectionTypeSimulationResult} {
		if err := q.InsertProjectDataCollectionTaskIfNotExists(ctx, tokensqlc.InsertProjectDataCollectionTaskIfNotExistsParams{
			ProjectID: row.ID,
			DataType:  dataType,
		}); err != nil {
			return nil, fmt.Errorf("insert project data collection task %s: %w", dataType, err)
		}
	}
	if err := q.InsertProjectReportIfNotExists(ctx, row.ID); err != nil {
		return nil, fmt.Errorf("insert project report: %w", err)
	}
	for _, wallet := range relatedWallets {
		if wallet.Wallet == (common.Address{}) || wallet.Role == "" {
			continue
		}
		if _, err := q.UpsertProjectRelatedWallet(ctx, tokensqlc.UpsertProjectRelatedWalletParams{
			ProjectID: row.ID,
			Wallet:    wallet.Wallet.Bytes(),
			Role:      wallet.Role,
		}); err != nil {
			return nil, fmt.Errorf("upsert project related wallet %s %s: %w", wallet.Role, wallet.Wallet.Hex(), err)
		}
	}
	if _, err := q.DeleteProjectInitialRecipientsByProject(ctx, row.ID); err != nil {
		return nil, fmt.Errorf("delete project initial recipients: %w", err)
	}
	for _, recipient := range initialRecipients {
		if recipient.Wallet == (common.Address{}) {
			continue
		}
		sourceBlockNumber, err := uint64ToInt64("source_block_number", recipient.SourceBlockNumber)
		if err != nil {
			return nil, err
		}
		if _, err := q.UpsertProjectInitialRecipient(ctx, tokensqlc.UpsertProjectInitialRecipientParams{
			ProjectID:         row.ID,
			Wallet:            recipient.Wallet.Bytes(),
			RatioBps:          recipient.RatioBPS,
			RankIndex:         recipient.RankIndex,
			SourceTxHash:      recipient.SourceTxHash.Bytes(),
			SourceBlockNumber: sourceBlockNumber,
		}); err != nil {
			return nil, fmt.Errorf("upsert project initial recipient %s: %w", recipient.Wallet.Hex(), err)
		}
	}
	fetchedAt := time.Now().UTC()
	for _, walletState := range walletAssetStates {
		if walletState.Wallet == (common.Address{}) {
			continue
		}
		if walletState.ChainID == 0 {
			walletState.ChainID = candidate.ChainID
		}
		if walletState.FetchedAt.IsZero() {
			walletState.FetchedAt = fetchedAt
		}
		if _, err := q.UpsertWalletAssetState(ctx, tokensqlc.UpsertWalletAssetStateParams{
			ChainID:       walletState.ChainID,
			Wallet:        walletState.Wallet.Bytes(),
			WethBalance:   numericFromBigInt(walletState.WethBalance),
			UsdtBalance:   numericFromBigInt(walletState.UsdtBalance),
			NativeBalance: numericFromBigInt(walletState.NativeBalance),
			UsdtValue:     numericFromBigInt(walletState.UsdtValue),
			FetchedAt:     nullableTime(walletState.FetchedAt),
		}); err != nil {
			return nil, fmt.Errorf("upsert wallet asset state %s: %w", walletState.Wallet.Hex(), err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit qualify project candidate transaction: %w", err)
	}
	mapped, err := mapProject(row)
	if err != nil {
		return nil, fmt.Errorf("map project: %w", err)
	}
	return mapped, nil
}
