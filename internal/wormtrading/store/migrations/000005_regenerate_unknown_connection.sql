-- +goose Up

ALTER TABLE worm_wallet_connection_attempts
  DROP CONSTRAINT worm_wallet_connection_attempts_kind_check;

ALTER TABLE worm_wallet_connection_attempts
  ADD CONSTRAINT worm_wallet_connection_attempts_kind_check
  CHECK (kind IN ('CONNECT', 'RECONNECT', 'REGENERATE'));

-- +goose Down

UPDATE worm_wallet_connections AS connection
SET
  state = attempt.previous_connection_state,
  warning_code = 'CONNECT_OUTCOME_UNKNOWN',
  connected_at = attempt.previous_connected_at,
  updated_at = now()
FROM worm_wallet_connection_attempts AS attempt
WHERE attempt.wallet_id = connection.wallet_id
  AND attempt.address = connection.address
  AND attempt.kind = 'REGENERATE'
  AND attempt.state IN ('PREPARED', 'COMPLETING');

DELETE FROM worm_wallet_connection_attempts
WHERE kind = 'REGENERATE';

ALTER TABLE worm_wallet_connection_attempts
  DROP CONSTRAINT worm_wallet_connection_attempts_kind_check;

ALTER TABLE worm_wallet_connection_attempts
  ADD CONSTRAINT worm_wallet_connection_attempts_kind_check
  CHECK (kind IN ('CONNECT', 'RECONNECT'));
