package postgres

import (
	"context"

	tokensqlc "github.com/useryege/athena/internal/token/adapters/postgres/sqlc"
	"github.com/useryege/athena/internal/token/research"
)

func (s *Database) ListProjectResearchStatesPage(ctx context.Context, chainID, projectID int64, status string, page, pageSize int32) (*research.ResearchStatePage, error) {
	q, e := s.querier()
	if e != nil {
		return nil, e
	}
	page, pageSize, offset := normalizePage(page, pageSize)
	p := tokensqlc.CountProjectResearchStatesParams{ChainID: chainID, ProjectID: projectID, Status: status}
	total, e := q.CountProjectResearchStates(ctx, p)
	if e != nil {
		return nil, e
	}
	rows, e := q.ListProjectResearchStates(ctx, tokensqlc.ListProjectResearchStatesParams{ChainID: chainID, ProjectID: projectID, Status: status, Offset: offset, Limit: pageSize})
	if e != nil {
		return nil, e
	}
	items := make([]research.ProjectResearchState, 0, len(rows))
	for _, r := range rows {
		items = append(items, mapProjectResearchState(r))
	}
	return &research.ResearchStatePage{Items: items, Total: total, Page: page, PageSize: pageSize}, nil
}
