-- name: DeleteProjectGenesisWalletsByContract :exec
DELETE FROM project_genesis_wallet
WHERE project_contract = $1;

-- name: InsertProjectGenesisWallet :exec
INSERT INTO project_genesis_wallet (
  project_contract,
  wallet,
  net_amount,
  ratio_bps,
  rank_index,
  total_supply,
  source_tx_hash,
  source_block_number
) VALUES ($1, $2, $3::numeric, $4, $5, $6::numeric, $7, $8);

-- name: ListProjectGenesisWalletsByContract :many
SELECT
  id,
  project_contract,
  wallet,
  net_amount::text AS net_amount,
  ratio_bps,
  rank_index,
  total_supply::text AS total_supply,
  source_tx_hash,
  source_block_number,
  created_at
FROM project_genesis_wallet
WHERE project_contract = $1
ORDER BY rank_index ASC, id ASC;

-- name: ListProjectGenesisWalletsByContracts :many
SELECT
  id,
  project_contract,
  wallet,
  net_amount::text AS net_amount,
  ratio_bps,
  rank_index,
  total_supply::text AS total_supply,
  source_tx_hash,
  source_block_number,
  created_at
FROM project_genesis_wallet
WHERE project_contract = ANY($1::bytea[])
ORDER BY project_contract ASC, rank_index ASC, id ASC;

-- name: ListProjectGenesisWalletsByWallet :many
SELECT
  id,
  project_contract,
  wallet,
  net_amount::text AS net_amount,
  ratio_bps,
  rank_index,
  total_supply::text AS total_supply,
  source_tx_hash,
  source_block_number,
  created_at
FROM project_genesis_wallet
WHERE wallet = $1
ORDER BY ratio_bps DESC, project_contract ASC, id ASC;
