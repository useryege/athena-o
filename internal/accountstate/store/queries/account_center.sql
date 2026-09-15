-- name: GetAccountProfile :one
SELECT display_name, account_tier, avatar_object_key, avatar_content_type,
       avatar_etag, avatar_size_bytes, revision
FROM account_profile
WHERE account_id = sqlc.arg(account_id)::uuid;

-- name: ListAvatarObjectKeys :many
SELECT avatar_object_key
FROM account_profile
WHERE avatar_object_key <> ''
ORDER BY avatar_object_key;

-- name: CreateAccountProfile :one
INSERT INTO account_profile (
  account_id, display_name, account_tier, avatar_object_key,
  avatar_content_type, avatar_etag, avatar_size_bytes, revision
)
VALUES (
  sqlc.arg(account_id)::uuid,
  sqlc.arg(display_name)::text,
  sqlc.arg(account_tier)::text,
  sqlc.arg(avatar_object_key)::text,
  sqlc.arg(avatar_content_type)::text,
  sqlc.arg(avatar_etag)::text,
  sqlc.arg(avatar_size_bytes)::bigint,
  1
)
ON CONFLICT (account_id) DO NOTHING
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
WHERE account_id = sqlc.arg(account_id)::uuid
  AND sqlc.arg(expected_revision)::bigint > 0
  AND revision = sqlc.arg(expected_revision)::bigint
RETURNING display_name, account_tier, avatar_object_key, avatar_content_type,
          avatar_etag, avatar_size_bytes, revision;
