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

-- name: CountCurrentProjectReports :one
SELECT COUNT(*)::bigint
FROM project_research_state AS research
JOIN project ON project.id = research.project_id
JOIN project_report_revision AS report
  ON report.project_id = research.project_id
  AND report.revision = research.current_report_revision
WHERE (sqlc.arg('chain_id')::bigint = 0 OR project.chain_id = sqlc.arg('chain_id')::bigint)
  AND (sqlc.arg('project_id')::bigint = 0 OR project.id = sqlc.arg('project_id')::bigint)
  AND (sqlc.narg('contract')::bytea IS NULL OR project.contract = sqlc.narg('contract')::bytea)
  AND (sqlc.arg('build_status')::text = '' OR sqlc.arg('build_status')::text = 'succeeded');

-- name: ListCurrentProjectReports :many
SELECT
  report.*,
  project.chain_id,
  project.name,
  project.symbol,
  project.contract,
  'succeeded'::text AS build_status,
  0::int AS build_attempts,
  ''::text AS build_last_error,
  report.built_at AS build_updated_at
FROM project_research_state AS research
JOIN project ON project.id = research.project_id
JOIN project_report_revision AS report
  ON report.project_id = research.project_id
  AND report.revision = research.current_report_revision
WHERE (sqlc.arg('chain_id')::bigint = 0 OR project.chain_id = sqlc.arg('chain_id')::bigint)
  AND (sqlc.arg('project_id')::bigint = 0 OR project.id = sqlc.arg('project_id')::bigint)
  AND (sqlc.narg('contract')::bytea IS NULL OR project.contract = sqlc.narg('contract')::bytea)
  AND (sqlc.arg('build_status')::text = '' OR sqlc.arg('build_status')::text = 'succeeded')
ORDER BY report.built_at DESC, report.project_id DESC
LIMIT sqlc.arg('limit') OFFSET sqlc.arg('offset');

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
