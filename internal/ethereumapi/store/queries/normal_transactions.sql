-- name: GetNormalTransactionQueryCache :one
SELECT id, chain_id, address, start_block, end_block, page, page_size, sort_order, fetched_at, expires_at
FROM normal_transaction_query_cache
WHERE chain_id = @chain_id
  AND address = @address
  AND start_block = @start_block
  AND end_block = @end_block
  AND page = @page
  AND page_size = @page_size
  AND sort_order = @sort_order;

-- name: ListNormalTransactionQueryItems :many
SELECT
  item.position,
  tx.chain_id,
  tx.tx_hash,
  tx.block_number,
  tx.block_hash,
  tx.block_timestamp,
  tx.nonce,
  tx.transaction_index,
  tx.from_address,
  tx.to_address,
  tx.value,
  tx.gas,
  tx.gas_price,
  tx.input,
  tx.method_id,
  tx.function_name,
  tx.contract_address,
  tx.cumulative_gas_used,
  tx.tx_receipt_status,
  tx.gas_used,
  tx.confirmations,
  tx.is_error,
  tx.fetched_at
FROM normal_transaction_query_item AS item
JOIN normal_transaction AS tx
  ON tx.chain_id = item.chain_id
  AND tx.tx_hash = item.tx_hash
WHERE item.query_id = @query_id
ORDER BY item.position;

-- name: BatchUpsertNormalTransactions :exec
INSERT INTO normal_transaction (
  chain_id,
  tx_hash,
  block_number,
  block_hash,
  block_timestamp,
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
  tx_receipt_status,
  gas_used,
  confirmations,
  is_error,
  fetched_at
)
SELECT
  unnest(sqlc.arg('chain_id_values')::bigint[]),
  unnest(sqlc.arg('tx_hash_values')::bytea[]),
  unnest(sqlc.arg('block_number_values')::bigint[]),
  unnest(sqlc.arg('block_hash_values')::bytea[]),
  unnest(sqlc.arg('block_timestamp_values')::bigint[]),
  unnest(sqlc.arg('nonce_values')::numeric[]),
  unnest(sqlc.arg('transaction_index_values')::bigint[]),
  unnest(sqlc.arg('from_address_values')::bytea[]),
  unnest(sqlc.arg('to_address_values')::bytea[]),
  unnest(sqlc.arg('value_values')::numeric[]),
  unnest(sqlc.arg('gas_values')::numeric[]),
  unnest(sqlc.arg('gas_price_values')::numeric[]),
  unnest(sqlc.arg('input_values')::text[]),
  unnest(sqlc.arg('method_id_values')::bytea[]),
  unnest(sqlc.arg('function_name_values')::text[]),
  unnest(sqlc.arg('contract_address_values')::bytea[]),
  unnest(sqlc.arg('cumulative_gas_used_values')::numeric[]),
  unnest(sqlc.arg('tx_receipt_status_values')::smallint[]),
  unnest(sqlc.arg('gas_used_values')::numeric[]),
  unnest(sqlc.arg('confirmations_values')::numeric[]),
  unnest(sqlc.arg('is_error_values')::boolean[]),
  unnest(sqlc.arg('fetched_at_values')::timestamptz[])
ON CONFLICT (chain_id, tx_hash) DO UPDATE
SET block_number = EXCLUDED.block_number,
  block_hash = EXCLUDED.block_hash,
  block_timestamp = EXCLUDED.block_timestamp,
  nonce = EXCLUDED.nonce,
  transaction_index = EXCLUDED.transaction_index,
  from_address = EXCLUDED.from_address,
  to_address = EXCLUDED.to_address,
  value = EXCLUDED.value,
  gas = EXCLUDED.gas,
  gas_price = EXCLUDED.gas_price,
  input = EXCLUDED.input,
  method_id = EXCLUDED.method_id,
  function_name = EXCLUDED.function_name,
  contract_address = EXCLUDED.contract_address,
  cumulative_gas_used = EXCLUDED.cumulative_gas_used,
  tx_receipt_status = EXCLUDED.tx_receipt_status,
  gas_used = EXCLUDED.gas_used,
  confirmations = EXCLUDED.confirmations,
  is_error = EXCLUDED.is_error,
  fetched_at = EXCLUDED.fetched_at;

-- name: UpsertNormalTransactionQueryCache :one
INSERT INTO normal_transaction_query_cache (
  chain_id,
  address,
  start_block,
  end_block,
  page,
  page_size,
  sort_order,
  fetched_at,
  expires_at
) VALUES (
  @chain_id,
  @address,
  @start_block,
  @end_block,
  @page,
  @page_size,
  @sort_order,
  @fetched_at,
  @expires_at
)
ON CONFLICT (chain_id, address, start_block, end_block, page, page_size, sort_order) DO UPDATE
SET fetched_at = EXCLUDED.fetched_at,
  expires_at = EXCLUDED.expires_at
RETURNING id, chain_id, address, start_block, end_block, page, page_size, sort_order, fetched_at, expires_at;

-- name: DeleteNormalTransactionQueryItems :exec
DELETE FROM normal_transaction_query_item
WHERE query_id = @query_id;

-- name: BatchCreateNormalTransactionQueryItems :exec
INSERT INTO normal_transaction_query_item (
  query_id,
  position,
  chain_id,
  tx_hash
)
SELECT
  unnest(sqlc.arg('query_id_values')::bigint[]),
  unnest(sqlc.arg('position_values')::integer[]),
  unnest(sqlc.arg('chain_id_values')::bigint[]),
  unnest(sqlc.arg('tx_hash_values')::bytea[]);

-- name: DeleteExpiredNormalTransactionQueryCaches :execrows
DELETE FROM normal_transaction_query_cache
WHERE expires_at < @delete_before;
