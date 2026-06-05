-- name: UpsertProjectSimulationResult :exec
INSERT INTO project_simulation_result (
  project_id,
  can_mint_from_dead_via_transfer_from,
  can_mint_from_zero_via_transfer_from,
  can_mint_from_weth_pair_via_transfer_from,
  can_mint_from_usdt_pair_via_transfer_from,
  can_mint_via_transfer_to_weth_pair,
  can_mint_via_transfer_to_usdt_pair,
  fetched_at
) VALUES (
  (SELECT id FROM project WHERE chain_id = @chain_id AND contract = @project_contract),
  @can_mint_from_dead_via_transfer_from,
  @can_mint_from_zero_via_transfer_from,
  @can_mint_from_weth_pair_via_transfer_from,
  @can_mint_from_usdt_pair_via_transfer_from,
  @can_mint_via_transfer_to_weth_pair,
  @can_mint_via_transfer_to_usdt_pair,
  @fetched_at
)
ON CONFLICT (project_id) DO UPDATE
SET can_mint_from_dead_via_transfer_from = EXCLUDED.can_mint_from_dead_via_transfer_from,
  can_mint_from_zero_via_transfer_from = EXCLUDED.can_mint_from_zero_via_transfer_from,
  can_mint_from_weth_pair_via_transfer_from = EXCLUDED.can_mint_from_weth_pair_via_transfer_from,
  can_mint_from_usdt_pair_via_transfer_from = EXCLUDED.can_mint_from_usdt_pair_via_transfer_from,
  can_mint_via_transfer_to_weth_pair = EXCLUDED.can_mint_via_transfer_to_weth_pair,
  can_mint_via_transfer_to_usdt_pair = EXCLUDED.can_mint_via_transfer_to_usdt_pair,
  fetched_at = EXCLUDED.fetched_at,
  updated_at = now();

-- name: GetProjectSimulationResult :one
SELECT p.chain_id,
  p.contract AS project_contract,
  sr.can_mint_from_dead_via_transfer_from,
  sr.can_mint_from_zero_via_transfer_from,
  sr.can_mint_from_weth_pair_via_transfer_from,
  sr.can_mint_from_usdt_pair_via_transfer_from,
  sr.can_mint_via_transfer_to_weth_pair,
  sr.can_mint_via_transfer_to_usdt_pair,
  sr.fetched_at,
  sr.updated_at
FROM project_simulation_result sr
JOIN project p ON p.id = sr.project_id
WHERE p.chain_id = @chain_id
  AND p.contract = @project_contract;
