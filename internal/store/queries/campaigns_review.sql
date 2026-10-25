-- name: ComposerVersionsPage :many
-- One page of the version history, newest first: versions older than
-- @before (none: from the newest), only named ones when @named_only.
SELECT v.version, v.saved_at, v.label, v.source, v.size_bytes, v.name, coalesce(u.name, '')::text AS saved_by_name
FROM campaign_versions v
LEFT JOIN users u ON u.id = v.saved_by
WHERE v.campaign_id = @campaign_id
  AND (sqlc.narg(before)::int IS NULL OR v.version < sqlc.narg(before)::int)
  AND (NOT @named_only::bool OR v.name <> '')
ORDER BY v.version DESC
LIMIT @lim;

-- name: ComposerNamedVersionCount :one
SELECT count(*)::bigint FROM campaign_versions WHERE campaign_id = @campaign_id AND name <> '';

-- name: ComposerNameVersion :one
-- Names (or unnames, with '') one version; returns the name it had.
UPDATE campaign_versions v SET name = @name
FROM (SELECT x.name AS old_name FROM campaign_versions x WHERE x.campaign_id = @campaign_id AND x.version = @version FOR UPDATE) o
WHERE v.campaign_id = @campaign_id AND v.version = @version
RETURNING o.old_name::text AS old_name;

-- name: ComposerBumpVersion :exec
-- The current version number moves on without the content changing
-- ("Load theirs" keeps your edits in the history underneath).
UPDATE campaigns
SET version = @version, updated_by = @updated_by, updated_at = @now
WHERE workspace_id = @workspace_id AND id = @id;
