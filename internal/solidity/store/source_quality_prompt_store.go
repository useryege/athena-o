package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"
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
	created, err := s.insertSourceQualityPrompt(ctx, name, systemPrompt, true)
	if err != nil {
		if isUniqueViolation(err) {
			return s.GetActiveSourceQualityPrompt(ctx)
		}
		return nil, err
	}
	return created, nil
}

func (s *SQLStore) GetActiveSourceQualityPrompt(ctx context.Context) (*SourceQualityPrompt, error) {
	row := s.db.QueryRowContext(ctx, `
SELECT id, version, name, system_prompt, is_active, created_at, updated_at
FROM source_quality_prompt
WHERE is_active AND deleted_at IS NULL
ORDER BY version DESC
LIMIT 1
`)
	item, err := scanSourceQualityPrompt(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &item, nil
}

func (s *SQLStore) ListSourceQualityPrompts(ctx context.Context) ([]SourceQualityPrompt, error) {
	rows, err := s.db.QueryContext(ctx, `
SELECT id, version, name, system_prompt, is_active, created_at, updated_at
FROM source_quality_prompt
WHERE deleted_at IS NULL
ORDER BY is_active DESC, version DESC
`)
	if err != nil {
		return nil, fmt.Errorf("list source quality prompts: %w", err)
	}
	defer rows.Close()

	items := make([]SourceQualityPrompt, 0)
	for rows.Next() {
		item, err := scanSourceQualityPrompt(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate source quality prompts: %w", err)
	}
	return items, nil
}

func (s *SQLStore) GetSourceQualityPrompt(ctx context.Context, id int64) (*SourceQualityPrompt, error) {
	row := s.db.QueryRowContext(ctx, `
SELECT id, version, name, system_prompt, is_active, created_at, updated_at
FROM source_quality_prompt
WHERE id = $1 AND deleted_at IS NULL
`, id)
	item, err := scanSourceQualityPrompt(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &item, nil
}

func (s *SQLStore) CreateSourceQualityPrompt(ctx context.Context, name, systemPrompt string) (*SourceQualityPrompt, error) {
	return s.insertSourceQualityPrompt(ctx, name, systemPrompt, false)
}

func (s *SQLStore) UpdateSourceQualityPrompt(ctx context.Context, id int64, name, systemPrompt string) (*SourceQualityPrompt, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin update source quality prompt: %w", err)
	}
	defer tx.Rollback()

	existing, err := getSourceQualityPromptForUpdate(ctx, tx, id)
	if err != nil {
		return nil, err
	}
	if existing == nil {
		return nil, ErrSourceQualityPromptNotFound
	}
	if existing.IsActive {
		if _, err := tx.ExecContext(ctx, `
UPDATE source_quality_prompt
SET is_active = false, updated_at = now()
WHERE is_active AND deleted_at IS NULL
`); err != nil {
			return nil, fmt.Errorf("deactivate source quality prompt: %w", err)
		}
	}
	created, err := insertSourceQualityPromptTx(ctx, tx, name, systemPrompt, existing.IsActive)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit update source quality prompt: %w", err)
	}
	return created, nil
}

func (s *SQLStore) ActivateSourceQualityPrompt(ctx context.Context, id int64) (*SourceQualityPrompt, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin activate source quality prompt: %w", err)
	}
	defer tx.Rollback()

	existing, err := getSourceQualityPromptForUpdate(ctx, tx, id)
	if err != nil {
		return nil, err
	}
	if existing == nil {
		return nil, ErrSourceQualityPromptNotFound
	}
	if _, err := tx.ExecContext(ctx, `
UPDATE source_quality_prompt
SET is_active = false, updated_at = now()
WHERE is_active AND deleted_at IS NULL
`); err != nil {
		return nil, fmt.Errorf("deactivate source quality prompt: %w", err)
	}
	row := tx.QueryRowContext(ctx, `
UPDATE source_quality_prompt
SET is_active = true, updated_at = now()
WHERE id = $1 AND deleted_at IS NULL
RETURNING id, version, name, system_prompt, is_active, created_at, updated_at
`, id)
	activated, err := scanSourceQualityPrompt(row)
	if err != nil {
		return nil, fmt.Errorf("activate source quality prompt: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit activate source quality prompt: %w", err)
	}
	return &activated, nil
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
	result, err := s.db.ExecContext(ctx, `
UPDATE source_quality_prompt
SET deleted_at = now(), updated_at = now()
WHERE id = $1 AND deleted_at IS NULL AND NOT is_active
`, id)
	if err != nil {
		return fmt.Errorf("delete source quality prompt: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("rows affected for delete source quality prompt: %w", err)
	}
	if affected == 0 {
		return ErrSourceQualityPromptNotFound
	}
	return nil
}

func (s *SQLStore) insertSourceQualityPrompt(ctx context.Context, name, systemPrompt string, active bool) (*SourceQualityPrompt, error) {
	row := s.db.QueryRowContext(ctx, `
INSERT INTO source_quality_prompt (name, system_prompt, is_active)
VALUES ($1, $2, $3)
RETURNING id, version, name, system_prompt, is_active, created_at, updated_at
`, strings.TrimSpace(name), strings.TrimSpace(systemPrompt), active)
	item, err := scanSourceQualityPrompt(row)
	if err != nil {
		return nil, fmt.Errorf("insert source quality prompt: %w", err)
	}
	return &item, nil
}

func insertSourceQualityPromptTx(ctx context.Context, tx *sql.Tx, name, systemPrompt string, active bool) (*SourceQualityPrompt, error) {
	row := tx.QueryRowContext(ctx, `
INSERT INTO source_quality_prompt (name, system_prompt, is_active)
VALUES ($1, $2, $3)
RETURNING id, version, name, system_prompt, is_active, created_at, updated_at
`, strings.TrimSpace(name), strings.TrimSpace(systemPrompt), active)
	item, err := scanSourceQualityPrompt(row)
	if err != nil {
		return nil, fmt.Errorf("insert source quality prompt: %w", err)
	}
	return &item, nil
}

func getSourceQualityPromptForUpdate(ctx context.Context, tx *sql.Tx, id int64) (*SourceQualityPrompt, error) {
	row := tx.QueryRowContext(ctx, `
SELECT id, version, name, system_prompt, is_active, created_at, updated_at
FROM source_quality_prompt
WHERE id = $1 AND deleted_at IS NULL
FOR UPDATE
`, id)
	item, err := scanSourceQualityPrompt(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("get source quality prompt for update: %w", err)
	}
	return &item, nil
}

func scanSourceQualityPrompt(scanner rowScanner) (SourceQualityPrompt, error) {
	var item SourceQualityPrompt
	if err := scanner.Scan(&item.ID, &item.Version, &item.Name, &item.SystemPrompt, &item.IsActive, &item.CreatedAt, &item.UpdatedAt); err != nil {
		return SourceQualityPrompt{}, err
	}
	return item, nil
}
