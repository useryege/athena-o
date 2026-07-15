package store

import (
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/useryege/athena/internal/token/domain"
)

type ProjectDataCollectionTaskPage struct {
	Items    []domain.ProjectDataCollectionTask
	Total    int64
	Page     int32
	PageSize int32
}

type ProjectResearchStatePage struct {
	Items    []domain.ProjectResearchState
	Total    int64
	Page     int32
	PageSize int32
}

type ProjectReportRevisionPage struct {
	Items    []domain.ProjectReportRevision
	Total    int64
	Page     int32
	PageSize int32
}

type ProjectReportListItem struct {
	Report         domain.ProjectReportRevision
	ChainID        int64
	Name           string
	Symbol         string
	Contract       common.Address
	BuildStatus    string
	BuildAttempts  int32
	BuildLastError string
	BuildUpdatedAt time.Time
}

type ProjectReportPage struct {
	Items    []ProjectReportListItem
	Total    int64
	Page     int32
	PageSize int32
}

type ProjectSelectionPage struct {
	Items    []domain.ProjectSelection
	Total    int64
	Page     int32
	PageSize int32
}

type ProjectCandidatePage struct {
	Items    []domain.ProjectCandidate
	Total    int64
	Page     int32
	PageSize int32
}

type ContractCodePage struct {
	Items    []domain.ContractCode
	Total    int64
	Page     int32
	PageSize int32
}

type ProjectPage struct {
	Items    []domain.Project
	Total    int64
	Page     int32
	PageSize int32
}
