-- name: UpsertProjectInitialRecipient :one
INSERT INTO project_initial_recipient (
  project_id,
  wallet,
  ratio_bps,
  rank_index,
  source_tx_hash,
  source_block_number
) VALUES (
  @project_id,
  @wallet,
  @ratio_bps,
  @rank_index,
  @source_tx_hash,
  @source_block_number
)
ON CONFLICT (project_id, wallet) DO UPDATE
SET ratio_bps = EXCLUDED.ratio_bps,
  rank_index = EXCLUDED.rank_index,
  source_tx_hash = EXCLUDED.source_tx_hash,
  source_block_number = EXCLUDED.source_block_number
RETURNING *;

-- name: GetProjectInitialRecipient :one
SELECT *
FROM project_initial_recipient
WHERE project_id = @project_id
  AND wallet = @wallet;

-- name: ListProjectInitialRecipientsByProject :many
SELECT *
FROM project_initial_recipient
WHERE project_id = @project_id
ORDER BY rank_index, wallet;

-- name: ListProjectInitialRecipientsByWallet :many
SELECT *
FROM project_initial_recipient
WHERE wallet = @wallet
ORDER BY ratio_bps DESC, project_id;

-- name: DeleteProjectInitialRecipient :execrows
DELETE FROM project_initial_recipient
WHERE project_id = @project_id
  AND wallet = @wallet;

-- name: DeleteProjectInitialRecipientsByProject :execrows
DELETE FROM project_initial_recipient
WHERE project_id = @project_id;
