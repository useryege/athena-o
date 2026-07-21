-- One-time related-wallet BSC funding-source history persistence.
-- name: ListPendingProjectWalletFundingSourceHistoryWallets :many
SELECT DISTINCT related.wallet
FROM project_related_wallet AS related
LEFT JOIN project_wallet_funding_source_history AS history
  ON history.project_id = related.project_id
  AND history.wallet = related.wallet
WHERE related.project_id = @project_id
  AND history.project_id IS NULL
ORDER BY related.wallet;

-- name: InsertProjectWalletFundingSourceHistory :one
INSERT INTO project_wallet_funding_source_history (
  project_id,
  wallet,
  anchor_block_number,
  anchor_transaction_index,
  requested_transaction_count,
  collected_transaction_count,
  indexed_through_block,
  indexed_through_timestamp,
  fetched_at
) VALUES (
  @project_id,
  @wallet,
  @anchor_block_number,
  @anchor_transaction_index,
  @requested_transaction_count,
  @collected_transaction_count,
  @indexed_through_block,
  @indexed_through_timestamp,
  @fetched_at
)
ON CONFLICT (project_id, wallet) DO NOTHING
RETURNING *;

-- name: InsertProjectWalletFundingSourceTransaction :exec
INSERT INTO project_wallet_funding_source_transaction (
  project_id,
  wallet,
  rank_index,
  block_number,
  block_hash,
  block_timestamp,
  transaction_hash,
  transaction_index,
  from_address,
  to_address,
  value_wei
) VALUES (
  @project_id,
  @wallet,
  @rank_index,
  @block_number,
  @block_hash,
  @block_timestamp,
  @transaction_hash,
  @transaction_index,
  @from_address,
  @to_address,
  @value_wei
);
