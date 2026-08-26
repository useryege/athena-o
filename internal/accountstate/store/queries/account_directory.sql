-- name: ListAccountRecords :many
SELECT account_name,
       google_subject,
       verified_email,
       administrator,
       created_at,
       updated_at,
       last_login_at
FROM athena_account
ORDER BY account_name;

-- name: GetAccountRecord :one
SELECT account_name,
       google_subject,
       verified_email,
       administrator,
       created_at,
       updated_at,
       last_login_at
FROM athena_account
WHERE account_name = sqlc.arg(account_name)::text;

-- name: GetAccountByGoogleSubject :one
SELECT account_name,
       google_subject,
       verified_email,
       administrator,
       created_at,
       updated_at,
       last_login_at
FROM athena_account
WHERE google_subject = sqlc.arg(google_subject)::text;

-- name: AccountExists :one
SELECT EXISTS (
  SELECT 1
  FROM athena_account
  WHERE account_name = sqlc.arg(account_name)::text
);

-- name: CreateOrdinaryAccount :one
WITH inserted_account AS (
  INSERT INTO athena_account (
    account_name,
    google_subject,
    verified_email,
    administrator
  )
  VALUES (
    'user-' || gen_random_uuid()::text,
    sqlc.arg(google_subject)::text,
    sqlc.arg(verified_email)::text,
    FALSE
  )
  ON CONFLICT (google_subject) DO NOTHING
  RETURNING account_name,
            google_subject,
            verified_email,
            administrator,
            created_at,
            updated_at,
            last_login_at
), inserted_access AS (
  INSERT INTO account_access (
    account_name,
    login_enabled,
    api_key_enabled,
    profit_sharing_enabled,
    revision
  )
  SELECT account_name, TRUE, FALSE, FALSE, 1
  FROM inserted_account
  RETURNING account_name
), inserted_modules AS (
  INSERT INTO account_module_access (account_name, module, access_level)
  SELECT inserted_access.account_name, module.name, 'none'
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
  RETURNING account_name
), inserted_profile AS (
  INSERT INTO account_profile (
    account_name,
    display_name,
    account_tier,
    revision
  )
  SELECT account_name, left(sqlc.arg(verified_email)::text, 80), 'standard', 1
  FROM inserted_account
  RETURNING account_name
)
SELECT account_name,
       google_subject,
       verified_email,
       administrator,
       created_at,
       updated_at,
       last_login_at
FROM inserted_account
WHERE EXISTS (SELECT 1 FROM inserted_access)
  AND (SELECT COUNT(*) FROM inserted_modules) = 10
  AND EXISTS (SELECT 1 FROM inserted_profile);

-- name: GetAdministratorForUpdate :one
SELECT account_name,
       google_subject,
       verified_email,
       administrator,
       created_at,
       updated_at,
       last_login_at
FROM athena_account
WHERE account_name = 'admin'
FOR UPDATE;

-- name: ClaimAdministratorIdentity :one
UPDATE athena_account
SET google_subject = sqlc.arg(google_subject)::text,
    verified_email = sqlc.arg(verified_email)::text,
    updated_at = NOW()
WHERE account_name = 'admin'
  AND administrator
  AND google_subject IS NULL
RETURNING account_name,
          google_subject,
          verified_email,
          administrator,
          created_at,
          updated_at,
          last_login_at;

-- name: RecordAccountLogin :one
UPDATE athena_account
SET verified_email = sqlc.arg(verified_email)::text,
    last_login_at = NOW(),
    updated_at = NOW()
WHERE account_name = sqlc.arg(account_name)::text
  AND google_subject = sqlc.arg(google_subject)::text
  AND EXISTS (
    SELECT 1
    FROM account_access
    WHERE account_access.account_name = athena_account.account_name
      AND account_access.login_enabled
  )
RETURNING account_name,
          google_subject,
          verified_email,
          administrator,
          created_at,
          updated_at,
          last_login_at;

-- name: CountAccountDirectory :one
WITH directory AS (
  SELECT account.account_name,
         account.verified_email,
         account.administrator,
         access.login_enabled,
         access.profit_sharing_enabled,
         COALESCE(profile.display_name, account.account_name) AS display_name,
         CASE
           WHEN NOT access.login_enabled THEN 'blocked'
           WHEN access.profit_sharing_enabled OR EXISTS (
             SELECT 1
             FROM account_module_access AS module_access
             WHERE module_access.account_name = account.account_name
               AND module_access.access_level <> 'none'
           ) THEN 'active'
           ELSE 'pending'
         END::text AS status
  FROM athena_account AS account
  JOIN account_access AS access USING (account_name)
  LEFT JOIN account_profile AS profile USING (account_name)
)
SELECT COUNT(*)
FROM directory
WHERE (
    sqlc.arg(search_query)::text = ''
    OR position(lower(sqlc.arg(search_query)::text) IN lower(account_name)) > 0
    OR position(lower(sqlc.arg(search_query)::text) IN lower(verified_email)) > 0
    OR position(lower(sqlc.arg(search_query)::text) IN lower(display_name)) > 0
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
  SELECT account.account_name,
         account.verified_email,
         account.administrator,
         account.created_at,
         account.last_login_at,
         access.login_enabled,
         access.api_key_enabled,
         access.profit_sharing_enabled,
         access.revision,
         COALESCE(profile.display_name, account.account_name) AS display_name,
         CASE
           WHEN NOT access.login_enabled THEN 'blocked'
           WHEN access.profit_sharing_enabled OR EXISTS (
             SELECT 1
             FROM account_module_access AS module_access
             WHERE module_access.account_name = account.account_name
               AND module_access.access_level <> 'none'
           ) THEN 'active'
           ELSE 'pending'
         END::text AS status
  FROM athena_account AS account
  JOIN account_access AS access USING (account_name)
  LEFT JOIN account_profile AS profile USING (account_name)
)
SELECT account_name,
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
    OR position(lower(sqlc.arg(search_query)::text) IN lower(account_name)) > 0
    OR position(lower(sqlc.arg(search_query)::text) IN lower(verified_email)) > 0
    OR position(lower(sqlc.arg(search_query)::text) IN lower(display_name)) > 0
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
         account_name
LIMIT sqlc.arg(limit_count)::integer
OFFSET sqlc.arg(offset_count)::integer;
