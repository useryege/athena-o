-- name: AddProjectEventLog :exec
INSERT INTO project_event_log (
  contract,
  event_type,
  occurred_at,
  message,
  payload,
  idempotency_key
) VALUES ($1, $2, $3, $4, $5::jsonb, $6)
ON CONFLICT (contract, idempotency_key) DO NOTHING;

-- name: ListProjectEventLogsByContract :many
SELECT
  id,
  contract,
  event_type,
  occurred_at,
  message,
  payload,
  idempotency_key,
  created_at
FROM project_event_log
WHERE contract = $1
ORDER BY occurred_at DESC, id DESC;
