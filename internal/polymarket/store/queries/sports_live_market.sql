-- name: BatchUpsertSportsLiveMarkets :exec
INSERT INTO polymarket_sports_live_market (
  condition_id,
  market_slug,
  event_slug,
  title,
  image,
  score,
  period,
  elapsed,
  gamma_updated_at,
  liquidity_num,
  volume_num,
  fetched_at,
  last_seen_at
)
SELECT
  unnest(sqlc.arg('condition_ids')::text[]),
  unnest(sqlc.arg('market_slugs')::text[]),
  unnest(sqlc.arg('event_slugs')::text[]),
  unnest(sqlc.arg('titles')::text[]),
  unnest(sqlc.arg('images')::text[]),
  unnest(sqlc.arg('scores')::text[]),
  unnest(sqlc.arg('periods')::text[]),
  unnest(sqlc.arg('elapsed_values')::text[]),
  unnest(sqlc.arg('gamma_updated_at_values')::timestamptz[]),
  unnest(sqlc.arg('liquidity_num_values')::double precision[]),
  unnest(sqlc.arg('volume_num_values')::double precision[]),
  unnest(sqlc.arg('fetched_at_values')::timestamptz[]),
  unnest(sqlc.arg('last_seen_at_values')::timestamptz[])
ON CONFLICT (condition_id) DO UPDATE
SET market_slug = EXCLUDED.market_slug,
  event_slug = EXCLUDED.event_slug,
  title = EXCLUDED.title,
  image = EXCLUDED.image,
  score = EXCLUDED.score,
  period = EXCLUDED.period,
  elapsed = EXCLUDED.elapsed,
  gamma_updated_at = EXCLUDED.gamma_updated_at,
  liquidity_num = EXCLUDED.liquidity_num,
  volume_num = EXCLUDED.volume_num,
  fetched_at = EXCLUDED.fetched_at,
  last_seen_at = EXCLUDED.last_seen_at,
  updated_at = now();

-- name: DeleteSportsLiveMarketsNotSeenSince :execrows
DELETE FROM polymarket_sports_live_market
WHERE last_seen_at < @last_seen_at;

-- name: ListSportsLiveMarkets :many
SELECT *
FROM polymarket_sports_live_market
ORDER BY gamma_updated_at DESC NULLS LAST, liquidity_num DESC, condition_id
LIMIT sqlc.arg('limit');

-- name: GetPolymarketSyncState :one
SELECT last_success_at
FROM polymarket_sync_state
WHERE sync_name = @sync_name;

-- name: UpsertPolymarketSyncState :exec
INSERT INTO polymarket_sync_state (sync_name, last_success_at)
VALUES (@sync_name, @last_success_at)
ON CONFLICT (sync_name) DO UPDATE
SET last_success_at = EXCLUDED.last_success_at,
  updated_at = now();
