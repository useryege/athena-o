package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	tokensqlc "github.com/useryege/athena/internal/token/adapters/postgres/sqlc"
	"github.com/useryege/athena/internal/token/discovery"
)

const chainBlockProcessingRetentionSeconds = uint64(72 * time.Hour / time.Second)

func (s *ChainRepository) StartChainBlockProcessingAttempt(ctx context.Context, chainID int64, blockNumber uint64) (*discovery.ChainBlockProcessingAttempt, error) {
	if s == nil || s.pool == nil {
		return nil, fmt.Errorf("token chain repository is not configured")
	}
	blockNumberValue, err := uint64ToInt64("block_number", blockNumber)
	if err != nil {
		return nil, err
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin token chain block processing attempt: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	queries := tokensqlc.New(tx)
	checkpoint, err := queries.LockChainProcessingCheckpoint(ctx, chainID)
	if err != nil {
		return nil, fmt.Errorf("lock token chain processing checkpoint: %w", err)
	}
	if err := queries.ReconcileRunningChainBlockProcessingAttempts(ctx, tokensqlc.ReconcileRunningChainBlockProcessingAttemptsParams{
		CursorBlockNumber:   checkpoint.CursorBlockNumber,
		CheckpointUpdatedAt: checkpoint.UpdatedAt,
		ChainID:             chainID,
	}); err != nil {
		return nil, fmt.Errorf("reconcile running token chain block processing attempts: %w", err)
	}
	row, err := queries.CreateChainBlockProcessingAttempt(ctx, tokensqlc.CreateChainBlockProcessingAttemptParams{
		ChainID:     chainID,
		BlockNumber: blockNumberValue,
	})
	if err != nil {
		return nil, fmt.Errorf("create token chain block processing attempt: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit token chain block processing attempt: %w", err)
	}
	return mapChainBlockProcessingAttempt(row)
}

func (s *ChainRepository) CompleteChainBlockProcessingAttempt(ctx context.Context, completion discovery.ChainBlockProcessingAttemptCompletion) (*discovery.ChainBlockProcessingAttempt, error) {
	if s == nil || s.pool == nil {
		return nil, fmt.Errorf("token chain repository is not configured")
	}
	blockTime, err := nullableBlockTime(completion.BlockTime)
	if err != nil {
		return nil, err
	}
	checkpointDuration, err := nullableDurationMicroseconds("checkpoint_read_duration", completion.CheckpointReadDuration)
	if err != nil {
		return nil, err
	}
	discoveryDuration, err := nullableDurationMicroseconds("discovery_duration", completion.DiscoveryDuration)
	if err != nil {
		return nil, err
	}
	validationDuration, err := nullableDurationMicroseconds("validation_duration", completion.ValidationDuration)
	if err != nil {
		return nil, err
	}
	persistenceDuration, err := nullableDurationMicroseconds("persistence_duration", completion.PersistenceDuration)
	if err != nil {
		return nil, err
	}
	totalDuration := sumDurationMicroseconds(checkpointDuration, discoveryDuration, validationDuration, persistenceDuration)

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin complete token chain block processing attempt: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	queries := tokensqlc.New(tx)
	row, err := queries.CompleteChainBlockProcessingAttempt(ctx, tokensqlc.CompleteChainBlockProcessingAttemptParams{
		BlockTime:                blockTime,
		Status:                   string(completion.Status),
		TerminalStage:            string(completion.TerminalStage),
		ErrorMessage:             completion.ErrorMessage,
		CheckpointReadDurationUs: checkpointDuration,
		DiscoveryDurationUs:      discoveryDuration,
		ValidationDurationUs:     validationDuration,
		PersistenceDurationUs:    persistenceDuration,
		TotalDurationUs:          totalDuration,
		CandidateCount:           nullableInt32Pointer(completion.CandidateCount),
		ValidatedCount:           nullableInt32Pointer(completion.ValidatedCount),
		RejectedCount:            nullableInt32Pointer(completion.RejectedCount),
		TimingComplete:           completion.TimingComplete,
		ID:                       completion.AttemptID,
	})
	if err != nil {
		return nil, fmt.Errorf("complete token chain block processing attempt: %w", err)
	}
	if completion.Status == discovery.ChainBlockProcessingAttemptStatusSucceeded && completion.BlockTime > 0 {
		if err := queries.BackfillChainBlockProcessingAttemptBlockTime(ctx, tokensqlc.BackfillChainBlockProcessingAttemptBlockTimeParams{
			BlockTime:   blockTime,
			ChainID:     row.ChainID,
			BlockNumber: row.BlockNumber,
		}); err != nil {
			return nil, fmt.Errorf("backfill token chain block processing attempt time: %w", err)
		}
		cutoff := uint64(0)
		if completion.BlockTime > chainBlockProcessingRetentionSeconds {
			cutoff = completion.BlockTime - chainBlockProcessingRetentionSeconds
		}
		if _, err := queries.DeleteExpiredUnknownTimeChainBlockProcessingAttempts(ctx, tokensqlc.DeleteExpiredUnknownTimeChainBlockProcessingAttemptsParams{
			ChainID:         row.ChainID,
			CutoffBlockTime: pgtype.Int8{Int64: int64(cutoff), Valid: true},
		}); err != nil {
			return nil, fmt.Errorf("delete expired token chain block processing attempts without block time: %w", err)
		}
		if _, err := queries.DeleteExpiredChainBlockProcessingAttempts(ctx, tokensqlc.DeleteExpiredChainBlockProcessingAttemptsParams{
			ChainID:         row.ChainID,
			CutoffBlockTime: pgtype.Int8{Int64: int64(cutoff), Valid: true},
		}); err != nil {
			return nil, fmt.Errorf("delete expired token chain block processing attempts: %w", err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit completed token chain block processing attempt: %w", err)
	}
	return mapChainBlockProcessingAttempt(row)
}

func (s *ChainRepository) GetChainBlockProcessingSummary(ctx context.Context, filter discovery.ChainBlockProcessingFilter) (*discovery.ChainBlockProcessingSummary, error) {
	q, err := s.querier()
	if err != nil {
		return nil, err
	}
	blockNumber, err := uint64ToInt64("block_number", filter.BlockNumber)
	if err != nil {
		return nil, err
	}
	row, err := q.GetChainBlockProcessingSummary(ctx, tokensqlc.GetChainBlockProcessingSummaryParams{
		ChainID:       filter.ChainID,
		BlockNumber:   blockNumber,
		WindowSeconds: filter.WindowSeconds,
	})
	if err != nil {
		return nil, fmt.Errorf("get token chain block processing summary: %w", err)
	}
	return mapChainBlockProcessingSummary(row)
}

func (s *ChainRepository) ListChainBlockProcessingAttemptsPage(ctx context.Context, filter discovery.ChainBlockProcessingFilter, page, pageSize int32) (*discovery.ChainBlockProcessingAttemptPage, error) {
	q, err := s.querier()
	if err != nil {
		return nil, err
	}
	blockNumber, err := uint64ToInt64("block_number", filter.BlockNumber)
	if err != nil {
		return nil, err
	}
	page, pageSize, offset := normalizePage(page, pageSize)
	params := tokensqlc.ListChainBlockProcessingAttemptsParams{
		ChainID:       filter.ChainID,
		Status:        string(filter.Status),
		BlockNumber:   blockNumber,
		WindowSeconds: filter.WindowSeconds,
		PageOffset:    offset,
		PageSize:      pageSize,
	}
	rows, err := q.ListChainBlockProcessingAttempts(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("list token chain block processing attempts: %w", err)
	}
	total, err := q.CountChainBlockProcessingAttempts(ctx, tokensqlc.CountChainBlockProcessingAttemptsParams{
		ChainID:       params.ChainID,
		Status:        params.Status,
		BlockNumber:   params.BlockNumber,
		WindowSeconds: params.WindowSeconds,
	})
	if err != nil {
		return nil, fmt.Errorf("count token chain block processing attempts: %w", err)
	}
	items := make([]discovery.ChainBlockProcessingAttempt, 0, len(rows))
	for _, row := range rows {
		item, err := mapChainBlockProcessingAttempt(row)
		if err != nil {
			return nil, fmt.Errorf("map token chain block processing attempt: %w", err)
		}
		items = append(items, *item)
	}
	return &discovery.ChainBlockProcessingAttemptPage{Items: items, Total: total, Page: page, PageSize: pageSize}, nil
}

func mapChainBlockProcessingAttempt(row tokensqlc.ChainBlockProcessingAttempt) (*discovery.ChainBlockProcessingAttempt, error) {
	blockNumber, err := int64ToUint64("block_number", row.BlockNumber)
	if err != nil {
		return nil, err
	}
	blockTime, err := int64ToUint64("block_time", int64Value(row.BlockTime))
	if err != nil {
		return nil, err
	}
	return &discovery.ChainBlockProcessingAttempt{
		ID:                       row.ID,
		ChainID:                  row.ChainID,
		BlockNumber:              blockNumber,
		AttemptNumber:            row.AttemptNumber,
		BlockTime:                blockTime,
		Status:                   discovery.ChainBlockProcessingAttemptStatus(row.Status),
		TerminalStage:            discovery.ChainBlockProcessingStage(textValue(row.TerminalStage)),
		ErrorMessage:             textValue(row.ErrorMessage),
		CheckpointReadDurationUS: int64Value(row.CheckpointReadDurationUs),
		DiscoveryDurationUS:      int64Value(row.DiscoveryDurationUs),
		ValidationDurationUS:     int64Value(row.ValidationDurationUs),
		PersistenceDurationUS:    int64Value(row.PersistenceDurationUs),
		TotalDurationUS:          int64Value(row.TotalDurationUs),
		CandidateCount:           int32Value(row.CandidateCount),
		ValidatedCount:           int32Value(row.ValidatedCount),
		RejectedCount:            int32Value(row.RejectedCount),
		TimingComplete:           row.TimingComplete,
		StartedAt:                timeValue(row.StartedAt),
		CompletedAt:              timeValue(row.CompletedAt),
		CreatedAt:                timeValue(row.CreatedAt),
		UpdatedAt:                timeValue(row.UpdatedAt),
	}, nil
}

func mapChainBlockProcessingSummary(row tokensqlc.GetChainBlockProcessingSummaryRow) (*discovery.ChainBlockProcessingSummary, error) {
	rangeStart, err := int64ToUint64("range_start_block_time", row.RangeStartBlockTime)
	if err != nil {
		return nil, err
	}
	rangeEnd, err := int64ToUint64("range_end_block_time", row.RangeEndBlockTime)
	if err != nil {
		return nil, err
	}
	fastestBlock, err := int64ToUint64("fastest_block_number", row.FastestBlockNumber)
	if err != nil {
		return nil, err
	}
	slowestBlock, err := int64ToUint64("slowest_block_number", row.SlowestBlockNumber)
	if err != nil {
		return nil, err
	}
	return &discovery.ChainBlockProcessingSummary{
		ChainID:                         row.ChainID,
		RangeStartBlockTime:             rangeStart,
		RangeEndBlockTime:               rangeEnd,
		AttemptCount:                    row.AttemptCount,
		RunningCount:                    row.RunningCount,
		SucceededCount:                  row.SucceededCount,
		FailedCount:                     row.FailedCount,
		CancelledCount:                  row.CancelledCount,
		InterruptedCount:                row.InterruptedCount,
		IncompleteSucceededCount:        row.IncompleteSucceededCount,
		MeasuredSucceededCount:          row.MeasuredSucceededCount,
		FailureRateBPS:                  row.FailureRateBps,
		AverageDurationUS:               row.AverageDurationUs,
		AverageCheckpointReadDurationUS: row.AverageCheckpointReadDurationUs,
		AverageDiscoveryDurationUS:      row.AverageDiscoveryDurationUs,
		AverageValidationDurationUS:     row.AverageValidationDurationUs,
		AveragePersistenceDurationUS:    row.AveragePersistenceDurationUs,
		FastestBlockNumber:              fastestBlock,
		FastestDurationUS:               row.FastestDurationUs,
		SlowestBlockNumber:              slowestBlock,
		SlowestDurationUS:               row.SlowestDurationUs,
	}, nil
}

func nullableBlockTime(value uint64) (pgtype.Int8, error) {
	if value == 0 {
		return pgtype.Int8{}, nil
	}
	mapped, err := uint64ToInt64("block_time", value)
	return pgtype.Int8{Int64: mapped, Valid: err == nil}, err
}

func nullableDurationMicroseconds(field string, value *time.Duration) (pgtype.Int8, error) {
	if value == nil {
		return pgtype.Int8{}, nil
	}
	microseconds := value.Microseconds()
	if microseconds < 0 {
		return pgtype.Int8{}, fmt.Errorf("%s must not be negative", field)
	}
	return pgtype.Int8{Int64: microseconds, Valid: true}, nil
}

func sumDurationMicroseconds(values ...pgtype.Int8) pgtype.Int8 {
	var total int64
	valid := false
	for _, value := range values {
		if value.Valid {
			total += value.Int64
			valid = true
		}
	}
	return pgtype.Int8{Int64: total, Valid: valid}
}

func nullableInt32Pointer(value *int32) pgtype.Int4 {
	if value == nil {
		return pgtype.Int4{}
	}
	return pgtype.Int4{Int32: *value, Valid: true}
}

func nullableInt64Pointer(value *int64) pgtype.Int8 {
	if value == nil {
		return pgtype.Int8{}
	}
	return pgtype.Int8{Int64: *value, Valid: true}
}

func int32Value(value pgtype.Int4) int32 {
	if !value.Valid {
		return 0
	}
	return value.Int32
}
