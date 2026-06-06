-- name: UpsertProjectAveDetail :exec
INSERT INTO project_ave_detail (
  project_id,
  ave_response,
  fetched_at
) VALUES (
  (SELECT id FROM project WHERE chain_id = @chain_id AND contract = @project_contract),
  @ave_response::jsonb,
  @fetched_at
)
ON CONFLICT (project_id) DO UPDATE
SET ave_response = EXCLUDED.ave_response,
  fetched_at = EXCLUDED.fetched_at,
  updated_at = now();

-- name: ListProjectAveDetailsByContracts :many
SELECT p.chain_id,
  p.contract AS project_contract,
  d.ave_response,
  d.fetched_at,
  d.updated_at
FROM project_ave_detail d
JOIN project p ON p.id = d.project_id
WHERE p.chain_id = @chain_id
  AND p.contract = ANY(@project_contracts::bytea[]);
