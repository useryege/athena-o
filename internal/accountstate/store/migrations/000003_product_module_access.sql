-- +goose Up

CREATE TABLE account_module_access_override (
  account_name TEXT NOT NULL,
  module TEXT NOT NULL,
  access_level TEXT NOT NULL,
  CONSTRAINT account_module_access_override_pk
    PRIMARY KEY (account_name, module),
  CONSTRAINT account_module_access_override_account_fk
    FOREIGN KEY (account_name)
    REFERENCES account_access_override (account_name)
    ON DELETE CASCADE,
  CONSTRAINT account_module_access_override_module_check
    CHECK (module IN (
      'market_radar',
      'sports_live',
      'sports_history',
      'managed_oo',
      'worm_markets',
      'fifa_market_dashboard',
      'world_cup_corners',
      'token',
      'wallet',
      'notifications'
    )),
  CONSTRAINT account_module_access_override_level_check
    CHECK (access_level IN ('none', 'read', 'read_write')),
  CONSTRAINT account_module_access_override_max_level_check
    CHECK (
      access_level <> 'read_write'
      OR module IN (
        'sports_history',
        'managed_oo',
        'fifa_market_dashboard',
        'token',
        'wallet',
        'notifications'
      )
    )
);

INSERT INTO account_module_access_override (account_name, module, access_level)
SELECT account.account_name, module.name, 'none'
FROM account_access_override AS account
CROSS JOIN (
  VALUES
    ('market_radar'),
    ('sports_live'),
    ('sports_history'),
    ('managed_oo'),
    ('worm_markets'),
    ('fifa_market_dashboard'),
    ('world_cup_corners'),
    ('token'),
    ('wallet'),
    ('notifications')
) AS module(name);

UPDATE account_access_override
SET revision = revision + 1,
    updated_at = NOW();

ALTER TABLE account_access_override
  DROP CONSTRAINT account_access_override_data_access_check,
  DROP COLUMN data_access;

-- +goose Down

ALTER TABLE account_access_override
  ADD COLUMN data_access TEXT NOT NULL DEFAULT 'none',
  ADD CONSTRAINT account_access_override_data_access_check
    CHECK (data_access IN ('none', 'read', 'read_write'));

DROP TABLE account_module_access_override;
