-- name: ListProjectMetas :many
SELECT
  p.chain_id,
  p.block_number,
  p.block_time,
  p.contract,
  p.creator,
  COALESCE(cs.weth_pair, decode(repeat('00', 20), 'hex')) AS weth_pair,
  COALESCE(cs.usdt_pair, decode(repeat('00', 20), 'hex')) AS usdt_pair,
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
  COALESCE(cs.weth_pair, decode(repeat('00', 20), 'hex')) AS weth_pair,
  COALESCE(cs.usdt_pair, decode(repeat('00', 20), 'hex')) AS usdt_pair,
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
  AND (cs.weth_pair = ANY(@pairs::bytea[]) OR cs.usdt_pair = ANY(@pairs::bytea[]))
ORDER BY p.block_number, p.tx_index, p.id;

-- name: ListProjectMetasByCreator :many
SELECT
  p.chain_id,
  p.block_number,
  p.block_time,
  p.contract,
  p.creator,
  COALESCE(cs.weth_pair, decode(repeat('00', 20), 'hex')) AS weth_pair,
  COALESCE(cs.usdt_pair, decode(repeat('00', 20), 'hex')) AS usdt_pair,
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
  COALESCE(cs.weth_pair, decode(repeat('00', 20), 'hex')) AS weth_pair,
  COALESCE(cs.usdt_pair, decode(repeat('00', 20), 'hex')) AS usdt_pair,
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
  COALESCE(cs.weth_pair, decode(repeat('00', 20), 'hex')) AS weth_pair,
  COALESCE(cs.usdt_pair, decode(repeat('00', 20), 'hex')) AS usdt_pair,
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
