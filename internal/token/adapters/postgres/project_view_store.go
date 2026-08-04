package postgres

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	tokensqlc "github.com/useryege/athena/internal/token/adapters/postgres/sqlc"
	"github.com/useryege/athena/internal/token/projectview"
	"github.com/useryege/athena/internal/token/research"
	"github.com/useryege/athena/internal/token/selection"
	"github.com/useryege/athena/internal/token/shared"
)

func (repository *ProjectViewRepository) GetProjectDetail(ctx context.Context, projectID int64) (*projectview.Detail, error) {
	queries, err := repository.querier()
	if err != nil {
		return nil, err
	}
	projectRow, err := queries.GetProject(ctx, projectID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get project detail project: %w", err)
	}
	project, err := mapProject(projectRow)
	if err != nil {
		return nil, fmt.Errorf("map project detail project: %w", err)
	}
	detail := &projectview.Detail{Project: *project}

	researchRows, err := queries.ListProjectResearchStates(ctx, tokensqlc.ListProjectResearchStatesParams{
		ProjectID: projectID,
		Limit:     1,
	})
	if err != nil {
		return nil, fmt.Errorf("list project detail research state: %w", err)
	}
	if len(researchRows) > 0 {
		state := mapProjectResearchState(researchRows[0])
		detail.ResearchState = &state
		if state.CurrentReportRevision > 0 {
			reportRow, reportErr := queries.GetProjectReportRevision(ctx, tokensqlc.GetProjectReportRevisionParams{
				ProjectID: projectID,
				Revision:  state.CurrentReportRevision,
			})
			if reportErr != nil && !errors.Is(reportErr, pgx.ErrNoRows) {
				return nil, fmt.Errorf("get project detail current report: %w", reportErr)
			}
			if reportErr == nil {
				report, mapErr := mapProjectReportRevision(reportRow)
				if mapErr != nil {
					return nil, fmt.Errorf("map project detail current report: %w", mapErr)
				}
				report.ChainID = project.ChainID
				report.Contract = project.Contract
				detail.CurrentReport = &report

				evaluationRow, evaluationErr := queries.GetProjectSelectionEvaluationTask(ctx, tokensqlc.GetProjectSelectionEvaluationTaskParams{
					ProjectID:      projectID,
					ReportRevision: state.CurrentReportRevision,
				})
				if evaluationErr != nil && !errors.Is(evaluationErr, pgx.ErrNoRows) {
					return nil, fmt.Errorf("get project detail current report evaluation: %w", evaluationErr)
				}
				if evaluationErr == nil {
					detail.CurrentReportEvaluation = mapProjectSelectionEvaluationTask(evaluationRow)
				}
			}
		}
		if state.CurrentSelectionID > 0 {
			selectionRow, selectionErr := queries.GetProjectSelectionByID(ctx, tokensqlc.GetProjectSelectionByIDParams{
				ID:        state.CurrentSelectionID,
				ProjectID: projectID,
			})
			if selectionErr != nil && !errors.Is(selectionErr, pgx.ErrNoRows) {
				return nil, fmt.Errorf("get project detail current selection: %w", selectionErr)
			}
			if selectionErr == nil {
				currentSelection := mapProjectSelection(selectionRow)
				currentSelection.ChainID = project.ChainID
				currentSelection.Contract = project.Contract
				detail.CurrentSelection = &currentSelection
			}
		}
		if detail.CurrentReportEvaluation != nil &&
			detail.CurrentReportEvaluation.Status == selection.TaskStatusSucceeded &&
			state.LastEvaluatedReportRevision == state.CurrentReportRevision &&
			detail.CurrentSelection != nil {
			detail.CurrentReportEvaluation.Outcome = detail.CurrentSelection.Outcome
			detail.CurrentReportEvaluation.EvaluatedAt = state.LastEvaluatedAt
		}
	}

	observationRows, err := queries.ListCurrentProjectObservations(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("list project detail current observations: %w", err)
	}
	detail.CurrentObservations, err = mapCurrentProjectObservations(observationRows)
	if err != nil {
		return nil, fmt.Errorf("map project detail current observations: %w", err)
	}

	relatedWalletRows, err := queries.ListProjectRelatedWalletsByProject(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("list project detail related wallets: %w", err)
	}
	detail.RelatedWallets = mapProjectRelatedWallets(relatedWalletRows)

	initialRecipientRows, err := queries.ListProjectInitialRecipientsByProject(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("list project detail initial recipients: %w", err)
	}
	detail.InitialRecipients, err = mapProjectInitialRecipients(initialRecipientRows)
	if err != nil {
		return nil, fmt.Errorf("map project detail initial recipients: %w", err)
	}

	scheduleRows, err := queries.ListProjectDataCollectionSchedulesByProject(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("list project detail collection schedules: %w", err)
	}
	detail.CollectionSchedules = make([]research.ProjectDataCollectionSchedule, 0, len(scheduleRows))
	for _, scheduleRow := range scheduleRows {
		detail.CollectionSchedules = append(detail.CollectionSchedules, mapProjectDataCollectionSchedule(scheduleRow))
	}

	detail.TransactionCount, err = queries.CountProjectWalletNormalTransactions(ctx, tokensqlc.CountProjectWalletNormalTransactionsParams{
		ProjectID: projectID,
	})
	if err != nil {
		return nil, fmt.Errorf("count project detail wallet transactions: %w", err)
	}
	countRows, err := queries.CountProjectWalletNormalTransactionsByWallet(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("count project detail wallet transactions by wallet: %w", err)
	}
	detail.WalletTransactionCounts = make([]projectview.WalletTransactionCount, 0, len(countRows))
	for _, countRow := range countRows {
		detail.WalletTransactionCounts = append(detail.WalletTransactionCounts, projectview.WalletTransactionCount{
			Wallet:           bytesToAddress(countRow.Wallet),
			TransactionCount: countRow.TransactionCount,
		})
	}
	return detail, nil
}

func (repository *ProjectViewRepository) ListProjectObservationsPage(
	ctx context.Context,
	projectID int64,
	dataType string,
	page, pageSize int32,
) (*projectview.ObservationPage, error) {
	queries, err := repository.querier()
	if err != nil {
		return nil, err
	}
	page, pageSize, offset := normalizePage(page, pageSize)
	dataType = strings.TrimSpace(dataType)
	total, err := queries.CountProjectObservations(ctx, tokensqlc.CountProjectObservationsParams{
		ProjectID: projectID,
		DataType:  dataType,
	})
	if err != nil {
		return nil, fmt.Errorf("count project observations: %w", err)
	}
	rows, err := queries.ListProjectObservations(ctx, tokensqlc.ListProjectObservationsParams{
		ProjectID: projectID,
		DataType:  dataType,
		Offset:    offset,
		Limit:     pageSize,
	})
	if err != nil {
		return nil, fmt.Errorf("list project observations: %w", err)
	}
	items := make([]research.ProjectObservation, 0, len(rows))
	for _, row := range rows {
		item, mapErr := mapProjectObservation(row)
		if mapErr != nil {
			return nil, fmt.Errorf("map project observation: %w", mapErr)
		}
		items = append(items, item)
	}
	return &projectview.ObservationPage{Items: items, Total: total, Page: page, PageSize: pageSize}, nil
}

func (repository *ProjectViewRepository) ListProjectTrendObservations(ctx context.Context, projectID int64, observedFrom time.Time) ([]research.ProjectObservation, error) {
	queries, err := repository.querier()
	if err != nil {
		return nil, err
	}
	rows, err := queries.ListProjectTrendObservations(ctx, tokensqlc.ListProjectTrendObservationsParams{
		ProjectID:    projectID,
		ObservedFrom: nullableTime(observedFrom),
	})
	if err != nil {
		return nil, fmt.Errorf("list project trend observations: %w", err)
	}
	items := make([]research.ProjectObservation, 0, len(rows))
	for _, row := range rows {
		item, mapErr := mapProjectObservation(row)
		if mapErr != nil {
			return nil, fmt.Errorf("map project trend observation: %w", mapErr)
		}
		items = append(items, item)
	}
	return items, nil
}

func (repository *ProjectViewRepository) ListProjectWalletNormalTransactionsPage(
	ctx context.Context,
	projectID int64,
	wallet shared.Address,
	receiptStatus string,
	methodID string,
	page, pageSize int32,
) (*projectview.WalletNormalTransactionPage, error) {
	queries, err := repository.querier()
	if err != nil {
		return nil, err
	}
	page, pageSize, offset := normalizePage(page, pageSize)
	receiptStatus = strings.TrimSpace(receiptStatus)
	methodID = strings.TrimSpace(methodID)
	params := tokensqlc.CountProjectWalletNormalTransactionsParams{
		ProjectID:     projectID,
		Wallet:        optionalAddressBytes(wallet),
		ReceiptStatus: receiptStatus,
		MethodID:      methodID,
	}
	total, err := queries.CountProjectWalletNormalTransactions(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("count project wallet normal transactions: %w", err)
	}
	rows, err := queries.ListProjectWalletNormalTransactions(ctx, tokensqlc.ListProjectWalletNormalTransactionsParams{
		ProjectID:     projectID,
		Wallet:        params.Wallet,
		ReceiptStatus: receiptStatus,
		MethodID:      methodID,
		Offset:        offset,
		Limit:         pageSize,
	})
	if err != nil {
		return nil, fmt.Errorf("list project wallet normal transactions: %w", err)
	}
	items := make([]research.WalletNormalTransaction, 0, len(rows))
	for _, row := range rows {
		item, mapErr := mapWalletNormalTransaction(row)
		if mapErr != nil {
			return nil, fmt.Errorf("map project wallet normal transaction: %w", mapErr)
		}
		items = append(items, item)
	}
	return &projectview.WalletNormalTransactionPage{Items: items, Total: total, Page: page, PageSize: pageSize}, nil
}

func mapWalletNormalTransaction(row tokensqlc.ProjectWalletNormalTransaction) (research.WalletNormalTransaction, error) {
	blockNumber, err := int64ToUint64("block_number", row.BlockNumber)
	if err != nil {
		return research.WalletNormalTransaction{}, err
	}
	blockTimestamp, err := int64ToUint64("block_timestamp", row.BlockTimestamp)
	if err != nil {
		return research.WalletNormalTransaction{}, err
	}
	transactionIndex, err := int64ToUint64("transaction_index", row.TransactionIndex)
	if err != nil {
		return research.WalletNormalTransaction{}, err
	}
	nonce, err := int64ToUint64("nonce", row.Nonce)
	if err != nil {
		return research.WalletNormalTransaction{}, err
	}
	gas, err := int64ToUint64("gas", row.Gas)
	if err != nil {
		return research.WalletNormalTransaction{}, err
	}
	gasUsed, err := int64ToUint64("gas_used", row.GasUsed)
	if err != nil {
		return research.WalletNormalTransaction{}, err
	}
	return research.WalletNormalTransaction{
		Wallet:           bytesToAddress(row.Wallet),
		TransactionHash:  bytesToHash(row.TransactionHash),
		BlockNumber:      blockNumber,
		BlockTimestamp:   blockTimestamp,
		TransactionIndex: transactionIndex,
		Nonce:            nonce,
		FromAddress:      bytesToAddress(row.FromAddress),
		ToAddress:        bytesToAddress(row.ToAddress),
		Value:            bigIntFromNumeric(row.Value),
		Gas:              gas,
		GasPrice:         bigIntFromNumeric(row.GasPrice),
		GasUsed:          gasUsed,
		Input:            row.Input,
		MethodID:         row.MethodID,
		FunctionName:     row.FunctionName,
		ReceiptStatus:    research.NormalTransactionReceiptStatus(row.ReceiptStatus),
		IsError:          row.IsError,
		CollectedAt:      timeValue(row.CollectedAt),
	}, nil
}
