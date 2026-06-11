package projectreportevaluator

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

func (r *reportEvaluatorRunner) processTask(ctx context.Context, task tokenstore.ProjectReportEvaluationTask) {
	report, err := evaluateProjectReport(task)
	if err == nil {
		var updated bool
		_, updated, err = r.store.CompleteProjectReportEvaluation(ctx, task, report)
		if err == nil {
			if updated {
				log.WithFields(log.Fields{
					"project_id":      task.ProjectID,
					"revision":        task.Revision,
					"has_chain_state": task.HasChainState,
				}).Info("token project report evaluator completed report")
			}
			return
		}
	}
	updatedTask, updated, updateErr := r.store.MarkProjectReportEvaluationTaskFailed(ctx, task.ProjectID, task.Revision, err.Error())
	if updateErr != nil {
		log.WithError(updateErr).WithFields(log.Fields{
			"project_id": task.ProjectID,
			"revision":   task.Revision,
		}).Error("token project report evaluator failed to update task failure")
		return
	}
	if !updated {
		return
	}
	log.WithError(err).WithFields(log.Fields{
		"project_id": task.ProjectID,
		"revision":   task.Revision,
		"attempts":   updatedTask.Attempts,
		"status":     updatedTask.Status,
	}).Warn("token project report evaluator task failed")
}

func evaluateProjectReport(task tokenstore.ProjectReportEvaluationTask) (tokenstore.ProjectReport, error) {
	report := tokenstore.ProjectReport{
		ProjectID:       task.ProjectID,
		SourceUpdatedAt: task.SourceUpdatedAt,
		EvaluatedAt:     time.Now().UTC(),
	}
	if !task.HasChainState {
		return report, nil
	}
	var state athenacontract.AthenaProjectState
	if err := json.Unmarshal(task.ChainState, &state); err != nil {
		return tokenstore.ProjectReport{}, fmt.Errorf("unmarshal ATHENA chain state: %w", err)
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
	return report, nil
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
