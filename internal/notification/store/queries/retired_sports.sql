-- name: CountRetiredSports :one
SELECT count(*) FILTER (WHERE status = 'pending') AS pending,
       count(*) FILTER (WHERE status = 'sending') AS sending
FROM system_notification_deliveries
WHERE source IN (
  'polymarket.sports-live-score',
  'polymarket.sports-live-price-alert-85-15',
  'polymarket.sports-live-price-alert-90-10',
  'polymarket.sports-live-price-alert-95-5',
  'polymarket.sports-live-price-alert-97-3',
  'polymarket.sports-live-price-alert-99-1'
);

-- name: CancelRetiredSportsPending :execrows
WITH chosen AS (
  SELECT id FROM system_notification_deliveries
  WHERE status = 'pending' AND source IN (
    'polymarket.sports-live-score',
    'polymarket.sports-live-price-alert-85-15',
    'polymarket.sports-live-price-alert-90-10',
    'polymarket.sports-live-price-alert-95-5',
    'polymarket.sports-live-price-alert-97-3',
    'polymarket.sports-live-price-alert-99-1'
  )
  ORDER BY id LIMIT $1 FOR UPDATE SKIP LOCKED
)
UPDATE system_notification_deliveries d
SET status = 'cancelled', error_message = 'source retired: sports removal',
    locked_at = NULL, locked_by = NULL
FROM chosen c WHERE d.id = c.id AND d.status = 'pending';
