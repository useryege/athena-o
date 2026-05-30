-- name: UpsertProjectBytecodeFact :exec
INSERT INTO project_bytecode_fact (
  project_contract,
  code_hash,
  is_bytecode_blacklisted,
  fetched_at
) VALUES (
  $1,
  sqlc.narg('code_hash')::bytea,
  $2,
  $3
)
ON CONFLICT (project_contract) DO UPDATE
SET code_hash = EXCLUDED.code_hash,
  is_bytecode_blacklisted = EXCLUDED.is_bytecode_blacklisted,
  fetched_at = EXCLUDED.fetched_at,
  updated_at = now();

-- name: GetProjectBytecodeFact :one
SELECT project_contract, code_hash, is_bytecode_blacklisted, fetched_at, updated_at
FROM project_bytecode_fact
WHERE project_contract = $1;
