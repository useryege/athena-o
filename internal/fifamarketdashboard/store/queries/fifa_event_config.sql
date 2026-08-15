-- name: GetFIFAEventConfig :one
SELECT worm_event_id, event_ref, updated_at
FROM fifa_market_dashboard_event_config
WHERE singleton = true;

-- name: UpdateFIFAEventConfig :one
INSERT INTO fifa_market_dashboard_event_config (singleton, worm_event_id, event_ref)
VALUES (true, @worm_event_id, @event_ref)
ON CONFLICT (singleton) DO UPDATE
SET worm_event_id = EXCLUDED.worm_event_id,
  event_ref = EXCLUDED.event_ref,
  updated_at = now()
RETURNING worm_event_id, event_ref, updated_at;
