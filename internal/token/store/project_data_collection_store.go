package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/jackc/pgx/v5"
	tokensqlc "github.com/useryege/athena/internal/token/store/sqlc"
)

func (s *SQLStore) ListDueProjectDataCollectionTasks(ctx context.Context, dataType string, chainIDs []int64, limit int32) ([]ProjectDataCollectionTaskWithProject, error) {
	if len(chainIDs) == 0 {
		return nil, nil
	}
	q, err := s.querier()
	if err != nil {
		return nil, err
	}
	rows, err := q.ListDueProjectDataCollectionTasks(ctx, tokensqlc.ListDueProjectDataCollectionTasksParams{
		DataType: dataType,
		ChainIds: chainIDs,
		Limit:    limit,
	})
	if err != nil {
		return nil, fmt.Errorf("list due project data collection tasks: %w", err)
	}
	items, err := mapDueProjectDataCollectionTasks(rows)
	if err != nil {
		return nil, fmt.Errorf("map due project data collection tasks: %w", err)
	}
	return items, nil
}

func (s *SQLStore) GetProjectDataCollectionTask(ctx context.Context, projectID int64, dataType string) (*ProjectDataCollectionTask, error) {
	q, err := s.querier()
	if err != nil {
		return nil, err
	}
	row, err := q.GetProjectDataCollectionTask(ctx, tokensqlc.GetProjectDataCollectionTaskParams{
		ProjectID: projectID,
		DataType:  dataType,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get project data collection task: %w", err)
	}
	return mapProjectDataCollectionTask(row), nil
}

func (s *SQLStore) ListProjectDataCollectionTasks(ctx context.Context, projectID int64, dataType, status string, page, pageSize int32) (*ProjectDataCollectionTaskPage, error) {
	q, err := s.querier()
	if err != nil {
		return nil, err
	}
	page, pageSize, offset := normalizePage(page, pageSize)
	projectIDFilter := nullableInt64(projectID)
	dataTypeFilter := nullableText(dataType)
	statusFilter := nullableText(status)
	total, err := q.CountProjectDataCollectionTasks(ctx, tokensqlc.CountProjectDataCollectionTasksParams{
		ProjectID: projectIDFilter,
		DataType:  dataTypeFilter,
		Status:    statusFilter,
	})
	if err != nil {
		return nil, fmt.Errorf("count project data collection tasks: %w", err)
	}
	rows, err := q.ListProjectDataCollectionTasks(ctx, tokensqlc.ListProjectDataCollectionTasksParams{
		ProjectID: projectIDFilter,
		DataType:  dataTypeFilter,
		Status:    statusFilter,
		Offset:    offset,
		Limit:     pageSize,
	})
	if err != nil {
		return nil, fmt.Errorf("list project data collection tasks: %w", err)
	}
	items := make([]ProjectDataCollectionTask, 0, len(rows))
	for _, row := range rows {
		items = append(items, *mapProjectDataCollectionTask(row))
	}
	return &ProjectDataCollectionTaskPage{
		Items:    items,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}, nil
}

func (s *SQLStore) MarkProjectDataCollectionTaskSucceeded(ctx context.Context, projectID int64, dataType string) (*ProjectDataCollectionTask, error) {
	q, err := s.querier()
	if err != nil {
		return nil, err
	}
	row, err := q.MarkProjectDataCollectionTaskSucceeded(ctx, tokensqlc.MarkProjectDataCollectionTaskSucceededParams{
		ProjectID: projectID,
		DataType:  dataType,
	})
	if err != nil {
		return nil, fmt.Errorf("mark project data collection task succeeded: %w", err)
	}
	return mapProjectDataCollectionTask(row), nil
}

func (s *SQLStore) MarkProjectDataCollectionTaskFailed(ctx context.Context, projectID int64, dataType, lastError string) (*ProjectDataCollectionTask, error) {
	q, err := s.querier()
	if err != nil {
		return nil, err
	}
	row, err := q.MarkProjectDataCollectionTaskFailed(ctx, tokensqlc.MarkProjectDataCollectionTaskFailedParams{
		ProjectID: projectID,
		DataType:  dataType,
		LastError: nullableText(lastError),
	})
	if err != nil {
		return nil, fmt.Errorf("mark project data collection task failed: %w", err)
	}
	return mapProjectDataCollectionTask(row), nil
}

func (s *SQLStore) CompleteProjectAveDataCollection(ctx context.Context, projectID int64, payload json.RawMessage, fetchedAt time.Time) (*ProjectAveData, error) {
	if s == nil || s.pool == nil {
		return nil, fmt.Errorf("token postgres database is not configured")
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin project ave data collection transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()
	q := tokensqlc.New(tx)
	row, err := q.UpsertProjectAveData(ctx, tokensqlc.UpsertProjectAveDataParams{
		ProjectID:   projectID,
		AveResponse: []byte(payload),
		FetchedAt:   nullableTime(fetchedAt),
	})
	if err != nil {
		return nil, fmt.Errorf("upsert project ave data: %w", err)
	}
	if _, err := q.MarkProjectDataCollectionTaskSucceeded(ctx, tokensqlc.MarkProjectDataCollectionTaskSucceededParams{
		ProjectID: projectID,
		DataType:  ProjectDataCollectionTypeAve,
	}); err != nil {
		return nil, fmt.Errorf("mark project ave data collection succeeded: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit project ave data collection transaction: %w", err)
	}
	return mapProjectAveData(row), nil
}

func (s *SQLStore) CompleteProjectContractCodeSourceCollection(ctx context.Context, projectID int64, codeHash common.Hash, sourceCode string, fetchedAt time.Time) error {
	if s == nil || s.pool == nil {
		return fmt.Errorf("token postgres database is not configured")
	}
	if fetchedAt.IsZero() {
		fetchedAt = time.Now().UTC()
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin project contract code source collection transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()
	q := tokensqlc.New(tx)
	if _, err := q.UpdateContractCodeSource(ctx, tokensqlc.UpdateContractCodeSourceParams{
		CodeHash:            codeHash.Bytes(),
		SourceCode:          nullableText(sourceCode),
		SourceCodeFetchedAt: nullableTime(fetchedAt),
	}); err != nil {
		return fmt.Errorf("update contract code source %s: %w", codeHash.Hex(), err)
	}
	if _, err := q.MarkProjectDataCollectionTaskSucceeded(ctx, tokensqlc.MarkProjectDataCollectionTaskSucceededParams{
		ProjectID: projectID,
		DataType:  ProjectDataCollectionTypeContractCodeSource,
	}); err != nil {
		return fmt.Errorf("mark project contract code source collection succeeded: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit project contract code source collection transaction: %w", err)
	}
	return nil
}

func (s *SQLStore) CompleteProjectChainStateCollection(ctx context.Context, projectID int64, payload json.RawMessage, fetchedAt time.Time) (*ProjectChainStateData, error) {
	if s == nil || s.pool == nil {
		return nil, fmt.Errorf("token postgres database is not configured")
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin project chain state collection transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()
	q := tokensqlc.New(tx)
	row, err := q.UpsertProjectChainState(ctx, tokensqlc.UpsertProjectChainStateParams{
		ProjectID:  projectID,
		ChainState: []byte(payload),
		FetchedAt:  nullableTime(fetchedAt),
	})
	if err != nil {
		return nil, fmt.Errorf("upsert project chain state: %w", err)
	}
	if _, err := q.MarkProjectDataCollectionTaskSucceeded(ctx, tokensqlc.MarkProjectDataCollectionTaskSucceededParams{
		ProjectID: projectID,
		DataType:  ProjectDataCollectionTypeChainState,
	}); err != nil {
		return nil, fmt.Errorf("mark project chain state collection succeeded: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit project chain state collection transaction: %w", err)
	}
	return mapProjectChainStateData(row), nil
}

func (s *SQLStore) CompleteProjectWalletAssetStateCollection(ctx context.Context, projectID int64, states []WalletAssetState, fetchedAt time.Time) error {
	if s == nil || s.pool == nil {
		return fmt.Errorf("token postgres database is not configured")
	}
	if fetchedAt.IsZero() {
		fetchedAt = time.Now().UTC()
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin project wallet asset state collection transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()
	q := tokensqlc.New(tx)
	for _, state := range states {
		if state.Wallet == (common.Address{}) {
			continue
		}
		if state.FetchedAt.IsZero() {
			state.FetchedAt = fetchedAt
		}
		if _, err := q.UpsertWalletAssetState(ctx, tokensqlc.UpsertWalletAssetStateParams{
			ChainID:       state.ChainID,
			Wallet:        state.Wallet.Bytes(),
			WethBalance:   numericFromBigInt(state.WethBalance),
			UsdtBalance:   numericFromBigInt(state.UsdtBalance),
			NativeBalance: numericFromBigInt(state.NativeBalance),
			UsdtValue:     numericFromBigInt(state.UsdtValue),
			FetchedAt:     nullableTime(state.FetchedAt),
		}); err != nil {
			return fmt.Errorf("upsert wallet asset state %s: %w", state.Wallet.Hex(), err)
		}
	}
	if _, err := q.MarkProjectDataCollectionTaskSucceeded(ctx, tokensqlc.MarkProjectDataCollectionTaskSucceededParams{
		ProjectID: projectID,
		DataType:  ProjectDataCollectionTypeWalletAssetState,
	}); err != nil {
		return fmt.Errorf("mark project wallet asset state collection succeeded: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit project wallet asset state collection transaction: %w", err)
	}
	return nil
}

func (s *SQLStore) CompleteProjectSimulationResultCollection(ctx context.Context, projectID int64, results []ProjectSimulationResult, fetchedAt time.Time) error {
	if s == nil || s.pool == nil {
		return fmt.Errorf("token postgres database is not configured")
	}
	if fetchedAt.IsZero() {
		fetchedAt = time.Now().UTC()
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin project simulation result collection transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()
	q := tokensqlc.New(tx)
	for _, result := range results {
		if result.Wallet == (common.Address{}) {
			continue
		}
		if result.ProjectID == 0 {
			result.ProjectID = projectID
		}
		if result.FetchedAt.IsZero() {
			result.FetchedAt = fetchedAt
		}
		if _, err := q.UpsertProjectSimulationResult(ctx, tokensqlc.UpsertProjectSimulationResultParams{
			ProjectID:                          result.ProjectID,
			Wallet:                             result.Wallet.Bytes(),
			CanMintFromDeadViaTransferFrom:     result.CanMintFromDeadViaTransferFrom,
			CanMintFromZeroViaTransferFrom:     result.CanMintFromZeroViaTransferFrom,
			CanMintFromWethPairViaTransferFrom: result.CanMintFromWethPairViaTransferFrom,
			CanMintFromUsdtPairViaTransferFrom: result.CanMintFromUsdtPairViaTransferFrom,
			CanMintViaTransferToWethPair:       result.CanMintViaTransferToWethPair,
			CanMintViaTransferToUsdtPair:       result.CanMintViaTransferToUsdtPair,
			FetchedAt:                          nullableTime(result.FetchedAt),
		}); err != nil {
			return fmt.Errorf("upsert project simulation result %s: %w", result.Wallet.Hex(), err)
		}
	}
	if _, err := q.MarkProjectDataCollectionTaskSucceeded(ctx, tokensqlc.MarkProjectDataCollectionTaskSucceededParams{
		ProjectID: projectID,
		DataType:  ProjectDataCollectionTypeSimulationResult,
	}); err != nil {
		return fmt.Errorf("mark project simulation result collection succeeded: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit project simulation result collection transaction: %w", err)
	}
	return nil
}
