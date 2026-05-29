package store

import (
	"context"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/ethereum/go-ethereum/common"
	appsqlc "github.com/useryege/athena/internal/application/store/sqlc"
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

	if s.queries == nil {
		return ProjectComment{}, fmt.Errorf("application postgres database is not configured")
	}
	row, err := s.queries.AddProjectComment(ctx, appsqlc.AddProjectCommentParams{
		ProjectContract: item.Contract.Bytes(),
		Username:        strings.TrimSpace(item.Username),
		Content:         strings.TrimSpace(item.Content),
	})
	if err != nil {
		return ProjectComment{}, fmt.Errorf("add project comment: %w", err)
	}
	return projectCommentFromSQLC(row), nil
}

func (s *SQLStore) ListProjectCommentsByContract(ctx context.Context, contract common.Address, page int32, pageSize int32) ([]ProjectComment, int64, int32, int32, error) {
	page, pageSize = normalizeProjectCommentPage(page, pageSize)

	if s.queries == nil {
		return nil, 0, 0, 0, fmt.Errorf("application postgres database is not configured")
	}
	total, err := s.queries.CountProjectCommentsByContract(ctx, contract.Bytes())
	if err != nil {
		return nil, 0, 0, 0, fmt.Errorf("count project comments: %w", err)
	}

	offset := int64(page-1) * int64(pageSize)
	rows, err := s.queries.ListProjectCommentsByContract(ctx, appsqlc.ListProjectCommentsByContractParams{
		ProjectContract: contract.Bytes(),
		Limit:           pageSize,
		Offset:          int32(offset),
	})
	if err != nil {
		return nil, 0, 0, 0, fmt.Errorf("list project comments: %w", err)
	}

	items := make([]ProjectComment, 0, len(rows))
	for _, row := range rows {
		items = append(items, projectCommentFromSQLC(row))
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

func projectCommentFromSQLC(row appsqlc.ProjectComment) ProjectComment {
	return ProjectComment{
		ID:        row.ID,
		Contract:  common.BytesToAddress(row.ProjectContract),
		Username:  row.Username,
		Content:   row.Content,
		CreatedAt: row.CreatedAt.Time.UTC(),
	}
}
