-- +goose Up

-- Preview policy v3 has a different digest and mandatory exposure guards.
-- Preserve plans already consumed by a run, but make every unconsumed older
-- preview impossible to execute after this migration.
UPDATE worm_execution_plans AS plans
SET state = 'FAILED',
    build_stage = '',
    failure_code = 'PLAN_CONTRACT_CHANGED',
    worker_id = '',
    locked_at = NULL,
    lease_expires_at = NULL,
    completed_at = CURRENT_TIMESTAMP,
    updated_at = CURRENT_TIMESTAMP
WHERE plans.state = 'BUILDING'
  AND NOT EXISTS (
    SELECT 1 FROM worm_execution_runs AS runs WHERE runs.plan_id = plans.id
  );

UPDATE worm_execution_plans AS plans
SET expires_at = LEAST(plans.expires_at, CURRENT_TIMESTAMP),
    updated_at = CURRENT_TIMESTAMP
WHERE plans.state = 'READY'
  AND NOT EXISTS (
    SELECT 1 FROM worm_execution_runs AS runs WHERE runs.plan_id = plans.id
  );

ALTER TABLE worm_execution_plans
  DROP COLUMN skip_opposite_side_exposure,
  DROP COLUMN skip_in_flight_request,
  DROP COLUMN skip_already_held;

ALTER TABLE worm_execution_runs
  DROP COLUMN skip_opposite_side_exposure,
  DROP COLUMN skip_in_flight_request,
  DROP COLUMN skip_already_held;

UPDATE worm_execution_plan_steps
SET advisory_codes = CASE
  WHEN 'LIQUIDITY_INSUFFICIENT' = ANY(advisory_codes)
    THEN ARRAY['LIQUIDITY_INSUFFICIENT']::text[]
  ELSE '{}'::text[]
END;

UPDATE worm_execution_run_steps
SET advisory_codes = CASE
  WHEN 'LIQUIDITY_INSUFFICIENT' = ANY(advisory_codes)
    THEN ARRAY['LIQUIDITY_INSUFFICIENT']::text[]
  ELSE '{}'::text[]
END;

ALTER TABLE worm_execution_run_steps
  ADD COLUMN completion_source TEXT NOT NULL DEFAULT '' CHECK (
    completion_source IN ('', 'OPEN_POSITION')
  ),
  ADD COLUMN completion_position_pubkey TEXT NOT NULL DEFAULT '' CHECK (
    completion_position_pubkey = ''
    OR (
      completion_position_pubkey = btrim(completion_position_pubkey)
      AND char_length(completion_position_pubkey) BETWEEN 32 AND 64
    )
  ),
  ADD COLUMN completion_position_request_pubkey TEXT NOT NULL DEFAULT '' CHECK (
    completion_position_request_pubkey = ''
    OR (
      completion_position_request_pubkey = btrim(completion_position_request_pubkey)
      AND char_length(completion_position_request_pubkey) BETWEEN 32 AND 64
    )
  ),
  ADD COLUMN completion_position_created_at TIMESTAMPTZ;

-- The v2 checks coupled completion to Worm Web state and required transaction
-- metadata even when a completed Web request did not expose it. Locate these
-- unnamed constraints by definition so the migration works for both existing
-- databases and a fresh migration chain.
-- +goose StatementBegin
DO $$
DECLARE
  constraint_name TEXT;
BEGIN
  FOR constraint_name IN
    SELECT conname
    FROM pg_constraint
    WHERE conrelid = 'worm_execution_run_steps'::regclass
      AND contype = 'c'
      AND (
        (
          pg_get_constraintdef(oid) ILIKE '%provider_state%'
          AND pg_get_constraintdef(oid) ILIKE '%completed%'
        )
        OR pg_get_constraintdef(oid) ILIKE '%position_request_id IS NOT NULL%'
        OR pg_get_constraintdef(oid) ILIKE '%transaction_message_sha256 IS NOT NULL%'
      )
  LOOP
    EXECUTE format(
      'ALTER TABLE worm_execution_run_steps DROP CONSTRAINT %I',
      constraint_name
    );
  END LOOP;
END $$;
-- +goose StatementEnd

ALTER TABLE worm_execution_run_steps
  ADD CONSTRAINT worm_execution_run_steps_position_request_state_check CHECK (
    (
      state IN ('OPENED', 'SIGNING', 'FINALIZING', 'AWAITING_COMPLETION')
      AND position_request_id IS NOT NULL
    )
    OR state NOT IN ('OPENED', 'SIGNING', 'FINALIZING', 'AWAITING_COMPLETION')
  ),
  ADD CONSTRAINT worm_execution_run_steps_transaction_state_check CHECK (
    (
      state IN ('OPENED', 'SIGNING', 'FINALIZING', 'AWAITING_COMPLETION')
      AND transaction_message_sha256 IS NOT NULL
    )
    OR state NOT IN ('OPENED', 'SIGNING', 'FINALIZING', 'AWAITING_COMPLETION')
  ),
  ADD CONSTRAINT worm_execution_run_steps_completion_evidence_check CHECK (
    (
      state = 'COMPLETED'
      AND completion_source = 'OPEN_POSITION'
      AND completion_position_pubkey <> ''
      AND completion_position_created_at IS NOT NULL
      AND completion_position_created_at > TIMESTAMPTZ '1970-01-01 00:00:00+00'
    )
    OR (
      state <> 'COMPLETED'
      AND completion_source = ''
      AND completion_position_pubkey = ''
      AND completion_position_request_pubkey = ''
      AND completion_position_created_at IS NULL
    )
  );

CREATE UNIQUE INDEX worm_execution_run_steps_completion_position_pubkey_unique
  ON worm_execution_run_steps (completion_position_pubkey)
  WHERE completion_position_pubkey <> '';

CREATE UNIQUE INDEX worm_execution_run_steps_completion_request_pubkey_unique
  ON worm_execution_run_steps (completion_position_request_pubkey)
  WHERE completion_position_request_pubkey <> '';

-- +goose Down

DROP INDEX IF EXISTS worm_execution_run_steps_completion_request_pubkey_unique;
DROP INDEX IF EXISTS worm_execution_run_steps_completion_position_pubkey_unique;

ALTER TABLE worm_execution_run_steps
  DROP CONSTRAINT IF EXISTS worm_execution_run_steps_completion_evidence_check,
  DROP CONSTRAINT IF EXISTS worm_execution_run_steps_transaction_state_check,
  DROP CONSTRAINT IF EXISTS worm_execution_run_steps_position_request_state_check;

-- v3 completion evidence cannot be represented truthfully by the v2 schema.
-- Refuse that destructive rollback instead of fabricating provider metadata.
-- +goose StatementBegin
DO $$
BEGIN
  IF EXISTS (
    SELECT 1
    FROM worm_execution_run_steps
    WHERE state = 'COMPLETED' AND completion_source = 'OPEN_POSITION'
  ) THEN
    RAISE EXCEPTION 'cannot roll back execution completion v3 after an open-position completion was recorded';
  END IF;
END $$;
-- +goose StatementEnd

ALTER TABLE worm_execution_run_steps
  DROP COLUMN completion_position_created_at,
  DROP COLUMN completion_position_request_pubkey,
  DROP COLUMN completion_position_pubkey,
  DROP COLUMN completion_source,
  ADD CONSTRAINT worm_execution_run_steps_provider_completed_check CHECK (
    (state = 'COMPLETED' AND lower(provider_state) = 'completed')
    OR state <> 'COMPLETED'
  ),
  ADD CONSTRAINT worm_execution_run_steps_position_request_state_check CHECK (
    (
      state IN ('OPENED', 'SIGNING', 'FINALIZING', 'AWAITING_COMPLETION', 'COMPLETED')
      AND position_request_id IS NOT NULL
    )
    OR state NOT IN ('OPENED', 'SIGNING', 'FINALIZING', 'AWAITING_COMPLETION', 'COMPLETED')
  ),
  ADD CONSTRAINT worm_execution_run_steps_transaction_state_check CHECK (
    (
      state IN ('OPENED', 'SIGNING', 'FINALIZING', 'AWAITING_COMPLETION', 'COMPLETED')
      AND transaction_message_sha256 IS NOT NULL
    )
    OR state NOT IN ('OPENED', 'SIGNING', 'FINALIZING', 'AWAITING_COMPLETION', 'COMPLETED')
  );

ALTER TABLE worm_execution_runs
  ADD COLUMN skip_already_held BOOLEAN NOT NULL DEFAULT TRUE,
  ADD COLUMN skip_in_flight_request BOOLEAN NOT NULL DEFAULT TRUE,
  ADD COLUMN skip_opposite_side_exposure BOOLEAN NOT NULL DEFAULT TRUE;

ALTER TABLE worm_execution_plans
  ADD COLUMN skip_already_held BOOLEAN NOT NULL DEFAULT TRUE,
  ADD COLUMN skip_in_flight_request BOOLEAN NOT NULL DEFAULT TRUE,
  ADD COLUMN skip_opposite_side_exposure BOOLEAN NOT NULL DEFAULT TRUE;
