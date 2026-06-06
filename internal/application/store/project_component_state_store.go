package store

import (
	"context"
	"errors"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/jackc/pgx/v5"
	appsqlc "github.com/useryege/athena/internal/application/store/sqlc"
)

func (s *SQLStore) UpsertProjectComponentState(ctx context.Context, item ProjectComponentState) error {
	queries, err := s.querier()
	if err != nil {
		return err
	}
	err = queries.UpsertProjectComponentState(ctx, appsqlc.UpsertProjectComponentStateParams{
		ChainID:         s.chainIDForProject(item.ChainID),
		ProjectContract: item.ProjectContract.Bytes(),
		Component:       item.Component,
		Status:          item.Status,
		LastAttemptAt:   pgTime(item.LastAttemptAt),
		LastSuccessAt:   pgTime(item.LastSuccessAt),
		NextRunAt:       pgTime(item.NextRunAt),
		LastError:       optionalPgText(item.LastError),
	})
	if err != nil {
		return fmt.Errorf("upsert project component state: %w", err)
	}
	return nil
}

func (s *SQLStore) GetProjectComponentState(ctx context.Context, chainID int64, contract common.Address, component string) (*ProjectComponentState, error) {
	queries, err := s.querier()
	if err != nil {
		return nil, err
	}
	row, err := queries.GetProjectComponentState(ctx, appsqlc.GetProjectComponentStateParams{
		ChainID:         s.chainIDForProject(chainID),
		ProjectContract: contract.Bytes(),
		Component:       component,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("get project component state: %w", err)
	}
	item := projectComponentStateFromFields(row.ChainID, row.ProjectContract, row.Component, row.Status, row.LastAttemptAt, row.LastSuccessAt, row.NextRunAt, row.LastError, row.UpdatedAt)
	return &item, nil
}
