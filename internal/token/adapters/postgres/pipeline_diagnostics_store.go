package postgres

import (
	"context"
	"fmt"
	"time"
)

type PipelineQueueMetric struct {
	Queue             string
	Status            string
	Count             int64
	OldestAvailableAt time.Time
}

func (s *DiagnosticsRepository) Ping(ctx context.Context) error {
	if s == nil || s.pool == nil {
		return fmt.Errorf("token diagnostics repository is not configured")
	}
	return s.pool.Ping(ctx)
}

func (s *DiagnosticsRepository) ListPipelineQueueMetrics(ctx context.Context) ([]PipelineQueueMetric, error) {
	q, err := s.querier()
	if err != nil {
		return nil, err
	}
	rows, err := q.ListPipelineQueueMetrics(ctx)
	if err != nil {
		return nil, fmt.Errorf("list token pipeline queue metrics: %w", err)
	}
	metrics := make([]PipelineQueueMetric, 0, len(rows))
	for _, row := range rows {
		metrics = append(metrics, PipelineQueueMetric{Queue: row.Queue, Status: row.Status, Count: row.ItemCount, OldestAvailableAt: timeValue(row.OldestAvailableAt)})
	}
	return metrics, nil
}
