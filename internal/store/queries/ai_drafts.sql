-- name: AIDraftInsert :exec
INSERT INTO ai_draft_proposals (id, workspace_id, user_id, kind, target_id, base_revision, context, content, created_at, expires_at)
VALUES (@id, @workspace_id, @user_id, @kind, sqlc.narg(target_id), @base_revision, @context, @content, @created_at, @expires_at);

-- name: AIDraftGet :one
SELECT * FROM ai_draft_proposals
WHERE id = @id AND workspace_id = @workspace_id AND user_id = @user_id AND expires_at > @now;

-- name: AIDraftPending :many
SELECT id FROM ai_draft_proposals
WHERE workspace_id = @workspace_id AND user_id = @user_id AND kind = @kind
  AND (@all_targets::boolean OR target_id IS NOT DISTINCT FROM sqlc.narg(target_id)::uuid)
  AND decision = '' AND expires_at > @now
  AND (sqlc.narg(before_id)::uuid IS NULL OR id < sqlc.narg(before_id)::uuid)
ORDER BY id DESC LIMIT 20;

-- name: AIDraftLock :one
SELECT * FROM ai_draft_proposals
WHERE id = @id AND workspace_id = @workspace_id AND user_id = @user_id AND expires_at > @now
FOR UPDATE;

-- name: AIDraftDecide :exec
UPDATE ai_draft_proposals SET decision = @decision, result_id = sqlc.narg(result_id), result_revision = sqlc.narg(result_revision)
WHERE id = @id;

-- name: AIDraftPurge :execrows
DELETE FROM ai_draft_proposals WHERE expires_at <= @now;

-- name: AIDraftCampaign :one
SELECT c.id, c.name, c.subject, c.preheader, c.mjml, c.version, c.status, c.ab_test
FROM campaigns c
LEFT JOIN LATERAL (SELECT email, domain_id FROM senders WHERE id = c.sender_id FOR SHARE) s ON true
LEFT JOIN LATERAL (SELECT domain FROM sending_domains WHERE id = s.domain_id FOR SHARE) d ON true
WHERE c.workspace_id = @workspace_id AND c.id = @id
  AND c.deleted_at IS NULL AND c.channel = 'email'
  AND (cardinality(@scope_lists::uuid[]) = 0 OR (
    coalesce(jsonb_array_length(c.audience->'segments'), 0) = 0
    AND NOT EXISTS (SELECT 1 FROM jsonb_array_elements_text(coalesce(c.audience->'lists', '[]'::jsonb)) l(id)
                    WHERE NOT (l.id::uuid = ANY(@scope_lists::uuid[])))
  ))
  AND (cardinality(@scope_domains::text[]) = 0 OR c.sender_id IS NULL OR
       lower(coalesce(nullif(d.domain, ''), nullif(split_part(s.email, '@', 2), ''), '')) = ANY(@scope_domains::text[]))
FOR NO KEY UPDATE OF c;

-- name: AIDraftTemplate :one
SELECT mjml FROM templates
WHERE workspace_id = @workspace_id AND id = @id AND kind = 'template' AND deleted_at IS NULL
FOR SHARE;

-- name: AIDraftList :one
SELECT id FROM lists
WHERE workspace_id = @workspace_id AND id = @id AND deleted_at IS NULL
  AND (cardinality(@scope_lists::uuid[]) = 0 OR id = ANY(@scope_lists::uuid[]))
FOR SHARE;

-- name: AIDraftAutomationEmail :one
-- Only the exact version referenced by the selected flow, never recipients.
SELECT e.id, e.name, v.version, v.subject, v.preheader, v.mjml
FROM automation_emails e
JOIN automation_email_versions v ON v.email_id = e.id AND v.version = @version
LEFT JOIN LATERAL (SELECT default_sender_id FROM workspaces WHERE id = e.workspace_id FOR SHARE) w ON true
LEFT JOIN LATERAL (SELECT email, domain_id FROM senders WHERE id = coalesce(e.sender_id, w.default_sender_id) FOR SHARE) s ON true
LEFT JOIN LATERAL (SELECT domain FROM sending_domains WHERE id = s.domain_id FOR SHARE) d ON true
WHERE e.workspace_id = @workspace_id AND e.id = @id AND e.deleted_at IS NULL
  AND (cardinality(@scope_domains::text[]) = 0 OR
       lower(coalesce(nullif(d.domain, ''), nullif(split_part(s.email, '@', 2), ''), '')) = ANY(@scope_domains::text[]))
FOR SHARE OF e, v;
