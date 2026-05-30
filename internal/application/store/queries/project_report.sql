-- name: UpsertProjectReportState :exec
INSERT INTO project_report (
  project_contract,
  is_report_evaluated,
  is_report_complete,
  is_blacklisted_creator_wallet,
  is_blacklisted_genesis_wallet,
  is_blacklisted_bytecode,
  has_mint_risk,
  evaluated_at
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
ON CONFLICT (project_contract) DO UPDATE
SET is_report_evaluated = EXCLUDED.is_report_evaluated,
  is_report_complete = EXCLUDED.is_report_complete,
  is_blacklisted_creator_wallet = EXCLUDED.is_blacklisted_creator_wallet,
  is_blacklisted_genesis_wallet = EXCLUDED.is_blacklisted_genesis_wallet,
  is_blacklisted_bytecode = EXCLUDED.is_blacklisted_bytecode,
  has_mint_risk = EXCLUDED.has_mint_risk,
  evaluated_at = EXCLUDED.evaluated_at,
  updated_at = now();

-- name: GetProjectReportState :one
SELECT project_contract, is_report_evaluated, is_report_complete, is_blacklisted_creator_wallet,
  is_blacklisted_genesis_wallet, is_blacklisted_bytecode, has_mint_risk,
  evaluated_at, updated_at
FROM project_report
WHERE project_contract = $1;

-- name: ListProjectReportStatesByContracts :many
SELECT project_contract, is_report_evaluated, is_report_complete, is_blacklisted_creator_wallet,
  is_blacklisted_genesis_wallet, is_blacklisted_bytecode, has_mint_risk,
  evaluated_at, updated_at
FROM project_report
WHERE project_contract = ANY($1::bytea[]);
