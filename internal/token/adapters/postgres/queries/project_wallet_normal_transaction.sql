-- One-time pre-deployment normal transactions for project-related wallets.
-- name: InsertProjectWalletNormalTransaction :exec
INSERT INTO project_wallet_normal_transaction (
  project_id,
  wallet,
  transaction_hash,
  block_number,
  block_timestamp,
  transaction_index,
  nonce,
  from_address,
  to_address,
  value,
  gas,
  gas_price,
  gas_used,
  input,
  method_id,
  function_name,
  receipt_status,
  is_error,
  collected_at
) VALUES (
  @project_id,
  @wallet,
  @transaction_hash,
  @block_number,
  @block_timestamp,
  @transaction_index,
  @nonce,
  @from_address,
  @to_address,
  @value,
  @gas,
  @gas_price,
  @gas_used,
  @input,
  @method_id,
  @function_name,
  @receipt_status,
  @is_error,
  @collected_at
)
ON CONFLICT (project_id, wallet, transaction_hash) DO NOTHING;

-- name: CountProjectWalletNormalTransactions :one
SELECT COUNT(*)::bigint
FROM project_wallet_normal_transaction
WHERE project_id = @project_id
  AND (sqlc.narg('wallet')::bytea IS NULL OR wallet = sqlc.narg('wallet')::bytea)
  AND (sqlc.arg('receipt_status')::text = '' OR receipt_status = sqlc.arg('receipt_status')::text)
  AND (sqlc.arg('method_id')::text = '' OR method_id = sqlc.arg('method_id')::text);

-- name: ListProjectWalletNormalTransactions :many
SELECT *
FROM project_wallet_normal_transaction
WHERE project_id = @project_id
  AND (sqlc.narg('wallet')::bytea IS NULL OR wallet = sqlc.narg('wallet')::bytea)
  AND (sqlc.arg('receipt_status')::text = '' OR receipt_status = sqlc.arg('receipt_status')::text)
  AND (sqlc.arg('method_id')::text = '' OR method_id = sqlc.arg('method_id')::text)
ORDER BY block_number DESC, transaction_index DESC, transaction_hash
LIMIT sqlc.arg('limit') OFFSET sqlc.arg('offset');

-- name: CountProjectWalletNormalTransactionsByWallet :many
SELECT wallet, COUNT(*)::bigint AS transaction_count
FROM project_wallet_normal_transaction
WHERE project_id = @project_id
GROUP BY wallet
ORDER BY wallet;
