package projectreportbuilder

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"math/big"
	"sort"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	log "github.com/sirupsen/logrus"
	"github.com/useryege/athena/internal/token/domain"
	"github.com/useryege/athena/internal/token/telemetry"
)

type Store interface {
	ClaimProjectReportBuildTasks(context.Context, int32) ([]domain.ProjectReportBuildTask, error)
	ListCurrentProjectObservations(context.Context, int64) ([]domain.ProjectObservation, error)
	CompleteProjectReportBuild(context.Context, domain.ProjectReportBuildTask, domain.ProjectReportRevision) (*domain.ProjectReportRevision, bool, error)
	MarkProjectReportBuildTaskFailed(context.Context, domain.ProjectReportBuildTask, string) error
}

type Options struct {
	Store        Store
	PollInterval time.Duration
	TaskLimit    int32
	Telemetry    telemetry.Reporter
}

type Worker struct {
	opts   Options
	mu     sync.Mutex
	cancel context.CancelFunc
	done   chan struct{}
}

func NewWorker(opts Options) *Worker {
	if opts.PollInterval <= 0 {
		opts.PollInterval = time.Second
	}
	if opts.TaskLimit <= 0 {
		opts.TaskLimit = 20
	}
	return &Worker{opts: opts}
}

func (w *Worker) Start(ctx context.Context) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.cancel != nil {
		return nil
	}
	if w.opts.Store == nil {
		return fmt.Errorf("token report builder store is required")
	}
	telemetry.Register(w.opts.Telemetry, telemetry.Scope{Component: "report_builder"})
	runCtx, cancel := context.WithCancel(ctx)
	w.cancel = cancel
	w.done = make(chan struct{})
	go w.run(runCtx)
	return nil
}

func (w *Worker) Stop(ctx context.Context) error {
	w.mu.Lock()
	cancel, done := w.cancel, w.done
	w.cancel = nil
	w.done = nil
	w.mu.Unlock()
	if cancel != nil {
		cancel()
	}
	if done != nil {
		select {
		case <-done:
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	return nil
}

func (w *Worker) run(ctx context.Context) {
	defer close(w.done)
	scope := telemetry.Scope{Component: "report_builder"}
	telemetry.Register(w.opts.Telemetry, scope)
	ticker := time.NewTicker(w.opts.PollInterval)
	defer ticker.Stop()
	for {
		tasks, err := w.opts.Store.ClaimProjectReportBuildTasks(ctx, w.opts.TaskLimit)
		if err != nil {
			telemetry.Failure(w.opts.Telemetry, scope, err)
			log.WithError(err).Error("token report builder claim failed")
		} else {
			for _, task := range tasks {
				w.process(ctx, task)
			}
			telemetry.Success(w.opts.Telemetry, scope)
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func (w *Worker) process(ctx context.Context, task domain.ProjectReportBuildTask) {
	observations, err := w.opts.Store.ListCurrentProjectObservations(ctx, task.ProjectID)
	if err != nil {
		_ = w.opts.Store.MarkProjectReportBuildTaskFailed(ctx, task, err.Error())
		return
	}
	report, err := buildReport(task.ProjectID, observations)
	if err != nil {
		_ = w.opts.Store.MarkProjectReportBuildTaskFailed(ctx, task, err.Error())
		return
	}
	_, created, err := w.opts.Store.CompleteProjectReportBuild(ctx, task, report)
	if err != nil {
		_ = w.opts.Store.MarkProjectReportBuildTaskFailed(ctx, task, err.Error())
		return
	}
	log.WithFields(log.Fields{"project_id": task.ProjectID, "evidence_revision": task.EvidenceRevision, "created": created}).Info("token project report build completed")
}

func buildReport(projectID int64, items []domain.ProjectObservation) (domain.ProjectReportRevision, error) {
	sort.Slice(items, func(i, j int) bool {
		if items[i].DataType != items[j].DataType {
			return items[i].DataType < items[j].DataType
		}
		return items[i].ID < items[j].ID
	})
	evidence := make([]domain.EvidenceReference, 0, len(items))
	freshness := make([]domain.ObservationFreshness, 0, len(items))
	observations := domain.ResearchObservationsV1{}
	var maxBlock *uint64
	for _, item := range items {
		if item.SchemaVersion != domain.ObservationSchemaVersionV1 {
			return domain.ProjectReportRevision{}, fmt.Errorf("unsupported %s observation schema version %d", item.DataType, item.SchemaVersion)
		}
		evidence = append(evidence, domain.EvidenceReference{ObservationID: item.ID, DataType: item.DataType, SchemaVersion: item.SchemaVersion, ContentHash: item.ContentHash.Hex(), BlockNumber: item.BlockNumber})
		freshness = append(freshness, domain.ObservationFreshness{DataType: item.DataType, LastCheckedAt: item.LastCheckedAt})
		if item.BlockNumber != nil && (maxBlock == nil || *item.BlockNumber > *maxBlock) {
			v := *item.BlockNumber
			maxBlock = &v
		}
		if err := decodeObservationV1(&observations, item); err != nil {
			return domain.ProjectReportRevision{}, err
		}
	}

	completeness := "complete"
	if observations.Ave == nil || observations.ChainState == nil || observations.WalletAssets == nil || observations.Simulation == nil || observations.ContractSource == nil {
		completeness = "incomplete"
	}
	risk := riskSummary(observations.ChainState)
	report := domain.ResearchReportV1{SchemaVersion: domain.ReportSchemaVersionV1, ProjectID: projectID, CompletenessStatus: completeness, Observations: observations, Freshness: freshness, RiskSummary: risk}
	canonical, err := json.Marshal(struct {
		SchemaVersion int32                      `json:"schemaVersion"`
		Evidence      []domain.EvidenceReference `json:"evidence"`
		Report        domain.ResearchReportV1    `json:"report"`
	}{SchemaVersion: domain.ReportSchemaVersionV1, Evidence: evidence, Report: report})
	if err != nil {
		return domain.ProjectReportRevision{}, err
	}
	digest := sha256.Sum256(canonical)
	return domain.ProjectReportRevision{
		ProjectID:           projectID,
		SchemaVersion:       domain.ReportSchemaVersionV1,
		ContentHash:         common.BytesToHash(digest[:]),
		CompletenessStatus:  completeness,
		Evidence:            evidence,
		Report:              report,
		ObservedBlockNumber: maxBlock,
		BuiltAt:             time.Now().UTC(),
	}, nil
}

func decodeObservationV1(target *domain.ResearchObservationsV1, item domain.ProjectObservation) error {
	var destination any
	switch item.DataType {
	case domain.DataCollectionTypeAve:
		target.Ave = &domain.AveObservationV1{}
		destination = target.Ave
	case domain.DataCollectionTypeChainState:
		target.ChainState = &domain.ChainStateObservationV1{}
		destination = target.ChainState
	case domain.DataCollectionTypeWalletAssetState:
		target.WalletAssets = &domain.WalletAssetObservationV1{}
		destination = target.WalletAssets
	case domain.DataCollectionTypeSimulationResult:
		target.Simulation = &domain.SimulationObservationV1{}
		destination = target.Simulation
	case domain.DataCollectionTypeContractCodeSource:
		target.ContractSource = &domain.ContractSourceObservationV1{}
		destination = target.ContractSource
	default:
		return fmt.Errorf("unsupported observation data type %q", item.DataType)
	}
	if err := json.Unmarshal(item.Payload, destination); err != nil {
		return fmt.Errorf("decode %s observation schema version %d: %w", item.DataType, item.SchemaVersion, err)
	}
	return nil
}

func riskSummary(state *domain.ChainStateObservationV1) domain.ReportRiskSummary {
	if state == nil {
		return domain.ReportRiskSummary{}
	}
	return domain.ReportRiskSummary{
		WethPairIsCreated:         boolPtr(state.WethPair.IsCreated),
		WethPairIsRemoveLiquidity: boolPtr(state.WethReport.IsRemoveLiquidity),
		WethPairIsMint:            boolPtr(state.WethReport.IsMint),
		WethPairQuoteUsdtValueInt: cloneBigInt(state.WethPair.QuoteUsdtValueInt),
		WethPairLastSwapTimestamp: uint64Ptr(uint64(state.WethPair.LastSwapTimestamp)),
		UsdtPairIsCreated:         boolPtr(state.UsdtPair.IsCreated),
		UsdtPairIsRemoveLiquidity: boolPtr(state.UsdtReport.IsRemoveLiquidity),
		UsdtPairIsMint:            boolPtr(state.UsdtReport.IsMint),
		UsdtPairQuoteUsdtValueInt: cloneBigInt(state.UsdtPair.QuoteUsdtValueInt),
		UsdtPairLastSwapTimestamp: uint64Ptr(uint64(state.UsdtPair.LastSwapTimestamp)),
	}
}

func boolPtr(v bool) *bool       { return &v }
func uint64Ptr(v uint64) *uint64 { return &v }
func cloneBigInt(v *big.Int) *big.Int {
	if v == nil {
		return new(big.Int)
	}
	return new(big.Int).Set(v)
}
