package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/jackc/pgx/v5"
	tokensqlc "github.com/useryege/athena/internal/token/adapters/postgres/sqlc"
	"github.com/useryege/athena/internal/token/discovery"
)

func (s *Database) UpsertProjectCandidate(ctx context.Context, item discovery.ProjectCandidate) (*discovery.ProjectCandidate, error) {
	q, err := s.querier()
	if err != nil {
		return nil, err
	}
	status := item.Status
	if status == "" {
		status = discovery.ProjectCandidateStatusPending
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
		TxSender:    item.TxSender.Bytes(),
		TxHash:      item.TxHash.Bytes(),
		TxIndex:     txIndex,
		BlockNumber: blockNumber,
		BlockTime:   blockTime,
		Status:      string(status),
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

func (s *Database) BatchUpsertProjectCandidates(ctx context.Context, items []discovery.ProjectCandidate) error {
	if len(items) == 0 {
		return nil
	}
	q, err := s.querier()
	if err != nil {
		return err
	}
	params, err := batchUpsertProjectCandidatesParams(items)
	if err != nil {
		return err
	}
	if err := q.BatchUpsertProjectCandidates(ctx, params); err != nil {
		return fmt.Errorf("batch upsert project candidates: %w", err)
	}
	return nil
}

func batchUpsertProjectCandidatesParams(items []discovery.ProjectCandidate) (tokensqlc.BatchUpsertProjectCandidatesParams, error) {
	chainIDs := make([]int64, 0, len(items))
	contracts := make([][]byte, 0, len(items))
	txSenders := make([][]byte, 0, len(items))
	txHashes := make([][]byte, 0, len(items))
	txIndexes := make([]int64, 0, len(items))
	blockNumbers := make([]int64, 0, len(items))
	blockTimes := make([]int64, 0, len(items))
	statuses := make([]string, 0, len(items))
	for _, item := range items {
		status := item.Status
		if status == "" {
			status = discovery.ProjectCandidateStatusPending
		}
		txIndex, err := uint64ToInt64("tx_index", item.TxIndex)
		if err != nil {
			return tokensqlc.BatchUpsertProjectCandidatesParams{}, err
		}
		blockNumber, err := uint64ToInt64("block_number", item.BlockNumber)
		if err != nil {
			return tokensqlc.BatchUpsertProjectCandidatesParams{}, err
		}
		blockTime, err := uint64ToInt64("block_time", item.BlockTime)
		if err != nil {
			return tokensqlc.BatchUpsertProjectCandidatesParams{}, err
		}
		chainIDs = append(chainIDs, item.ChainID)
		contracts = append(contracts, item.Contract.Bytes())
		txSenders = append(txSenders, item.TxSender.Bytes())
		txHashes = append(txHashes, item.TxHash.Bytes())
		txIndexes = append(txIndexes, txIndex)
		blockNumbers = append(blockNumbers, blockNumber)
		blockTimes = append(blockTimes, blockTime)
		statuses = append(statuses, string(status))
	}
	return tokensqlc.BatchUpsertProjectCandidatesParams{
		ChainIds:     chainIDs,
		Contracts:    contracts,
		TxSenders:    txSenders,
		TxHashes:     txHashes,
		TxIndexes:    txIndexes,
		BlockNumbers: blockNumbers,
		BlockTimes:   blockTimes,
		Statuses:     statuses,
	}, nil
}

func (s *Database) GetProjectCandidate(ctx context.Context, id int64) (*discovery.ProjectCandidate, error) {
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

func (s *Database) GetProjectCandidateByContract(ctx context.Context, chainID int64, contract common.Address) (*discovery.ProjectCandidate, error) {
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

func (s *Database) CountProjectCandidates(ctx context.Context, chainID int64, status discovery.ProjectCandidateStatus) (int64, error) {
	q, err := s.querier()
	if err != nil {
		return 0, err
	}
	total, err := q.CountProjectCandidates(ctx, tokensqlc.CountProjectCandidatesParams{
		ChainID: chainID,
		Status:  nullableText(string(status)),
	})
	if err != nil {
		return 0, fmt.Errorf("count project candidates: %w", err)
	}
	return total, nil
}

func (s *Database) ListProjectCandidates(ctx context.Context, chainID int64, status discovery.ProjectCandidateStatus, page, pageSize int32) (*discovery.CandidatePage, error) {
	q, err := s.querier()
	if err != nil {
		return nil, err
	}
	page, pageSize, offset := normalizePage(page, pageSize)
	total, err := q.CountProjectCandidates(ctx, tokensqlc.CountProjectCandidatesParams{
		ChainID: chainID,
		Status:  nullableText(string(status)),
	})
	if err != nil {
		return nil, fmt.Errorf("count project candidates: %w", err)
	}
	rows, err := q.ListProjectCandidates(ctx, tokensqlc.ListProjectCandidatesParams{
		ChainID: chainID,
		Status:  nullableText(string(status)),
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
	return &discovery.CandidatePage{
		Items:    items,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}, nil
}

func (s *Database) ClaimProjectCandidateValidations(ctx context.Context, chainID int64, lockToken string, lease time.Duration, limit int32) ([]discovery.ProjectCandidate, error) {
	q, err := s.querier()
	if err != nil {
		return nil, err
	}
	if !uuidParam(lockToken).Valid {
		return nil, fmt.Errorf("project candidate validation lock token is required")
	}
	if lease <= 0 {
		return nil, fmt.Errorf("project candidate validation lease must be positive")
	}
	if limit <= 0 {
		limit = defaultPageSize
	}
	rows, err := q.ClaimProjectCandidateValidations(ctx, tokensqlc.ClaimProjectCandidateValidationsParams{ValidationLockToken: uuidParam(lockToken), LeaseSeconds: int64(lease / time.Second), ChainID: chainID, LimitCount: limit})
	if err != nil {
		return nil, fmt.Errorf("claim project candidate validations: %w", err)
	}
	items, err := mapProjectCandidates(rows)
	if err != nil {
		return nil, fmt.Errorf("map claimed project candidates: %w", err)
	}
	return items, nil
}

func (s *Database) RenewProjectCandidateValidationClaims(ctx context.Context, lockToken string, lease time.Duration) error {
	q, err := s.querier()
	if err != nil {
		return err
	}
	if !uuidParam(lockToken).Valid || lease <= 0 {
		return fmt.Errorf("project candidate validation lock token and lease are required")
	}
	_, err = q.RenewProjectCandidateValidationClaims(ctx, tokensqlc.RenewProjectCandidateValidationClaimsParams{LeaseSeconds: int64(lease / time.Second), ValidationLockToken: uuidParam(lockToken)})
	if err != nil {
		return fmt.Errorf("renew project candidate validation claims: %w", err)
	}
	return nil
}

func (s *Database) ReleaseProjectCandidateValidationClaims(ctx context.Context, lockToken string) error {
	q, err := s.querier()
	if err != nil {
		return err
	}
	if lockToken == "" {
		return nil
	}
	_, err = q.ReleaseProjectCandidateValidationClaims(ctx, uuidParam(lockToken))
	if err != nil {
		return fmt.Errorf("release project candidate validation claims: %w", err)
	}
	return nil
}

func (s *Database) ListProjectCandidatesByStatus(ctx context.Context, status discovery.ProjectCandidateStatus, limit int32) ([]discovery.ProjectCandidate, error) {
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
		Status:     string(status),
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

func (s *Database) RejectProjectCandidate(ctx context.Context, candidate discovery.ProjectCandidate) error {
	q, err := s.querier()
	if err != nil {
		return err
	}
	_, err = q.CompleteProjectCandidateValidation(ctx, tokensqlc.CompleteProjectCandidateValidationParams{
		ID:                  candidate.ID,
		Status:              string(discovery.ProjectCandidateStatusRejected),
		ValidationLockToken: uuidParam(candidate.ValidationLockToken),
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return fmt.Errorf("project candidate validation claim lost for candidate %d", candidate.ID)
	}
	if err != nil {
		return fmt.Errorf("reject project candidate: %w", err)
	}
	return nil
}

func (s *Database) DeleteProjectCandidate(ctx context.Context, id int64) (int64, error) {
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
