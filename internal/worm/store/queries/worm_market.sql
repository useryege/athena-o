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
  live_state = worm_market.live_state,
  live_checked_at = worm_market.live_checked_at,
  live_price_change = worm_market.live_price_change,
  raw = EXCLUDED.raw,
  fetched_at = EXCLUDED.fetched_at,
  last_seen_at = EXCLUDED.last_seen_at,
  updated_at = now()
RETURNING *;

-- name: BatchUpsertWormMarkets :many
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
  live_state = worm_market.live_state,
  live_checked_at = worm_market.live_checked_at,
  live_price_change = worm_market.live_price_change,
  raw = EXCLUDED.raw,
  fetched_at = EXCLUDED.fetched_at,
  last_seen_at = EXCLUDED.last_seen_at,
  updated_at = now()
RETURNING condition_id, (xmax = 0)::boolean AS inserted;

-- name: CountWormMarkets :one
SELECT COUNT(*)::bigint
FROM worm_market;

-- name: ListWormEventConditionIDs :many
SELECT DISTINCT event_condition_id
FROM worm_market
WHERE event_condition_id <> ''
ORDER BY event_condition_id;

-- name: GetWormMarket :one
SELECT *
FROM worm_market
WHERE condition_id = @condition_id;

-- name: ListWormMarkets :many
SELECT *
FROM worm_market
ORDER BY (live_state = 'live') DESC, created DESC, condition_id;

-- name: ListWormLiveMarketsForPriceAlerts :many
SELECT *
FROM worm_market
WHERE live_state = 'live'
  AND state = 'open'
ORDER BY condition_id;

-- name: UpdateWormMarketPriceAlertBand :execrows
UPDATE worm_market
SET price_alert_band = @price_alert_band,
  updated_at = now()
WHERE condition_id = @condition_id
  AND price_alert_band = @expected_price_alert_band;

-- name: ListWormMarketConditionIDsMissingRules :many
SELECT condition_id
FROM worm_market
WHERE rules IS NULL
ORDER BY condition_id;

-- name: SetWormMarketRulesIfMissing :execrows
UPDATE worm_market
SET rules = @rules,
  updated_at = now()
WHERE condition_id = @condition_id
  AND rules IS NULL;

-- name: CountWormEvents :one
SELECT COUNT(DISTINCT event_condition_id)::bigint
FROM worm_market
WHERE event_condition_id <> '';

-- name: ListWormEventsPage :many
SELECT
  event_condition_id,
  COALESCE(
    (ARRAY_AGG(event_title ORDER BY created DESC, condition_id) FILTER (WHERE event_title <> ''))[1],
    ''
  )::text AS event_title,
  COALESCE(
    (ARRAY_AGG(event_logo ORDER BY created DESC, condition_id) FILTER (WHERE event_logo <> ''))[1],
    ''
  )::text AS event_logo,
  BOOL_OR(live_state = 'live')::boolean AS live,
  COUNT(*)::bigint AS market_count,
  MAX(created)::bigint AS newest_created,
  MAX(fetched_at)::timestamptz AS fetched_at
FROM worm_market
WHERE event_condition_id <> ''
GROUP BY event_condition_id
ORDER BY BOOL_OR(live_state = 'live') DESC, MAX(created) DESC, event_condition_id
LIMIT sqlc.arg('limit') OFFSET sqlc.arg('offset');

-- name: ListWormMarketsByEventConditionIDs :many
SELECT *
FROM worm_market
WHERE event_condition_id = ANY(sqlc.arg('event_condition_ids')::text[])
ORDER BY event_condition_id, (live_state = 'live') DESC, created DESC, condition_id;

-- name: BatchInsertWormMarketPriceHistory :exec
INSERT INTO worm_market_price_history (
  condition_id,
  price,
  sampled_at
)
SELECT
  unnest(sqlc.arg('condition_ids')::text[]),
  unnest(sqlc.arg('prices')::text[])::numeric,
  unnest(sqlc.arg('sampled_at_values')::timestamptz[])
ON CONFLICT (condition_id, sampled_at) DO UPDATE
SET price = EXCLUDED.price;

-- name: DeleteWormMarketPriceHistoryBefore :execrows
DELETE FROM worm_market_price_history
WHERE sampled_at < @sampled_at;

-- name: ListWormMarketLivePriceChanges :many
SELECT
  market.condition_id,
  market.title,
  market.event_condition_id,
  market.event_title,
  COUNT(history.price)::bigint AS sample_count,
  COALESCE((MAX(history.price) - MIN(history.price))::text, '')::text AS price_change,
  COALESCE(MAX(history.price) - MIN(history.price) > 0.05, false)::boolean AS is_live,
  EXISTS (
    SELECT 1
    FROM worm_market AS live_market
    WHERE live_market.event_condition_id = market.event_condition_id
      AND live_market.live_state = 'live'
  )::boolean AS event_has_live
FROM worm_market AS market
LEFT JOIN worm_market_price_history AS history
  ON history.condition_id = market.condition_id
  AND history.sampled_at >= @sampled_at
WHERE market.live_state <> 'live'
  AND market.state = 'open'
GROUP BY market.condition_id, market.title, market.event_condition_id, market.event_title
ORDER BY market.condition_id;

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
