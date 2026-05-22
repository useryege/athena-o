package store

import (
	"context"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/ethereum/go-ethereum/common"
)

const (
	defaultProjectCommentPageSize  int32 = 5
	maxProjectCommentPageSize      int32 = 5
	MaxProjectCommentContentLength       = 1000
)

func (s *SQLStore) AddProjectComment(ctx context.Context, item ProjectComment) (ProjectComment, error) {
	if strings.TrimSpace(item.Username) == "" {
		return ProjectComment{}, fmt.Errorf("project comment username is empty")
	}
	if strings.TrimSpace(item.Content) == "" {
		return ProjectComment{}, fmt.Errorf("project comment content is empty")
	}
	if utf8.RuneCountInString(strings.TrimSpace(item.Content)) > MaxProjectCommentContentLength {
		return ProjectComment{}, fmt.Errorf("project comment content exceeds max length %d", MaxProjectCommentContentLength)
	}

	row := s.db.QueryRowContext(ctx, `
INSERT INTO project_comment (
  project_contract,
  username,
  content
) VALUES ($1, $2, $3)
RETURNING id, project_contract, username, content, created_at
`, item.Contract.Bytes(), strings.TrimSpace(item.Username), strings.TrimSpace(item.Content))

	created, err := scanProjectCommentRow(row)
	if err != nil {
		return ProjectComment{}, fmt.Errorf("add project comment: %w", err)
	}
	return created, nil
}

func (s *SQLStore) ListProjectCommentsByContract(ctx context.Context, contract common.Address, page int32, pageSize int32) ([]ProjectComment, int64, int32, int32, error) {
	page, pageSize = normalizeProjectCommentPage(page, pageSize)

	var total int64
	if err := s.db.QueryRowContext(ctx, `
SELECT COUNT(*)
FROM project_comment
WHERE project_contract = $1
`, contract.Bytes()).Scan(&total); err != nil {
		return nil, 0, 0, 0, fmt.Errorf("count project comments: %w", err)
	}

	offset := int64(page-1) * int64(pageSize)
	rows, err := s.db.QueryContext(ctx, `
SELECT
  id,
  project_contract,
  username,
  content,
  created_at
FROM project_comment
WHERE project_contract = $1
ORDER BY created_at DESC, id DESC
LIMIT $2 OFFSET $3
`, contract.Bytes(), pageSize, offset)
	if err != nil {
		return nil, 0, 0, 0, fmt.Errorf("list project comments: %w", err)
	}
	defer rows.Close()

	items := make([]ProjectComment, 0)
	for rows.Next() {
		item, err := scanProjectCommentRow(rows)
		if err != nil {
			return nil, 0, 0, 0, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, 0, 0, fmt.Errorf("iterate project comments: %w", err)
	}
	return items, total, page, pageSize, nil
}

func normalizeProjectCommentPage(page int32, pageSize int32) (int32, int32) {
	if page < 1 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = defaultProjectCommentPageSize
	}
	if pageSize > maxProjectCommentPageSize {
		pageSize = maxProjectCommentPageSize
	}
	return page, pageSize
}

func scanProjectCommentRow(scanner rowScanner) (ProjectComment, error) {
	var (
		item     ProjectComment
		contract []byte
	)
	if err := scanner.Scan(&item.ID, &contract, &item.Username, &item.Content, &item.CreatedAt); err != nil {
		return ProjectComment{}, fmt.Errorf("scan project comment: %w", err)
	}
	item.Contract = common.BytesToAddress(contract)
	item.CreatedAt = item.CreatedAt.UTC()
	return item, nil
}
