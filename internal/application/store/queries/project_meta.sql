-- name: ListProjectMetas :many
SELECT
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
  COALESCE(pr.is_report_evaluated, false) AS is_report_evaluated,
  COALESCE(pr.is_report_complete, false) AS is_report_complete,
  COALESCE(pr.is_blacklisted_creator_wallet, false) AS is_blacklisted_creator_wallet,
  COALESCE(pr.is_blacklisted_genesis_wallet, false) AS is_blacklisted_genesis_wallet,
  COALESCE(pr.is_blacklisted_bytecode, false) AS is_blacklisted_bytecode,
  COALESCE(pr.has_mint_risk, false) AS has_mint_risk,
  gw.last_success_at AS genesis_wallets_fetched_at,
  ch.last_success_at AS creator_historical_projects_fetched_at
FROM project p
LEFT JOIN project_chain_state cs ON cs.project_contract = p.contract
LEFT JOIN project_simulation_result sr ON sr.project_contract = p.contract
LEFT JOIN project_report pr ON pr.project_contract = p.contract
LEFT JOIN project_component_state gw ON gw.project_contract = p.contract AND gw.component = 'genesis_wallet'
LEFT JOIN project_component_state ch ON ch.project_contract = p.contract AND ch.component = 'creator_history'
ORDER BY p.block_number, p.tx_index, p.id;

-- name: ListProjectMetasByPairAddresses :many
SELECT
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
  COALESCE(pr.is_report_evaluated, false) AS is_report_evaluated,
  COALESCE(pr.is_report_complete, false) AS is_report_complete,
  COALESCE(pr.is_blacklisted_creator_wallet, false) AS is_blacklisted_creator_wallet,
  COALESCE(pr.is_blacklisted_genesis_wallet, false) AS is_blacklisted_genesis_wallet,
  COALESCE(pr.is_blacklisted_bytecode, false) AS is_blacklisted_bytecode,
  COALESCE(pr.has_mint_risk, false) AS has_mint_risk,
  gw.last_success_at AS genesis_wallets_fetched_at,
  ch.last_success_at AS creator_historical_projects_fetched_at
FROM project p
LEFT JOIN project_chain_state cs ON cs.project_contract = p.contract
LEFT JOIN project_simulation_result sr ON sr.project_contract = p.contract
LEFT JOIN project_report pr ON pr.project_contract = p.contract
LEFT JOIN project_component_state gw ON gw.project_contract = p.contract AND gw.component = 'genesis_wallet'
LEFT JOIN project_component_state ch ON ch.project_contract = p.contract AND ch.component = 'creator_history'
WHERE cs.weth_pair = ANY($1::bytea[]) OR cs.usdt_pair = ANY($1::bytea[])
ORDER BY p.block_number, p.tx_index, p.id;

-- name: ListProjectMetasByCreator :many
SELECT
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
  COALESCE(pr.is_report_evaluated, false) AS is_report_evaluated,
  COALESCE(pr.is_report_complete, false) AS is_report_complete,
  COALESCE(pr.is_blacklisted_creator_wallet, false) AS is_blacklisted_creator_wallet,
  COALESCE(pr.is_blacklisted_genesis_wallet, false) AS is_blacklisted_genesis_wallet,
  COALESCE(pr.is_blacklisted_bytecode, false) AS is_blacklisted_bytecode,
  COALESCE(pr.has_mint_risk, false) AS has_mint_risk,
  gw.last_success_at AS genesis_wallets_fetched_at,
  ch.last_success_at AS creator_historical_projects_fetched_at
FROM project p
LEFT JOIN project_chain_state cs ON cs.project_contract = p.contract
LEFT JOIN project_simulation_result sr ON sr.project_contract = p.contract
LEFT JOIN project_report pr ON pr.project_contract = p.contract
LEFT JOIN project_component_state gw ON gw.project_contract = p.contract AND gw.component = 'genesis_wallet'
LEFT JOIN project_component_state ch ON ch.project_contract = p.contract AND ch.component = 'creator_history'
WHERE p.creator = $1
ORDER BY p.block_number, p.tx_index, p.id;

-- name: ListProjectMetasByCreatorBefore :many
SELECT
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
  COALESCE(pr.is_report_evaluated, false) AS is_report_evaluated,
  COALESCE(pr.is_report_complete, false) AS is_report_complete,
  COALESCE(pr.is_blacklisted_creator_wallet, false) AS is_blacklisted_creator_wallet,
  COALESCE(pr.is_blacklisted_genesis_wallet, false) AS is_blacklisted_genesis_wallet,
  COALESCE(pr.is_blacklisted_bytecode, false) AS is_blacklisted_bytecode,
  COALESCE(pr.has_mint_risk, false) AS has_mint_risk,
  gw.last_success_at AS genesis_wallets_fetched_at,
  ch.last_success_at AS creator_historical_projects_fetched_at
FROM project p
LEFT JOIN project_chain_state cs ON cs.project_contract = p.contract
LEFT JOIN project_simulation_result sr ON sr.project_contract = p.contract
LEFT JOIN project_report pr ON pr.project_contract = p.contract
LEFT JOIN project_component_state gw ON gw.project_contract = p.contract AND gw.component = 'genesis_wallet'
LEFT JOIN project_component_state ch ON ch.project_contract = p.contract AND ch.component = 'creator_history'
WHERE p.creator = $1
  AND (p.block_number < $2 OR (p.block_number = $2 AND p.tx_index < $3))
ORDER BY p.block_number, p.tx_index, p.id;

-- name: GetProjectMetaByContract :one
SELECT
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
  COALESCE(pr.is_report_evaluated, false) AS is_report_evaluated,
  COALESCE(pr.is_report_complete, false) AS is_report_complete,
  COALESCE(pr.is_blacklisted_creator_wallet, false) AS is_blacklisted_creator_wallet,
  COALESCE(pr.is_blacklisted_genesis_wallet, false) AS is_blacklisted_genesis_wallet,
  COALESCE(pr.is_blacklisted_bytecode, false) AS is_blacklisted_bytecode,
  COALESCE(pr.has_mint_risk, false) AS has_mint_risk,
  gw.last_success_at AS genesis_wallets_fetched_at,
  ch.last_success_at AS creator_historical_projects_fetched_at
FROM project p
LEFT JOIN project_chain_state cs ON cs.project_contract = p.contract
LEFT JOIN project_simulation_result sr ON sr.project_contract = p.contract
LEFT JOIN project_report pr ON pr.project_contract = p.contract
LEFT JOIN project_component_state gw ON gw.project_contract = p.contract AND gw.component = 'genesis_wallet'
LEFT JOIN project_component_state ch ON ch.project_contract = p.contract AND ch.component = 'creator_history'
WHERE p.contract = $1;
