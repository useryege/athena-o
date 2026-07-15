-- Worker host diagnostics read model.
-- name: ListPipelineQueueMetrics :many
WITH queue_items AS (
  SELECT
    'candidate_validation'::text AS queue,
    CASE
      WHEN validation_lease_expires_at IS NOT NULL AND validation_lease_expires_at > now() THEN 'running'::text
      ELSE 'pending'::text
    END AS status,
    CASE
      WHEN validation_lease_expires_at IS NOT NULL AND validation_lease_expires_at > now() THEN validation_lease_expires_at
      ELSE created_at
    END AS available_at
  FROM project_candidate
  WHERE status = 'pending'

  UNION ALL

  SELECT 'data_collection:' || data_type, status, available_at
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
