-- name: GetManagedOOChainLogCursor :one
SELECT
  sync_name,
  contract_address,
  topic,
  last_block_number,
  last_polled_at,
  created_at,
  updated_at
FROM managed_oo_chain_log_cursor
WHERE sync_name = @sync_name;

-- name: UpsertManagedOOChainLogCursor :one
INSERT INTO managed_oo_chain_log_cursor (
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
INSERT INTO managed_oo_propose_price_log (
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

-- name: BatchUpsertManagedOODisputePriceLogs :exec
INSERT INTO managed_oo_dispute_price_log (
  tx_hash,
  log_index,
  block_number,
  block_hash,
  tx_index,
  contract_address,
  topic,
  requester,
  proposer,
  disputer,
  identifier,
  request_timestamp,
  ancillary_data_hex,
  ancillary_data_text,
  market_id,
  proposed_price,
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
  unnest(sqlc.arg('disputers')::text[]),
  unnest(sqlc.arg('identifiers')::text[]),
  unnest(sqlc.arg('request_timestamps')::bigint[]),
  unnest(sqlc.arg('ancillary_data_hex_values')::text[]),
  unnest(sqlc.arg('ancillary_data_text_values')::text[]),
  unnest(sqlc.arg('market_ids')::text[]),
  unnest(sqlc.arg('proposed_prices')::text[]),
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
  disputer = EXCLUDED.disputer,
  identifier = EXCLUDED.identifier,
  request_timestamp = EXCLUDED.request_timestamp,
  ancillary_data_hex = EXCLUDED.ancillary_data_hex,
  ancillary_data_text = EXCLUDED.ancillary_data_text,
  market_id = EXCLUDED.market_id,
  proposed_price = EXCLUDED.proposed_price,
  raw_topics = EXCLUDED.raw_topics,
  raw_data = EXCLUDED.raw_data,
  fetched_at = EXCLUDED.fetched_at,
  updated_at = now();

-- name: CountManagedOOProposePriceLogs :one
SELECT COUNT(*)::bigint
FROM managed_oo_propose_price_log
WHERE @block_number::bigint = 0 OR block_number = @block_number;

-- name: ListManagedOOProposePriceLogs :many
SELECT
  log.tx_hash,
  log.log_index,
  log.block_number,
  log.block_hash,
  log.tx_index,
  log.contract_address,
  log.topic,
  log.requester,
  log.proposer,
  log.identifier,
  log.request_timestamp,
  log.ancillary_data_hex,
  log.ancillary_data_text,
  log.market_id,
  log.proposed_price,
  log.expiration_timestamp,
  log.currency,
  log.raw_topics,
  log.raw_data,
  log.fetched_at,
  COALESCE(market.condition_id, '')::text AS condition_id,
  COALESCE(NULLIF(market.event_slug, ''), NULLIF(market.raw #>> '{events,0,slug}', ''), '')::text AS event_slug,
  COALESCE(market.slug, '')::text AS market_slug,
  COALESCE(market.question, '')::text AS question
FROM managed_oo_propose_price_log AS log
LEFT JOIN managed_oo_market AS market
  ON market.market_id = log.market_id
WHERE @block_number::bigint = 0 OR log.block_number = @block_number
ORDER BY log.block_number DESC, log.log_index DESC
LIMIT @limit_value OFFSET @offset_value;

-- name: CountManagedOODisputePriceLogs :one
SELECT COUNT(*)::bigint
FROM managed_oo_dispute_price_log
WHERE @block_number::bigint = 0 OR block_number = @block_number;

-- name: ListManagedOODisputePriceLogs :many
SELECT
  log.tx_hash,
  log.log_index,
  log.block_number,
  log.block_hash,
  log.tx_index,
  log.contract_address,
  log.topic,
  log.requester,
  log.proposer,
  log.disputer,
  log.identifier,
  log.request_timestamp,
  log.ancillary_data_hex,
  log.ancillary_data_text,
  log.market_id,
  log.proposed_price,
  log.raw_topics,
  log.raw_data,
  log.fetched_at,
  COALESCE(market.condition_id, '')::text AS condition_id,
  COALESCE(NULLIF(market.event_slug, ''), NULLIF(market.raw #>> '{events,0,slug}', ''), '')::text AS event_slug,
  COALESCE(market.slug, '')::text AS market_slug,
  COALESCE(market.question, '')::text AS question
FROM managed_oo_dispute_price_log AS log
LEFT JOIN managed_oo_market AS market
  ON market.market_id = log.market_id
WHERE @block_number::bigint = 0 OR log.block_number = @block_number
ORDER BY log.block_number DESC, log.log_index DESC
LIMIT @limit_value OFFSET @offset_value;

-- name: ListManagedOOMarketIDsNeedingRefresh :many
WITH market_activity AS (
  SELECT
    activity.market_id,
    max(activity.block_number) AS latest_block_number
  FROM (
    SELECT market_id, block_number
    FROM managed_oo_propose_price_log
    UNION ALL
    SELECT market_id, block_number
    FROM managed_oo_dispute_price_log
  ) AS activity
  WHERE activity.market_id <> ''
    AND activity.market_id ~ '^[0-9]+$'
  GROUP BY activity.market_id
), pending_disputed_markets AS (
  SELECT DISTINCT log.market_id
  FROM managed_oo_dispute_price_log AS log
  LEFT JOIN managed_oo_dispute_price_alert_state AS state
    ON state.tx_hash = log.tx_hash
    AND state.log_index = log.log_index
  WHERE state.tx_hash IS NULL
    AND log.market_id <> ''
    AND log.market_id ~ '^[0-9]+$'
)
SELECT market_activity.market_id
FROM market_activity
LEFT JOIN managed_oo_market AS market
  ON market.market_id = market_activity.market_id
LEFT JOIN pending_disputed_markets
  ON pending_disputed_markets.market_id = market_activity.market_id
WHERE market.market_id IS NULL
  OR (
    market.fetched_at <= @retry_before
    AND (
      market.fetch_status <> 'ok'
      OR btrim(market.slug) = ''
    )
  )
ORDER BY
  CASE
    WHEN pending_disputed_markets.market_id IS NOT NULL AND market.market_id IS NULL THEN 0
    WHEN pending_disputed_markets.market_id IS NOT NULL THEN 1
    WHEN market.market_id IS NULL THEN 2
    ELSE 3
  END,
  market_activity.latest_block_number DESC,
  market.fetched_at ASC NULLS FIRST,
  market_activity.market_id
LIMIT @limit_value;

-- name: UpsertManagedOOMarket :exec
INSERT INTO managed_oo_market (
  market_id,
  condition_id,
  slug,
  event_slug,
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
  @event_slug,
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
  event_slug = EXCLUDED.event_slug,
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

-- name: DeleteManagedOOMarketLabelsByMarketID :exec
DELETE FROM managed_oo_market_label
WHERE market_id = @market_id;

-- name: BatchUpsertManagedOOMarketLabels :exec
INSERT INTO managed_oo_market_label (
  market_id,
  label,
  tag_id,
  slug,
  position,
  fetched_at
)
SELECT
  unnest(sqlc.arg('market_ids')::text[]),
  unnest(sqlc.arg('labels')::text[]),
  unnest(sqlc.arg('tag_ids')::text[]),
  unnest(sqlc.arg('slugs')::text[]),
  unnest(sqlc.arg('positions')::bigint[]),
  unnest(sqlc.arg('fetched_at_values')::timestamptz[])
ON CONFLICT (market_id, label) DO UPDATE
SET tag_id = EXCLUDED.tag_id,
  slug = EXCLUDED.slug,
  position = EXCLUDED.position,
  fetched_at = EXCLUDED.fetched_at,
  updated_at = now();

-- name: UpsertManagedOOMarketNotFound :exec
INSERT INTO managed_oo_market (
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

-- name: ListManagedOOProposePriceAlertCandidates :many
WITH matched_labels AS (
  SELECT
    market_id,
    string_agg(label, ', ' ORDER BY position) AS labels
  FROM managed_oo_market_label
  WHERE label IN ('Politics', 'Iran', 'Geopolitics')
  GROUP BY market_id
)
SELECT
  log.tx_hash,
  log.log_index,
  log.block_number,
  log.market_id,
  log.proposer,
  log.proposed_price,
  log.request_timestamp,
  log.expiration_timestamp,
  log.ancillary_data_text,
  market.condition_id,
  COALESCE(NULLIF(market.event_slug, ''), NULLIF(market.raw #>> '{events,0,slug}', ''), '')::text AS event_slug,
  market.slug AS market_slug,
  market.question,
  matched_labels.labels AS matched_labels
FROM managed_oo_propose_price_log AS log
JOIN managed_oo_market AS market
  ON market.market_id = log.market_id
JOIN matched_labels
  ON matched_labels.market_id = log.market_id
LEFT JOIN managed_oo_propose_price_alert_state AS state
  ON state.tx_hash = log.tx_hash
  AND state.log_index = log.log_index
WHERE state.tx_hash IS NULL
  AND btrim(market.slug) <> ''
ORDER BY log.block_number ASC, log.log_index ASC
LIMIT @limit_value;

-- name: UpsertManagedOOProposePriceAlertState :exec
INSERT INTO managed_oo_propose_price_alert_state (
  tx_hash,
  log_index,
  notification_id,
  notified_at
) VALUES (
  @tx_hash,
  @log_index,
  @notification_id,
  @notified_at
)
ON CONFLICT (tx_hash, log_index) DO UPDATE
SET notification_id = EXCLUDED.notification_id,
  notified_at = EXCLUDED.notified_at,
  updated_at = now();

-- name: ListManagedOODisputePriceAlertCandidates :many
WITH matched_labels AS (
  SELECT
    market_id,
    string_agg(label, ', ' ORDER BY position) AS labels
  FROM managed_oo_market_label
  GROUP BY market_id
)
SELECT
  log.tx_hash,
  log.log_index,
  log.block_number,
  log.market_id,
  log.requester,
  log.proposer,
  log.disputer,
  log.proposed_price,
  log.request_timestamp,
  log.ancillary_data_text,
  COALESCE(market.condition_id, '')::text AS condition_id,
  COALESCE(NULLIF(market.event_slug, ''), NULLIF(market.raw #>> '{events,0,slug}', ''), '')::text AS event_slug,
  COALESCE(market.slug, '')::text AS market_slug,
  COALESCE(market.question, '')::text AS question,
  COALESCE(matched_labels.labels, '')::text AS matched_labels
FROM managed_oo_dispute_price_log AS log
JOIN managed_oo_market AS market
  ON market.market_id = log.market_id
LEFT JOIN matched_labels
  ON matched_labels.market_id = log.market_id
LEFT JOIN managed_oo_dispute_price_alert_state AS state
  ON state.tx_hash = log.tx_hash
  AND state.log_index = log.log_index
WHERE state.tx_hash IS NULL
  AND btrim(market.slug) <> ''
ORDER BY log.block_number ASC, log.log_index ASC
LIMIT @limit_value;

-- name: UpsertManagedOODisputePriceAlertState :exec
INSERT INTO managed_oo_dispute_price_alert_state (
  tx_hash,
  log_index,
  notification_id,
  notified_at
) VALUES (
  @tx_hash,
  @log_index,
  @notification_id,
  @notified_at
)
ON CONFLICT (tx_hash, log_index) DO UPDATE
SET notification_id = EXCLUDED.notification_id,
  notified_at = EXCLUDED.notified_at,
  updated_at = now();
