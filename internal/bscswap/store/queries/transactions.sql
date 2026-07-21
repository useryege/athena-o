-- name: InsertSwapTransaction :exec
INSERT INTO bsc_swap_transaction (
  transaction_hash,
  block_number,
  block_hash,
  block_timestamp,
  transaction_index,
  from_address
) VALUES (
  @transaction_hash,
  @block_number,
  @block_hash,
  @block_timestamp,
  @transaction_index,
  @from_address
)
ON CONFLICT (transaction_hash) DO NOTHING;

-- name: ListSwapTransactionsBeforeBlock :many
SELECT transaction_hash, block_number, block_hash, block_timestamp,
       transaction_index, from_address
FROM bsc_swap_transaction
WHERE from_address = @from_address
  AND block_number < @before_block_number
ORDER BY block_number DESC, transaction_index DESC
LIMIT @result_limit;

-- name: ListSwapTransactionsBeforePosition :many
SELECT transaction_hash, block_number, block_hash, block_timestamp,
       transaction_index, from_address
FROM bsc_swap_transaction
WHERE from_address = @from_address
  AND (block_number, transaction_index) < (@before_block_number, @before_transaction_index)
ORDER BY block_number DESC, transaction_index DESC
LIMIT @result_limit;
