-- name: DeleteProjectCreatorHistoricalProjectsByContract :exec
DELETE FROM project_creator_historical_project
WHERE project_contract = $1;

-- name: InsertProjectCreatorHistoricalProject :exec
INSERT INTO project_creator_historical_project (
  project_contract,
  historical_project_contract,
  rank_index
) VALUES ($1, $2, $3);

-- name: ListProjectCreatorHistoricalProjectsByContract :many
SELECT
  id,
  project_contract,
  historical_project_contract,
  rank_index,
  created_at
FROM project_creator_historical_project
WHERE project_contract = $1
ORDER BY rank_index ASC, id ASC;

-- name: ListProjectCreatorHistoricalProjectsByContracts :many
SELECT
  id,
  project_contract,
  historical_project_contract,
  rank_index,
  created_at
FROM project_creator_historical_project
WHERE project_contract = ANY($1::bytea[])
ORDER BY project_contract ASC, rank_index ASC, id ASC;
