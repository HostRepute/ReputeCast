-- name: BrandVoiceLatest :one
SELECT v.*, coalesce(u.name, '')::text AS created_by_name
FROM brand_voice_revisions v LEFT JOIN users u ON u.id = v.created_by
WHERE v.workspace_id = @workspace_id
ORDER BY v.revision DESC LIMIT 1;

-- name: BrandVoiceRevision :one
SELECT v.*, coalesce(u.name, '')::text AS created_by_name
FROM brand_voice_revisions v LEFT JOIN users u ON u.id = v.created_by
WHERE v.workspace_id = @workspace_id AND v.revision = @revision;

-- name: BrandVoiceHistory :many
SELECT v.*, coalesce(u.name, '')::text AS created_by_name
FROM brand_voice_revisions v LEFT JOIN users u ON u.id = v.created_by
WHERE v.workspace_id = @workspace_id AND (@before_revision::integer = 0 OR v.revision < @before_revision)
ORDER BY v.revision DESC LIMIT 20;

-- name: BrandVoiceLock :exec
-- Serializes first saves as well as later revisions. No lock spans a model call.
SELECT id FROM workspaces WHERE id = @workspace_id FOR NO KEY UPDATE;

-- name: BrandVoiceInsert :exec
INSERT INTO brand_voice_revisions (workspace_id, revision, content, source_campaigns, request_id, created_by, created_at)
VALUES (@workspace_id, @revision, @content, @source_campaigns, sqlc.narg(request_id), sqlc.narg(created_by), @created_at);

-- name: BrandVoiceAccepted :one
SELECT revision FROM brand_voice_revisions WHERE workspace_id = @workspace_id AND request_id = @request_id;

-- name: BrandVoiceSuggest :exec
INSERT INTO brand_voice_suggestions (id, workspace_id, user_id, base_revision, content, source_campaigns, expires_at)
VALUES (@id, @workspace_id, @user_id, @base_revision, @content, @source_campaigns, @expires_at);

-- name: BrandVoiceSuggestion :one
SELECT * FROM brand_voice_suggestions
WHERE id = @id AND workspace_id = @workspace_id AND user_id = @user_id AND expires_at > @now
FOR UPDATE;

-- name: BrandVoiceSuggestionDelete :execrows
DELETE FROM brand_voice_suggestions WHERE id = @id AND workspace_id = @workspace_id AND user_id = @user_id;

-- name: BrandVoiceSuggestionsPurge :execrows
DELETE FROM brand_voice_suggestions WHERE expires_at <= @now;

-- name: BrandVoiceActor :one
-- Recheck current membership and user status after an external model call.
SELECT r.key, r.builtin, r.permissions, m.scope
FROM workspace_members m JOIN roles r ON r.id = m.role_id JOIN users u ON u.id = m.user_id
WHERE m.workspace_id = @workspace_id AND m.user_id = @user_id AND m.status = 'active' AND u.status = 'active'
FOR SHARE OF m, r, u;

-- name: BrandVoiceSources :many
-- Only explicitly selected campaigns. No recipients, rendered HTML, URLs,
-- names or sender addresses are returned to the learning service.
SELECT c.id, c.subject, c.preheader, c.mjml
FROM campaigns c
LEFT JOIN LATERAL (SELECT email, domain_id FROM senders WHERE id = c.sender_id FOR SHARE) s ON true
LEFT JOIN LATERAL (SELECT domain FROM sending_domains WHERE id = s.domain_id FOR SHARE) d ON true
WHERE c.workspace_id = @workspace_id AND c.id = ANY(@ids::uuid[]) AND c.status = 'sent'
  AND c.deleted_at IS NULL AND c.channel = 'email'
  AND (cardinality(@scope_lists::uuid[]) = 0 OR (
    coalesce(jsonb_array_length(c.audience->'segments'), 0) = 0
    AND NOT EXISTS (SELECT 1 FROM jsonb_array_elements_text(coalesce(c.audience->'lists', '[]'::jsonb)) l(id)
                    WHERE NOT (l.id::uuid = ANY(@scope_lists::uuid[])))
  ))
  AND (cardinality(@scope_domains::text[]) = 0 OR
       lower(coalesce(nullif(d.domain, ''), nullif(split_part(s.email, '@', 2), ''), '')) = ANY(@scope_domains::text[]))
ORDER BY c.id
FOR SHARE OF c;
