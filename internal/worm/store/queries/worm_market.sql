-- name: UpsertWormMarket :one
INSERT INTO worm_market (
  condition_id,
  title,
  description,
  logo,
  last_trade_price,
  state,
  category,
  sort_option,
  created,
  event_title,
  event_condition_id,
  event_logo,
  margin_enabled,
  ignored,
  live_state,
  live_checked_at,
  live_price_change,
  raw,
  fetched_at,
  last_seen_at
) VALUES (
  @condition_id,
  @title,
  @description,
  @logo,
  @last_trade_price,
  @state,
  @category,
  @sort_option,
  @created,
  @event_title,
  @event_condition_id,
  @event_logo,
  @margin_enabled,
  @ignored,
  @live_state,
  @live_checked_at,
  @live_price_change,
  @raw,
  @fetched_at,
  @last_seen_at
)
ON CONFLICT (condition_id) DO UPDATE
SET title = EXCLUDED.title,
  description = EXCLUDED.description,
  logo = EXCLUDED.logo,
  last_trade_price = EXCLUDED.last_trade_price,
  state = EXCLUDED.state,
  category = EXCLUDED.category,
  sort_option = EXCLUDED.sort_option,
  created = EXCLUDED.created,
  event_title = EXCLUDED.event_title,
  event_condition_id = EXCLUDED.event_condition_id,
  event_logo = EXCLUDED.event_logo,
  margin_enabled = EXCLUDED.margin_enabled,
  ignored = worm_market.ignored,
  live_state = worm_market.live_state,
  live_checked_at = worm_market.live_checked_at,
  live_price_change = worm_market.live_price_change,
  raw = EXCLUDED.raw,
  fetched_at = EXCLUDED.fetched_at,
  last_seen_at = EXCLUDED.last_seen_at,
  updated_at = now()
RETURNING *;

-- name: BatchUpsertWormMarkets :exec
INSERT INTO worm_market (
  condition_id,
  title,
  description,
  logo,
  last_trade_price,
  state,
  category,
  sort_option,
  created,
  event_title,
  event_condition_id,
  event_logo,
  margin_enabled,
  ignored,
  live_state,
  live_checked_at,
  live_price_change,
  raw,
  fetched_at,
  last_seen_at
)
SELECT
  unnest(sqlc.arg('condition_ids')::text[]),
  unnest(sqlc.arg('titles')::text[]),
  unnest(sqlc.arg('descriptions')::text[]),
  unnest(sqlc.arg('logos')::text[]),
  unnest(sqlc.arg('last_trade_prices')::text[]),
  unnest(sqlc.arg('states')::text[]),
  unnest(sqlc.arg('categories')::text[]),
  unnest(sqlc.arg('sort_options')::text[]),
  unnest(sqlc.arg('created_values')::bigint[]),
  unnest(sqlc.arg('event_titles')::text[]),
  unnest(sqlc.arg('event_condition_ids')::text[]),
  unnest(sqlc.arg('event_logos')::text[]),
  unnest(sqlc.arg('margin_enabled_values')::boolean[]),
  unnest(sqlc.arg('ignored_values')::boolean[]),
  unnest(sqlc.arg('live_states')::text[]),
  unnest(sqlc.arg('live_checked_at_values')::timestamptz[]),
  unnest(sqlc.arg('live_price_changes')::text[]),
  unnest(sqlc.arg('raw_values')::jsonb[]),
  unnest(sqlc.arg('fetched_at_values')::timestamptz[]),
  unnest(sqlc.arg('last_seen_at_values')::timestamptz[])
ON CONFLICT (condition_id) DO UPDATE
SET title = EXCLUDED.title,
  description = EXCLUDED.description,
  logo = EXCLUDED.logo,
  last_trade_price = EXCLUDED.last_trade_price,
  state = EXCLUDED.state,
  category = EXCLUDED.category,
  sort_option = EXCLUDED.sort_option,
  created = EXCLUDED.created,
  event_title = EXCLUDED.event_title,
  event_condition_id = EXCLUDED.event_condition_id,
  event_logo = EXCLUDED.event_logo,
  margin_enabled = EXCLUDED.margin_enabled,
  ignored = worm_market.ignored,
  live_state = worm_market.live_state,
  live_checked_at = worm_market.live_checked_at,
  live_price_change = worm_market.live_price_change,
  raw = EXCLUDED.raw,
  fetched_at = EXCLUDED.fetched_at,
  last_seen_at = EXCLUDED.last_seen_at,
  updated_at = now();

-- name: GetWormMarket :one
SELECT *
FROM worm_market
WHERE condition_id = @condition_id;

-- name: CountWormMarkets :one
SELECT COUNT(*)::bigint
FROM worm_market
WHERE (sqlc.narg('condition_id')::text IS NULL OR condition_id = sqlc.narg('condition_id')::text)
  AND (sqlc.narg('event_condition_id')::text IS NULL OR event_condition_id = sqlc.narg('event_condition_id')::text)
  AND (sqlc.narg('ignored')::boolean IS NULL OR ignored = sqlc.narg('ignored')::boolean);

-- name: ListWormMarkets :many
SELECT *
FROM worm_market
ORDER BY ignored ASC, (live_state = 'live') DESC, created DESC, condition_id;

-- name: ListWormMarketsPage :many
SELECT *
FROM worm_market
WHERE (sqlc.narg('condition_id')::text IS NULL OR condition_id = sqlc.narg('condition_id')::text)
  AND (sqlc.narg('event_condition_id')::text IS NULL OR event_condition_id = sqlc.narg('event_condition_id')::text)
  AND (sqlc.narg('ignored')::boolean IS NULL OR ignored = sqlc.narg('ignored')::boolean)
ORDER BY (live_state = 'live') DESC, created DESC, condition_id
LIMIT sqlc.arg('limit') OFFSET sqlc.arg('offset');

-- name: ListWormMarketsPendingLiveCheck :many
SELECT *
FROM worm_market
WHERE ignored = false
  AND live_state <> 'live'
  AND state = 'open'
ORDER BY live_checked_at ASC NULLS FIRST, created DESC, condition_id;

-- name: BatchUpdateWormMarketsIgnored :execrows
UPDATE worm_market
SET ignored = @ignored,
  updated_at = now()
WHERE condition_id = ANY(sqlc.arg('condition_ids')::text[]);

-- name: UpdateWormMarketLiveState :one
UPDATE worm_market
SET live_state = CASE WHEN live_state = 'live' THEN live_state ELSE @live_state END,
  live_checked_at = CASE WHEN live_state = 'live' THEN live_checked_at ELSE @live_checked_at END,
  live_price_change = CASE WHEN live_state = 'live' THEN live_price_change ELSE @live_price_change END,
  updated_at = now()
WHERE condition_id = @condition_id
RETURNING *;

-- name: UpdateWormMarket :one
UPDATE worm_market
SET title = @title,
  description = @description,
  logo = @logo,
  last_trade_price = @last_trade_price,
  state = @state,
  category = @category,
  sort_option = @sort_option,
  created = @created,
  event_title = @event_title,
  event_condition_id = @event_condition_id,
  event_logo = @event_logo,
  margin_enabled = @margin_enabled,
  ignored = @ignored,
  live_state = @live_state,
  live_checked_at = @live_checked_at,
  live_price_change = @live_price_change,
  raw = @raw,
  fetched_at = @fetched_at,
  last_seen_at = @last_seen_at,
  updated_at = now()
WHERE condition_id = @condition_id
RETURNING *;

-- name: DeleteWormMarket :execrows
DELETE FROM worm_market
WHERE condition_id = @condition_id;

-- name: DeleteWormMarketsNotSeenSince :execrows
DELETE FROM worm_market
WHERE last_seen_at < @last_seen_at;
