-- +goose Up

-- +goose StatementBegin
CREATE FUNCTION athena_is_canonical_solana_public_key(value TEXT)
RETURNS BOOLEAN
LANGUAGE plpgsql
IMMUTABLE
STRICT
PARALLEL SAFE
AS $$
DECLARE
  alphabet CONSTANT TEXT := '123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz';
  encoded_length INTEGER := char_length(value);
  leading_zero_bytes INTEGER := 0;
  accumulator NUMERIC := 0;
  remaining NUMERIC;
  decoded_nonzero_bytes INTEGER := 0;
  position_index INTEGER;
  digit INTEGER;
BEGIN
  IF encoded_length < 32 OR encoded_length > 44 THEN
    RETURN FALSE;
  END IF;

  FOR position_index IN 1..encoded_length LOOP
    digit := strpos(alphabet, substr(value, position_index, 1)) - 1;
    IF digit < 0 THEN
      RETURN FALSE;
    END IF;
    IF position_index = leading_zero_bytes + 1 AND digit = 0 THEN
      leading_zero_bytes := leading_zero_bytes + 1;
    END IF;
    accumulator := accumulator * 58 + digit;
  END LOOP;

  remaining := accumulator;
  WHILE remaining > 0 LOOP
    decoded_nonzero_bytes := decoded_nonzero_bytes + 1;
    remaining := trunc(remaining / 256);
  END LOOP;

  RETURN leading_zero_bytes + decoded_nonzero_bytes = 32;
END;
$$;
-- +goose StatementEnd

CREATE TABLE athena_account (
  account_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  username TEXT NOT NULL,
  identity_provider TEXT NOT NULL,
  identity_subject TEXT,
  verified_email TEXT NOT NULL DEFAULT '',
  administrator BOOLEAN NOT NULL DEFAULT FALSE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  last_login_at TIMESTAMPTZ,
  CONSTRAINT athena_account_username_check CHECK (
    char_length(username) BETWEEN 3 AND 42
    AND username ~ '^[A-Za-z0-9.-]+$'
    AND username ~ '[A-Za-z0-9]'
    AND username !~* '^0x[0-9a-f]{40}$'
  ),
  CONSTRAINT athena_account_admin_username_check CHECK (
    lower(username) <> 'admin'
    OR (administrator AND username = 'admin')
  ),
  CONSTRAINT athena_account_identity_provider_check CHECK (
    identity_provider IN ('google', 'solana_wallet', 'development')
  ),
  CONSTRAINT athena_account_identity_subject_check CHECK (
    identity_subject IS NULL
    OR (
      identity_subject = btrim(identity_subject)
      AND identity_subject <> ''
      AND identity_subject !~ '[[:cntrl:]]'
    )
  ),
  CONSTRAINT athena_account_verified_email_check CHECK (
    verified_email = ''
    OR (
      verified_email = btrim(verified_email)
      AND verified_email !~ '[[:cntrl:]]'
    )
  ),
  CONSTRAINT athena_account_identity_binding_check CHECK (
    (
      identity_provider = 'google'
      AND identity_subject IS NOT NULL
      AND verified_email <> ''
    )
    OR (
      identity_provider = 'solana_wallet'
      AND identity_subject IS NOT NULL
      AND athena_is_canonical_solana_public_key(identity_subject)
      AND verified_email = ''
      AND NOT administrator
    )
    OR (
      identity_provider = 'development'
      AND identity_subject IS NULL
      AND verified_email = ''
      AND administrator
      AND username = 'local-admin'
    )
  ),
  CONSTRAINT athena_account_administrator_provider_check CHECK (
    NOT administrator OR identity_provider IN ('google', 'development')
  ),
  CONSTRAINT athena_account_last_login_check CHECK (
    last_login_at IS NULL OR last_login_at >= created_at
  )
);

CREATE UNIQUE INDEX athena_account_username_lower_uidx
  ON athena_account (lower(username));

CREATE UNIQUE INDEX athena_account_identity_uidx
  ON athena_account (identity_provider, identity_subject)
  WHERE identity_subject IS NOT NULL;

CREATE UNIQUE INDEX athena_account_single_administrator_uidx
  ON athena_account (administrator)
  WHERE administrator;

CREATE TABLE account_access (
  account_id UUID PRIMARY KEY,
  login_enabled BOOLEAN NOT NULL,
  api_key_enabled BOOLEAN NOT NULL,
  profit_sharing_enabled BOOLEAN NOT NULL,
  revision BIGINT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT account_access_account_fk
    FOREIGN KEY (account_id)
    REFERENCES athena_account (account_id)
    ON DELETE CASCADE,
  CONSTRAINT account_access_revision_check CHECK (revision > 0)
);

CREATE TABLE account_module_access (
  account_id UUID NOT NULL,
  module TEXT NOT NULL,
  access_level TEXT NOT NULL,
  CONSTRAINT account_module_access_pk
    PRIMARY KEY (account_id, module),
  CONSTRAINT account_module_access_account_fk
    FOREIGN KEY (account_id)
    REFERENCES account_access (account_id)
    ON DELETE CASCADE,
  CONSTRAINT account_module_access_module_check
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
  CONSTRAINT account_module_access_level_check
    CHECK (access_level IN ('none', 'read', 'read_write')),
  CONSTRAINT account_module_access_max_level_check CHECK (
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

CREATE TABLE account_profile (
  account_id UUID PRIMARY KEY,
  display_name TEXT NOT NULL,
  account_tier TEXT NOT NULL DEFAULT 'standard',
  avatar_object_key TEXT NOT NULL DEFAULT '',
  avatar_content_type TEXT NOT NULL DEFAULT '',
  avatar_etag TEXT NOT NULL DEFAULT '',
  avatar_size_bytes BIGINT NOT NULL DEFAULT 0,
  revision BIGINT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT account_profile_account_fk
    FOREIGN KEY (account_id)
    REFERENCES athena_account (account_id)
    ON DELETE CASCADE,
  CONSTRAINT account_profile_display_name_check CHECK (
    char_length(display_name) BETWEEN 1 AND 80
    AND display_name !~ '[[:cntrl:]]'
  ),
  CONSTRAINT account_profile_tier_check CHECK (account_tier IN ('standard', 'pro')),
  CONSTRAINT account_profile_avatar_check CHECK (
    (
      avatar_object_key = ''
      AND avatar_content_type = ''
      AND avatar_etag = ''
      AND avatar_size_bytes = 0
    )
    OR (
      avatar_object_key <> ''
      AND avatar_content_type IN ('image/jpeg', 'image/png', 'image/webp')
      AND avatar_etag <> ''
      AND avatar_size_bytes > 0
    )
  ),
  CONSTRAINT account_profile_revision_check CHECK (revision > 0)
);

CREATE TABLE account_preferences (
  account_id UUID PRIMARY KEY,
  theme TEXT NOT NULL,
  revision BIGINT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT account_preferences_account_fk
    FOREIGN KEY (account_id)
    REFERENCES athena_account (account_id)
    ON DELETE CASCADE,
  CONSTRAINT account_preferences_theme_check CHECK (theme IN ('system', 'light', 'dark')),
  CONSTRAINT account_preferences_revision_check CHECK (revision > 0)
);

CREATE TABLE account_api_key (
  account_id UUID NOT NULL,
  display_id TEXT NOT NULL,
  jti TEXT NOT NULL UNIQUE,
  issued_at TIMESTAMPTZ NOT NULL,
  expires_at TIMESTAMPTZ,
  CONSTRAINT account_api_key_pk PRIMARY KEY (account_id, display_id),
  CONSTRAINT account_api_key_account_fk
    FOREIGN KEY (account_id)
    REFERENCES athena_account (account_id)
    ON DELETE CASCADE,
  CONSTRAINT account_api_key_display_id_check CHECK (
    display_id ~ '^[A-Za-z0-9][A-Za-z0-9._-]{0,63}$'
  ),
  CONSTRAINT account_api_key_jti_check CHECK (
    jti <> '' AND jti !~ '[[:cntrl:]]'
  ),
  CONSTRAINT account_api_key_expiry_check CHECK (
    expires_at IS NULL OR expires_at > issued_at
  )
);

CREATE INDEX account_api_key_account_issued_idx
  ON account_api_key (account_id, issued_at DESC, display_id);

CREATE INDEX athena_account_recent_login_idx
  ON athena_account (last_login_at DESC NULLS LAST, account_id);

-- Normal authentication starts with no account rows. External registration creates
-- a complete account aggregate, while disabled-auth development explicitly
-- creates its isolated local administrator through the account-state store.

-- +goose Down

DROP TABLE account_api_key;
DROP TABLE account_preferences;
DROP TABLE account_profile;
DROP TABLE account_module_access;
DROP TABLE account_access;
DROP TABLE athena_account;
DROP FUNCTION athena_is_canonical_solana_public_key(TEXT);
