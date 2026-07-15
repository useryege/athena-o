-- Discovery validation persistence.
-- name: UpsertProjectRelatedWallet :one
INSERT INTO project_related_wallet (
  project_id,
  wallet,
  role
) VALUES (
  @project_id,
  @wallet,
  @role
)
ON CONFLICT (project_id, wallet, role) DO UPDATE
SET role = EXCLUDED.role
RETURNING *;

-- name: ListProjectRelatedWalletsByProject :many
SELECT *
FROM project_related_wallet
WHERE project_id = @project_id
ORDER BY created_at, wallet, role;

-- name: ListProjectRelatedWalletsByWallet :many
SELECT *
FROM project_related_wallet
WHERE wallet = @wallet
ORDER BY created_at, project_id, role;

-- name: DeleteProjectRelatedWallet :execrows
DELETE FROM project_related_wallet
WHERE project_id = @project_id
  AND wallet = @wallet
  AND role = @role;

-- name: DeleteProjectRelatedWalletsByProject :execrows
DELETE FROM project_related_wallet
WHERE project_id = @project_id;
