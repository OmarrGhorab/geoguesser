-- +goose Up
ALTER TABLE locations ADD COLUMN heading INT;

-- +goose Down
ALTER TABLE locations DROP COLUMN heading;
