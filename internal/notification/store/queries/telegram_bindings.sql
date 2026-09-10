-- name: LockTelegramBindingAccount :exec
SELECT pg_advisory_xact_lock(hashtextextended('athena:account:' || sqlc.arg('account_id')::uuid::text, 0));

-- name: LockTelegramBindingIdentity :exec
SELECT pg_advisory_xact_lock(
  hashtextextended(
    'telegram-binding:' || sqlc.arg('telegram_user_id')::bigint::text || ':' || sqlc.arg('telegram_chat_id')::bigint::text,
    0
  )
);

-- name: GetTelegramBinding :one
SELECT account_id, telegram_user_id, telegram_chat_id,
  COALESCE(telegram_username, '') AS telegram_username,
  telegram_display_name, status, revision, bound_at, updated_at,
  COALESCE(last_error, '') AS last_error
FROM telegram_bindings
WHERE account_id = $1;

-- name: GetTelegramBindingForShare :one
SELECT account_id, telegram_user_id, telegram_chat_id,
  COALESCE(telegram_username, '') AS telegram_username,
  telegram_display_name, status, revision, bound_at, updated_at,
  COALESCE(last_error, '') AS last_error
FROM telegram_bindings
WHERE account_id = $1
FOR SHARE;

-- name: GetTelegramBindingByIdentity :one
SELECT account_id, telegram_user_id, telegram_chat_id,
  COALESCE(telegram_username, '') AS telegram_username,
  telegram_display_name, status, revision, bound_at, updated_at,
  COALESCE(last_error, '') AS last_error
FROM telegram_bindings
WHERE telegram_user_id = $1
  AND telegram_chat_id = $2;

-- name: ReplaceTelegramBinding :one
INSERT INTO telegram_bindings (
  account_id, telegram_user_id, telegram_chat_id, telegram_username,
  telegram_display_name, status, revision
)
VALUES ($1, $2, $3, $4, $5, 'connected', $6)
ON CONFLICT (account_id) DO UPDATE
SET telegram_user_id = EXCLUDED.telegram_user_id,
    telegram_chat_id = EXCLUDED.telegram_chat_id,
    telegram_username = EXCLUDED.telegram_username,
    telegram_display_name = EXCLUDED.telegram_display_name,
    status = 'connected',
    revision = EXCLUDED.revision,
    bound_at = NOW(),
    updated_at = NOW(),
    last_error = NULL
RETURNING account_id, telegram_user_id, telegram_chat_id,
  COALESCE(telegram_username, '') AS telegram_username,
  telegram_display_name, status, revision, bound_at, updated_at,
  COALESCE(last_error, '') AS last_error;

-- name: DeleteTelegramBinding :one
DELETE FROM telegram_bindings
WHERE account_id = $1
RETURNING account_id, telegram_user_id, telegram_chat_id,
  COALESCE(telegram_username, '') AS telegram_username,
  telegram_display_name, status, revision, bound_at, updated_at,
  COALESCE(last_error, '') AS last_error;

-- name: MarkTelegramBindingUnreachable :execrows
UPDATE telegram_bindings
SET status = 'unreachable',
    updated_at = NOW(),
    last_error = $4
WHERE account_id = $1
  AND telegram_chat_id = $2
  AND revision = $3
  AND status <> 'unreachable';

-- name: MarkTelegramBindingUnreachableByIdentity :one
UPDATE telegram_bindings
SET status = 'unreachable',
    updated_at = NOW(),
    last_error = $3
WHERE telegram_user_id = $1
  AND telegram_chat_id = $2
  AND status <> 'unreachable'
RETURNING account_id, telegram_chat_id, revision;

-- name: MarkTelegramBindingConnectedByIdentity :execrows
UPDATE telegram_bindings
SET status = 'connected',
    updated_at = NOW(),
    last_error = NULL
WHERE telegram_user_id = $1
  AND telegram_chat_id = $2
  AND status <> 'connected';

-- name: ListTelegramBindingsByIdentityForShare :many
SELECT account_id, telegram_user_id, telegram_chat_id,
  COALESCE(telegram_username, '') AS telegram_username,
  telegram_display_name, status, revision, bound_at, updated_at,
  COALESCE(last_error, '') AS last_error
FROM telegram_bindings
WHERE telegram_user_id = $1
   OR telegram_chat_id = $2
FOR SHARE;

-- name: NextTelegramBindingRevision :one
INSERT INTO telegram_binding_versions (account_id, revision)
VALUES ($1, 1)
ON CONFLICT (account_id) DO UPDATE
SET revision = telegram_binding_versions.revision + 1
RETURNING revision;

-- name: UpsertTelegramBindingAttempt :one
INSERT INTO telegram_binding_attempts (
  id, account_id, token_digest, status, failure_reason, expires_at
)
VALUES ($1, $2, $3, 'pending', NULL, $4)
ON CONFLICT (account_id) DO UPDATE
SET id = EXCLUDED.id,
    token_digest = EXCLUDED.token_digest,
    status = 'pending',
    failure_reason = NULL,
    expires_at = EXCLUDED.expires_at,
    created_at = NOW(),
    updated_at = NOW()
RETURNING id, account_id, token_digest, status, failure_reason, expires_at, created_at, updated_at;

-- name: GetTelegramBindingAttempt :one
SELECT id, account_id, token_digest, status, failure_reason, expires_at, created_at, updated_at
FROM telegram_binding_attempts
WHERE account_id = $1;

-- name: GetTelegramBindingAttemptByTokenForUpdate :one
SELECT id, account_id, token_digest, status, failure_reason, expires_at, created_at, updated_at
FROM telegram_binding_attempts
WHERE token_digest = $1
FOR UPDATE;

-- name: GetTelegramBindingAttemptAccountByToken :one
SELECT account_id
FROM telegram_binding_attempts
WHERE token_digest = $1;

-- name: FailTelegramBindingAttempt :one
UPDATE telegram_binding_attempts
SET status = 'failed',
    failure_reason = $2,
    updated_at = NOW()
WHERE id = $1
RETURNING id, account_id, token_digest, status, failure_reason, expires_at, created_at, updated_at;

-- name: DeleteTelegramBindingAttemptByAccount :execrows
DELETE FROM telegram_binding_attempts
WHERE account_id = $1;

-- name: DeleteTelegramBindingAttemptByID :execrows
DELETE FROM telegram_binding_attempts
WHERE id = $1;

-- name: GetTelegramPollingState :one
SELECT next_update_id, last_poll_at, last_update_at, updated_at
FROM telegram_polling_state
WHERE singleton = TRUE;

-- name: RecordTelegramPoll :one
UPDATE telegram_polling_state
SET last_poll_at = $1,
    updated_at = NOW()
WHERE singleton = TRUE
RETURNING next_update_id, last_poll_at, last_update_at, updated_at;

-- name: AdvanceTelegramPollingState :one
UPDATE telegram_polling_state
SET next_update_id = GREATEST(next_update_id, $1),
    last_update_at = $2,
    updated_at = NOW()
WHERE singleton = TRUE
RETURNING next_update_id, last_poll_at, last_update_at, updated_at;
