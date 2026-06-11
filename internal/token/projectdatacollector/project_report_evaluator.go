package projectdatacollector

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"time"

	log "github.com/sirupsen/logrus"
	tokenstore "github.com/useryege/athena/internal/token/store"
	athenacontract "github.com/useryege/athena/pkg/abi/ATHENA"
)

func (r *dataCollectorRunner) processProjectReports(ctx context.Context) error {
	candidates, err := r.opts.store.ListProjectReportsDueForEvaluation(ctx, projectReportTaskLimit)
	if err != nil {
		return err
	}
	if len(candidates) == 0 {
		log.Debug("token project data collector has no project reports due for evaluation")
		return nil
	}
	for _, candidate := range candidates {
		if err := ctx.Err(); err != nil {
			return err
		}
		if err := r.evaluateProjectReport(ctx, candidate); err != nil {
			log.WithError(err).WithField("project_id", candidate.ProjectID).Error("token project data collector failed to evaluate project report")
		}
	}
	log.WithField("report_count", len(candidates)).Debug("token project data collector processed project report batch")
	return nil
}

func (r *dataCollectorRunner) evaluateProjectReport(ctx context.Context, candidate tokenstore.ProjectReportEvaluationCandidate) error {
	report := tokenstore.ProjectReport{
		ProjectID:       candidate.ProjectID,
		IsComplete:      candidate.IsComplete,
		SourceUpdatedAt: candidate.SourceUpdatedAt,
		EvaluatedAt:     time.Now().UTC(),
	}
	if candidate.HasChainState {
		var state athenacontract.AthenaProjectState
		if err := json.Unmarshal(candidate.ChainState, &state); err != nil {
			return fmt.Errorf("unmarshal ATHENA chain state: %w", err)
		}
		report.WethPairIsCreated = boolPointer(state.WethPair.IsCreated)
		report.WethPairIsRemoveLiquidity = boolPointer(state.WethReport.IsRemoveLiquidity)
		report.WethPairIsMint = boolPointer(state.WethReport.IsMint)
		report.WethPairQuoteUsdtValueInt = bigIntOrZero(state.WethPair.QuoteUsdtValueInt)
		report.WethPairLastSwapTimestamp = uint64Pointer(uint64(state.WethPair.LastSwapTimestamp))
		report.UsdtPairIsCreated = boolPointer(state.UsdtPair.IsCreated)
		report.UsdtPairIsRemoveLiquidity = boolPointer(state.UsdtReport.IsRemoveLiquidity)
		report.UsdtPairIsMint = boolPointer(state.UsdtReport.IsMint)
		report.UsdtPairQuoteUsdtValueInt = bigIntOrZero(state.UsdtPair.QuoteUsdtValueInt)
		report.UsdtPairLastSwapTimestamp = uint64Pointer(uint64(state.UsdtPair.LastSwapTimestamp))
	}
	if _, err := r.opts.store.UpdateProjectReportEvaluation(ctx, report); err != nil {
		return err
	}
	log.WithFields(log.Fields{
		"project_id":      candidate.ProjectID,
		"is_complete":     candidate.IsComplete,
		"has_chain_state": candidate.HasChainState,
	}).Info("token project data collector evaluated project report")
	return nil
}

func boolPointer(value bool) *bool {
	return &value
}

func uint64Pointer(value uint64) *uint64 {
	return &value
}

func bigIntOrZero(value *big.Int) *big.Int {
	if value == nil {
		return new(big.Int)
	}
	return new(big.Int).Set(value)
}
