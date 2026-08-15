-- name: GetSportsHistorySyncState :one
SELECT last_success_at
FROM sports_history_sync_state
WHERE sync_name = @sync_name;

-- name: UpsertSportsHistorySyncState :exec
INSERT INTO sports_history_sync_state (sync_name, last_success_at)
VALUES (@sync_name, @last_success_at)
ON CONFLICT (sync_name) DO UPDATE
SET last_success_at = EXCLUDED.last_success_at,
  updated_at = now();
