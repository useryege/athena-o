-- name: UpsertProjectSimulationResult :exec
INSERT INTO project_simulation_result (
  project_contract,
  can_mint_from_dead_via_transfer_from,
  can_mint_from_zero_via_transfer_from,
  can_mint_from_weth_pair_via_transfer_from,
  can_mint_from_usdt_pair_via_transfer_from,
  can_mint_via_transfer_to_weth_pair,
  can_mint_via_transfer_to_usdt_pair,
  fetched_at
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
ON CONFLICT (project_contract) DO UPDATE
SET can_mint_from_dead_via_transfer_from = EXCLUDED.can_mint_from_dead_via_transfer_from,
  can_mint_from_zero_via_transfer_from = EXCLUDED.can_mint_from_zero_via_transfer_from,
  can_mint_from_weth_pair_via_transfer_from = EXCLUDED.can_mint_from_weth_pair_via_transfer_from,
  can_mint_from_usdt_pair_via_transfer_from = EXCLUDED.can_mint_from_usdt_pair_via_transfer_from,
  can_mint_via_transfer_to_weth_pair = EXCLUDED.can_mint_via_transfer_to_weth_pair,
  can_mint_via_transfer_to_usdt_pair = EXCLUDED.can_mint_via_transfer_to_usdt_pair,
  fetched_at = EXCLUDED.fetched_at,
  updated_at = now();

-- name: GetProjectSimulationResult :one
SELECT project_contract,
  can_mint_from_dead_via_transfer_from,
  can_mint_from_zero_via_transfer_from,
  can_mint_from_weth_pair_via_transfer_from,
  can_mint_from_usdt_pair_via_transfer_from,
  can_mint_via_transfer_to_weth_pair,
  can_mint_via_transfer_to_usdt_pair,
  fetched_at,
  updated_at
FROM project_simulation_result
WHERE project_contract = $1;
