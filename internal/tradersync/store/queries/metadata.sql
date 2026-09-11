-- name: GetTradeMetadata :one
SELECT metadata_json FROM trader_sync_market_metadata WHERE cache_key=$1;

-- name: SaveTradeMetadata :exec
INSERT INTO trader_sync_market_metadata(cache_key,metadata_json) VALUES($1,$2)
ON CONFLICT(cache_key) DO UPDATE SET metadata_json=EXCLUDED.metadata_json,updated_at=clock_timestamp();

-- name: LookupComboPosition :many
SELECT market_id,condition_id,position_ids FROM trader_sync_combo_leg_index WHERE position_id=$1 ORDER BY market_id;

-- name: UpsertComboPosition :exec
INSERT INTO trader_sync_combo_leg_index(position_id,market_id,condition_id,position_ids) VALUES($1,$2,$3,$4)
ON CONFLICT(position_id,market_id) DO UPDATE SET condition_id=EXCLUDED.condition_id,position_ids=EXCLUDED.position_ids,seen_at=clock_timestamp();

-- name: EnsureComboDirectory :exec
INSERT INTO trader_sync_directory_refresh(name) VALUES('combo_markets') ON CONFLICT(name) DO NOTHING;

-- name: LockComboDirectory :one
SELECT *,clock_timestamp()::timestamptz AS database_now FROM trader_sync_directory_refresh WHERE name='combo_markets' FOR UPDATE;

-- name: StartComboRound :exec
UPDATE trader_sync_directory_refresh SET round_started_at=clock_timestamp(),visited_cursors='{}'
WHERE name='combo_markets' AND cursor='' AND (round_started_at IS NULL OR round_completed_at>=round_started_at);

-- name: DelayComboPage :one
UPDATE trader_sync_directory_refresh SET next_page_at=clock_timestamp()+interval '1 second',admission_id=NULL
WHERE name='combo_markets' AND admission_id=sqlc.arg(admission_id)::uuid AND cursor=sqlc.arg(expected_cursor)::text
RETURNING next_page_at,clock_timestamp()::timestamptz AS database_now;

-- name: AdvanceComboPage :one
UPDATE trader_sync_directory_refresh SET cursor=sqlc.arg(next_cursor),admission_id=NULL,
 visited_cursors=array_append(visited_cursors,cursor),
 round_completed_at=CASE WHEN sqlc.arg(next_cursor)::text='' THEN clock_timestamp() ELSE round_completed_at END,
 next_page_at=CASE WHEN sqlc.arg(next_cursor)::text='' THEN greatest(round_started_at+interval '10 minutes',clock_timestamp()+interval '1 second') ELSE clock_timestamp()+interval '1 second' END
WHERE name='combo_markets' AND cursor=sqlc.arg(expected_cursor) AND admission_id=sqlc.arg(admission_id)::uuid
RETURNING next_page_at,clock_timestamp()::timestamptz AS database_now;

-- name: LockTradeMetadata :exec
SELECT pg_advisory_xact_lock(hashtextextended('athena:metadata:' || sqlc.arg(cache_key)::text,0));

-- name: AdmitComboPage :one
UPDATE trader_sync_directory_refresh SET admission_id=sqlc.arg(admission_id)::uuid,next_page_at=clock_timestamp()+interval '6 seconds'
WHERE name='combo_markets' AND cursor=sqlc.arg(expected_cursor)::text
RETURNING next_page_at,clock_timestamp()::timestamptz AS database_now;
