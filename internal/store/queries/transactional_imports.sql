-- Template imports (X03): one row per run of the Import templates dialog.

-- name: TxnImportInsert :exec
INSERT INTO transactional_template_imports (id, workspace_id, source, status, credentials_enc, connection, available, picked, files, report, created_by, created_at, updated_at)
VALUES (@id, @workspace_id, @source, @status, sqlc.narg(credentials_enc), @connection, @available, @picked, @files, @report, sqlc.narg(created_by), @now, @now);

-- name: TxnImportGet :one
SELECT * FROM transactional_template_imports WHERE workspace_id = @workspace_id AND id = @id;

-- name: TxnImportForUpdate :one
SELECT * FROM transactional_template_imports WHERE workspace_id = @workspace_id AND id = @id FOR UPDATE;

-- name: TxnImportSave :exec
UPDATE transactional_template_imports
SET status = @status, picked = @picked, report = @report, error = sqlc.narg(error), result = @result,
    credentials_enc = sqlc.narg(credentials_enc), updated_at = @now, finished_at = sqlc.narg(finished_at)
WHERE workspace_id = @workspace_id AND id = @id;

-- name: TxnTemplateKeysInUse :many
SELECT key FROM transactional_templates WHERE workspace_id = @workspace_id AND deleted_at IS NULL;
