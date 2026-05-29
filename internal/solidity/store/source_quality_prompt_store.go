package store

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	soliditysqlc "github.com/useryege/athena/internal/solidity/store/sqlc"
)

var (
	ErrSourceQualityPromptNotFound     = errors.New("source quality prompt not found")
	ErrSourceQualityPromptActiveDelete = errors.New("active source quality prompt cannot be deleted")
)

type SourceQualityPrompt struct {
	ID           int64
	Version      int64
	Name         string
	SystemPrompt string
	IsActive     bool
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

func (s *SQLStore) EnsureDefaultSourceQualityPrompt(ctx context.Context, name, systemPrompt string) (*SourceQualityPrompt, error) {
	active, err := s.GetActiveSourceQualityPrompt(ctx)
	if err != nil {
		return nil, err
	}
	if active != nil {
		return active, nil
	}
	created, err := s.insertSourceQualityPrompt(ctx, s.queries, name, systemPrompt, true)
	if err != nil {
		if isUniqueViolation(err) {
			return s.GetActiveSourceQualityPrompt(ctx)
		}
		return nil, err
	}
	return created, nil
}

func (s *SQLStore) GetActiveSourceQualityPrompt(ctx context.Context) (*SourceQualityPrompt, error) {
	if s.queries == nil {
		return nil, fmt.Errorf("solidity postgres database is not configured")
	}
	row, err := s.queries.GetActiveSourceQualityPrompt(ctx)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	item := sourceQualityPromptFromActiveRow(row)
	return &item, nil
}

func (s *SQLStore) ListSourceQualityPrompts(ctx context.Context) ([]SourceQualityPrompt, error) {
	if s.queries == nil {
		return nil, fmt.Errorf("solidity postgres database is not configured")
	}
	rows, err := s.queries.ListSourceQualityPrompts(ctx)
	if err != nil {
		return nil, fmt.Errorf("list source quality prompts: %w", err)
	}

	items := make([]SourceQualityPrompt, 0, len(rows))
	for _, row := range rows {
		items = append(items, sourceQualityPromptFromListRow(row))
	}
	return items, nil
}

func (s *SQLStore) GetSourceQualityPrompt(ctx context.Context, id int64) (*SourceQualityPrompt, error) {
	if s.queries == nil {
		return nil, fmt.Errorf("solidity postgres database is not configured")
	}
	row, err := s.queries.GetSourceQualityPrompt(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	item := sourceQualityPromptFromGetRow(row)
	return &item, nil
}

func (s *SQLStore) CreateSourceQualityPrompt(ctx context.Context, name, systemPrompt string) (*SourceQualityPrompt, error) {
	return s.insertSourceQualityPrompt(ctx, s.queries, name, systemPrompt, false)
}

func (s *SQLStore) UpdateSourceQualityPrompt(ctx context.Context, id int64, name, systemPrompt string) (*SourceQualityPrompt, error) {
	if s.pool == nil {
		return nil, fmt.Errorf("solidity postgres database is not configured")
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin update source quality prompt: %w", err)
	}
	defer tx.Rollback(ctx)
	queries := soliditysqlc.New(tx)

	existing, err := getSourceQualityPromptForUpdate(ctx, queries, id)
	if err != nil {
		return nil, err
	}
	if existing == nil {
		return nil, ErrSourceQualityPromptNotFound
	}
	if existing.IsActive {
		if err := queries.DeactivateActiveSourceQualityPrompts(ctx); err != nil {
			return nil, fmt.Errorf("deactivate source quality prompt: %w", err)
		}
	}
	created, err := s.insertSourceQualityPrompt(ctx, queries, name, systemPrompt, existing.IsActive)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit update source quality prompt: %w", err)
	}
	return created, nil
}

func (s *SQLStore) ActivateSourceQualityPrompt(ctx context.Context, id int64) (*SourceQualityPrompt, error) {
	if s.pool == nil {
		return nil, fmt.Errorf("solidity postgres database is not configured")
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin activate source quality prompt: %w", err)
	}
	defer tx.Rollback(ctx)
	queries := soliditysqlc.New(tx)

	existing, err := getSourceQualityPromptForUpdate(ctx, queries, id)
	if err != nil {
		return nil, err
	}
	if existing == nil {
		return nil, ErrSourceQualityPromptNotFound
	}
	if err := queries.DeactivateActiveSourceQualityPrompts(ctx); err != nil {
		return nil, fmt.Errorf("deactivate source quality prompt: %w", err)
	}
	row, err := queries.ActivateSourceQualityPrompt(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("activate source quality prompt: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit activate source quality prompt: %w", err)
	}
	item := sourceQualityPromptFromActivateRow(row)
	return &item, nil
}

func (s *SQLStore) DeleteSourceQualityPrompt(ctx context.Context, id int64) error {
	item, err := s.GetSourceQualityPrompt(ctx, id)
	if err != nil {
		return err
	}
	if item == nil {
		return ErrSourceQualityPromptNotFound
	}
	if item.IsActive {
		return ErrSourceQualityPromptActiveDelete
	}
	affected, err := s.queries.DeleteSourceQualityPrompt(ctx, id)
	if err != nil {
		return fmt.Errorf("delete source quality prompt: %w", err)
	}
	if affected == 0 {
		return ErrSourceQualityPromptNotFound
	}
	return nil
}

func (s *SQLStore) insertSourceQualityPrompt(ctx context.Context, queries soliditysqlc.Querier, name, systemPrompt string, active bool) (*SourceQualityPrompt, error) {
	if queries == nil {
		return nil, fmt.Errorf("solidity postgres database is not configured")
	}
	row, err := queries.InsertSourceQualityPrompt(ctx, soliditysqlc.InsertSourceQualityPromptParams{
		Name:         strings.TrimSpace(name),
		SystemPrompt: strings.TrimSpace(systemPrompt),
		IsActive:     active,
	})
	if err != nil {
		return nil, fmt.Errorf("insert source quality prompt: %w", err)
	}
	item := sourceQualityPromptFromInsertRow(row)
	return &item, nil
}

func getSourceQualityPromptForUpdate(ctx context.Context, queries soliditysqlc.Querier, id int64) (*SourceQualityPrompt, error) {
	row, err := queries.GetSourceQualityPromptForUpdate(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("get source quality prompt for update: %w", err)
	}
	item := sourceQualityPromptFromForUpdateRow(row)
	return &item, nil
}

func sourceQualityPromptFromActiveRow(row soliditysqlc.GetActiveSourceQualityPromptRow) SourceQualityPrompt {
	return SourceQualityPrompt{
		ID:           row.ID,
		Version:      row.Version,
		Name:         row.Name,
		SystemPrompt: row.SystemPrompt,
		IsActive:     row.IsActive,
		CreatedAt:    timestamptzTime(row.CreatedAt),
		UpdatedAt:    timestamptzTime(row.UpdatedAt),
	}
}

func sourceQualityPromptFromListRow(row soliditysqlc.ListSourceQualityPromptsRow) SourceQualityPrompt {
	return SourceQualityPrompt{
		ID:           row.ID,
		Version:      row.Version,
		Name:         row.Name,
		SystemPrompt: row.SystemPrompt,
		IsActive:     row.IsActive,
		CreatedAt:    timestamptzTime(row.CreatedAt),
		UpdatedAt:    timestamptzTime(row.UpdatedAt),
	}
}

func sourceQualityPromptFromGetRow(row soliditysqlc.GetSourceQualityPromptRow) SourceQualityPrompt {
	return SourceQualityPrompt{
		ID:           row.ID,
		Version:      row.Version,
		Name:         row.Name,
		SystemPrompt: row.SystemPrompt,
		IsActive:     row.IsActive,
		CreatedAt:    timestamptzTime(row.CreatedAt),
		UpdatedAt:    timestamptzTime(row.UpdatedAt),
	}
}

func sourceQualityPromptFromInsertRow(row soliditysqlc.InsertSourceQualityPromptRow) SourceQualityPrompt {
	return SourceQualityPrompt{
		ID:           row.ID,
		Version:      row.Version,
		Name:         row.Name,
		SystemPrompt: row.SystemPrompt,
		IsActive:     row.IsActive,
		CreatedAt:    timestamptzTime(row.CreatedAt),
		UpdatedAt:    timestamptzTime(row.UpdatedAt),
	}
}

func sourceQualityPromptFromForUpdateRow(row soliditysqlc.GetSourceQualityPromptForUpdateRow) SourceQualityPrompt {
	return SourceQualityPrompt{
		ID:           row.ID,
		Version:      row.Version,
		Name:         row.Name,
		SystemPrompt: row.SystemPrompt,
		IsActive:     row.IsActive,
		CreatedAt:    timestamptzTime(row.CreatedAt),
		UpdatedAt:    timestamptzTime(row.UpdatedAt),
	}
}

func sourceQualityPromptFromActivateRow(row soliditysqlc.ActivateSourceQualityPromptRow) SourceQualityPrompt {
	return SourceQualityPrompt{
		ID:           row.ID,
		Version:      row.Version,
		Name:         row.Name,
		SystemPrompt: row.SystemPrompt,
		IsActive:     row.IsActive,
		CreatedAt:    timestamptzTime(row.CreatedAt),
		UpdatedAt:    timestamptzTime(row.UpdatedAt),
	}
}
