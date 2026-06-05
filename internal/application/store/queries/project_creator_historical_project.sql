-- name: DeleteProjectCreatorHistoricalProjectsByContract :exec
DELETE FROM project_creator_historical_project
WHERE project_id = (SELECT id FROM project WHERE chain_id = @chain_id AND contract = @project_contract);

-- name: InsertProjectCreatorHistoricalProject :exec
INSERT INTO project_creator_historical_project (
  project_id,
  historical_project_contract,
  rank_index
) VALUES (
  (SELECT id FROM project WHERE chain_id = @chain_id AND contract = @project_contract),
  @historical_project_contract,
  @rank_index
);

-- name: ListProjectCreatorHistoricalProjectsByContract :many
SELECT
  h.id,
  p.chain_id,
  p.contract AS project_contract,
  h.historical_project_contract,
  h.rank_index,
  h.created_at
FROM project_creator_historical_project h
JOIN project p ON p.id = h.project_id
WHERE p.chain_id = @chain_id
  AND p.contract = @project_contract
ORDER BY h.rank_index ASC, h.id ASC;

-- name: ListProjectCreatorHistoricalProjectsByContracts :many
SELECT
  h.id,
  p.chain_id,
  p.contract AS project_contract,
  h.historical_project_contract,
  h.rank_index,
  h.created_at
FROM project_creator_historical_project h
JOIN project p ON p.id = h.project_id
WHERE p.chain_id = @chain_id
  AND p.contract = ANY(@project_contracts::bytea[])
ORDER BY p.contract ASC, h.rank_index ASC, h.id ASC;
