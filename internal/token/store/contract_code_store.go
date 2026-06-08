package store

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/jackc/pgx/v5"
	tokensqlc "github.com/useryege/athena/internal/token/store/sqlc"
)

func (s *SQLStore) UpsertContractCode(ctx context.Context, codeHash common.Hash) error {
	q, err := s.querier()
	if err != nil {
		return err
	}
	if err := q.UpsertContractCode(ctx, codeHash.Bytes()); err != nil {
		return fmt.Errorf("upsert contract code: %w", err)
	}
	return nil
}

func (s *SQLStore) GetContractCode(ctx context.Context, codeHash common.Hash) (*ContractCode, error) {
	q, err := s.querier()
	if err != nil {
		return nil, err
	}
	row, err := q.GetContractCode(ctx, codeHash.Bytes())
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get contract code: %w", err)
	}
	return mapContractCode(row), nil
}

func (s *SQLStore) CountContractCodes(ctx context.Context, codeHash common.Hash) (int64, error) {
	q, err := s.querier()
	if err != nil {
		return 0, err
	}
	total, err := q.CountContractCodes(ctx, optionalHashBytes(codeHash))
	if err != nil {
		return 0, fmt.Errorf("count contract codes: %w", err)
	}
	return total, nil
}

func (s *SQLStore) ListContractCodes(ctx context.Context, codeHash common.Hash, page, pageSize int32) (*ContractCodePage, error) {
	q, err := s.querier()
	if err != nil {
		return nil, err
	}
	page, pageSize, offset := normalizePage(page, pageSize)
	filter := optionalHashBytes(codeHash)
	total, err := q.CountContractCodes(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("count contract codes: %w", err)
	}
	rows, err := q.ListContractCodes(ctx, tokensqlc.ListContractCodesParams{
		CodeHash: filter,
		Offset:   offset,
		Limit:    pageSize,
	})
	if err != nil {
		return nil, fmt.Errorf("list contract codes: %w", err)
	}
	return &ContractCodePage{
		Items:    mapContractCodes(rows),
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}, nil
}

func (s *SQLStore) ListContractCodesByDeploymentCount(ctx context.Context, page, pageSize int32) (*ContractCodePage, error) {
	q, err := s.querier()
	if err != nil {
		return nil, err
	}
	page, pageSize, offset := normalizePage(page, pageSize)
	total, err := q.CountContractCodes(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("count contract codes: %w", err)
	}
	rows, err := q.ListContractCodesByDeploymentCount(ctx, tokensqlc.ListContractCodesByDeploymentCountParams{
		Offset: offset,
		Limit:  pageSize,
	})
	if err != nil {
		return nil, fmt.Errorf("list contract codes by deployment count: %w", err)
	}
	return &ContractCodePage{
		Items:    mapContractCodes(rows),
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}, nil
}

func (s *SQLStore) UpdateContractCodeSource(ctx context.Context, codeHash common.Hash, sourceCode string, sourceCodeHash common.Hash, fetchedAt time.Time) (*ContractCode, error) {
	q, err := s.querier()
	if err != nil {
		return nil, err
	}
	row, err := q.UpdateContractCodeSource(ctx, tokensqlc.UpdateContractCodeSourceParams{
		CodeHash:            codeHash.Bytes(),
		SourceCode:          nullableText(sourceCode),
		SourceCodeHash:      optionalHashBytes(sourceCodeHash),
		SourceCodeFetchedAt: nullableTime(fetchedAt),
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("update contract code source: %w", err)
	}
	return mapContractCode(row), nil
}
