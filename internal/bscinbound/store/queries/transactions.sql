-- name: InsertInboundNormalTransaction :exec
INSERT INTO inbound_normal_transaction (
  transaction_hash,
  block_number,
  block_hash,
  block_timestamp,
  transaction_index,
  from_address,
  to_address,
  value_wei
) VALUES (
  @transaction_hash,
  @block_number,
  @block_hash,
  @block_timestamp,
  @transaction_index,
  @from_address,
  @to_address,
  @value_wei
)
ON CONFLICT (transaction_hash) DO NOTHING;

-- name: ListLatestInboundNormalTransactions :many
SELECT transaction_hash, block_number, block_hash, block_timestamp,
       transaction_index, from_address, to_address, value_wei
FROM inbound_normal_transaction
WHERE to_address = @to_address
ORDER BY block_number DESC, transaction_index DESC
LIMIT @result_limit;

-- name: ListInboundNormalTransactionsBefore :many
SELECT transaction_hash, block_number, block_hash, block_timestamp,
       transaction_index, from_address, to_address, value_wei
FROM inbound_normal_transaction
WHERE to_address = @to_address
  AND (block_number, transaction_index) < (@before_block_number, @before_transaction_index)
ORDER BY block_number DESC, transaction_index DESC
LIMIT @result_limit;
