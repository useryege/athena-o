-- Discovery processing persistence.
-- name: UpsertProjectCandidate :exec
INSERT INTO project_candidate (
  chain_id,
  contract,
  tx_sender,
  tx_hash,
  tx_index,
  block_number,
  block_time,
  status
) VALUES (
  @chain_id,
  @contract,
  @tx_sender,
  @tx_hash,
  @tx_index,
  @block_number,
  @block_time,
  @status
)
ON CONFLICT (chain_id, contract) DO UPDATE
SET tx_sender = EXCLUDED.tx_sender,
  tx_hash = EXCLUDED.tx_hash,
  tx_index = EXCLUDED.tx_index,
  block_number = EXCLUDED.block_number,
  block_time = EXCLUDED.block_time,
  status = EXCLUDED.status;
