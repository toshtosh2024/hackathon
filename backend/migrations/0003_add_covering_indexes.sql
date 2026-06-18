-- +goose Up
-- Indexes are managed programmatically inside main.go's ensureIndex to prevent Duplicate Key errors (Error 1061) on MySQL
SELECT 1;

-- +goose Down
SELECT 1;
