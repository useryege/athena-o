package store

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/jackc/pgx/v5"
	appsqlc "github.com/useryege/athena/internal/application/store/sqlc"
)

func (s *SQLStore) GetProjectCollectionState(ctx context.Context, chainID int64, contract common.Address) (*ProjectCollectionState, error) {
	queries, err := s.querier()
	if err != nil {
		return nil, err
	}
	row, err := queries.GetProjectCollectionState(ctx, appsqlc.GetProjectCollectionStateParams{
		ChainID:         s.chainIDForProject(chainID),
		ProjectContract: contract.Bytes(),
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("get project collection state: %w", err)
	}
	item := ProjectCollectionState{
		ChainID:         row.ChainID,
		ProjectContract: common.BytesToAddress(row.ProjectContract),
		Status:          row.Status,
		WorkflowID:      row.WorkflowID.String,
		LastError:       row.LastError.String,
	}
	if row.LastRequestedAt.Valid {
		item.LastRequestedAt = row.LastRequestedAt.Time
	}
	if row.LastStartedAt.Valid {
		item.LastStartedAt = row.LastStartedAt.Time
	}
	if row.LastCompletedAt.Valid {
		item.LastCompletedAt = row.LastCompletedAt.Time
	}
	if row.NextRunAt.Valid {
		item.NextRunAt = row.NextRunAt.Time
	}
	if row.UpdatedAt.Valid {
		item.UpdatedAt = row.UpdatedAt.Time
	}
	components, err := s.ListProjectComponentStates(ctx, row.ChainID, common.BytesToAddress(row.ProjectContract))
	if err != nil {
		return nil, err
	}
	item.ComponentStates = components
	return &item, nil
}

func (s *SQLStore) ListProjectComponentStates(ctx context.Context, chainID int64, contract common.Address) ([]ProjectComponentState, error) {
	queries, err := s.querier()
	if err != nil {
		return nil, err
	}
	rows, err := queries.ListProjectComponentStates(ctx, appsqlc.ListProjectComponentStatesParams{
		ChainID:         s.chainIDForProject(chainID),
		ProjectContract: contract.Bytes(),
	})
	if err != nil {
		return nil, fmt.Errorf("list project component states: %w", err)
	}
	items := make([]ProjectComponentState, 0, len(rows))
	for _, row := range rows {
		items = append(items, projectComponentStateFromFields(
			row.ChainID,
			row.ProjectContract,
			row.Component,
			row.Status,
			row.LastAttemptAt,
			row.LastSuccessAt,
			row.NextRunAt,
			row.LastError,
			row.UpdatedAt,
		))
	}
	return items, nil
}

func (s *SQLStore) MarkProjectCollectionRunning(ctx context.Context, chainID int64, contract common.Address, workflowID string, startedAt time.Time) error {
	workflowID = strings.TrimSpace(workflowID)
	if workflowID == "" {
		return fmt.Errorf("project collection workflow_id is required")
	}
	if startedAt.IsZero() {
		startedAt = time.Now().UTC()
	}
	queries, err := s.querier()
	if err != nil {
		return err
	}
	if err := queries.MarkProjectCollectionRunning(ctx, appsqlc.MarkProjectCollectionRunningParams{
		WorkflowID:      pgText(workflowID),
		StartedAt:       pgTime(startedAt),
		ChainID:         s.chainIDForProject(chainID),
		ProjectContract: contract.Bytes(),
	}); err != nil {
		return fmt.Errorf("mark project collection running: %w", err)
	}
	return nil
}

func (s *SQLStore) MarkProjectCollectionCompleted(ctx context.Context, chainID int64, contract common.Address, completedAt time.Time) error {
	if completedAt.IsZero() {
		completedAt = time.Now().UTC()
	}
	queries, err := s.querier()
	if err != nil {
		return err
	}
	if err := queries.MarkProjectCollectionCompleted(ctx, appsqlc.MarkProjectCollectionCompletedParams{
		CompletedAt:     pgTime(completedAt),
		ChainID:         s.chainIDForProject(chainID),
		ProjectContract: contract.Bytes(),
	}); err != nil {
		return fmt.Errorf("mark project collection completed: %w", err)
	}
	return nil
}

func (s *SQLStore) MarkProjectCollectionFailed(ctx context.Context, chainID int64, contract common.Address, nextRunAt time.Time, lastError string) error {
	lastError = strings.TrimSpace(lastError)
	if lastError == "" {
		lastError = "project collection failed"
	}
	queries, err := s.querier()
	if err != nil {
		return err
	}
	if err := queries.MarkProjectCollectionFailed(ctx, appsqlc.MarkProjectCollectionFailedParams{
		NextRunAt:       pgTime(nextRunAt),
		LastError:       pgText(lastError),
		ChainID:         s.chainIDForProject(chainID),
		ProjectContract: contract.Bytes(),
	}); err != nil {
		return fmt.Errorf("mark project collection failed: %w", err)
	}
	return nil
}
