-- Reporting persistence and read model.
-- name: GetLatestProjectReportRevision :one
SELECT *
FROM project_report_revision
WHERE project_id = @project_id
ORDER BY revision DESC
LIMIT 1;

-- name: GetProjectReportRevision :one
SELECT *
FROM project_report_revision
WHERE project_id = @project_id
  AND revision = @revision;

-- name: InsertProjectReportRevision :one
INSERT INTO project_report_revision (
  project_id,
  revision,
  schema_version,
  content_hash,
  completeness_status,
  evidence,
  report,
  observed_block_number,
  weth_pair_is_created,
  weth_pair_is_remove_liquidity,
  weth_pair_is_mint,
  weth_pair_quote_usdt_value_int,
  weth_pair_last_swap_timestamp,
  usdt_pair_is_created,
  usdt_pair_is_remove_liquidity,
  usdt_pair_is_mint,
  usdt_pair_quote_usdt_value_int,
  usdt_pair_last_swap_timestamp,
  built_at
) VALUES (
  @project_id,
  @revision,
  @schema_version,
  @content_hash,
  @completeness_status,
  @evidence::jsonb,
  @report::jsonb,
  sqlc.narg('observed_block_number'),
  sqlc.narg('weth_pair_is_created'),
  sqlc.narg('weth_pair_is_remove_liquidity'),
  sqlc.narg('weth_pair_is_mint'),
  sqlc.narg('weth_pair_quote_usdt_value_int'),
  sqlc.narg('weth_pair_last_swap_timestamp'),
  sqlc.narg('usdt_pair_is_created'),
  sqlc.narg('usdt_pair_is_remove_liquidity'),
  sqlc.narg('usdt_pair_is_mint'),
  sqlc.narg('usdt_pair_quote_usdt_value_int'),
  sqlc.narg('usdt_pair_last_swap_timestamp'),
  @built_at
)
RETURNING *;

-- name: CountProjectReportRevisions :one
SELECT COUNT(*)::bigint
FROM project_report_revision AS report
JOIN project ON project.id = report.project_id
WHERE (sqlc.arg('chain_id')::bigint = 0 OR project.chain_id = sqlc.arg('chain_id')::bigint)
  AND (sqlc.arg('project_id')::bigint = 0 OR report.project_id = sqlc.arg('project_id')::bigint);

-- name: ListProjectReportRevisions :many
SELECT
  report.*,
  project.chain_id,
  project.contract
FROM project_report_revision AS report
JOIN project ON project.id = report.project_id
WHERE (sqlc.arg('chain_id')::bigint = 0 OR project.chain_id = sqlc.arg('chain_id')::bigint)
  AND (sqlc.arg('project_id')::bigint = 0 OR report.project_id = sqlc.arg('project_id')::bigint)
ORDER BY report.built_at DESC, report.project_id DESC, report.revision DESC
LIMIT sqlc.arg('limit') OFFSET sqlc.arg('offset');
