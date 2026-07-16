-- One-time related-wallet normal transaction history persistence.
-- name: ListPendingProjectWalletNormalTransactionHistoryWallets :many
SELECT DISTINCT related.wallet
FROM project_related_wallet AS related
LEFT JOIN project_wallet_normal_transaction_history AS history
  ON history.project_id = related.project_id
  AND history.wallet = related.wallet
WHERE related.project_id = @project_id
  AND history.project_id IS NULL
ORDER BY related.wallet;

-- name: InsertProjectWalletNormalTransactionHistory :one
INSERT INTO project_wallet_normal_transaction_history (
  project_id,
  wallet,
  anchor_block_number,
  anchor_transaction_index,
  requested_transaction_count,
  collected_transaction_count,
  fetched_at
) VALUES (
  @project_id,
  @wallet,
  @anchor_block_number,
  @anchor_transaction_index,
  @requested_transaction_count,
  @collected_transaction_count,
  @fetched_at
)
ON CONFLICT (project_id, wallet) DO NOTHING
RETURNING *;

-- name: InsertProjectWalletNormalTransaction :exec
INSERT INTO project_wallet_normal_transaction (
  project_id,
  wallet,
  rank_index,
  block_number,
  block_hash,
  block_timestamp,
  transaction_hash,
  nonce,
  transaction_index,
  from_address,
  to_address,
  value,
  gas,
  gas_price,
  input,
  method_id,
  function_name,
  contract_address,
  cumulative_gas_used,
  receipt_status,
  gas_used,
  confirmations,
  is_error
) VALUES (
  @project_id,
  @wallet,
  @rank_index,
  @block_number,
  @block_hash,
  @block_timestamp,
  @transaction_hash,
  @nonce,
  @transaction_index,
  @from_address,
  sqlc.narg('to_address'),
  @value,
  @gas,
  @gas_price,
  @input,
  sqlc.narg('method_id'),
  @function_name,
  sqlc.narg('contract_address'),
  @cumulative_gas_used,
  @receipt_status,
  @gas_used,
  @confirmations,
  @is_error
);
