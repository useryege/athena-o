-- name: ListAccountAPIKeys :many
SELECT display_id, issued_at, expires_at
FROM account_api_key
WHERE account_name = sqlc.arg(account_name)::text
ORDER BY issued_at DESC, display_id;

-- name: ListAccountAPIKeyRecords :many
SELECT account_name, display_id, jti, issued_at, expires_at
FROM account_api_key
ORDER BY account_name, issued_at DESC, display_id;

-- name: GetAccountAPIKeyByJTI :one
SELECT key.account_name,
       key.display_id,
       key.jti,
       key.issued_at,
       key.expires_at,
       access.login_enabled,
       access.api_key_enabled
FROM account_api_key AS key
JOIN account_access AS access USING (account_name)
WHERE key.jti = sqlc.arg(jti)::text;

-- name: GetUsableAccountAPIKeyByJTI :one
SELECT key.account_name,
       key.display_id,
       key.jti,
       key.issued_at,
       key.expires_at
FROM account_api_key AS key
JOIN account_access AS access USING (account_name)
WHERE key.jti = sqlc.arg(jti)::text
  AND access.login_enabled
  AND access.api_key_enabled
  AND (key.expires_at IS NULL OR key.expires_at > NOW());

-- name: CreateAccountAPIKey :one
INSERT INTO account_api_key (
  account_name,
  display_id,
  jti,
  issued_at,
  expires_at
)
SELECT access.account_name,
       sqlc.arg(display_id)::text,
       sqlc.arg(jti)::text,
       sqlc.arg(issued_at)::timestamptz,
       sqlc.narg(expires_at)::timestamptz
FROM account_access AS access
WHERE access.account_name = sqlc.arg(account_name)::text
  AND access.login_enabled
  AND access.api_key_enabled
RETURNING account_name, display_id, jti, issued_at, expires_at;

-- name: DeleteAccountAPIKey :one
DELETE FROM account_api_key
WHERE account_name = sqlc.arg(account_name)::text
  AND display_id = sqlc.arg(display_id)::text
RETURNING jti;
