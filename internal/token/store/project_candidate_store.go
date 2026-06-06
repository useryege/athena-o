package store

import (
	"context"
	"errors"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/jackc/pgx/v5"
	tokensqlc "github.com/useryege/athena/internal/token/store/sqlc"
)

func (s *SQLStore) UpsertProjectCandidate(ctx context.Context, item ProjectCandidate) (*ProjectCandidate, error) {
	q, err := s.querier()
	if err != nil {
		return nil, err
	}
	status := item.Status
	if status == "" {
		status = ProjectCandidateStatusPending
	}
	txIndex, err := uint64ToInt64("tx_index", item.TxIndex)
	if err != nil {
		return nil, err
	}
	blockNumber, err := uint64ToInt64("block_number", item.BlockNumber)
	if err != nil {
		return nil, err
	}
	blockTime, err := uint64ToInt64("block_time", item.BlockTime)
	if err != nil {
		return nil, err
	}
	row, err := q.UpsertProjectCandidate(ctx, tokensqlc.UpsertProjectCandidateParams{
		ChainID:     item.ChainID,
		Contract:    item.Contract.Bytes(),
		Creator:     item.Creator.Bytes(),
		TxHash:      item.TxHash.Bytes(),
		TxIndex:     txIndex,
		BlockNumber: blockNumber,
		BlockTime:   blockTime,
		Status:      status,
	})
	if err != nil {
		return nil, fmt.Errorf("upsert project candidate: %w", err)
	}
	mapped, err := mapProjectCandidate(row)
	if err != nil {
		return nil, fmt.Errorf("map project candidate: %w", err)
	}
	return mapped, nil
}

func (s *SQLStore) GetProjectCandidate(ctx context.Context, id int64) (*ProjectCandidate, error) {
	q, err := s.querier()
	if err != nil {
		return nil, err
	}
	row, err := q.GetProjectCandidate(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get project candidate: %w", err)
	}
	item, err := mapProjectCandidate(row)
	if err != nil {
		return nil, fmt.Errorf("map project candidate: %w", err)
	}
	return item, nil
}

func (s *SQLStore) GetProjectCandidateByContract(ctx context.Context, chainID int64, contract common.Address) (*ProjectCandidate, error) {
	q, err := s.querier()
	if err != nil {
		return nil, err
	}
	row, err := q.GetProjectCandidateByContract(ctx, tokensqlc.GetProjectCandidateByContractParams{
		ChainID:  chainID,
		Contract: contract.Bytes(),
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get project candidate by contract: %w", err)
	}
	item, err := mapProjectCandidate(row)
	if err != nil {
		return nil, fmt.Errorf("map project candidate: %w", err)
	}
	return item, nil
}

func (s *SQLStore) CountProjectCandidates(ctx context.Context, chainID int64, status string) (int64, error) {
	q, err := s.querier()
	if err != nil {
		return 0, err
	}
	total, err := q.CountProjectCandidates(ctx, tokensqlc.CountProjectCandidatesParams{
		ChainID: chainID,
		Status:  nullableText(status),
	})
	if err != nil {
		return 0, fmt.Errorf("count project candidates: %w", err)
	}
	return total, nil
}

func (s *SQLStore) ListProjectCandidates(ctx context.Context, chainID int64, status string, page, pageSize int32) (*ProjectCandidatePage, error) {
	q, err := s.querier()
	if err != nil {
		return nil, err
	}
	page, pageSize, offset := normalizePage(page, pageSize)
	total, err := q.CountProjectCandidates(ctx, tokensqlc.CountProjectCandidatesParams{
		ChainID: chainID,
		Status:  nullableText(status),
	})
	if err != nil {
		return nil, fmt.Errorf("count project candidates: %w", err)
	}
	rows, err := q.ListProjectCandidates(ctx, tokensqlc.ListProjectCandidatesParams{
		ChainID: chainID,
		Status:  nullableText(status),
		Offset:  offset,
		Limit:   pageSize,
	})
	if err != nil {
		return nil, fmt.Errorf("list project candidates: %w", err)
	}
	items, err := mapProjectCandidates(rows)
	if err != nil {
		return nil, fmt.Errorf("map project candidates: %w", err)
	}
	return &ProjectCandidatePage{
		Items:    items,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}, nil
}

func (s *SQLStore) ListProjectCandidatesByStatus(ctx context.Context, status string, limit int32) ([]ProjectCandidate, error) {
	q, err := s.querier()
	if err != nil {
		return nil, err
	}
	if limit <= 0 {
		limit = defaultPageSize
	}
	if limit > maxPageSize {
		limit = maxPageSize
	}
	rows, err := q.ListProjectCandidatesByStatus(ctx, tokensqlc.ListProjectCandidatesByStatusParams{
		Status:     status,
		LimitCount: limit,
	})
	if err != nil {
		return nil, fmt.Errorf("list project candidates by status: %w", err)
	}
	items, err := mapProjectCandidates(rows)
	if err != nil {
		return nil, fmt.Errorf("map project candidates: %w", err)
	}
	return items, nil
}

func (s *SQLStore) MarkProjectCandidateStatus(ctx context.Context, id int64, status string) (*ProjectCandidate, error) {
	q, err := s.querier()
	if err != nil {
		return nil, err
	}
	row, err := q.MarkProjectCandidateStatus(ctx, tokensqlc.MarkProjectCandidateStatusParams{
		ID:     id,
		Status: status,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("mark project candidate status: %w", err)
	}
	item, err := mapProjectCandidate(row)
	if err != nil {
		return nil, fmt.Errorf("map project candidate: %w", err)
	}
	return item, nil
}

func (s *SQLStore) DeleteProjectCandidate(ctx context.Context, id int64) (int64, error) {
	q, err := s.querier()
	if err != nil {
		return 0, err
	}
	rowsAffected, err := q.DeleteProjectCandidate(ctx, id)
	if err != nil {
		return 0, fmt.Errorf("delete project candidate: %w", err)
	}
	return rowsAffected, nil
}
