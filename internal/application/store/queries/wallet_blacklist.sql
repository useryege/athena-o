-- name: ListWalletBlacklistEntries :many
SELECT wallet, note, created_at
FROM wallet_blacklist
ORDER BY created_at DESC, wallet;

-- name: AddWalletBlacklistEntry :exec
INSERT INTO wallet_blacklist (wallet, note)
VALUES ($1, $2);

-- name: UpdateWalletBlacklistEntryNote :execrows
UPDATE wallet_blacklist
SET note = $2
WHERE wallet = $1;

-- name: DeleteWalletBlacklistEntry :execrows
DELETE FROM wallet_blacklist
WHERE wallet = $1;

-- name: GetWalletBlacklistEntry :one
SELECT wallet, note, created_at
FROM wallet_blacklist
WHERE wallet = $1;
