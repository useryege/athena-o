-- name: GetWormPolyFIFAEventConfig :one
SELECT worm_event_id, event_ref, updated_at
FROM worm_poly_fifa_event_config
WHERE singleton = true;

-- name: UpdateWormPolyFIFAEventConfig :one
UPDATE worm_poly_fifa_event_config
SET worm_event_id = @worm_event_id,
  event_ref = @event_ref,
  updated_at = now()
WHERE singleton = true
RETURNING worm_event_id, event_ref, updated_at;
