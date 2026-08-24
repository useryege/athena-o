-- name: GetAccountProfile :one
SELECT display_name, account_tier, avatar_object_key, avatar_content_type,
       avatar_etag, avatar_size_bytes, revision
FROM account_profile
WHERE account_name = sqlc.arg(account_name)::text;

-- name: ListAvatarObjectKeys :many
SELECT avatar_object_key
FROM account_profile
WHERE avatar_object_key <> ''
ORDER BY avatar_object_key;

-- name: CreateAccountProfile :one
INSERT INTO account_profile (
  account_name, display_name, account_tier, avatar_object_key,
  avatar_content_type, avatar_etag, avatar_size_bytes, revision
)
VALUES (
  sqlc.arg(account_name)::text,
  sqlc.arg(display_name)::text,
  sqlc.arg(account_tier)::text,
  sqlc.arg(avatar_object_key)::text,
  sqlc.arg(avatar_content_type)::text,
  sqlc.arg(avatar_etag)::text,
  sqlc.arg(avatar_size_bytes)::bigint,
  1
)
ON CONFLICT (account_name) DO NOTHING
RETURNING display_name, account_tier, avatar_object_key, avatar_content_type,
          avatar_etag, avatar_size_bytes, revision;

-- name: UpdateAccountProfile :one
UPDATE account_profile
SET display_name = sqlc.arg(display_name)::text,
    account_tier = sqlc.arg(account_tier)::text,
    avatar_object_key = sqlc.arg(avatar_object_key)::text,
    avatar_content_type = sqlc.arg(avatar_content_type)::text,
    avatar_etag = sqlc.arg(avatar_etag)::text,
    avatar_size_bytes = sqlc.arg(avatar_size_bytes)::bigint,
    revision = account_profile.revision + 1,
    updated_at = NOW()
WHERE account_name = sqlc.arg(account_name)::text
  AND sqlc.arg(expected_revision)::bigint > 0
  AND revision = sqlc.arg(expected_revision)::bigint
RETURNING display_name, account_tier, avatar_object_key, avatar_content_type,
          avatar_etag, avatar_size_bytes, revision;

-- name: GetAccountPreferences :one
SELECT theme, revision
FROM account_preferences
WHERE account_name = sqlc.arg(account_name)::text;

-- name: CreateAccountPreferences :one
INSERT INTO account_preferences (account_name, theme, revision)
VALUES (sqlc.arg(account_name)::text, sqlc.arg(theme)::text, 1)
ON CONFLICT (account_name) DO NOTHING
RETURNING theme, revision;

-- name: UpdateAccountPreferences :one
UPDATE account_preferences
SET theme = sqlc.arg(theme)::text,
    revision = account_preferences.revision + 1,
    updated_at = NOW()
WHERE account_name = sqlc.arg(account_name)::text
  AND sqlc.arg(expected_revision)::bigint > 0
  AND revision = sqlc.arg(expected_revision)::bigint
RETURNING theme, revision;
