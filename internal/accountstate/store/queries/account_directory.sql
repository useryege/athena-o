-- name: ListAccountRecords :many
SELECT account_id,
       username,
       identity_provider,
       identity_subject,
       verified_email,
       administrator,
       created_at,
       updated_at,
       last_login_at
FROM athena_account
ORDER BY account_id;

-- name: GetAccountRecord :one
SELECT account_id,
       username,
       identity_provider,
       identity_subject,
       verified_email,
       administrator,
       created_at,
       updated_at,
       last_login_at
FROM athena_account
WHERE account_id = sqlc.arg(account_id)::uuid;

-- name: GetAccountByIdentity :one
SELECT account_id,
       username,
       identity_provider,
       identity_subject,
       verified_email,
       administrator,
       created_at,
       updated_at,
       last_login_at
FROM athena_account
WHERE identity_provider = sqlc.arg(identity_provider)::text
  AND identity_subject = sqlc.arg(identity_subject)::text;

-- name: GetDevelopmentAdministrator :one
SELECT account_id,
       username,
       identity_provider,
       identity_subject,
       verified_email,
       administrator,
       created_at,
       updated_at,
       last_login_at
FROM athena_account
WHERE identity_provider = 'development'
  AND username = 'local-admin'
  AND administrator;

-- name: UsernameExists :one
SELECT EXISTS (
  SELECT 1
  FROM athena_account
  WHERE lower(username) = lower(sqlc.arg(username)::text)
);

-- name: AccountExists :one
SELECT EXISTS (
  SELECT 1
  FROM athena_account
  WHERE account_id = sqlc.arg(account_id)::uuid
);

-- name: CreateOrdinaryAccount :one
WITH inserted_account AS (
  INSERT INTO athena_account (
    username,
    identity_provider,
    identity_subject,
    verified_email,
    administrator
  )
  VALUES (
    sqlc.arg(username)::text,
    sqlc.arg(identity_provider)::text,
    sqlc.arg(identity_subject)::text,
    sqlc.arg(verified_email)::text,
    FALSE
  )
  ON CONFLICT (identity_provider, identity_subject) WHERE identity_subject IS NOT NULL DO NOTHING
  RETURNING account_id,
            username,
            identity_provider,
            identity_subject,
            verified_email,
            administrator,
            created_at,
            updated_at,
            last_login_at
), inserted_access AS (
  INSERT INTO account_access (
    account_id,
    login_enabled,
    api_key_enabled,
    profit_sharing_enabled,
    revision
  )
  SELECT account_id, TRUE, FALSE, FALSE, 1
  FROM inserted_account
  RETURNING account_id
), inserted_modules AS (
  INSERT INTO account_module_access (account_id, module, access_level)
  SELECT inserted_access.account_id, module.name, 'none'
  FROM inserted_access
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
  ) AS module(name)
  RETURNING account_id
), inserted_profile AS (
  INSERT INTO account_profile (
    account_id,
    display_name,
    account_tier,
    revision
  )
  SELECT account_id, username, 'standard', 1
  FROM inserted_account
  RETURNING account_id
), inserted_preferences AS (
  INSERT INTO account_preferences (account_id, theme, revision)
  SELECT account_id, 'system', 1
  FROM inserted_account
  RETURNING account_id
)
SELECT account_id,
       username,
       identity_provider,
       identity_subject,
       verified_email,
       administrator,
       created_at,
       updated_at,
       last_login_at
FROM inserted_account
WHERE EXISTS (SELECT 1 FROM inserted_access)
  AND (SELECT COUNT(*) FROM inserted_modules) = 10
  AND EXISTS (SELECT 1 FROM inserted_profile)
  AND EXISTS (SELECT 1 FROM inserted_preferences);

-- name: CreateAdministratorAccount :one
WITH inserted_account AS (
  INSERT INTO athena_account (
    username,
    identity_provider,
    identity_subject,
    verified_email,
    administrator
  )
  VALUES (
    sqlc.arg(username)::text,
    sqlc.arg(identity_provider)::text,
    sqlc.arg(identity_subject)::text,
    sqlc.arg(verified_email)::text,
    TRUE
  )
  ON CONFLICT (identity_provider, identity_subject) WHERE identity_subject IS NOT NULL DO NOTHING
  RETURNING account_id,
            username,
            identity_provider,
            identity_subject,
            verified_email,
            administrator,
            created_at,
            updated_at,
            last_login_at
), inserted_access AS (
  INSERT INTO account_access (
    account_id,
    login_enabled,
    api_key_enabled,
    profit_sharing_enabled,
    revision
  )
  SELECT account_id, TRUE, FALSE, TRUE, 1
  FROM inserted_account
  RETURNING account_id
), inserted_modules AS (
  INSERT INTO account_module_access (account_id, module, access_level)
  SELECT inserted_access.account_id, module.name, module.access_level
  FROM inserted_access
  CROSS JOIN (
    VALUES
      ('market_radar', 'read'),
      ('sports_live', 'read'),
      ('sports_history', 'read_write'),
      ('managed_oo', 'read_write'),
      ('worm_markets', 'read'),
      ('fifa_market_dashboard', 'read_write'),
      ('world_cup_corners', 'read'),
      ('token', 'read_write'),
      ('wallet', 'read_write'),
      ('notifications', 'read_write')
  ) AS module(name, access_level)
  RETURNING account_id
), inserted_profile AS (
  INSERT INTO account_profile (
    account_id,
    display_name,
    account_tier,
    revision
  )
  SELECT account_id, username, 'standard', 1
  FROM inserted_account
  RETURNING account_id
), inserted_preferences AS (
  INSERT INTO account_preferences (account_id, theme, revision)
  SELECT account_id, 'system', 1
  FROM inserted_account
  RETURNING account_id
)
SELECT account_id,
       username,
       identity_provider,
       identity_subject,
       verified_email,
       administrator,
       created_at,
       updated_at,
       last_login_at
FROM inserted_account
WHERE EXISTS (SELECT 1 FROM inserted_access)
  AND (SELECT COUNT(*) FROM inserted_modules) = 10
  AND EXISTS (SELECT 1 FROM inserted_profile)
  AND EXISTS (SELECT 1 FROM inserted_preferences);

-- name: CreateDevelopmentAdministrator :one
WITH inserted_account AS (
  INSERT INTO athena_account (
    username,
    identity_provider,
    identity_subject,
    verified_email,
    administrator
  )
  VALUES ('local-admin', 'development', NULL, '', TRUE)
  RETURNING account_id,
            username,
            identity_provider,
            identity_subject,
            verified_email,
            administrator,
            created_at,
            updated_at,
            last_login_at
), inserted_access AS (
  INSERT INTO account_access (
    account_id,
    login_enabled,
    api_key_enabled,
    profit_sharing_enabled,
    revision
  )
  SELECT account_id, TRUE, FALSE, TRUE, 1
  FROM inserted_account
  RETURNING account_id
), inserted_modules AS (
  INSERT INTO account_module_access (account_id, module, access_level)
  SELECT inserted_access.account_id, module.name, module.access_level
  FROM inserted_access
  CROSS JOIN (
    VALUES
      ('market_radar', 'read'),
      ('sports_live', 'read'),
      ('sports_history', 'read_write'),
      ('managed_oo', 'read_write'),
      ('worm_markets', 'read'),
      ('fifa_market_dashboard', 'read_write'),
      ('world_cup_corners', 'read'),
      ('token', 'read_write'),
      ('wallet', 'read_write'),
      ('notifications', 'read_write')
  ) AS module(name, access_level)
  RETURNING account_id
), inserted_profile AS (
  INSERT INTO account_profile (
    account_id,
    display_name,
    account_tier,
    revision
  )
  SELECT account_id, username, 'standard', 1
  FROM inserted_account
  RETURNING account_id
), inserted_preferences AS (
  INSERT INTO account_preferences (account_id, theme, revision)
  SELECT account_id, 'system', 1
  FROM inserted_account
  RETURNING account_id
)
SELECT account_id,
       username,
       identity_provider,
       identity_subject,
       verified_email,
       administrator,
       created_at,
       updated_at,
       last_login_at
FROM inserted_account
WHERE EXISTS (SELECT 1 FROM inserted_access)
  AND (SELECT COUNT(*) FROM inserted_modules) = 10
  AND EXISTS (SELECT 1 FROM inserted_profile)
  AND EXISTS (SELECT 1 FROM inserted_preferences);

-- name: RecordAccountLogin :one
UPDATE athena_account
SET verified_email = sqlc.arg(verified_email)::text,
    last_login_at = NOW(),
    updated_at = NOW()
WHERE account_id = sqlc.arg(account_id)::uuid
  AND identity_provider = sqlc.arg(identity_provider)::text
  AND identity_subject = sqlc.arg(identity_subject)::text
  AND EXISTS (
    SELECT 1
    FROM account_access
    WHERE account_access.account_id = athena_account.account_id
      AND account_access.login_enabled
  )
RETURNING account_id,
          username,
          identity_provider,
          identity_subject,
          verified_email,
          administrator,
          created_at,
          updated_at,
          last_login_at;

-- name: CountAccountDirectory :one
WITH directory AS (
  SELECT account.account_id,
         account.username,
         account.identity_provider,
         account.identity_subject,
         account.verified_email,
         account.administrator,
         access.login_enabled,
         access.profit_sharing_enabled,
         COALESCE(profile.display_name, account.username) AS display_name,
         CASE
           WHEN NOT access.login_enabled THEN 'blocked'
           WHEN access.profit_sharing_enabled OR EXISTS (
             SELECT 1
             FROM account_module_access AS module_access
             WHERE module_access.account_id = account.account_id
               AND module_access.access_level <> 'none'
           ) THEN 'active'
           ELSE 'pending'
         END::text AS status
  FROM athena_account AS account
  JOIN account_access AS access USING (account_id)
  LEFT JOIN account_profile AS profile USING (account_id)
)
SELECT COUNT(*)
FROM directory
WHERE (
    sqlc.arg(search_query)::text = ''
    OR position(lower(sqlc.arg(search_query)::text) IN lower(username)) > 0
    OR position(lower(sqlc.arg(search_query)::text) IN lower(verified_email)) > 0
    OR position(lower(sqlc.arg(search_query)::text) IN lower(display_name)) > 0
    OR (
      identity_provider = 'solana_wallet'
      AND position(sqlc.arg(search_query)::text IN identity_subject) > 0
    )
    OR lower(sqlc.arg(search_query)::text) = account_id::text
  )
  AND (
    sqlc.arg(status_filter)::text IN ('', 'all')
    OR status = sqlc.arg(status_filter)::text
  )
  AND (
    NOT sqlc.arg(profit_sharing_eligible_only)::boolean
    OR (
      login_enabled
      AND profit_sharing_enabled
      AND NOT administrator
    )
  );

-- name: ListAccountDirectoryPage :many
WITH directory AS (
  SELECT account.account_id,
         account.username,
         account.identity_provider,
         account.identity_subject,
         account.verified_email,
         account.administrator,
         account.created_at,
         account.last_login_at,
         access.login_enabled,
         access.api_key_enabled,
         access.profit_sharing_enabled,
         access.revision,
         COALESCE(profile.display_name, account.username) AS display_name,
         CASE
           WHEN NOT access.login_enabled THEN 'blocked'
           WHEN access.profit_sharing_enabled OR EXISTS (
             SELECT 1
             FROM account_module_access AS module_access
             WHERE module_access.account_id = account.account_id
               AND module_access.access_level <> 'none'
           ) THEN 'active'
           ELSE 'pending'
         END::text AS status
  FROM athena_account AS account
  JOIN account_access AS access USING (account_id)
  LEFT JOIN account_profile AS profile USING (account_id)
)
SELECT account_id,
       username,
       verified_email,
       administrator,
       created_at,
       last_login_at,
       login_enabled,
       api_key_enabled,
       profit_sharing_enabled,
       revision,
       display_name,
       status
FROM directory
WHERE (
    sqlc.arg(search_query)::text = ''
    OR position(lower(sqlc.arg(search_query)::text) IN lower(username)) > 0
    OR position(lower(sqlc.arg(search_query)::text) IN lower(verified_email)) > 0
    OR position(lower(sqlc.arg(search_query)::text) IN lower(display_name)) > 0
    OR (
      identity_provider = 'solana_wallet'
      AND position(sqlc.arg(search_query)::text IN identity_subject) > 0
    )
    OR lower(sqlc.arg(search_query)::text) = account_id::text
  )
  AND (
    sqlc.arg(status_filter)::text IN ('', 'all')
    OR status = sqlc.arg(status_filter)::text
  )
  AND (
    NOT sqlc.arg(profit_sharing_eligible_only)::boolean
    OR (
      login_enabled
      AND profit_sharing_enabled
      AND NOT administrator
    )
  )
ORDER BY CASE WHEN status = 'pending' THEN 0 ELSE 1 END,
         last_login_at DESC NULLS LAST,
         username,
         account_id
LIMIT sqlc.arg(limit_count)::integer
OFFSET sqlc.arg(offset_count)::integer;
