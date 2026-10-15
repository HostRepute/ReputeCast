-- Welcome-series prompts (add-on transactional §6).

-- name: WelcomeLists :many
-- The workspace's lists, newest first.
SELECT id, name, created_at FROM lists
WHERE workspace_id = @workspace_id AND deleted_at IS NULL
ORDER BY created_at DESC, id DESC;

-- name: WelcomeListTriggers :many
-- Automations that start on a list or a form, with their status.
SELECT id, name, status, trigger FROM automations
WHERE workspace_id = @workspace_id AND deleted_at IS NULL
  AND trigger->>'kind' IN ('joined_list', 'form_submitted')
ORDER BY created_at, id;

-- name: WelcomeDismissedLists :many
SELECT list_id FROM welcome_prompt_dismissals WHERE workspace_id = @workspace_id;

-- name: WelcomeDismiss :exec
INSERT INTO welcome_prompt_dismissals (workspace_id, list_id, dismissed_by, dismissed_at)
VALUES (@workspace_id, @list_id, sqlc.narg(dismissed_by), @dismissed_at)
ON CONFLICT (workspace_id, list_id) DO UPDATE SET dismissed_by = EXCLUDED.dismissed_by, dismissed_at = EXCLUDED.dismissed_at;

-- name: WelcomeForm :one
SELECT id, name, status, list_ids FROM forms WHERE workspace_id = @workspace_id AND id = @id;

-- name: WelcomeImport :one
SELECT id, status, options FROM imports WHERE workspace_id = @workspace_id AND id = @id;
