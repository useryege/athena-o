-- +goose Up

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
  CONSTRAINT account_profile_name_check CHECK (account_name <> ''),
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
  CONSTRAINT account_preferences_name_check CHECK (account_name <> ''),
  CONSTRAINT account_preferences_theme_check CHECK (theme IN ('system', 'light', 'dark')),
  CONSTRAINT account_preferences_revision_check CHECK (revision > 0)
);

-- +goose Down

DROP TABLE account_preferences;
DROP TABLE account_profile;
