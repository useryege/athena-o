-- name: CreateWalletBlocklistEntry :exec
INSERT INTO wallet_blocklist (
  wallet,
  note
) VALUES (
  @wallet,
  sqlc.narg('note')
);

-- name: GetWalletBlocklistEntry :one
SELECT *
FROM wallet_blocklist
WHERE wallet = @wallet;

-- name: ListWalletBlocklistEntries :many
SELECT *
FROM wallet_blocklist
ORDER BY created_at DESC, wallet;

-- name: UpdateWalletBlocklistEntryNote :execrows
UPDATE wallet_blocklist
SET note = sqlc.narg('note')
WHERE wallet = @wallet;

-- name: DeleteWalletBlocklistEntry :execrows
DELETE FROM wallet_blocklist
WHERE wallet = @wallet;

-- name: IsWalletBlocked :one
SELECT EXISTS(
  SELECT 1
  FROM wallet_blocklist
  WHERE wallet = @wallet
)::boolean;
