-- name: ListAccountEnabledOverrides :many
SELECT account_name, enabled
FROM account_enabled_override
ORDER BY account_name;

-- name: UpsertAccountEnabledOverride :exec
INSERT INTO account_enabled_override (account_name, enabled)
VALUES ($1, $2)
ON CONFLICT (account_name) DO UPDATE
SET enabled = EXCLUDED.enabled,
    updated_at = NOW();
