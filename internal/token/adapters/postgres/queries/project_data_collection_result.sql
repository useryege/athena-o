-- Immutable evidence produced by one-time project data collection.

-- name: InsertProjectDataCollectionResult :one
INSERT INTO project_data_collection_result (
  task_id,
  project_id,
  data_type,
  schema_version,
  payload,
  content_hash,
  block_number,
  collected_at
) VALUES (
  @task_id,
  @project_id,
  @data_type,
  @schema_version,
  @payload::jsonb,
  @content_hash,
  sqlc.narg('block_number')::bigint,
  @collected_at
)
ON CONFLICT (task_id) DO UPDATE
SET task_id = project_data_collection_result.task_id
WHERE project_data_collection_result.content_hash = EXCLUDED.content_hash
RETURNING *;

-- name: GetProjectDataCollectionResultByTaskID :one
SELECT *
FROM project_data_collection_result
WHERE task_id = @task_id;

-- name: GetProjectDataCollectionResult :one
SELECT *
FROM project_data_collection_result
WHERE project_id = @project_id
  AND data_type = @data_type;

-- name: ListProjectDataCollectionResultsByProject :many
SELECT *
FROM project_data_collection_result
WHERE project_id = @project_id
ORDER BY CASE data_type
  WHEN 'chain_state' THEN 1
  WHEN 'wallet_asset_state' THEN 2
  WHEN 'simulation_result' THEN 3
  WHEN 'ave' THEN 4
  WHEN 'contract_code_source' THEN 5
  WHEN 'wallet_normal_transactions' THEN 6
  ELSE 7
END;

-- name: GetProjectDataCollectionTaskWithResult :one
SELECT
  task.*,
  result.schema_version AS result_schema_version,
  result.payload AS result_payload,
  result.content_hash AS result_content_hash,
  result.block_number AS result_block_number,
  result.collected_at AS result_collected_at,
  result.created_at AS result_created_at
FROM project_data_collection_task AS task
LEFT JOIN project_data_collection_result AS result ON result.task_id = task.id
WHERE task.id = @task_id;
