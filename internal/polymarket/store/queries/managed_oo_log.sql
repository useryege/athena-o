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
  market_id,
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
  unnest(sqlc.arg('market_ids')::text[]),
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
  market_id = EXCLUDED.market_id,
  proposed_price = EXCLUDED.proposed_price,
  expiration_timestamp = EXCLUDED.expiration_timestamp,
  currency = EXCLUDED.currency,
  raw_topics = EXCLUDED.raw_topics,
  raw_data = EXCLUDED.raw_data,
  fetched_at = EXCLUDED.fetched_at,
  updated_at = now();

-- name: ListManagedOOMarketIDsMissingData :many
SELECT DISTINCT log.market_id
FROM polymarket_managed_oo_propose_price_log AS log
LEFT JOIN polymarket_managed_oo_market AS market
  ON market.market_id = log.market_id
WHERE log.market_id <> ''
  AND log.market_id ~ '^[0-9]+$'
  AND market.market_id IS NULL
ORDER BY log.market_id
LIMIT @limit_value;

-- name: UpsertManagedOOMarket :exec
INSERT INTO polymarket_managed_oo_market (
  market_id,
  condition_id,
  slug,
  question,
  description,
  resolution_source,
  question_id,
  sports_market_type,
  group_item_title,
  image,
  icon,
  outcomes,
  outcome_prices,
  clob_token_ids,
  active,
  closed,
  archived,
  restricted,
  enable_order_book,
  accepting_orders,
  volume,
  volume_num,
  liquidity_num,
  volume_24hr,
  volume_1wk,
  volume_1mo,
  volume_1yr,
  spread,
  best_bid,
  best_ask,
  last_trade_price,
  start_date,
  end_date,
  created_at_gamma,
  updated_at_gamma,
  tags,
  raw,
  fetch_status,
  last_error,
  last_error_at,
  fetched_at
) VALUES (
  @market_id,
  @condition_id,
  @slug,
  @question,
  @description,
  @resolution_source,
  @question_id,
  @sports_market_type,
  @group_item_title,
  @image,
  @icon,
  @outcomes,
  @outcome_prices,
  @clob_token_ids,
  @active,
  @closed,
  @archived,
  @restricted,
  @enable_order_book,
  @accepting_orders,
  @volume,
  @volume_num,
  @liquidity_num,
  @volume_24hr,
  @volume_1wk,
  @volume_1mo,
  @volume_1yr,
  @spread,
  @best_bid,
  @best_ask,
  @last_trade_price,
  @start_date,
  @end_date,
  @created_at_gamma,
  @updated_at_gamma,
  @tags,
  @raw,
  'ok',
  '',
  NULL,
  @fetched_at
)
ON CONFLICT (market_id) DO UPDATE
SET condition_id = EXCLUDED.condition_id,
  slug = EXCLUDED.slug,
  question = EXCLUDED.question,
  description = EXCLUDED.description,
  resolution_source = EXCLUDED.resolution_source,
  question_id = EXCLUDED.question_id,
  sports_market_type = EXCLUDED.sports_market_type,
  group_item_title = EXCLUDED.group_item_title,
  image = EXCLUDED.image,
  icon = EXCLUDED.icon,
  outcomes = EXCLUDED.outcomes,
  outcome_prices = EXCLUDED.outcome_prices,
  clob_token_ids = EXCLUDED.clob_token_ids,
  active = EXCLUDED.active,
  closed = EXCLUDED.closed,
  archived = EXCLUDED.archived,
  restricted = EXCLUDED.restricted,
  enable_order_book = EXCLUDED.enable_order_book,
  accepting_orders = EXCLUDED.accepting_orders,
  volume = EXCLUDED.volume,
  volume_num = EXCLUDED.volume_num,
  liquidity_num = EXCLUDED.liquidity_num,
  volume_24hr = EXCLUDED.volume_24hr,
  volume_1wk = EXCLUDED.volume_1wk,
  volume_1mo = EXCLUDED.volume_1mo,
  volume_1yr = EXCLUDED.volume_1yr,
  spread = EXCLUDED.spread,
  best_bid = EXCLUDED.best_bid,
  best_ask = EXCLUDED.best_ask,
  last_trade_price = EXCLUDED.last_trade_price,
  start_date = EXCLUDED.start_date,
  end_date = EXCLUDED.end_date,
  created_at_gamma = EXCLUDED.created_at_gamma,
  updated_at_gamma = EXCLUDED.updated_at_gamma,
  tags = EXCLUDED.tags,
  raw = EXCLUDED.raw,
  fetch_status = EXCLUDED.fetch_status,
  last_error = EXCLUDED.last_error,
  last_error_at = EXCLUDED.last_error_at,
  fetched_at = EXCLUDED.fetched_at,
  updated_at = now();

-- name: UpsertManagedOOMarketNotFound :exec
INSERT INTO polymarket_managed_oo_market (
  market_id,
  fetch_status,
  last_error,
  last_error_at,
  fetched_at
) VALUES (
  @market_id,
  'not_found',
  @last_error,
  @last_error_at,
  @fetched_at
)
ON CONFLICT (market_id) DO UPDATE
SET fetch_status = EXCLUDED.fetch_status,
  last_error = EXCLUDED.last_error,
  last_error_at = EXCLUDED.last_error_at,
  fetched_at = EXCLUDED.fetched_at,
  updated_at = now();
