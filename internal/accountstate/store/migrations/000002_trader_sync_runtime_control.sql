-- +goose Up
CREATE TABLE trader_sync_runtime_control (
    singleton BOOLEAN PRIMARY KEY DEFAULT true CHECK (singleton),
    owner_id UUID,
    generation BIGINT NOT NULL DEFAULT 0 CHECK (generation >= 0)
);
INSERT INTO trader_sync_runtime_control(singleton) VALUES(true);

-- +goose Down
DROP TABLE trader_sync_runtime_control;
