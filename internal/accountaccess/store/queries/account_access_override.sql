-- name: ListAccountAccessOverrides :many
SELECT account_name, login_enabled, data_access, revision
FROM account_access_override
ORDER BY account_name;

-- name: UpdateAccountAccessOverride :one
WITH updated AS (
  UPDATE account_access_override
  SET login_enabled = sqlc.arg(login_enabled)::boolean,
      data_access = sqlc.arg(data_access)::text,
      revision = account_access_override.revision + 1,
      updated_at = NOW()
  WHERE account_name = sqlc.arg(account_name)::text
    AND sqlc.arg(expected_revision)::bigint > 0
    AND revision = sqlc.arg(expected_revision)::bigint
  RETURNING login_enabled, data_access, revision
), inserted AS (
  INSERT INTO account_access_override (
    account_name,
    login_enabled,
    data_access,
    revision
  )
  SELECT
    sqlc.arg(account_name)::text,
    sqlc.arg(login_enabled)::boolean,
    sqlc.arg(data_access)::text,
    1
  WHERE sqlc.arg(expected_revision)::bigint = 0
  ON CONFLICT (account_name) DO NOTHING
  RETURNING login_enabled, data_access, revision
)
SELECT login_enabled, data_access, revision FROM updated
UNION ALL
SELECT login_enabled, data_access, revision FROM inserted;
