-- Operations read model.
-- name: UpsertChain :one
INSERT INTO chain (id, name, enabled)
VALUES ($1, $2, $3)
ON CONFLICT (id) DO UPDATE SET
  name = EXCLUDED.name,
  enabled = EXCLUDED.enabled
RETURNING *;

-- name: ListChains :many
SELECT *
FROM chain
ORDER BY id;
