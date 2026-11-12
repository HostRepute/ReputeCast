-- Countdown timers in email (M59 G-21): one row per workspace and set of
-- settings, drawn as an animated GIF when the email is opened.

-- name: CountdownInsert :one
-- A new set of settings; no row when the workspace has it already.
INSERT INTO email_countdowns (id, workspace_id, hash, settings, ends_at, created_by, created_at)
VALUES (@id, @workspace_id, @hash, @settings, @ends_at, sqlc.narg(created_by), @created_at)
ON CONFLICT (workspace_id, hash) DO NOTHING
RETURNING id;

-- name: CountdownByHash :one
SELECT id, ends_at FROM email_countdowns WHERE workspace_id = @workspace_id AND hash = @hash;

-- name: CountdownGet :one
-- The image's row, for the workspace its token names.
SELECT id, workspace_id, hash, settings, ends_at FROM email_countdowns WHERE id = @id AND workspace_id = @workspace_id;

-- name: CountdownsByIDs :many
-- The countdowns an email shows (the composer's and Send Readiness's checks).
SELECT id, hash, settings, ends_at FROM email_countdowns WHERE workspace_id = @workspace_id AND id = ANY(@ids::uuid[]) ORDER BY ends_at, id;

-- name: CountdownServed :exec
-- When the image was last opened, at most once a minute (before is a
-- minute earlier than at).
UPDATE email_countdowns SET last_served_at = @at
WHERE id = @id AND (last_served_at IS NULL OR last_served_at < @before);

-- name: CountdownCampaignSchedule :one
-- When a campaign's send starts (Send Readiness's countdown rows).
SELECT schedule FROM campaigns WHERE id = @id AND workspace_id = @workspace_id;
