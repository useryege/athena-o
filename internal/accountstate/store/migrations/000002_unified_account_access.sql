-- +goose Up

ALTER TABLE account_enabled_override RENAME TO account_access_override;

ALTER TABLE account_access_override RENAME COLUMN enabled TO login_enabled;

ALTER TABLE account_access_override
  ADD COLUMN data_access TEXT NOT NULL DEFAULT 'none',
  ADD COLUMN revision BIGINT NOT NULL DEFAULT 1;

ALTER TABLE account_access_override
  ADD CONSTRAINT account_access_override_data_access_check
    CHECK (data_access IN ('none', 'read', 'read_write')),
  ADD CONSTRAINT account_access_override_revision_check
    CHECK (revision > 0);

-- +goose Down

ALTER TABLE account_access_override
  DROP CONSTRAINT IF EXISTS account_access_override_revision_check,
  DROP CONSTRAINT IF EXISTS account_access_override_data_access_check,
  DROP COLUMN IF EXISTS revision,
  DROP COLUMN IF EXISTS data_access;

ALTER TABLE account_access_override RENAME COLUMN login_enabled TO enabled;

ALTER TABLE account_access_override RENAME TO account_enabled_override;
