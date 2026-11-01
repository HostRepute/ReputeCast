-- +goose Up
-- Review 2026-11 AT-15: how long a watched segment's last evaluation took,
-- so a slow one is checked less often and after the fast ones.
ALTER TABLE segments ADD COLUMN watch_ms integer;

-- +goose Down
ALTER TABLE segments DROP COLUMN watch_ms;
