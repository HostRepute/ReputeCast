-- Retries of an issue's blocked addresses (review AN-7, R5.L5).

-- name: RetryBlockedContacts :many
-- Who a retry of an issue may go to: the contacts whose email a mailbox
-- provider blocked for the sender's reputation or policy and never
-- delivered, who haven't been sent a retry of it yet. The retry being built
-- (retry_id) doesn't count, so a build that continues keeps its own.
SELECT DISTINCT m.contact_id::uuid AS contact_id
FROM delivery_events e
JOIN messages m ON m.id = e.message_id
WHERE e.campaign_id = @campaign_id AND e.workspace_id = @workspace_id AND e.type IN ('hard_bounce', 'soft_bounce')
  AND e.bounce_class IN ('policy_block', 'spam_block', 'reputation')
  AND m.contact_id IS NOT NULL AND m.delivered_at IS NULL
  AND NOT EXISTS (
      SELECT 1 FROM messages r
      JOIN campaigns rc ON rc.id = r.campaign_id
      WHERE r.contact_id = m.contact_id AND r.sent_at IS NOT NULL AND rc.workspace_id = @workspace_id
        AND rc.id <> sqlc.arg(retry_id)::uuid AND (rc.audience ->> 'retry_of') = m.campaign_id::text);

-- name: RetryCampaignOf :one
-- An issue's retry, when it has one that isn't deleted.
SELECT c.id, c.name, c.status, c.started_at, c.completed_at
FROM campaigns c
WHERE c.workspace_id = @workspace_id AND c.deleted_at IS NULL AND (c.audience ->> 'retry_of') = sqlc.arg(campaign_id)::text;

-- name: RetryIssueName :one
-- The name of the issue a retry retries (deleted or not).
SELECT name FROM campaigns WHERE workspace_id = @workspace_id AND id = @id;

-- name: RetryLockIssue :one
-- Takes an issue for a retry: two people asking at once make one retry.
SELECT id, status, audience FROM campaigns WHERE workspace_id = @workspace_id AND id = @id AND deleted_at IS NULL FOR UPDATE;

-- name: RetryInsertCampaign :exec
-- A retry: a draft copy of its issue as everyone else got it (subject,
-- preheader and content come in: an A/B test's winner, with no test of its
-- own), from the same sender with the same settings, to the issue's
-- audience kept to its blocked addresses.
INSERT INTO campaigns (id, workspace_id, name, status, subject, preheader, sender_id, reply_to, audience, editor_mode, mjml, html, plain_text,
    version, tracking, thresholds, content_settings, content_hash, created_by, updated_by, created_at, updated_at, autosaved_at)
SELECT @id, c.workspace_id, @name, 'draft', @subject, @preheader, c.sender_id, c.reply_to, @audience, c.editor_mode, @mjml, @html, @plain_text,
    1, c.tracking, c.thresholds, c.content_settings, @content_hash, sqlc.narg(created_by), sqlc.narg(created_by), @now, @now, @now
FROM campaigns c
WHERE c.workspace_id = @workspace_id AND c.id = sqlc.arg(issue)::uuid;

-- name: RetryCopyLanguageSource :exec
-- A retry has its issue's language versions: the source's blocks first.
INSERT INTO content_sources (workspace_id, owner_kind, owner_id, owner_version, language, blocks, subject_hash, preheader_hash, version, next_key, updated_at)
SELECT src.workspace_id, src.owner_kind, sqlc.arg(to_id)::uuid, src.owner_version, src.language, src.blocks, src.subject_hash, src.preheader_hash,
    src.version, src.next_key, src.updated_at
FROM content_sources src
WHERE src.owner_kind = 'campaign' AND src.owner_id = sqlc.arg(from_id)::uuid AND src.owner_version = 0
ON CONFLICT (owner_kind, owner_id, owner_version) DO NOTHING;

-- name: RetryCopyTranslation :exec
-- Then each language version, as everyone else got it (subject, preheader
-- and content come in: an A/B test's winner, with no test of its own).
INSERT INTO content_translations (id, workspace_id, owner_kind, owner_id, owner_version, language, status, held, subject, preheader, mjml, html,
    plain_text, ab_test, links, version, content_hash, translated_from, translated_by, translated_at, created_by, created_at, updated_by, updated_at,
    autosaved_at)
SELECT @id, src.workspace_id, src.owner_kind, sqlc.arg(to_id)::uuid, src.owner_version, src.language, src.status, src.held, @subject, @preheader,
    @mjml, @html, @plain_text, NULL, src.links, src.version, @content_hash, src.translated_from, src.translated_by, src.translated_at, src.created_by,
    src.created_at, src.updated_by, src.updated_at, src.autosaved_at
FROM content_translations src
WHERE src.id = sqlc.arg(src_id)::uuid;
