package projectreportbuilder

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"math/big"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	log "github.com/sirupsen/logrus"
	tokenstore "github.com/useryege/athena/internal/token/store"
	athenacontract "github.com/useryege/athena/pkg/abi/ATHENA"
)

type Options struct {
	Store        *tokenstore.SQLStore
	PollInterval time.Duration
	TaskLimit    int32
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
	ticker := time.NewTicker(w.opts.PollInterval)
	defer ticker.Stop()
	for {
		tasks, err := w.opts.Store.ClaimProjectReportBuildTasks(ctx, w.opts.TaskLimit)
		if err != nil {
			log.WithError(err).Error("token report builder claim failed")
		} else {
			for _, task := range tasks {
				w.process(ctx, task)
			}
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}
func (w *Worker) process(ctx context.Context, task tokenstore.ProjectReportBuildTask) {
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

type evidenceItem struct {
	ObservationID int64   `json:"observationId"`
	DataType      string  `json:"dataType"`
	ContentHash   string  `json:"contentHash"`
	BlockNumber   *uint64 `json:"blockNumber,omitempty"`
}
type freshnessItem struct {
	DataType      string    `json:"dataType"`
	LastCheckedAt time.Time `json:"lastCheckedAt"`
}
type normalizedReport struct {
	ProjectID    int64                      `json:"projectId"`
	Observations map[string]json.RawMessage `json:"observations"`
	Freshness    []freshnessItem            `json:"freshness"`
}

func buildReport(projectID int64, items []tokenstore.ProjectObservation) (tokenstore.ProjectReportRevision, error) {
	evidence := make([]evidenceItem, 0, len(items))
	freshness := make([]freshnessItem, 0, len(items))
	payloads := make(map[string]json.RawMessage, len(items))
	present := make(map[string]bool, len(items))
	var maxBlock *uint64
	for _, item := range items {
		evidence = append(evidence, evidenceItem{ObservationID: item.ID, DataType: item.DataType, ContentHash: item.ContentHash.Hex(), BlockNumber: item.BlockNumber})
		freshness = append(freshness, freshnessItem{DataType: item.DataType, LastCheckedAt: item.LastCheckedAt})
		payloads[item.DataType] = item.Payload
		present[item.DataType] = true
		if item.BlockNumber != nil && (maxBlock == nil || *item.BlockNumber > *maxBlock) {
			v := *item.BlockNumber
			maxBlock = &v
		}
	}
	evidenceJSON, err := json.Marshal(evidence)
	if err != nil {
		return tokenstore.ProjectReportRevision{}, err
	}
	reportJSON, err := json.Marshal(normalizedReport{ProjectID: projectID, Observations: payloads, Freshness: freshness})
	if err != nil {
		return tokenstore.ProjectReportRevision{}, err
	}
	combined, _ := json.Marshal(struct {
		Evidence json.RawMessage `json:"evidence"`
		Report   json.RawMessage `json:"report"`
	}{evidenceJSON, reportJSON})
	digest := sha256.Sum256(combined)
	completeness := "complete"
	for _, required := range []string{tokenstore.ProjectDataCollectionTypeAve, tokenstore.ProjectDataCollectionTypeChainState, tokenstore.ProjectDataCollectionTypeWalletAssetState, tokenstore.ProjectDataCollectionTypeSimulationResult, tokenstore.ProjectDataCollectionTypeContractCodeSource} {
		if !present[required] {
			completeness = "incomplete"
			break
		}
	}
	result := tokenstore.ProjectReportRevision{ProjectID: projectID, ContentHash: common.BytesToHash(digest[:]), CompletenessStatus: completeness, Evidence: evidenceJSON, Report: reportJSON, ObservedBlockNumber: maxBlock, BuiltAt: time.Now().UTC()}
	if chainPayload := payloads[tokenstore.ProjectDataCollectionTypeChainState]; len(chainPayload) > 0 {
		if err = applyRiskSummary(&result, chainPayload); err != nil {
			return tokenstore.ProjectReportRevision{}, err
		}
	}
	return result, nil
}
func applyRiskSummary(report *tokenstore.ProjectReportRevision, payload []byte) error {
	var state athenacontract.AthenaProjectState
	if err := json.Unmarshal(payload, &state); err != nil {
		return fmt.Errorf("unmarshal chain state: %w", err)
	}
	report.WethPairIsCreated = boolPtr(state.WethPair.IsCreated)
	report.WethPairIsRemoveLiquidity = boolPtr(state.WethReport.IsRemoveLiquidity)
	report.WethPairIsMint = boolPtr(state.WethReport.IsMint)
	report.WethPairQuoteUsdtValueInt = bigInt(state.WethPair.QuoteUsdtValueInt)
	report.WethPairLastSwapTimestamp = uint64Ptr(uint64(state.WethPair.LastSwapTimestamp))
	report.UsdtPairIsCreated = boolPtr(state.UsdtPair.IsCreated)
	report.UsdtPairIsRemoveLiquidity = boolPtr(state.UsdtReport.IsRemoveLiquidity)
	report.UsdtPairIsMint = boolPtr(state.UsdtReport.IsMint)
	report.UsdtPairQuoteUsdtValueInt = bigInt(state.UsdtPair.QuoteUsdtValueInt)
	report.UsdtPairLastSwapTimestamp = uint64Ptr(uint64(state.UsdtPair.LastSwapTimestamp))
	return nil
}
func boolPtr(v bool) *bool       { return &v }
func uint64Ptr(v uint64) *uint64 { return &v }
func bigInt(v *big.Int) *big.Int {
	if v == nil {
		return new(big.Int)
	}
	return new(big.Int).Set(v)
}
