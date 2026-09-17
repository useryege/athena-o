-- name: GetModuleAccessSetting :one
SELECT s.module_key, s.is_open, s.updated_by_account_id, s.updated_at,
       COALESCE(a.username, '')::text AS updated_by_username
FROM athena_module_access_setting s
LEFT JOIN athena_account a ON a.account_id = s.updated_by_account_id
WHERE s.module_key = $1;

-- name: ListModuleAccessSettings :many
SELECT s.module_key, s.is_open, s.updated_by_account_id, s.updated_at,
       COALESCE(a.username, '')::text AS updated_by_username
FROM athena_module_access_setting s
LEFT JOIN athena_account a ON a.account_id = s.updated_by_account_id;

-- name: UpsertModuleAccessSetting :exec
INSERT INTO athena_module_access_setting (module_key, is_open, updated_by_account_id, updated_at)
VALUES ($1, $2, $3, clock_timestamp())
ON CONFLICT (module_key) DO UPDATE
SET is_open = EXCLUDED.is_open,
    updated_by_account_id = EXCLUDED.updated_by_account_id,
    updated_at = clock_timestamp();
