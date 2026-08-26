-- +goose Up

CREATE TABLE athena_account (
  account_name TEXT PRIMARY KEY,
  google_subject TEXT UNIQUE,
  verified_email TEXT NOT NULL DEFAULT '',
  administrator BOOLEAN NOT NULL DEFAULT FALSE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  last_login_at TIMESTAMPTZ,
  CONSTRAINT athena_account_name_check CHECK (
    account_name = 'admin'
    OR account_name ~ '^user-[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$'
  ),
  CONSTRAINT athena_account_name_colon_check CHECK (position(':' IN account_name) = 0),
  CONSTRAINT athena_account_administrator_check CHECK (
    administrator = (account_name = 'admin')
  ),
  CONSTRAINT athena_account_google_subject_check CHECK (
    google_subject IS NULL
    OR (
      google_subject = btrim(google_subject)
      AND google_subject <> ''
      AND google_subject !~ '[[:cntrl:]]'
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
    (google_subject IS NULL AND verified_email = '' AND administrator)
    OR (google_subject IS NOT NULL AND verified_email <> '')
  ),
  CONSTRAINT athena_account_last_login_check CHECK (
    last_login_at IS NULL OR last_login_at >= created_at
  )
);

CREATE TABLE account_access (
  account_name TEXT PRIMARY KEY,
  login_enabled BOOLEAN NOT NULL,
  api_key_enabled BOOLEAN NOT NULL,
  profit_sharing_enabled BOOLEAN NOT NULL,
  revision BIGINT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT account_access_account_fk
    FOREIGN KEY (account_name)
    REFERENCES athena_account (account_name)
    ON DELETE CASCADE,
  CONSTRAINT account_access_revision_check CHECK (revision > 0)
);

CREATE TABLE account_module_access (
  account_name TEXT NOT NULL,
  module TEXT NOT NULL,
  access_level TEXT NOT NULL,
  CONSTRAINT account_module_access_pk
    PRIMARY KEY (account_name, module),
  CONSTRAINT account_module_access_account_fk
    FOREIGN KEY (account_name)
    REFERENCES account_access (account_name)
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
  account_name TEXT PRIMARY KEY,
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
    FOREIGN KEY (account_name)
    REFERENCES athena_account (account_name)
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
  account_name TEXT PRIMARY KEY,
  theme TEXT NOT NULL,
  revision BIGINT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT account_preferences_account_fk
    FOREIGN KEY (account_name)
    REFERENCES athena_account (account_name)
    ON DELETE CASCADE,
  CONSTRAINT account_preferences_theme_check CHECK (theme IN ('system', 'light', 'dark')),
  CONSTRAINT account_preferences_revision_check CHECK (revision > 0)
);

CREATE TABLE account_api_key (
  account_name TEXT NOT NULL,
  display_id TEXT NOT NULL,
  jti TEXT NOT NULL UNIQUE,
  issued_at TIMESTAMPTZ NOT NULL,
  expires_at TIMESTAMPTZ,
  CONSTRAINT account_api_key_pk PRIMARY KEY (account_name, display_id),
  CONSTRAINT account_api_key_account_fk
    FOREIGN KEY (account_name)
    REFERENCES athena_account (account_name)
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
  ON account_api_key (account_name, issued_at DESC, display_id);

CREATE INDEX athena_account_recent_login_idx
  ON athena_account (last_login_at DESC NULLS LAST, account_name);

INSERT INTO athena_account (
  account_name,
  google_subject,
  verified_email,
  administrator
)
VALUES ('admin', NULL, '', TRUE);

INSERT INTO account_access (
  account_name,
  login_enabled,
  api_key_enabled,
  profit_sharing_enabled,
  revision
)
VALUES ('admin', TRUE, FALSE, TRUE, 1);

INSERT INTO account_module_access (account_name, module, access_level)
VALUES
  ('admin', 'market_radar', 'read'),
  ('admin', 'sports_live', 'read'),
  ('admin', 'sports_history', 'read_write'),
  ('admin', 'managed_oo', 'read_write'),
  ('admin', 'worm_markets', 'read'),
  ('admin', 'fifa_market_dashboard', 'read_write'),
  ('admin', 'world_cup_corners', 'read'),
  ('admin', 'token', 'read_write'),
  ('admin', 'wallet', 'read_write'),
  ('admin', 'notifications', 'read_write');

INSERT INTO account_profile (
  account_name,
  display_name,
  account_tier,
  revision
)
VALUES ('admin', 'admin', 'standard', 1);

-- +goose Down

DROP TABLE account_api_key;
DROP TABLE account_preferences;
DROP TABLE account_profile;
DROP TABLE account_module_access;
DROP TABLE account_access;
DROP TABLE athena_account;
