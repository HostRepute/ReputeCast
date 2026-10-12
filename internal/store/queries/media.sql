-- name: InsertMediaAsset :one
-- Uploading the same bytes twice returns the existing asset (and shows it again).
INSERT INTO media_assets (id, workspace_id, file_name, content_type, size_bytes, width, height, has_alpha, sha256, data, created_by, created_at)
VALUES (@id, @workspace_id, @file_name, @content_type, @size_bytes, @width, @height, @has_alpha, @sha256, @data, sqlc.narg(created_by), @created_at)
ON CONFLICT (workspace_id, sha256) DO UPDATE SET hidden_at = NULL
RETURNING id, file_name, content_type, size_bytes, width, height, has_alpha, created_at, (xmax = 0) AS inserted;

-- name: ListMediaAssets :many
SELECT a.id, a.file_name, a.content_type, a.size_bytes, a.width, a.height, a.has_alpha, a.created_at,
       coalesce(u.name, '')::text AS created_by_name
FROM media_assets a
LEFT JOIN users u ON u.id = a.created_by
WHERE a.workspace_id = @workspace_id AND a.hidden_at IS NULL
ORDER BY a.created_at DESC
LIMIT @lim;

-- name: GetMediaAsset :one
SELECT id, workspace_id, file_name, content_type, size_bytes, width, height, has_alpha, created_at
FROM media_assets WHERE workspace_id = @workspace_id AND id = @id;

-- name: GetMediaData :one
-- Public serving: by id only (ids are unguessable UUIDv7s embedded in emails).
SELECT id, file_name, content_type, size_bytes, sha256, data, created_at FROM media_assets WHERE id = @id;

-- name: HideMediaAsset :execrows
UPDATE media_assets SET hidden_at = @now WHERE workspace_id = @workspace_id AND id = @id AND hidden_at IS NULL;
