-- name: GetActiveSourceQualityPrompt :one
SELECT id, version, name, system_prompt, is_active, created_at, updated_at
FROM source_quality_prompt
WHERE is_active AND deleted_at IS NULL
ORDER BY version DESC
LIMIT 1;

-- name: ListSourceQualityPrompts :many
SELECT id, version, name, system_prompt, is_active, created_at, updated_at
FROM source_quality_prompt
WHERE deleted_at IS NULL
ORDER BY is_active DESC, version DESC;

-- name: GetSourceQualityPrompt :one
SELECT id, version, name, system_prompt, is_active, created_at, updated_at
FROM source_quality_prompt
WHERE id = $1 AND deleted_at IS NULL;

-- name: InsertSourceQualityPrompt :one
INSERT INTO source_quality_prompt (name, system_prompt, is_active)
VALUES ($1, $2, $3)
RETURNING id, version, name, system_prompt, is_active, created_at, updated_at;

-- name: GetSourceQualityPromptForUpdate :one
SELECT id, version, name, system_prompt, is_active, created_at, updated_at
FROM source_quality_prompt
WHERE id = $1 AND deleted_at IS NULL
FOR UPDATE;

-- name: DeactivateActiveSourceQualityPrompts :exec
UPDATE source_quality_prompt
SET is_active = false, updated_at = now()
WHERE is_active AND deleted_at IS NULL;

-- name: ActivateSourceQualityPrompt :one
UPDATE source_quality_prompt
SET is_active = true, updated_at = now()
WHERE id = $1 AND deleted_at IS NULL
RETURNING id, version, name, system_prompt, is_active, created_at, updated_at;

-- name: DeleteSourceQualityPrompt :execrows
UPDATE source_quality_prompt
SET deleted_at = now(), updated_at = now()
WHERE id = $1 AND deleted_at IS NULL AND NOT is_active;
