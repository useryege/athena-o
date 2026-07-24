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
