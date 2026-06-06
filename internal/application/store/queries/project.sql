-- name: InsertProject :exec
INSERT INTO project (
  chain_id,
  block_number,
  block_time,
  contract,
  creator,
  tx_hash,
  tx_index
) VALUES (@chain_id, @block_number, @block_time, @contract, @creator, @tx_hash, @tx_index)
ON CONFLICT DO NOTHING;

-- name: UpsertProjectFromDiscovery :exec
INSERT INTO project (
  chain_id,
  block_number,
  block_time,
  contract,
  creator,
  tx_hash,
  tx_index,
  weth_pair,
  usdt_pair,
  source,
  status,
  reason,
  payload,
  discovered_at,
  code_hash
) VALUES (
  @chain_id,
  @block_number,
  @block_time,
  @contract,
  @creator,
  @tx_hash,
  @tx_index,
  sqlc.narg('weth_pair')::bytea,
  sqlc.narg('usdt_pair')::bytea,
  @source,
  'processed',
  sqlc.narg('reason')::text,
  @payload::jsonb,
  now(),
  (
    SELECT d.code_hash
    FROM contract_bytecode_deployment d
    WHERE d.chain_id = @chain_id
      AND d.contract = @contract
  )
)
ON CONFLICT (chain_id, contract) DO UPDATE
SET weth_pair = COALESCE(project.weth_pair, EXCLUDED.weth_pair),
  usdt_pair = COALESCE(project.usdt_pair, EXCLUDED.usdt_pair),
  code_hash = COALESCE(EXCLUDED.code_hash, project.code_hash),
  source = EXCLUDED.source,
  status = 'processed',
  reason = COALESCE(EXCLUDED.reason, project.reason),
  payload = project.payload || EXCLUDED.payload,
  updated_at = now();

-- name: GetMaxProjectBlockNumber :one
SELECT
  COALESCE(MAX(block_number), 0)::bigint AS max_block,
  (COUNT(*)::bigint > 0) AS has_value
FROM project
WHERE chain_id = @chain_id;

-- name: ListProjects :many
SELECT chain_id, block_number, block_time, contract, creator, tx_hash, tx_index, created_at
FROM project
WHERE chain_id = @chain_id
ORDER BY block_number, tx_index, id;

-- name: CountProjects :one
SELECT COUNT(*)::bigint
FROM project
WHERE chain_id = @chain_id;

-- name: ListProjectsPage :many
SELECT chain_id, block_number, block_time, contract, creator, tx_hash, tx_index, created_at
FROM project
WHERE chain_id = @chain_id
ORDER BY block_number, tx_index, id
LIMIT @limit_count OFFSET @offset_count;

-- name: GetProjectByContract :one
SELECT chain_id, block_number, block_time, contract, creator, tx_hash, tx_index, created_at
FROM project
WHERE chain_id = @chain_id
  AND contract = @contract;

-- name: ListProjectsByCreatorBefore :many
SELECT chain_id, block_number, block_time, contract, creator, tx_hash, tx_index, created_at
FROM project
WHERE chain_id = @chain_id
  AND creator = @creator
  AND (block_number < @block_number OR (block_number = @block_number AND tx_index < @tx_index))
ORDER BY block_number, tx_index, id;

-- name: ListProjectMetas :many
SELECT
  p.chain_id,
  p.block_number,
  p.block_time,
  p.contract,
  p.creator,
  COALESCE(cs.weth_pair, p.weth_pair, decode(repeat('00', 20), 'hex')) AS weth_pair,
  COALESCE(cs.usdt_pair, p.usdt_pair, decode(repeat('00', 20), 'hex')) AS usdt_pair,
  COALESCE(cs.fetched_at, p.created_at) AS fetch_at,
  p.tx_hash,
  p.tx_index,
  COALESCE(sr.can_mint_from_dead_via_transfer_from, false) AS can_mint_from_dead_via_transfer_from,
  COALESCE(sr.can_mint_from_zero_via_transfer_from, false) AS can_mint_from_zero_via_transfer_from,
  COALESCE(sr.can_mint_from_weth_pair_via_transfer_from, false) AS can_mint_from_weth_pair_via_transfer_from,
  COALESCE(sr.can_mint_from_usdt_pair_via_transfer_from, false) AS can_mint_from_usdt_pair_via_transfer_from,
  COALESCE(sr.can_mint_via_transfer_to_weth_pair, false) AS can_mint_via_transfer_to_weth_pair,
  COALESCE(sr.can_mint_via_transfer_to_usdt_pair, false) AS can_mint_via_transfer_to_usdt_pair,
  gw.last_success_at AS genesis_wallets_fetched_at,
  ch.last_success_at AS creator_historical_projects_fetched_at
FROM project p
LEFT JOIN project_chain_state cs ON cs.project_id = p.id
LEFT JOIN project_simulation_result sr ON sr.project_id = p.id
LEFT JOIN project_component_state gw ON gw.project_id = p.id AND gw.component = 'genesis_wallet'
LEFT JOIN project_component_state ch ON ch.project_id = p.id AND ch.component = 'creator_history'
WHERE p.chain_id = @chain_id
ORDER BY p.block_number, p.tx_index, p.id;

-- name: ListProjectMetasByPairAddresses :many
SELECT
  p.chain_id,
  p.block_number,
  p.block_time,
  p.contract,
  p.creator,
  COALESCE(cs.weth_pair, p.weth_pair, decode(repeat('00', 20), 'hex')) AS weth_pair,
  COALESCE(cs.usdt_pair, p.usdt_pair, decode(repeat('00', 20), 'hex')) AS usdt_pair,
  COALESCE(cs.fetched_at, p.created_at) AS fetch_at,
  p.tx_hash,
  p.tx_index,
  COALESCE(sr.can_mint_from_dead_via_transfer_from, false) AS can_mint_from_dead_via_transfer_from,
  COALESCE(sr.can_mint_from_zero_via_transfer_from, false) AS can_mint_from_zero_via_transfer_from,
  COALESCE(sr.can_mint_from_weth_pair_via_transfer_from, false) AS can_mint_from_weth_pair_via_transfer_from,
  COALESCE(sr.can_mint_from_usdt_pair_via_transfer_from, false) AS can_mint_from_usdt_pair_via_transfer_from,
  COALESCE(sr.can_mint_via_transfer_to_weth_pair, false) AS can_mint_via_transfer_to_weth_pair,
  COALESCE(sr.can_mint_via_transfer_to_usdt_pair, false) AS can_mint_via_transfer_to_usdt_pair,
  gw.last_success_at AS genesis_wallets_fetched_at,
  ch.last_success_at AS creator_historical_projects_fetched_at
FROM project p
LEFT JOIN project_chain_state cs ON cs.project_id = p.id
LEFT JOIN project_simulation_result sr ON sr.project_id = p.id
LEFT JOIN project_component_state gw ON gw.project_id = p.id AND gw.component = 'genesis_wallet'
LEFT JOIN project_component_state ch ON ch.project_id = p.id AND ch.component = 'creator_history'
WHERE p.chain_id = @chain_id
  AND (cs.weth_pair = ANY(@pairs::bytea[])
    OR cs.usdt_pair = ANY(@pairs::bytea[])
    OR p.weth_pair = ANY(@pairs::bytea[])
    OR p.usdt_pair = ANY(@pairs::bytea[]))
ORDER BY p.block_number, p.tx_index, p.id;

-- name: ListProjectMetasByCreator :many
SELECT
  p.chain_id,
  p.block_number,
  p.block_time,
  p.contract,
  p.creator,
  COALESCE(cs.weth_pair, p.weth_pair, decode(repeat('00', 20), 'hex')) AS weth_pair,
  COALESCE(cs.usdt_pair, p.usdt_pair, decode(repeat('00', 20), 'hex')) AS usdt_pair,
  COALESCE(cs.fetched_at, p.created_at) AS fetch_at,
  p.tx_hash,
  p.tx_index,
  COALESCE(sr.can_mint_from_dead_via_transfer_from, false) AS can_mint_from_dead_via_transfer_from,
  COALESCE(sr.can_mint_from_zero_via_transfer_from, false) AS can_mint_from_zero_via_transfer_from,
  COALESCE(sr.can_mint_from_weth_pair_via_transfer_from, false) AS can_mint_from_weth_pair_via_transfer_from,
  COALESCE(sr.can_mint_from_usdt_pair_via_transfer_from, false) AS can_mint_from_usdt_pair_via_transfer_from,
  COALESCE(sr.can_mint_via_transfer_to_weth_pair, false) AS can_mint_via_transfer_to_weth_pair,
  COALESCE(sr.can_mint_via_transfer_to_usdt_pair, false) AS can_mint_via_transfer_to_usdt_pair,
  gw.last_success_at AS genesis_wallets_fetched_at,
  ch.last_success_at AS creator_historical_projects_fetched_at
FROM project p
LEFT JOIN project_chain_state cs ON cs.project_id = p.id
LEFT JOIN project_simulation_result sr ON sr.project_id = p.id
LEFT JOIN project_component_state gw ON gw.project_id = p.id AND gw.component = 'genesis_wallet'
LEFT JOIN project_component_state ch ON ch.project_id = p.id AND ch.component = 'creator_history'
WHERE p.chain_id = @chain_id
  AND p.creator = @creator
ORDER BY p.block_number, p.tx_index, p.id;

-- name: ListProjectMetasByCreatorBefore :many
SELECT
  p.chain_id,
  p.block_number,
  p.block_time,
  p.contract,
  p.creator,
  COALESCE(cs.weth_pair, p.weth_pair, decode(repeat('00', 20), 'hex')) AS weth_pair,
  COALESCE(cs.usdt_pair, p.usdt_pair, decode(repeat('00', 20), 'hex')) AS usdt_pair,
  COALESCE(cs.fetched_at, p.created_at) AS fetch_at,
  p.tx_hash,
  p.tx_index,
  COALESCE(sr.can_mint_from_dead_via_transfer_from, false) AS can_mint_from_dead_via_transfer_from,
  COALESCE(sr.can_mint_from_zero_via_transfer_from, false) AS can_mint_from_zero_via_transfer_from,
  COALESCE(sr.can_mint_from_weth_pair_via_transfer_from, false) AS can_mint_from_weth_pair_via_transfer_from,
  COALESCE(sr.can_mint_from_usdt_pair_via_transfer_from, false) AS can_mint_from_usdt_pair_via_transfer_from,
  COALESCE(sr.can_mint_via_transfer_to_weth_pair, false) AS can_mint_via_transfer_to_weth_pair,
  COALESCE(sr.can_mint_via_transfer_to_usdt_pair, false) AS can_mint_via_transfer_to_usdt_pair,
  gw.last_success_at AS genesis_wallets_fetched_at,
  ch.last_success_at AS creator_historical_projects_fetched_at
FROM project p
LEFT JOIN project_chain_state cs ON cs.project_id = p.id
LEFT JOIN project_simulation_result sr ON sr.project_id = p.id
LEFT JOIN project_component_state gw ON gw.project_id = p.id AND gw.component = 'genesis_wallet'
LEFT JOIN project_component_state ch ON ch.project_id = p.id AND ch.component = 'creator_history'
WHERE p.chain_id = @chain_id
  AND p.creator = @creator
  AND (p.block_number < @block_number OR (p.block_number = @block_number AND p.tx_index < @tx_index))
ORDER BY p.block_number, p.tx_index, p.id;

-- name: GetProjectMetaByContract :one
SELECT
  p.chain_id,
  p.block_number,
  p.block_time,
  p.contract,
  p.creator,
  COALESCE(cs.weth_pair, p.weth_pair, decode(repeat('00', 20), 'hex')) AS weth_pair,
  COALESCE(cs.usdt_pair, p.usdt_pair, decode(repeat('00', 20), 'hex')) AS usdt_pair,
  COALESCE(cs.fetched_at, p.created_at) AS fetch_at,
  p.tx_hash,
  p.tx_index,
  COALESCE(sr.can_mint_from_dead_via_transfer_from, false) AS can_mint_from_dead_via_transfer_from,
  COALESCE(sr.can_mint_from_zero_via_transfer_from, false) AS can_mint_from_zero_via_transfer_from,
  COALESCE(sr.can_mint_from_weth_pair_via_transfer_from, false) AS can_mint_from_weth_pair_via_transfer_from,
  COALESCE(sr.can_mint_from_usdt_pair_via_transfer_from, false) AS can_mint_from_usdt_pair_via_transfer_from,
  COALESCE(sr.can_mint_via_transfer_to_weth_pair, false) AS can_mint_via_transfer_to_weth_pair,
  COALESCE(sr.can_mint_via_transfer_to_usdt_pair, false) AS can_mint_via_transfer_to_usdt_pair,
  gw.last_success_at AS genesis_wallets_fetched_at,
  ch.last_success_at AS creator_historical_projects_fetched_at
FROM project p
LEFT JOIN project_chain_state cs ON cs.project_id = p.id
LEFT JOIN project_simulation_result sr ON sr.project_id = p.id
LEFT JOIN project_component_state gw ON gw.project_id = p.id AND gw.component = 'genesis_wallet'
LEFT JOIN project_component_state ch ON ch.project_id = p.id AND ch.component = 'creator_history'
WHERE p.chain_id = @chain_id
  AND p.contract = @contract;
