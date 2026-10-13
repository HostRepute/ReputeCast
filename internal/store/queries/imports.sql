-- Imports: the migration wizard's drafts, staged rows, live log and reports.

-- name: CreateImport :one
INSERT INTO imports (id, workspace_id, source, status, step, options, field_map, consent_map, created_by)
VALUES (@id, @workspace_id, @source, 'draft', 'source', @options, '{}', '{}', @created_by)
RETURNING *;

-- name: GetImport :one
SELECT * FROM imports WHERE workspace_id = @workspace_id AND id = @id;

-- name: GetImportByID :one
SELECT * FROM imports WHERE id = @id;

-- name: ListImports :many
SELECT i.id, i.source, i.status, i.step, i.file_name, i.stats, i.created_at, i.started_at, i.finished_at,
       coalesce(u.name, '')::text AS created_by_name
FROM imports i
LEFT JOIN users u ON u.id = i.created_by
WHERE i.workspace_id = @workspace_id
ORDER BY i.created_at DESC
LIMIT 50;

-- name: SetImportConnection :exec
UPDATE imports SET connection = @connection, credentials_enc = @credentials_enc, error = '', updated_at = now()
WHERE id = @id;

-- name: SetImportFile :exec
UPDATE imports SET file_name = @file_name, file_data = @file_data, connection = @connection, staged_at = NULL, error = '', updated_at = now()
WHERE id = @id;

-- name: SetImportOptions :exec
UPDATE imports SET options = @options, updated_at = now() WHERE id = @id;

-- name: SetImportFieldMap :exec
UPDATE imports SET field_map = @field_map, updated_at = now() WHERE id = @id;

-- name: SetImportConsentMap :exec
UPDATE imports SET consent_map = @consent_map, updated_at = now() WHERE id = @id;

-- name: SetImportStep :exec
UPDATE imports SET step = @step, updated_at = now() WHERE id = @id;

-- name: SetImportStatus :exec
UPDATE imports SET status = @status, error = @error, resume_at = @resume_at, updated_at = now() WHERE id = @id;

-- name: SetImportStaged :exec
UPDATE imports SET status = 'draft', staged_at = now(), error = '', resume_at = NULL, connection = @connection, updated_at = now()
WHERE id = @id;

-- name: SetImportPreview :exec
UPDATE imports SET preview = @preview, updated_at = now() WHERE id = @id;

-- name: StartImport :exec
UPDATE imports SET status = 'running', step = 'import', started_at = now(), finished_at = NULL, error = '', progress = @progress, updated_at = now()
WHERE id = @id;

-- name: SetImportProgress :exec
UPDATE imports SET progress = @progress, updated_at = now() WHERE id = @id;

-- name: FinishImport :exec
UPDATE imports SET status = @status, step = @step, stats = @stats, progress = @progress, report_csv = @report_csv,
       skipped_csv = @skipped_csv, finished_at = now(), error = @error, updated_at = now()
WHERE id = @id;

-- name: DeleteImport :execrows
DELETE FROM imports WHERE workspace_id = @workspace_id AND id = @id AND status IN ('draft', 'failed', 'cancelled', 'paused');

-- name: InsertImportLog :exec
INSERT INTO import_log (id, import_id, occurred_at, kind, message) VALUES (@id, @import_id, @occurred_at, @kind, @message);

-- name: ListImportLog :many
SELECT * FROM import_log WHERE import_id = @import_id ORDER BY occurred_at DESC, id DESC LIMIT 30;

-- name: CountImportRows :one
SELECT count(*)::bigint FROM import_rows WHERE import_id = @import_id;

-- name: DeleteImportRows :exec
DELETE FROM import_rows WHERE import_id = @import_id;

-- name: ImportRowsAfter :many
SELECT * FROM import_rows WHERE import_id = @import_id AND row_no > @after ORDER BY row_no LIMIT @lim;

-- name: ImportStatusCounts :many
SELECT status, list_ref, count(*)::bigint AS n FROM import_rows WHERE import_id = @import_id GROUP BY status, list_ref;

-- name: LatestFinishedImport :one
SELECT * FROM imports WHERE workspace_id = @workspace_id AND status = 'done' ORDER BY finished_at DESC NULLS LAST LIMIT 1;

-- name: ActiveImport :one
-- The draft or running import the wizard resumes.
SELECT * FROM imports WHERE workspace_id = @workspace_id AND status IN ('draft', 'staging', 'running', 'paused')
ORDER BY created_at DESC LIMIT 1;

-- name: GetMXCache :many
SELECT domain, has_mx, provider FROM mx_cache WHERE domain = ANY(@domains::text[]) AND checked_at > @fresh_after;

-- name: PutMXCache :exec
INSERT INTO mx_cache (domain, has_mx, provider, checked_at) VALUES (@domain, @has_mx, @provider, now())
ON CONFLICT (domain) DO UPDATE SET has_mx = excluded.has_mx, provider = excluded.provider, checked_at = now();

-- name: ImportJobs :many
-- Imports running in the background, and those that ended in the last two
-- hours, for the jobs tray.
SELECT i.id, i.source, i.status, i.file_name, i.progress, i.stats, i.error, i.connection, i.resume_at,
       i.created_at, i.started_at, i.finished_at
FROM imports i
WHERE i.workspace_id = @workspace_id
  AND (i.status IN ('staging', 'running', 'paused') OR (i.finished_at IS NOT NULL AND i.finished_at > @ended_after))
ORDER BY i.created_at DESC
LIMIT 20;
