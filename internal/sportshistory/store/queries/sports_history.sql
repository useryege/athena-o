-- name: BatchUpsertSportsHistoryEvents :exec
INSERT INTO sports_history_event (
  event_key,
  event_id,
  league,
  slug,
  title,
  image,
  icon,
  score,
  period,
  elapsed,
  game_status,
  start_time,
  finished_at,
  updated_at_gamma,
  liquidity,
  volume,
  teams,
  raw,
  fetched_at,
  last_seen_at
)
SELECT
  unnest(sqlc.arg('event_keys')::text[]),
  unnest(sqlc.arg('event_ids')::text[]),
  unnest(sqlc.arg('leagues')::text[]),
  unnest(sqlc.arg('slugs')::text[]),
  unnest(sqlc.arg('titles')::text[]),
  unnest(sqlc.arg('images')::text[]),
  unnest(sqlc.arg('icons')::text[]),
  unnest(sqlc.arg('scores')::text[]),
  unnest(sqlc.arg('periods')::text[]),
  unnest(sqlc.arg('elapsed_values')::text[]),
  unnest(sqlc.arg('game_statuses')::text[]),
  unnest(sqlc.arg('start_time_values')::timestamptz[]),
  unnest(sqlc.arg('finished_at_values')::timestamptz[]),
  unnest(sqlc.arg('updated_at_gamma_values')::timestamptz[]),
  unnest(sqlc.arg('liquidity_values')::double precision[]),
  unnest(sqlc.arg('volume_values')::double precision[]),
  unnest(sqlc.arg('teams_values')::jsonb[]),
  unnest(sqlc.arg('raw_values')::jsonb[]),
  unnest(sqlc.arg('fetched_at_values')::timestamptz[]),
  unnest(sqlc.arg('last_seen_at_values')::timestamptz[])
ON CONFLICT (event_key) DO UPDATE
SET event_id = EXCLUDED.event_id,
  league = EXCLUDED.league,
  slug = EXCLUDED.slug,
  title = EXCLUDED.title,
  image = EXCLUDED.image,
  icon = EXCLUDED.icon,
  score = EXCLUDED.score,
  period = EXCLUDED.period,
  elapsed = EXCLUDED.elapsed,
  game_status = EXCLUDED.game_status,
  start_time = EXCLUDED.start_time,
  finished_at = EXCLUDED.finished_at,
  updated_at_gamma = EXCLUDED.updated_at_gamma,
  liquidity = EXCLUDED.liquidity,
  volume = EXCLUDED.volume,
  teams = EXCLUDED.teams,
  raw = EXCLUDED.raw,
  fetched_at = EXCLUDED.fetched_at,
  last_seen_at = EXCLUDED.last_seen_at,
  updated_at = now();

-- name: BatchUpsertSportsHistoryMarkets :exec
INSERT INTO sports_history_market (
  market_key,
  event_key,
  condition_id,
  slug,
  question,
  sports_market_type,
  outcomes,
  outcome_prices,
  clob_token_ids,
  best_bid,
  best_ask,
  last_trade_price,
  spread,
  liquidity_num,
  volume_num,
  updated_at_gamma,
  raw,
  fetched_at,
  last_seen_at
)
SELECT
  unnest(sqlc.arg('market_keys')::text[]),
  unnest(sqlc.arg('event_keys')::text[]),
  unnest(sqlc.arg('condition_ids')::text[]),
  unnest(sqlc.arg('slugs')::text[]),
  unnest(sqlc.arg('questions')::text[]),
  unnest(sqlc.arg('sports_market_types')::text[]),
  unnest(sqlc.arg('outcomes_values')::text[]),
  unnest(sqlc.arg('outcome_prices_values')::text[]),
  unnest(sqlc.arg('clob_token_ids_values')::text[]),
  unnest(sqlc.arg('best_bid_values')::double precision[]),
  unnest(sqlc.arg('best_ask_values')::double precision[]),
  unnest(sqlc.arg('last_trade_price_values')::double precision[]),
  unnest(sqlc.arg('spread_values')::double precision[]),
  unnest(sqlc.arg('liquidity_num_values')::double precision[]),
  unnest(sqlc.arg('volume_num_values')::double precision[]),
  unnest(sqlc.arg('updated_at_gamma_values')::timestamptz[]),
  unnest(sqlc.arg('raw_values')::jsonb[]),
  unnest(sqlc.arg('fetched_at_values')::timestamptz[]),
  unnest(sqlc.arg('last_seen_at_values')::timestamptz[])
ON CONFLICT (market_key) DO UPDATE
SET event_key = EXCLUDED.event_key,
  condition_id = EXCLUDED.condition_id,
  slug = EXCLUDED.slug,
  question = EXCLUDED.question,
  sports_market_type = EXCLUDED.sports_market_type,
  outcomes = EXCLUDED.outcomes,
  outcome_prices = EXCLUDED.outcome_prices,
  clob_token_ids = EXCLUDED.clob_token_ids,
  best_bid = EXCLUDED.best_bid,
  best_ask = EXCLUDED.best_ask,
  last_trade_price = EXCLUDED.last_trade_price,
  spread = EXCLUDED.spread,
  liquidity_num = EXCLUDED.liquidity_num,
  volume_num = EXCLUDED.volume_num,
  updated_at_gamma = EXCLUDED.updated_at_gamma,
  raw = EXCLUDED.raw,
  fetched_at = EXCLUDED.fetched_at,
  last_seen_at = EXCLUDED.last_seen_at,
  updated_at = now();

-- name: DeleteSportsHistoryMarketsNotSeenSince :execrows
DELETE FROM sports_history_market
WHERE last_seen_at < @last_seen_at;

-- name: DeleteSportsHistoryEventsNotSeenSince :execrows
DELETE FROM sports_history_event
WHERE last_seen_at < @last_seen_at;

-- name: ListSportsHistoryEvents :many
SELECT
  event.event_key,
  event.event_id,
  event.league,
  event.slug,
  event.title,
  COALESCE(NULLIF(event.image, ''), event.icon)::text AS image,
  event.score,
  event.period,
  event.elapsed,
  event.game_status,
  event.start_time,
  event.finished_at,
  event.updated_at_gamma,
  event.liquidity,
  event.volume,
  COUNT(market.market_key)::bigint AS market_count,
  event.teams,
  event.fetched_at,
  event.last_seen_at
FROM sports_history_event AS event
JOIN sports_history_market AS market ON market.event_key = event.event_key
  AND lower(market.sports_market_type) = 'moneyline'
GROUP BY event.event_key
ORDER BY CASE event.league WHEN 'ATP' THEN 0 ELSE 1 END, event.start_time DESC, event.event_key
LIMIT sqlc.arg('limit');

-- name: ListSportsHistoryMarketsByEventKeys :many
SELECT
  market.event_key,
  market.market_key,
  market.condition_id,
  market.slug,
  market.sports_market_type,
  market.question,
  market.outcomes,
  market.outcome_prices,
  market.best_bid,
  market.best_ask,
  market.last_trade_price,
  market.spread,
  market.liquidity_num,
  market.volume_num,
  market.updated_at_gamma
FROM sports_history_market AS market
WHERE market.event_key = ANY(sqlc.arg('event_keys')::text[])
  AND lower(market.sports_market_type) = 'moneyline'
ORDER BY market.event_key, market.liquidity_num DESC, market.volume_num DESC, market.market_key;

-- name: ListSportsHistoryMoneylineMarketsForPriceHistory :many
SELECT
  market.event_key,
  market.market_key,
  market.condition_id,
  market.outcomes,
  market.clob_token_ids,
  event.start_time,
  event.finished_at
FROM sports_history_market AS market
JOIN sports_history_event AS event ON event.event_key = market.event_key
WHERE lower(market.sports_market_type) = 'moneyline'
  AND btrim(market.clob_token_ids) <> ''
ORDER BY event.start_time DESC, market.market_key;

-- name: BatchUpsertSportsHistoryPricePoints :exec
INSERT INTO sports_history_price_point (
  token_id,
  market_key,
  event_key,
  condition_id,
  outcome,
  price_ts,
  price,
  fetched_at
)
SELECT
  unnest(sqlc.arg('token_ids')::text[]),
  unnest(sqlc.arg('market_keys')::text[]),
  unnest(sqlc.arg('event_keys')::text[]),
  unnest(sqlc.arg('condition_ids')::text[]),
  unnest(sqlc.arg('outcomes')::text[]),
  unnest(sqlc.arg('price_ts_values')::timestamptz[]),
  unnest(sqlc.arg('price_values')::double precision[]),
  unnest(sqlc.arg('fetched_at_values')::timestamptz[])
ON CONFLICT (token_id, price_ts) DO UPDATE
SET market_key = EXCLUDED.market_key,
  event_key = EXCLUDED.event_key,
  condition_id = EXCLUDED.condition_id,
  outcome = EXCLUDED.outcome,
  price = EXCLUDED.price,
  fetched_at = EXCLUDED.fetched_at,
  updated_at = now();

-- name: ListSportsHistoryPriceHistoryByMarketKeys :many
WITH ranked_points AS (
  SELECT
    point.market_key,
    point.token_id,
    point.outcome,
    point.price_ts,
    point.price,
    row_number() OVER (
      PARTITION BY point.market_key, point.token_id
      ORDER BY point.price_ts DESC
    ) AS row_num
  FROM sports_history_price_point AS point
  WHERE point.market_key = ANY(sqlc.arg('market_keys')::text[])
)
SELECT
  market_key,
  token_id,
  outcome,
  price_ts,
  price
FROM ranked_points
WHERE row_num <= sqlc.arg('limit_per_token')::int
ORDER BY market_key, token_id, price_ts;
