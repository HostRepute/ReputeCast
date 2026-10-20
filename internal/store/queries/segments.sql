-- name: ListSegments :many
SELECT s.id, s.name, s.description, s.mode, s.refresh, s.cached_count, s.counted_at, s.count_ms, s.tier_counts,
       s.created_at, s.updated_at
FROM segments s
WHERE s.workspace_id = @workspace_id
ORDER BY s.created_at, s.id;

-- name: CountWorkspaceSegments :one
SELECT count(*)::bigint FROM segments WHERE workspace_id = @workspace_id;

-- name: GetSegment :one
SELECT * FROM segments WHERE workspace_id = @workspace_id AND id = @id;

-- name: CreateSegment :one
INSERT INTO segments (id, workspace_id, name, description, mode, refresh, definition, sql, created_by, updated_by)
VALUES (@id, @workspace_id, @name, @description, @mode, @refresh, @definition, @sql, @created_by, @created_by)
RETURNING *;

-- name: UpdateSegment :one
UPDATE segments
SET name = @name, description = @description, mode = @mode, refresh = @refresh, definition = @definition, sql = @sql,
    updated_by = @updated_by, updated_at = now()
WHERE workspace_id = @workspace_id AND id = @id
RETURNING *;

-- name: DeleteSegment :execrows
DELETE FROM segments WHERE workspace_id = @workspace_id AND id = @id;

-- name: SetSegmentCount :exec
UPDATE segments
SET cached_count = @cached_count, counted_at = @counted_at, count_ms = @count_ms, tier_counts = @tier_counts
WHERE id = @id;

-- name: ClearSegmentMembers :exec
DELETE FROM segment_members WHERE segment_id = @segment_id;

-- name: AddSegmentMembers :exec
INSERT INTO segment_members (segment_id, contact_id)
SELECT @segment_id, unnest(@contact_ids::uuid[])
ON CONFLICT DO NOTHING;

-- name: SegmentMemberIDs :many
SELECT sm.contact_id FROM segment_members sm
JOIN contacts c ON c.id = sm.contact_id AND c.erased_at IS NULL
WHERE sm.segment_id = @segment_id;

-- name: RecentSentCampaigns :many
-- Campaigns a segment can refer to ("Opened campaign …").
SELECT id, name, coalesce(started_at, created_at)::timestamptz AS at
FROM campaigns
WHERE workspace_id = @workspace_id AND deleted_at IS NULL AND status IN ('sent', 'sending', 'paused') AND channel = 'email'
ORDER BY coalesce(started_at, created_at) DESC
LIMIT 50;

-- name: ContactSources :many
SELECT source, count(*)::bigint AS n
FROM contacts
WHERE workspace_id = @workspace_id AND erased_at IS NULL AND source <> ''
GROUP BY source
ORDER BY n DESC, source
LIMIT 40;
