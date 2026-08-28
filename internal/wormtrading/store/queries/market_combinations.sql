-- name: CreateMarketCombination :one
INSERT INTO worm_market_combinations (
  id,
  owner_account_id,
  name,
  created_at,
  updated_at
) VALUES (
  sqlc.arg(id)::uuid,
  sqlc.arg(owner_account_id)::uuid,
  sqlc.arg(name)::text,
  sqlc.arg(now)::timestamptz,
  sqlc.arg(now)::timestamptz
)
RETURNING *;

-- name: GetMarketCombination :one
SELECT *
FROM worm_market_combinations
WHERE id = sqlc.arg(id)::uuid
  AND owner_account_id = sqlc.arg(owner_account_id)::uuid;

-- name: GetMarketCombinationForUpdate :one
SELECT *
FROM worm_market_combinations
WHERE id = sqlc.arg(id)::uuid
  AND owner_account_id = sqlc.arg(owner_account_id)::uuid
FOR UPDATE;

-- name: ListMarketCombinations :many
SELECT *
FROM worm_market_combinations
WHERE owner_account_id = sqlc.arg(owner_account_id)::uuid
ORDER BY updated_at DESC, id DESC
LIMIT sqlc.arg(page_size)::integer
OFFSET sqlc.arg(page_offset)::bigint;

-- name: CountMarketCombinations :one
SELECT COUNT(*)::bigint
FROM worm_market_combinations
WHERE owner_account_id = sqlc.arg(owner_account_id)::uuid;

-- name: UpdateMarketCombination :one
UPDATE worm_market_combinations
SET name = sqlc.arg(name)::text,
    revision = revision + 1,
    updated_at = sqlc.arg(now)::timestamptz
WHERE id = sqlc.arg(id)::uuid
  AND owner_account_id = sqlc.arg(owner_account_id)::uuid
  AND revision = sqlc.arg(expected_revision)::bigint
RETURNING *;

-- name: DeleteMarketCombination :one
DELETE FROM worm_market_combinations
WHERE id = sqlc.arg(id)::uuid
  AND owner_account_id = sqlc.arg(owner_account_id)::uuid
  AND revision = sqlc.arg(expected_revision)::bigint
RETURNING id;

-- name: DeleteMarketCombinationItems :exec
DELETE FROM worm_market_combination_items
WHERE combination_id = sqlc.arg(combination_id)::uuid;

-- name: CreateMarketCombinationItem :exec
INSERT INTO worm_market_combination_items (
  combination_id,
  ordinal,
  event_condition_id,
  event_title,
  event_logo,
  market_condition_id,
  market_title,
  market_logo,
  is_yes,
  outcome_label
) VALUES (
  sqlc.arg(combination_id)::uuid,
  sqlc.arg(ordinal)::integer,
  sqlc.arg(event_condition_id)::text,
  sqlc.arg(event_title)::text,
  sqlc.arg(event_logo)::text,
  sqlc.arg(market_condition_id)::text,
  sqlc.arg(market_title)::text,
  sqlc.arg(market_logo)::text,
  sqlc.arg(is_yes)::boolean,
  sqlc.arg(outcome_label)::text
);

-- name: ListMarketCombinationItems :many
SELECT *
FROM worm_market_combination_items
WHERE combination_id = sqlc.arg(combination_id)::uuid
ORDER BY ordinal;
