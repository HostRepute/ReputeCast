-- name: InsertMediaAsset :one
-- Uploading the same bytes twice returns the existing asset (and shows it again).
-- sha256 is the uploaded file's; data may be its resized, compressed copy,
-- and the original_* columns then say what was uploaded.
INSERT INTO media_assets (id, workspace_id, file_name, content_type, size_bytes, width, height, has_alpha, sha256, data, created_by, created_at,
                          original_size_bytes, original_width, original_height, original_content_type)
VALUES (@id, @workspace_id, @file_name, @content_type, @size_bytes, @width, @height, @has_alpha, @sha256, @data, sqlc.narg(created_by), @created_at,
        sqlc.narg(original_size_bytes), sqlc.narg(original_width), sqlc.narg(original_height), sqlc.narg(original_content_type))
ON CONFLICT (workspace_id, sha256) DO UPDATE SET hidden_at = NULL
RETURNING id, file_name, content_type, size_bytes, width, height, has_alpha, created_at, (xmax = 0) AS inserted,
          original_size_bytes, original_width, original_height, original_content_type;

-- name: ListMediaAssets :many
SELECT a.id, a.file_name, a.content_type, a.size_bytes, a.width, a.height, a.has_alpha, a.created_at,
       a.original_size_bytes, a.original_width, a.original_height, a.original_content_type,
       coalesce(u.name, '')::text AS created_by_name
FROM media_assets a
LEFT JOIN users u ON u.id = a.created_by
WHERE a.workspace_id = @workspace_id AND a.hidden_at IS NULL
ORDER BY a.created_at DESC
LIMIT @lim;

-- name: GetMediaAsset :one
SELECT id, workspace_id, file_name, content_type, size_bytes, width, height, has_alpha, created_at
FROM media_assets WHERE workspace_id = @workspace_id AND id = @id;

-- name: GetMediaAssetData :one
-- Public serving: by id only (ids are unguessable UUIDv7s embedded in emails).
SELECT id, file_name, content_type, size_bytes, sha256, data, created_at FROM media_assets WHERE id = @id;

-- name: HideMediaAsset :execrows
UPDATE media_assets SET hidden_at = @now WHERE workspace_id = @workspace_id AND id = @id AND hidden_at IS NULL;
