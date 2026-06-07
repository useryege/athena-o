-- name: AddBytecodeBlacklistEntry :exec
INSERT INTO bytecode_blacklist (
  code_hash,
  note,
  source_chain_id,
  source_contract
) VALUES (
  @code_hash,
  sqlc.narg('note'),
  sqlc.narg('source_chain_id'),
  sqlc.narg('source_contract')
);

-- name: GetBytecodeBlacklistEntry :one
SELECT *
FROM bytecode_blacklist
WHERE code_hash = @code_hash;

-- name: ListBytecodeBlacklistEntries :many
SELECT *
FROM bytecode_blacklist
ORDER BY created_at DESC, code_hash;

-- name: UpdateBytecodeBlacklistNote :execrows
UPDATE bytecode_blacklist
SET note = sqlc.narg('note')
WHERE code_hash = @code_hash;

-- name: DeleteBytecodeBlacklistEntry :execrows
DELETE FROM bytecode_blacklist
WHERE code_hash = @code_hash;

-- name: IsBytecodeBlacklisted :one
SELECT EXISTS(
  SELECT 1
  FROM bytecode_blacklist
  WHERE code_hash = @code_hash
)::boolean;
