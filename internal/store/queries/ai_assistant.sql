-- The AI assistant (B3): its settings, threads and their attached context.
-- A thread is read only through its owner: every query that returns one
-- names the person.

-- name: AIAssistantSettings :one
SELECT * FROM ai_assistant_settings WHERE workspace_id = @workspace_id;

-- name: AIAssistantSettingsSave :exec
INSERT INTO ai_assistant_settings (workspace_id, enabled, tool_limit, token_limit, cost_limit_micros, per_person, per_workspace, updated_by, updated_at)
VALUES (@workspace_id, @enabled, @tool_limit, @token_limit, @cost_limit_micros, @per_person, @per_workspace, sqlc.narg(updated_by), @updated_at)
ON CONFLICT (workspace_id) DO UPDATE SET enabled = EXCLUDED.enabled, tool_limit = EXCLUDED.tool_limit, token_limit = EXCLUDED.token_limit,
    cost_limit_micros = EXCLUDED.cost_limit_micros, per_person = EXCLUDED.per_person, per_workspace = EXCLUDED.per_workspace,
    revision = ai_assistant_settings.revision + 1, updated_by = EXCLUDED.updated_by, updated_at = EXCLUDED.updated_at;

-- name: AIAssistantThreadInsert :exec
INSERT INTO ai_assistant_threads (id, workspace_id, user_id, model_id, title_sealed, screen, tool_limit, token_limit, cost_limit_micros,
    request_key, created_at, updated_at, last_activity_at, expires_at)
VALUES (@id, @workspace_id, @user_id, sqlc.narg(model_id), sqlc.narg(title_sealed), @screen, @tool_limit, @token_limit, @cost_limit_micros,
    sqlc.narg(request_key), @now, @now, @now, @expires_at);

-- name: AIAssistantThread :one
-- One of the person's threads; another person's isn't there, a manager's
-- view included.
SELECT * FROM ai_assistant_threads WHERE workspace_id = @workspace_id AND user_id = @user_id AND id = @id;

-- name: AIAssistantThreadLock :one
SELECT * FROM ai_assistant_threads WHERE workspace_id = @workspace_id AND user_id = @user_id AND id = @id FOR UPDATE;

-- name: AIAssistantThreadByRequest :one
SELECT * FROM ai_assistant_threads
WHERE workspace_id = @workspace_id AND user_id = @user_id AND request_key = @request_key;

-- name: AIAssistantThreads :many
-- The person's threads, the most recently active first.
SELECT * FROM ai_assistant_threads
WHERE workspace_id = @workspace_id AND user_id = @user_id
  AND (sqlc.narg(before)::timestamptz IS NULL OR (last_activity_at, id) < (sqlc.narg(before)::timestamptz, sqlc.narg(before_id)::uuid))
ORDER BY last_activity_at DESC, id DESC
LIMIT @lim;

-- name: AIAssistantThreadSave :exec
-- Saves what the person changed. Only an open thread changes.
UPDATE ai_assistant_threads SET title_sealed = sqlc.narg(title_sealed), model_id = sqlc.narg(model_id), updated_at = @now,
    last_activity_at = @now, expires_at = @expires_at
WHERE workspace_id = @workspace_id AND id = @id AND state = 'open';

-- name: AIAssistantThreadDelete :execrows
-- Deleting a thread removes its content at once: messages, contexts and
-- stream events go with it. The usage ledger stays (without the thread).
DELETE FROM ai_assistant_threads WHERE workspace_id = @workspace_id AND user_id = @user_id AND id = @id;

-- name: AIAssistantContextInsert :execrows
INSERT INTO ai_assistant_contexts (id, thread_id, kind, target_id, revision, snapshot_sealed, shared_at)
VALUES (@id, @thread_id, @kind, @target_id, @revision, sqlc.narg(snapshot_sealed), @shared_at)
ON CONFLICT (thread_id, kind, target_id) DO NOTHING;

-- name: AIAssistantContexts :many
SELECT * FROM ai_assistant_contexts WHERE thread_id = @thread_id ORDER BY shared_at, id;

-- name: AIAssistantContextDelete :one
DELETE FROM ai_assistant_contexts WHERE thread_id = @thread_id AND id = @id RETURNING kind, target_id;
