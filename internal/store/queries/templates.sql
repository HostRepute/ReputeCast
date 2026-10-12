-- name: ListTemplates :many
SELECT t.id, t.kind, t.name, t.description, t.thumb, t.size_bytes, t.source_campaign_id, t.created_at, t.updated_at,
       coalesce(u.name, '')::text AS updated_by_name
FROM templates t
LEFT JOIN users u ON u.id = coalesce(t.updated_by, t.created_by)
WHERE t.workspace_id = @workspace_id AND t.deleted_at IS NULL AND (@kind::text = '' OR t.kind = @kind)
ORDER BY t.updated_at DESC, t.name;

-- name: GetTemplate :one
SELECT t.id, t.workspace_id, t.kind, t.name, t.description, t.mjml, t.html, t.thumb, t.size_bytes, t.source_campaign_id,
       t.created_at, t.updated_at, coalesce(u.name, '')::text AS updated_by_name
FROM templates t
LEFT JOIN users u ON u.id = coalesce(t.updated_by, t.created_by)
WHERE t.workspace_id = @workspace_id AND t.id = @id AND t.deleted_at IS NULL;

-- name: InsertTemplate :exec
INSERT INTO templates (id, workspace_id, kind, name, description, mjml, html, thumb, size_bytes, source_campaign_id, created_by, updated_by, created_at, updated_at)
VALUES (@id, @workspace_id, @kind, @name, @description, @mjml, @html, @thumb, @size_bytes, sqlc.narg(source_campaign_id), sqlc.narg(created_by), sqlc.narg(created_by), @now, @now);

-- name: UpdateTemplate :exec
UPDATE templates
SET name = @name, description = @description, mjml = @mjml, html = @html, thumb = @thumb, size_bytes = @size_bytes,
    updated_by = sqlc.narg(updated_by), updated_at = @now
WHERE workspace_id = @workspace_id AND id = @id AND deleted_at IS NULL;

-- name: SoftDeleteTemplate :execrows
UPDATE templates SET deleted_at = @now, updated_by = sqlc.narg(updated_by)
WHERE workspace_id = @workspace_id AND id = @id AND deleted_at IS NULL;

-- name: TemplateNameTaken :one
SELECT EXISTS (SELECT 1 FROM templates WHERE workspace_id = @workspace_id AND kind = @kind AND lower(name) = lower(@name) AND deleted_at IS NULL AND id <> @id);
