-- name: GetScanCheckpoint :one
SELECT singleton, start_block_number, cursor_block_number, cursor_block_hash,
       cursor_block_timestamp, initialized_at, updated_at
FROM bsc_inbound_scan_checkpoint
WHERE singleton = TRUE;

-- name: InitializeScanCheckpoint :one
INSERT INTO bsc_inbound_scan_checkpoint (
  singleton,
  start_block_number,
  cursor_block_number,
  cursor_block_hash,
  cursor_block_timestamp
) VALUES (
  TRUE,
  @start_block_number,
  @cursor_block_number,
  @cursor_block_hash,
  @cursor_block_timestamp
)
ON CONFLICT (singleton) DO NOTHING
RETURNING *;

-- name: AdvanceScanCheckpoint :execrows
UPDATE bsc_inbound_scan_checkpoint
SET cursor_block_number = @cursor_block_number,
    cursor_block_hash = @cursor_block_hash,
    cursor_block_timestamp = @cursor_block_timestamp,
    updated_at = NOW()
WHERE singleton = TRUE
  AND cursor_block_number = @expected_cursor_block_number;
