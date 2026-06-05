-- name: UpsertProjectBytecodeFact :exec
INSERT INTO project_bytecode_fact (
  project_id,
  code_hash,
  is_bytecode_blacklisted,
  fetched_at
) VALUES (
  (SELECT id FROM project WHERE chain_id = @chain_id AND contract = @project_contract),
  sqlc.narg('code_hash')::bytea,
  @is_bytecode_blacklisted,
  @fetched_at
)
ON CONFLICT (project_id) DO UPDATE
SET code_hash = EXCLUDED.code_hash,
  is_bytecode_blacklisted = EXCLUDED.is_bytecode_blacklisted,
  fetched_at = EXCLUDED.fetched_at,
  updated_at = now();

-- name: GetProjectBytecodeFact :one
SELECT p.chain_id, p.contract AS project_contract, bf.code_hash, bf.is_bytecode_blacklisted, bf.fetched_at, bf.updated_at
FROM project_bytecode_fact bf
JOIN project p ON p.id = bf.project_id
WHERE p.chain_id = @chain_id
  AND p.contract = @project_contract;
