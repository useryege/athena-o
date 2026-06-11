-- +goose Up

ALTER TABLE worm_market
  ADD COLUMN price_alert_band TEXT NOT NULL DEFAULT 'none',
  ADD CONSTRAINT worm_market_price_alert_band_valid
    CHECK (price_alert_band IN ('none', 'a', 'b'));

-- +goose Down

ALTER TABLE worm_market
  DROP COLUMN price_alert_band;
