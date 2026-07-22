-- One-time related-wallet BSC V2 Swap transaction-hash history persistence.
-- name: ListPendingProjectWalletSwapTransactionHistoryWallets :many
SELECT DISTINCT related.wallet
FROM project_related_wallet AS related
LEFT JOIN project_wallet_swap_transaction_history AS history
  ON history.project_id = related.project_id
  AND history.wallet = related.wallet
WHERE related.project_id = @project_id
  AND history.project_id IS NULL
ORDER BY related.wallet;

-- name: InsertProjectWalletSwapTransactionHistory :one
INSERT INTO project_wallet_swap_transaction_history (
  project_id,
  wallet,
  anchor_block_number,
  requested_transaction_count,
  collected_transaction_count,
  indexed_through_block,
  indexed_through_timestamp,
  fetched_at
) VALUES (
  @project_id,
  @wallet,
  @anchor_block_number,
  @requested_transaction_count,
  @collected_transaction_count,
  @indexed_through_block,
  @indexed_through_timestamp,
  @fetched_at
)
ON CONFLICT (project_id, wallet) DO NOTHING
RETURNING *;

-- name: InsertProjectWalletSwapTransaction :exec
INSERT INTO project_wallet_swap_transaction (
  project_id,
  wallet,
  rank_index,
  transaction_hash
) VALUES (
  @project_id,
  @wallet,
  @rank_index,
  @transaction_hash
);
