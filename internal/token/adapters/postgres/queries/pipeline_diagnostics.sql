-- Worker host diagnostics read model.
-- name: ListPipelineQueueMetrics :many
WITH queue_items AS (
  SELECT
    ('data_collection:' || data_type)::text AS queue,
    status,
    available_at
  FROM project_data_collection_task
  WHERE status IN ('pending', 'running', 'failed')

  UNION ALL

  SELECT 'report_build', status, available_at
  FROM project_report_build_task
  WHERE status IN ('pending', 'running', 'failed')

  UNION ALL

  SELECT 'selection_evaluation', status, available_at
  FROM project_selection_evaluation_task
  WHERE status IN ('pending', 'running', 'failed')
)
SELECT
  queue,
  status,
  COUNT(*)::bigint AS item_count,
  MIN(available_at)::timestamptz AS oldest_available_at
FROM queue_items
GROUP BY queue, status
ORDER BY queue, status;
