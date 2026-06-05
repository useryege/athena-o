-- name: UpsertProjectChainState :exec
INSERT INTO project_chain_state (
  project_id,
  chain_state,
  weth_pair,
  usdt_pair,
  token_name,
  token_symbol,
  fetched_at
) VALUES (
  (SELECT id FROM project WHERE chain_id = @chain_id AND contract = @project_contract),
  @chain_state::jsonb,
  @weth_pair,
  @usdt_pair,
  sqlc.narg('token_name')::text,
  sqlc.narg('token_symbol')::text,
  @fetched_at
)
ON CONFLICT (project_id) DO UPDATE
SET chain_state = EXCLUDED.chain_state,
  weth_pair = EXCLUDED.weth_pair,
  usdt_pair = EXCLUDED.usdt_pair,
  token_name = EXCLUDED.token_name,
  token_symbol = EXCLUDED.token_symbol,
  fetched_at = EXCLUDED.fetched_at,
  updated_at = now();

-- name: GetProjectChainState :one
SELECT p.chain_id, p.contract AS project_contract, cs.chain_state, cs.weth_pair, cs.usdt_pair, cs.token_name, cs.token_symbol, cs.fetched_at, cs.updated_at
FROM project_chain_state cs
JOIN project p ON p.id = cs.project_id
WHERE p.chain_id = @chain_id
  AND p.contract = @project_contract;

-- name: ListProjectChainStatesByContracts :many
SELECT p.chain_id, p.contract AS project_contract, cs.chain_state, cs.weth_pair, cs.usdt_pair, cs.token_name, cs.token_symbol, cs.fetched_at, cs.updated_at
FROM project_chain_state cs
JOIN project p ON p.id = cs.project_id
WHERE p.chain_id = @chain_id
  AND p.contract = ANY(@project_contracts::bytea[]);

-- name: ListProjectChainStatesByPairAddresses :many
SELECT p.chain_id, p.contract AS project_contract, cs.chain_state, cs.weth_pair, cs.usdt_pair, cs.token_name, cs.token_symbol, cs.fetched_at, cs.updated_at
FROM project_chain_state cs
JOIN project p ON p.id = cs.project_id
WHERE p.chain_id = @chain_id
  AND (cs.weth_pair = ANY(@pairs::bytea[]) OR cs.usdt_pair = ANY(@pairs::bytea[]));
