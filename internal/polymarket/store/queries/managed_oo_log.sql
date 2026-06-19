-- name: GetPolymarketChainLogCursor :one
SELECT
  sync_name,
  contract_address,
  topic,
  last_block_number,
  last_polled_at,
  created_at,
  updated_at
FROM polymarket_chain_log_cursor
WHERE sync_name = @sync_name;

-- name: UpsertPolymarketChainLogCursor :one
INSERT INTO polymarket_chain_log_cursor (
  sync_name,
  contract_address,
  topic,
  last_block_number,
  last_polled_at
) VALUES (
  @sync_name,
  @contract_address,
  @topic,
  @last_block_number,
  @last_polled_at
)
ON CONFLICT (sync_name) DO UPDATE
SET contract_address = EXCLUDED.contract_address,
  topic = EXCLUDED.topic,
  last_block_number = EXCLUDED.last_block_number,
  last_polled_at = EXCLUDED.last_polled_at,
  updated_at = now()
RETURNING *;

-- name: BatchUpsertManagedOOProposePriceLogs :exec
INSERT INTO polymarket_managed_oo_propose_price_log (
  tx_hash,
  log_index,
  block_number,
  block_hash,
  tx_index,
  contract_address,
  topic,
  requester,
  proposer,
  identifier,
  request_timestamp,
  ancillary_data_hex,
  ancillary_data_text,
  proposed_price,
  expiration_timestamp,
  currency,
  raw_topics,
  raw_data,
  fetched_at
)
SELECT
  unnest(sqlc.arg('tx_hashes')::text[]),
  unnest(sqlc.arg('log_indexes')::bigint[]),
  unnest(sqlc.arg('block_numbers')::bigint[]),
  unnest(sqlc.arg('block_hashes')::text[]),
  unnest(sqlc.arg('tx_indexes')::bigint[]),
  unnest(sqlc.arg('contract_addresses')::text[]),
  unnest(sqlc.arg('topics')::text[]),
  unnest(sqlc.arg('requesters')::text[]),
  unnest(sqlc.arg('proposers')::text[]),
  unnest(sqlc.arg('identifiers')::text[]),
  unnest(sqlc.arg('request_timestamps')::bigint[]),
  unnest(sqlc.arg('ancillary_data_hex_values')::text[]),
  unnest(sqlc.arg('ancillary_data_text_values')::text[]),
  unnest(sqlc.arg('proposed_prices')::text[]),
  unnest(sqlc.arg('expiration_timestamps')::bigint[]),
  unnest(sqlc.arg('currencies')::text[]),
  unnest(sqlc.arg('raw_topics_values')::jsonb[]),
  unnest(sqlc.arg('raw_data_values')::text[]),
  unnest(sqlc.arg('fetched_at_values')::timestamptz[])
ON CONFLICT (tx_hash, log_index) DO UPDATE
SET block_number = EXCLUDED.block_number,
  block_hash = EXCLUDED.block_hash,
  tx_index = EXCLUDED.tx_index,
  contract_address = EXCLUDED.contract_address,
  topic = EXCLUDED.topic,
  requester = EXCLUDED.requester,
  proposer = EXCLUDED.proposer,
  identifier = EXCLUDED.identifier,
  request_timestamp = EXCLUDED.request_timestamp,
  ancillary_data_hex = EXCLUDED.ancillary_data_hex,
  ancillary_data_text = EXCLUDED.ancillary_data_text,
  proposed_price = EXCLUDED.proposed_price,
  expiration_timestamp = EXCLUDED.expiration_timestamp,
  currency = EXCLUDED.currency,
  raw_topics = EXCLUDED.raw_topics,
  raw_data = EXCLUDED.raw_data,
  fetched_at = EXCLUDED.fetched_at,
  updated_at = now();
