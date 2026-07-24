package projectview

import (
	"time"

	"github.com/useryege/athena/internal/token/catalog"
	"github.com/useryege/athena/internal/token/reporting"
	"github.com/useryege/athena/internal/token/research"
	"github.com/useryege/athena/internal/token/selection"
	"github.com/useryege/athena/internal/token/shared"
)

type WalletTransactionCount struct {
	Wallet           shared.Address
	TransactionCount int64
}

type Detail struct {
	Project                 catalog.Project
	ResearchState           *research.ProjectResearchState
	CurrentReport           *reporting.ProjectReportRevision
	CurrentSelection        *selection.ProjectSelection
	CurrentObservations     []research.ProjectObservation
	RelatedWallets          []catalog.ProjectRelatedWallet
	InitialRecipients       []catalog.ProjectInitialRecipient
	CollectionSchedules     []research.ProjectDataCollectionSchedule
	WalletTransactionCounts []WalletTransactionCount
	TransactionCount        int64
}

type ObservationPage struct {
	Items    []research.ProjectObservation
	Total    int64
	Page     int32
	PageSize int32
}

type WalletNormalTransactionPage struct {
	Items    []research.WalletNormalTransaction
	Total    int64
	Page     int32
	PageSize int32
}

type TrendPoint struct {
	ObservedAt time.Time
	Value      string
}

type TrendSeries struct {
	Key      string
	Label    string
	Unit     string
	DataType research.DataCollectionType
	Points   []TrendPoint
}

type TrendResult struct {
	Range        string
	ObservedFrom time.Time
	GeneratedAt  time.Time
	Series       []TrendSeries
}
