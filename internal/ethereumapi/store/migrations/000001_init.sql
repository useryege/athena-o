-- +goose Up

-- Reserve the initial migration version for the ethereumapi service.
SELECT 1;

-- +goose Down

SELECT 1;
