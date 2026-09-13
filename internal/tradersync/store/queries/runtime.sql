-- name: CheckRuntimeWrite :one
SELECT owner_id, generation FROM trader_sync_runtime_control
WHERE singleton AND owner_id = $1 AND generation = $2 FOR SHARE;

-- name: LockRuntimeControl :one
SELECT owner_id, generation FROM trader_sync_runtime_control WHERE singleton FOR UPDATE;

-- name: ClaimRuntimeOwnership :one
UPDATE trader_sync_runtime_control SET owner_id = $1, generation = generation + 1
WHERE singleton AND generation < 9223372036854775807 RETURNING owner_id, generation;

-- name: ReleaseRuntimeOwnership :execrows
UPDATE trader_sync_runtime_control SET owner_id = NULL
WHERE singleton AND owner_id = $1 AND generation = $2;
