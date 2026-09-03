-- Worker host diagnostics read model.
-- name: ListPipelineQueueMetrics :many
WITH queue_items AS (
  SELECT
    ('data_collection:' || data_type)::text AS queue,
    status,
    CASE WHEN status IN ('pending', 'running') THEN available_at END AS available_at,
    failure_count::bigint AS failure_count,
    GREATEST(
      claim_generation - failure_count - CASE WHEN status IN ('running', 'succeeded') THEN 1 ELSE 0 END,
      0
    )::bigint AS lease_recovery_count,
    CASE
      WHEN finished_at IS NOT NULL AND locked_at IS NOT NULL
      THEN EXTRACT(EPOCH FROM (finished_at - locked_at))::double precision
    END AS duration_seconds
  FROM project_data_collection_task

  UNION ALL

  SELECT
    'profile_build',
    status,
    CASE WHEN status IN ('pending', 'running') THEN available_at END,
    failure_count::bigint,
    GREATEST(
      claim_generation - failure_count - CASE WHEN status IN ('running', 'succeeded') THEN 1 ELSE 0 END,
      0
    )::bigint,
    CASE
      WHEN finished_at IS NOT NULL AND locked_at IS NOT NULL
      THEN EXTRACT(EPOCH FROM (finished_at - locked_at))::double precision
    END
  FROM project_profile_build_task
)
SELECT
  queue,
  status,
  COUNT(*)::bigint AS item_count,
  MIN(available_at)::timestamptz AS oldest_available_at,
  SUM(failure_count)::bigint AS failure_count,
  SUM(lease_recovery_count)::bigint AS lease_recovery_count,
  COALESCE(AVG(duration_seconds), 0)::double precision AS average_duration_seconds
FROM queue_items
GROUP BY queue, status
ORDER BY queue, status;
