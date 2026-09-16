-- name: CountWormMarketsNotifications :one
SELECT count(*) FILTER (
         WHERE status = 'cancelled' AND error_message = 'WORM_MARKETS_RETIRED'
       ) AS cancelled,
       count(*) FILTER (WHERE status = 'pending') AS pending,
       count(*) FILTER (WHERE status = 'sending') AS sending
FROM system_notification_deliveries
WHERE source IN (
  'worm-markets.new-event',
  'worm-markets.live-event',
  'worm-markets.price-alert-80-20',
  'worm-markets.price-alert-90-10',
  'worm-markets.price-alert-95-5'
);

-- name: CancelWormMarketsNotificationsPending :execrows
WITH candidates AS (
  SELECT id FROM system_notification_deliveries
  WHERE status = 'pending' AND source IN (
    'worm-markets.new-event',
    'worm-markets.live-event',
    'worm-markets.price-alert-80-20',
    'worm-markets.price-alert-90-10',
    'worm-markets.price-alert-95-5'
  )
  ORDER BY id LIMIT $1 FOR UPDATE SKIP LOCKED
)
UPDATE system_notification_deliveries d
SET status = 'cancelled', error_message = 'WORM_MARKETS_RETIRED',
    locked_at = NULL, locked_by = NULL
FROM candidates c WHERE d.id = c.id AND d.status = 'pending';
