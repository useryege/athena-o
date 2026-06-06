package store

import (
	"context"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	tokensqlc "github.com/useryege/athena/internal/token/store/sqlc"
)

func (s *SQLStore) QualifyProjectCandidate(ctx context.Context, candidate ProjectCandidate, codeHash common.Hash, wethPair, usdtPair common.Address) (*Project, error) {
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
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit qualify project candidate transaction: %w", err)
	}
	mapped, err := mapProject(row)
	if err != nil {
		return nil, fmt.Errorf("map project: %w", err)
	}
	return mapped, nil
}
