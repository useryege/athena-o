-- name: DeleteProjectGenesisWalletsByContract :exec
DELETE FROM project_genesis_wallet
WHERE project_id = (SELECT id FROM project WHERE chain_id = @chain_id AND contract = @project_contract);

-- name: InsertProjectGenesisWallet :exec
INSERT INTO project_genesis_wallet (
  project_id,
  wallet,
  net_amount,
  ratio_bps,
  rank_index,
  total_supply,
  source_tx_hash,
  source_block_number
) VALUES (
  (SELECT id FROM project WHERE chain_id = @chain_id AND contract = @project_contract),
  @wallet,
  @net_amount::numeric,
  @ratio_bps,
  @rank_index,
  @total_supply::numeric,
  @source_tx_hash,
  @source_block_number
);

-- name: ListProjectGenesisWalletsByContract :many
SELECT
  gw.id,
  p.chain_id,
  p.contract AS project_contract,
  gw.wallet,
  gw.net_amount::text AS net_amount,
  gw.ratio_bps,
  gw.rank_index,
  gw.total_supply::text AS total_supply,
  gw.source_tx_hash,
  gw.source_block_number,
  gw.created_at
FROM project_genesis_wallet gw
JOIN project p ON p.id = gw.project_id
WHERE p.chain_id = @chain_id
  AND p.contract = @project_contract
ORDER BY gw.rank_index ASC, gw.id ASC;

-- name: ListProjectGenesisWalletsByContracts :many
SELECT
  gw.id,
  p.chain_id,
  p.contract AS project_contract,
  gw.wallet,
  gw.net_amount::text AS net_amount,
  gw.ratio_bps,
  gw.rank_index,
  gw.total_supply::text AS total_supply,
  gw.source_tx_hash,
  gw.source_block_number,
  gw.created_at
FROM project_genesis_wallet gw
JOIN project p ON p.id = gw.project_id
WHERE p.chain_id = @chain_id
  AND p.contract = ANY(@project_contracts::bytea[])
ORDER BY p.contract ASC, gw.rank_index ASC, gw.id ASC;

-- name: ListProjectGenesisWalletsByWallet :many
SELECT
  gw.id,
  p.chain_id,
  p.contract AS project_contract,
  gw.wallet,
  gw.net_amount::text AS net_amount,
  gw.ratio_bps,
  gw.rank_index,
  gw.total_supply::text AS total_supply,
  gw.source_tx_hash,
  gw.source_block_number,
  gw.created_at
FROM project_genesis_wallet gw
JOIN project p ON p.id = gw.project_id
WHERE p.chain_id = @chain_id
  AND gw.wallet = @wallet
ORDER BY gw.ratio_bps DESC, p.contract ASC, gw.id ASC;
