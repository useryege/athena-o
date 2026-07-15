-- Policy persistence.
-- name: CreateContractCodeBlocklistEntry :exec
INSERT INTO contract_code_blocklist (
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

-- name: GetContractCodeBlocklistEntry :one
SELECT *
FROM contract_code_blocklist
WHERE code_hash = @code_hash;

-- name: ListContractCodeBlocklistEntries :many
SELECT *
FROM contract_code_blocklist
ORDER BY created_at DESC, code_hash;

-- name: UpdateContractCodeBlocklistEntryNote :execrows
UPDATE contract_code_blocklist
SET note = sqlc.narg('note')
WHERE code_hash = @code_hash;

-- name: DeleteContractCodeBlocklistEntry :execrows
DELETE FROM contract_code_blocklist
WHERE code_hash = @code_hash;

-- name: IsContractCodeBlocked :one
SELECT EXISTS(
  SELECT 1
  FROM contract_code_blocklist
  WHERE code_hash = @code_hash
)::boolean;
