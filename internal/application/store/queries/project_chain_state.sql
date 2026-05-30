-- name: UpsertProjectChainState :exec
INSERT INTO project_chain_state (
  project_contract,
  chain_state,
  weth_pair,
  usdt_pair,
  token_name,
  token_symbol,
  fetched_at
) VALUES (
  $1,
  $2::jsonb,
  $3,
  $4,
  sqlc.narg('token_name')::text,
  sqlc.narg('token_symbol')::text,
  $5
)
ON CONFLICT (project_contract) DO UPDATE
SET chain_state = EXCLUDED.chain_state,
  weth_pair = EXCLUDED.weth_pair,
  usdt_pair = EXCLUDED.usdt_pair,
  token_name = EXCLUDED.token_name,
  token_symbol = EXCLUDED.token_symbol,
  fetched_at = EXCLUDED.fetched_at,
  updated_at = now();

-- name: GetProjectChainState :one
SELECT project_contract, chain_state, weth_pair, usdt_pair, token_name, token_symbol, fetched_at, updated_at
FROM project_chain_state
WHERE project_contract = $1;

-- name: ListProjectChainStatesByContracts :many
SELECT project_contract, chain_state, weth_pair, usdt_pair, token_name, token_symbol, fetched_at, updated_at
FROM project_chain_state
WHERE project_contract = ANY($1::bytea[]);

-- name: ListProjectChainStatesByPairAddresses :many
SELECT project_contract, chain_state, weth_pair, usdt_pair, token_name, token_symbol, fetched_at, updated_at
FROM project_chain_state
WHERE weth_pair = ANY($1::bytea[]) OR usdt_pair = ANY($1::bytea[]);
