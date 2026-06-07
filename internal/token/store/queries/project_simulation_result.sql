-- name: UpsertProjectSimulationResult :one
INSERT INTO project_simulation_result (
  project_id,
  wallet,
  can_mint_from_dead_via_transfer_from,
  can_mint_from_zero_via_transfer_from,
  can_mint_from_weth_pair_via_transfer_from,
  can_mint_from_usdt_pair_via_transfer_from,
  can_mint_via_transfer_to_weth_pair,
  can_mint_via_transfer_to_usdt_pair,
  fetched_at
) VALUES (
  @project_id,
  @wallet,
  @can_mint_from_dead_via_transfer_from,
  @can_mint_from_zero_via_transfer_from,
  @can_mint_from_weth_pair_via_transfer_from,
  @can_mint_from_usdt_pair_via_transfer_from,
  @can_mint_via_transfer_to_weth_pair,
  @can_mint_via_transfer_to_usdt_pair,
  @fetched_at
)
ON CONFLICT (project_id, wallet) DO UPDATE
SET can_mint_from_dead_via_transfer_from = EXCLUDED.can_mint_from_dead_via_transfer_from,
  can_mint_from_zero_via_transfer_from = EXCLUDED.can_mint_from_zero_via_transfer_from,
  can_mint_from_weth_pair_via_transfer_from = EXCLUDED.can_mint_from_weth_pair_via_transfer_from,
  can_mint_from_usdt_pair_via_transfer_from = EXCLUDED.can_mint_from_usdt_pair_via_transfer_from,
  can_mint_via_transfer_to_weth_pair = EXCLUDED.can_mint_via_transfer_to_weth_pair,
  can_mint_via_transfer_to_usdt_pair = EXCLUDED.can_mint_via_transfer_to_usdt_pair,
  fetched_at = EXCLUDED.fetched_at
RETURNING *;

-- name: GetProjectSimulationResult :one
SELECT *
FROM project_simulation_result
WHERE project_id = @project_id
  AND wallet = @wallet;

-- name: ListProjectSimulationResultsByProject :many
SELECT *
FROM project_simulation_result
WHERE project_id = @project_id
ORDER BY (
    can_mint_from_dead_via_transfer_from
    OR can_mint_from_zero_via_transfer_from
    OR can_mint_from_weth_pair_via_transfer_from
    OR can_mint_from_usdt_pair_via_transfer_from
    OR can_mint_via_transfer_to_weth_pair
    OR can_mint_via_transfer_to_usdt_pair
  ) DESC,
  fetched_at DESC,
  wallet;

-- name: ListProjectSimulationResultsByWallet :many
SELECT *
FROM project_simulation_result
WHERE wallet = @wallet
ORDER BY fetched_at DESC, project_id;

-- name: DeleteProjectSimulationResult :execrows
DELETE FROM project_simulation_result
WHERE project_id = @project_id
  AND wallet = @wallet;

-- name: DeleteProjectSimulationResultsByProject :execrows
DELETE FROM project_simulation_result
WHERE project_id = @project_id;
