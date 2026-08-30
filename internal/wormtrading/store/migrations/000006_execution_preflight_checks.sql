-- +goose Up

ALTER TABLE worm_execution_plans
  ADD COLUMN skip_already_held BOOLEAN NOT NULL DEFAULT TRUE,
  ADD COLUMN skip_in_flight_request BOOLEAN NOT NULL DEFAULT TRUE,
  ADD COLUMN skip_opposite_side_exposure BOOLEAN NOT NULL DEFAULT TRUE,
  ADD COLUMN require_full_liquidity BOOLEAN NOT NULL DEFAULT TRUE;

ALTER TABLE worm_execution_plan_steps
  ADD COLUMN advisory_codes TEXT[] NOT NULL DEFAULT '{}';

ALTER TABLE worm_execution_runs
  ADD COLUMN skip_already_held BOOLEAN NOT NULL DEFAULT TRUE,
  ADD COLUMN skip_in_flight_request BOOLEAN NOT NULL DEFAULT TRUE,
  ADD COLUMN skip_opposite_side_exposure BOOLEAN NOT NULL DEFAULT TRUE,
  ADD COLUMN require_full_liquidity BOOLEAN NOT NULL DEFAULT TRUE;

ALTER TABLE worm_execution_run_steps
  ADD COLUMN advisory_codes TEXT[] NOT NULL DEFAULT '{}';

-- +goose Down

ALTER TABLE worm_execution_run_steps
  DROP COLUMN IF EXISTS advisory_codes;

ALTER TABLE worm_execution_runs
  DROP COLUMN IF EXISTS require_full_liquidity,
  DROP COLUMN IF EXISTS skip_opposite_side_exposure,
  DROP COLUMN IF EXISTS skip_in_flight_request,
  DROP COLUMN IF EXISTS skip_already_held;

ALTER TABLE worm_execution_plan_steps
  DROP COLUMN IF EXISTS advisory_codes;

ALTER TABLE worm_execution_plans
  DROP COLUMN IF EXISTS require_full_liquidity,
  DROP COLUMN IF EXISTS skip_opposite_side_exposure,
  DROP COLUMN IF EXISTS skip_in_flight_request,
  DROP COLUMN IF EXISTS skip_already_held;
