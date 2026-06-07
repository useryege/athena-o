-- name: AddWalletBlacklistEntry :exec
INSERT INTO wallet_blacklist (
  wallet,
  note
) VALUES (
  @wallet,
  sqlc.narg('note')
);

-- name: GetWalletBlacklistEntry :one
SELECT *
FROM wallet_blacklist
WHERE wallet = @wallet;

-- name: ListWalletBlacklistEntries :many
SELECT *
FROM wallet_blacklist
ORDER BY created_at DESC, wallet;

-- name: UpdateWalletBlacklistNote :execrows
UPDATE wallet_blacklist
SET note = sqlc.narg('note')
WHERE wallet = @wallet;

-- name: DeleteWalletBlacklistEntry :execrows
DELETE FROM wallet_blacklist
WHERE wallet = @wallet;

-- name: IsWalletBlacklisted :one
SELECT EXISTS(
  SELECT 1
  FROM wallet_blacklist
  WHERE wallet = @wallet
)::boolean;
