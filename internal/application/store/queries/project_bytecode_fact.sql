-- name: UpsertProjectBytecodeFact :exec
UPDATE project
SET code_hash = sqlc.narg('code_hash')::bytea,
  updated_at = now()
WHERE chain_id = @chain_id
  AND contract = @project_contract;

-- name: GetProjectBytecodeFact :one
SELECT
  p.chain_id,
  p.contract AS project_contract,
  p.code_hash,
  COALESCE(bf.last_success_at, p.updated_at) AS fetched_at,
  p.updated_at
FROM project p
LEFT JOIN project_component_state bf ON bf.project_id = p.id AND bf.component = 'bytecode_fact'
WHERE p.chain_id = @chain_id
  AND p.contract = @project_contract
  AND (p.code_hash IS NOT NULL OR bf.last_success_at IS NOT NULL);
