-- name: BatchUpsertUMAResolutionMarkets :exec
INSERT INTO polymarket_uma_resolution_market (
  market_key,
  market_id,
  condition_id,
  slug,
  event_id,
  event_slug,
  question,
  description,
  resolution_source,
  image,
  icon,
  uma_resolution_status,
  uma_resolution_statuses,
  outcomes,
  outcome_prices,
  clob_token_ids,
  active,
  closed,
  archived,
  restricted,
  enable_order_book,
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
  tags,
  raw,
  fetched_at,
  last_seen_at
)
SELECT
  unnest(sqlc.arg('market_keys')::text[]),
  unnest(sqlc.arg('market_ids')::text[]),
  unnest(sqlc.arg('condition_ids')::text[]),
  unnest(sqlc.arg('slugs')::text[]),
  unnest(sqlc.arg('event_ids')::text[]),
  unnest(sqlc.arg('event_slugs')::text[]),
  unnest(sqlc.arg('questions')::text[]),
  unnest(sqlc.arg('descriptions')::text[]),
  unnest(sqlc.arg('resolution_sources')::text[]),
  unnest(sqlc.arg('images')::text[]),
  unnest(sqlc.arg('icons')::text[]),
  unnest(sqlc.arg('uma_resolution_status_values')::text[]),
  unnest(sqlc.arg('uma_resolution_statuses_values')::text[]),
  unnest(sqlc.arg('outcomes_values')::text[]),
  unnest(sqlc.arg('outcome_prices_values')::text[]),
  unnest(sqlc.arg('clob_token_ids_values')::text[]),
  unnest(sqlc.arg('active_values')::boolean[]),
  unnest(sqlc.arg('closed_values')::boolean[]),
  unnest(sqlc.arg('archived_values')::boolean[]),
  unnest(sqlc.arg('restricted_values')::boolean[]),
  unnest(sqlc.arg('enable_order_book_values')::boolean[]),
  unnest(sqlc.arg('volume_values')::text[]),
  unnest(sqlc.arg('volume_num_values')::double precision[]),
  unnest(sqlc.arg('liquidity_num_values')::double precision[]),
  unnest(sqlc.arg('volume_24hr_values')::double precision[]),
  unnest(sqlc.arg('volume_1wk_values')::double precision[]),
  unnest(sqlc.arg('volume_1mo_values')::double precision[]),
  unnest(sqlc.arg('volume_1yr_values')::double precision[]),
  unnest(sqlc.arg('spread_values')::double precision[]),
  unnest(sqlc.arg('best_bid_values')::double precision[]),
  unnest(sqlc.arg('best_ask_values')::double precision[]),
  unnest(sqlc.arg('last_trade_price_values')::double precision[]),
  unnest(sqlc.arg('tags_values')::jsonb[]),
  unnest(sqlc.arg('raw_values')::jsonb[]),
  unnest(sqlc.arg('fetched_at_values')::timestamptz[]),
  unnest(sqlc.arg('last_seen_at_values')::timestamptz[])
ON CONFLICT (market_key) DO UPDATE
SET market_id = EXCLUDED.market_id,
  condition_id = EXCLUDED.condition_id,
  slug = EXCLUDED.slug,
  event_id = EXCLUDED.event_id,
  event_slug = EXCLUDED.event_slug,
  question = EXCLUDED.question,
  description = EXCLUDED.description,
  resolution_source = EXCLUDED.resolution_source,
  image = EXCLUDED.image,
  icon = EXCLUDED.icon,
  uma_resolution_status = EXCLUDED.uma_resolution_status,
  uma_resolution_statuses = EXCLUDED.uma_resolution_statuses,
  outcomes = EXCLUDED.outcomes,
  outcome_prices = EXCLUDED.outcome_prices,
  clob_token_ids = EXCLUDED.clob_token_ids,
  active = EXCLUDED.active,
  closed = EXCLUDED.closed,
  archived = EXCLUDED.archived,
  restricted = EXCLUDED.restricted,
  enable_order_book = EXCLUDED.enable_order_book,
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
  tags = EXCLUDED.tags,
  raw = EXCLUDED.raw,
  fetched_at = EXCLUDED.fetched_at,
  last_seen_at = EXCLUDED.last_seen_at,
  updated_at = now();

-- name: DeleteUMAResolutionMarketsNotSeenSince :execrows
DELETE FROM polymarket_uma_resolution_market
WHERE last_seen_at < @last_seen_at;

-- name: ListDisputedMarkets :many
SELECT
  market_key,
  market_id,
  condition_id,
  slug,
  event_id,
  event_slug,
  question,
  image,
  icon,
  uma_resolution_status,
  uma_resolution_statuses,
  active,
  closed,
  enable_order_book,
  volume_num,
  liquidity_num,
  volume_24hr,
  spread,
  best_bid,
  best_ask,
  last_trade_price,
  fetched_at,
  last_seen_at
FROM polymarket_uma_resolution_market
WHERE uma_resolution_status = 'disputed'
ORDER BY volume_24hr DESC, volume_num DESC, market_key
LIMIT sqlc.arg('limit');

-- name: ListPendingUMAResolutionNotificationCandidates :many
SELECT
  m.market_key,
  m.condition_id,
  m.slug,
  m.event_slug,
  m.question,
  m.uma_resolution_status,
  m.uma_resolution_statuses,
  m.volume_24hr,
  m.liquidity_num,
  m.fetched_at,
  m.last_seen_at
FROM polymarket_uma_resolution_market AS m
LEFT JOIN polymarket_uma_resolution_notification_state AS state
  ON state.market_key = m.market_key
    AND state.uma_resolution_status = m.uma_resolution_status
WHERE state.market_key IS NULL
ORDER BY
  CASE m.uma_resolution_status WHEN 'disputed' THEN 0 ELSE 1 END,
  m.volume_24hr DESC,
  m.market_key;

-- name: UpsertUMAResolutionNotificationBaselines :exec
INSERT INTO polymarket_uma_resolution_notification_state (
  market_key,
  uma_resolution_status,
  condition_id,
  slug,
  event_slug,
  question,
  first_seen_at,
  last_seen_at,
  baseline_at
)
SELECT
  m.market_key,
  m.uma_resolution_status,
  m.condition_id,
  m.slug,
  m.event_slug,
  m.question,
  m.last_seen_at,
  m.last_seen_at,
  sqlc.arg('baseline_at')::timestamptz
FROM polymarket_uma_resolution_market AS m
ON CONFLICT (market_key, uma_resolution_status) DO UPDATE
SET condition_id = EXCLUDED.condition_id,
  slug = EXCLUDED.slug,
  event_slug = EXCLUDED.event_slug,
  question = EXCLUDED.question,
  last_seen_at = EXCLUDED.last_seen_at,
  baseline_at = COALESCE(polymarket_uma_resolution_notification_state.baseline_at, EXCLUDED.baseline_at),
  updated_at = now();

-- name: UpsertUMAResolutionNotificationSent :exec
INSERT INTO polymarket_uma_resolution_notification_state (
  market_key,
  uma_resolution_status,
  condition_id,
  slug,
  event_slug,
  question,
  first_seen_at,
  last_seen_at,
  notified_at,
  notification_id
) VALUES (
  @market_key,
  @uma_resolution_status,
  @condition_id,
  @slug,
  @event_slug,
  @question,
  @first_seen_at,
  @last_seen_at,
  @notified_at,
  @notification_id
)
ON CONFLICT (market_key, uma_resolution_status) DO UPDATE
SET condition_id = EXCLUDED.condition_id,
  slug = EXCLUDED.slug,
  event_slug = EXCLUDED.event_slug,
  question = EXCLUDED.question,
  last_seen_at = EXCLUDED.last_seen_at,
  notified_at = COALESCE(polymarket_uma_resolution_notification_state.notified_at, EXCLUDED.notified_at),
  notification_id = CASE
    WHEN polymarket_uma_resolution_notification_state.notification_id <> 0 THEN polymarket_uma_resolution_notification_state.notification_id
    ELSE EXCLUDED.notification_id
  END,
  updated_at = now();
