-- +goose Up

CREATE TABLE IF NOT EXISTS worm_poly_fifa_event_config (
  singleton BOOLEAN PRIMARY KEY DEFAULT true,
  worm_event_id TEXT NOT NULL,
  event_ref TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  CONSTRAINT worm_poly_fifa_event_config_singleton CHECK (singleton),
  CONSTRAINT worm_poly_fifa_event_config_worm_event_id_not_empty CHECK (btrim(worm_event_id) <> ''),
  CONSTRAINT worm_poly_fifa_event_config_event_ref_not_empty CHECK (btrim(event_ref) <> '')
);

INSERT INTO worm_poly_fifa_event_config (singleton, worm_event_id, event_ref)
VALUES (
  true,
  '87UM8qJ3BwL9ZJA4HvqtcTLgMtipBxgwPU29V3LWkD89',
  'fifwc-ecu-kor-2026-06-20'
)
ON CONFLICT (singleton) DO NOTHING;

-- +goose Down

DROP TABLE IF EXISTS worm_poly_fifa_event_config;
