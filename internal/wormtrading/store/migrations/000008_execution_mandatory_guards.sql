-- +goose Up

-- The execution preview/run contract changed from a configurable liquidity
-- policy to mandatory exposure guards with partial fills always allowed. Runs
-- and previews are development-only data, so reset the complete execution
-- graph before removing the obsolete snapshot fields.
TRUNCATE TABLE worm_execution_plans CASCADE;

ALTER TABLE worm_execution_run_steps
  DROP COLUMN advisory_codes;

ALTER TABLE worm_execution_runs
  DROP COLUMN require_full_liquidity;

ALTER TABLE worm_execution_plan_steps
  DROP COLUMN advisory_codes;

ALTER TABLE worm_execution_plans
  DROP COLUMN require_full_liquidity;

-- +goose Down

ALTER TABLE worm_execution_plans
  ADD COLUMN require_full_liquidity BOOLEAN NOT NULL DEFAULT TRUE;

ALTER TABLE worm_execution_plan_steps
  ADD COLUMN advisory_codes TEXT[] NOT NULL DEFAULT '{}';

ALTER TABLE worm_execution_runs
  ADD COLUMN require_full_liquidity BOOLEAN NOT NULL DEFAULT TRUE;

ALTER TABLE worm_execution_run_steps
  ADD COLUMN advisory_codes TEXT[] NOT NULL DEFAULT '{}';
