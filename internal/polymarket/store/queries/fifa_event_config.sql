-- name: GetPolymarketFIFAEventConfig :one
SELECT worm_event_id, event_ref, updated_at
FROM polymarket_fifa_event_config
WHERE singleton = true;

-- name: UpdatePolymarketFIFAEventConfig :one
UPDATE polymarket_fifa_event_config
SET worm_event_id = @worm_event_id,
  event_ref = @event_ref,
  updated_at = now()
WHERE singleton = true
RETURNING worm_event_id, event_ref, updated_at;
