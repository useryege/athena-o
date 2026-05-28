package policy

import (
	"context"

	"github.com/ethereum/go-ethereum/common"
	"github.com/useryege/athena/internal/application/persistence"
	appstore "github.com/useryege/athena/internal/application/store"
)

const projectEventTypePolicyMatchAudit = 5

func projectEventIdempotencyPolicyMatch(rule string) string {
	return "project_policy_matched:" + rule
}

type persistencePublisherFake struct {
	projectReports   map[common.Address]ProjectReport
	projectEventLogs []appstore.ProjectEventLog
}

func (p *persistencePublisherFake) Publish(context.Context, persistence.PersistenceEvent) error {
	return nil
}

func (p *persistencePublisherFake) PublishProjectMetaSave(context.Context, appstore.ProjectMeta) error {
	return nil
}

func (p *persistencePublisherFake) PublishProjectEventLog(_ context.Context, item appstore.ProjectEventLog) error {
	p.projectEventLogs = append(p.projectEventLogs, item)
	return nil
}

func (p *persistencePublisherFake) PublishProjectAveDetailUpsert(context.Context, common.Address, appstore.ProjectAveDetail) error {
	return nil
}

func (p *persistencePublisherFake) PublishProjectCreatorResultUpdate(context.Context, common.Address, SimulateResult) error {
	return nil
}

func (p *persistencePublisherFake) PublishProjectReportUpdate(_ context.Context, contract common.Address, report ProjectReport) error {
	if p.projectReports == nil {
		p.projectReports = map[common.Address]ProjectReport{}
	}
	p.projectReports[contract] = report
	return nil
}

func (p *persistencePublisherFake) PublishProjectCreatorHistoricalProjectsReplace(context.Context, common.Address, []appstore.ProjectCreatorHistoricalProject) error {
	return nil
}

func paginateProjects(page int32, pageSize int32, projects *[]*Project) (int64, int32, int32) {
	normalizedPage, normalizedPageSize := normalizeProjectPage(page, pageSize)
	total := int64(len(*projects))
	start := int64(normalizedPage-1) * int64(normalizedPageSize)
	if start >= total {
		*projects = (*projects)[:0]
		return total, normalizedPage, normalizedPageSize
	}

	stop := start + int64(normalizedPageSize)
	if stop > total {
		stop = total
	}

	*projects = (*projects)[start:stop]
	return total, normalizedPage, normalizedPageSize
}

func normalizeProjectPage(page int32, pageSize int32) (int32, int32) {
	if page < 1 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > 200 {
		pageSize = 200
	}
	return page, pageSize
}
