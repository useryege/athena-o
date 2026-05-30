package store

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	appsqlc "github.com/useryege/athena/internal/application/store/sqlc"
)

func (s *SQLStore) ListProjectAveRefreshCandidates(ctx context.Context, staleBefore time.Time, now time.Time, limit int32) ([]common.Address, error) {
	if s.queries == nil {
		return nil, fmt.Errorf("application postgres database is not configured")
	}
	if limit <= 0 {
		return nil, nil
	}
	rows, err := s.queries.ListProjectAveRefreshCandidates(ctx, appsqlc.ListProjectAveRefreshCandidatesParams{
		StaleBefore: pgTime(staleBefore),
		Now:         pgTime(now),
		LimitCount:  limit,
	})
	if err != nil {
		return nil, fmt.Errorf("list project ave refresh candidates: %w", err)
	}
	items := make([]common.Address, 0, len(rows))
	for _, row := range rows {
		items = append(items, common.BytesToAddress(row))
	}
	return items, nil
}

func (s *SQLStore) ScheduleProjectAveRefresh(ctx context.Context, contract common.Address, nextRunAt time.Time) error {
	if s.queries == nil {
		return fmt.Errorf("application postgres database is not configured")
	}
	if err := s.queries.ScheduleProjectAveRefresh(ctx, appsqlc.ScheduleProjectAveRefreshParams{
		ProjectContract: contract.Bytes(),
		NextRunAt:       pgTime(nextRunAt),
	}); err != nil {
		return fmt.Errorf("schedule project ave refresh: %w", err)
	}
	return nil
}

func (s *SQLStore) MarkProjectAveRefreshRunning(ctx context.Context, contract common.Address, at time.Time) error {
	if s.queries == nil {
		return fmt.Errorf("application postgres database is not configured")
	}
	if err := s.queries.MarkProjectAveRefreshRunning(ctx, appsqlc.MarkProjectAveRefreshRunningParams{
		ProjectContract: contract.Bytes(),
		LastAttemptAt:   pgTime(at),
	}); err != nil {
		return fmt.Errorf("mark project ave refresh running: %w", err)
	}
	return nil
}

func (s *SQLStore) MarkProjectAveRefreshSuccess(ctx context.Context, contract common.Address, successAt time.Time, nextRunAt time.Time) error {
	if s.queries == nil {
		return fmt.Errorf("application postgres database is not configured")
	}
	if err := s.queries.MarkProjectAveRefreshSuccess(ctx, appsqlc.MarkProjectAveRefreshSuccessParams{
		ProjectContract: contract.Bytes(),
		LastAttemptAt:   pgTime(successAt),
		NextRunAt:       pgTime(nextRunAt),
	}); err != nil {
		return fmt.Errorf("mark project ave refresh success: %w", err)
	}
	return nil
}

func (s *SQLStore) MarkProjectAveRefreshFailed(ctx context.Context, contract common.Address, attemptAt time.Time, nextRunAt time.Time, lastError string) error {
	if s.queries == nil {
		return fmt.Errorf("application postgres database is not configured")
	}
	if err := s.queries.MarkProjectAveRefreshFailed(ctx, appsqlc.MarkProjectAveRefreshFailedParams{
		ProjectContract: contract.Bytes(),
		LastAttemptAt:   pgTime(attemptAt),
		NextRunAt:       pgTime(nextRunAt),
		LastError:       pgText(lastError),
	}); err != nil {
		return fmt.Errorf("mark project ave refresh failed: %w", err)
	}
	return nil
}

func (s *SQLStore) GetProjectAveComponentState(ctx context.Context, contract common.Address) (*ProjectComponentState, error) {
	if s.queries == nil {
		return nil, fmt.Errorf("application postgres database is not configured")
	}
	row, err := s.queries.GetProjectAveComponentState(ctx, contract.Bytes())
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("get project ave component state: %w", err)
	}
	item := projectComponentStateFromSQLC(row)
	return &item, nil
}

func projectComponentStateFromSQLC(row appsqlc.ProjectComponentState) ProjectComponentState {
	item := ProjectComponentState{
		ProjectContract: common.BytesToAddress(row.ProjectContract),
		Component:       row.Component,
		Status:          row.Status,
		LastError:       row.LastError.String,
	}
	if row.LastAttemptAt.Valid {
		item.LastAttemptAt = row.LastAttemptAt.Time
	}
	if row.LastSuccessAt.Valid {
		item.LastSuccessAt = row.LastSuccessAt.Time
	}
	if row.NextRunAt.Valid {
		item.NextRunAt = row.NextRunAt.Time
	}
	if row.UpdatedAt.Valid {
		item.UpdatedAt = row.UpdatedAt.Time
	}
	return item
}

func pgTime(value time.Time) pgtype.Timestamptz {
	if value.IsZero() {
		return pgtype.Timestamptz{}
	}
	return pgtype.Timestamptz{Time: value.UTC(), Valid: true}
}

func pgText(value string) pgtype.Text {
	value = strings.TrimSpace(value)
	if value == "" {
		return pgtype.Text{}
	}
	return pgtype.Text{String: value, Valid: true}
}
