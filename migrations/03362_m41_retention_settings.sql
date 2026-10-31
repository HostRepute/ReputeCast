-- +goose Up
-- Review 2026-11 DP-4 and DP-6 (decision D3), Settings › General › Data
-- retention: double opt-in signups never confirmed are erased after this
-- many days (0: never), and the IP address and user agent of opens and
-- clicks in campaigns and automations are cleared after this many days
-- (0: kept). Counts, scores and reports are untouched.
ALTER TABLE workspaces
    ADD COLUMN unconfirmed_erase_days integer NOT NULL DEFAULT 30 CHECK (unconfirmed_erase_days IN (0, 14, 30, 60, 90)),
    ADD COLUMN event_detail_days      integer NOT NULL DEFAULT 365 CHECK (event_detail_days IN (0, 30, 90, 180, 365, 730));

-- +goose Down
ALTER TABLE workspaces DROP COLUMN unconfirmed_erase_days, DROP COLUMN event_detail_days;
