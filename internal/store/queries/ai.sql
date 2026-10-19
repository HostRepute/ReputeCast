-- AI assistant (design expansion M22): the workspace's model, the AI
-- activity log, kept report summaries and AI translations.

-- name: AIConnection :one
SELECT * FROM ai_connections WHERE workspace_id = $1;

-- name: AIConnectionForUpdate :one
SELECT * FROM ai_connections WHERE workspace_id = $1 FOR UPDATE;

-- name: AIConnectionSave :exec
INSERT INTO ai_connections (
    workspace_id, provider, label, base_url, deployment, api_version, model, model_name, image_input,
    key_sealed, key_added_by, key_added_at, connected_by, connected_at, disconnected_by, disconnected_at,
    status, status_detail, status_at, last_test_at, last_test_ok, last_test_ms,
    cap_kind, cap_usd, cap_tokens, cap_set_by, cap_set_at, input_price, output_price, prices_from,
    features_off, notice_month, cap_notice_month, demo, updated_by, updated_at
) VALUES (
    @workspace_id, @provider, @label, @base_url, @deployment, @api_version, @model, @model_name, @image_input,
    @key_sealed, @key_added_by, @key_added_at, @connected_by, @connected_at, @disconnected_by, @disconnected_at,
    @status, @status_detail, @status_at, @last_test_at, @last_test_ok, @last_test_ms,
    @cap_kind, @cap_usd, @cap_tokens, @cap_set_by, @cap_set_at, @input_price, @output_price, @prices_from,
    @features_off, @notice_month, @cap_notice_month, @demo, @updated_by, @updated_at
)
ON CONFLICT (workspace_id) DO UPDATE SET
    provider = EXCLUDED.provider, label = EXCLUDED.label, base_url = EXCLUDED.base_url, deployment = EXCLUDED.deployment,
    api_version = EXCLUDED.api_version, model = EXCLUDED.model, model_name = EXCLUDED.model_name, image_input = EXCLUDED.image_input,
    key_sealed = EXCLUDED.key_sealed, key_added_by = EXCLUDED.key_added_by, key_added_at = EXCLUDED.key_added_at,
    connected_by = EXCLUDED.connected_by, connected_at = EXCLUDED.connected_at, disconnected_by = EXCLUDED.disconnected_by,
    disconnected_at = EXCLUDED.disconnected_at, status = EXCLUDED.status, status_detail = EXCLUDED.status_detail,
    status_at = EXCLUDED.status_at, last_test_at = EXCLUDED.last_test_at, last_test_ok = EXCLUDED.last_test_ok,
    last_test_ms = EXCLUDED.last_test_ms, cap_kind = EXCLUDED.cap_kind, cap_usd = EXCLUDED.cap_usd, cap_tokens = EXCLUDED.cap_tokens,
    cap_set_by = EXCLUDED.cap_set_by, cap_set_at = EXCLUDED.cap_set_at, input_price = EXCLUDED.input_price,
    output_price = EXCLUDED.output_price, prices_from = EXCLUDED.prices_from, features_off = EXCLUDED.features_off,
    notice_month = EXCLUDED.notice_month, cap_notice_month = EXCLUDED.cap_notice_month, demo = EXCLUDED.demo,
    updated_by = EXCLUDED.updated_by, updated_at = EXCLUDED.updated_at;

-- name: AIConnectionSetStatus :execrows
-- The provider rejected the saved key (or a passing request cleared it).
UPDATE ai_connections SET status = @status, status_detail = @status_detail, status_at = @status_at
WHERE workspace_id = @workspace_id AND status <> @status;

-- name: AIConnectionSetTest :exec
UPDATE ai_connections SET last_test_at = @last_test_at, last_test_ok = @last_test_ok, last_test_ms = @last_test_ms
WHERE workspace_id = @workspace_id;

-- name: AIConnectionClaimNotice :execrows
-- Claims this month's 80% notice, once.
UPDATE ai_connections SET notice_month = @month WHERE workspace_id = @workspace_id AND notice_month <> @month;

-- name: AIConnectionClaimCapNotice :execrows
-- Claims this month's cap-reached notice, once.
UPDATE ai_connections SET cap_notice_month = @month WHERE workspace_id = @workspace_id AND cap_notice_month <> @month;

-- name: AIRequestInsert :exec
INSERT INTO ai_requests (id, workspace_id, user_id, feature, action, target_kind, target_id, target_label, provider, model,
    input_tokens, output_tokens, cost_micros, outcome, outcome_detail, duration_ms, created_at, decided_at)
VALUES (@id, @workspace_id, @user_id, @feature, @action, @target_kind, @target_id, @target_label, @provider, @model,
    @input_tokens, @output_tokens, @cost_micros, @outcome, @outcome_detail, @duration_ms, @created_at, @decided_at);

-- name: AIRequest :one
SELECT * FROM ai_requests WHERE id = $1 AND workspace_id = $2;

-- name: AIRequestSetOutcome :execrows
-- A person accepted, kept, rejected or discarded a suggestion; only the
-- person who asked can, and only once it has an answer.
UPDATE ai_requests SET outcome = @outcome, outcome_detail = @outcome_detail, decided_at = @decided_at
WHERE id = @id AND workspace_id = @workspace_id AND user_id = @user_id
  AND outcome IN ('pending','accepted','accepted_edits','kept','rejected','discarded');

-- name: AIRequests :many
-- The AI activity log, newest first.
SELECT r.*, coalesce(u.name, '')::text AS user_name, coalesce(u.email, '')::text AS user_email, coalesce(u.avatar_color, '')::text AS user_color
FROM ai_requests r LEFT JOIN users u ON u.id = r.user_id
WHERE r.workspace_id = @workspace_id AND r.created_at >= @since
  AND (sqlc.narg(user_id)::uuid IS NULL OR r.user_id = sqlc.narg(user_id))
  AND (sqlc.narg(feature)::text IS NULL OR r.feature = sqlc.narg(feature))
  AND (sqlc.narg(outcomes)::text[] IS NULL OR r.outcome = ANY(sqlc.narg(outcomes)::text[]))
  AND (sqlc.narg(before)::timestamptz IS NULL OR (r.created_at, r.id) < (sqlc.narg(before)::timestamptz, sqlc.narg(before_id)::uuid))
ORDER BY r.created_at DESC, r.id DESC
LIMIT @lim;

-- name: AIRequestCount :one
SELECT count(*)::bigint FROM ai_requests WHERE workspace_id = @workspace_id AND created_at >= @since;

-- name: AIUsageTotal :one
SELECT coalesce(sum(cost_micros), 0)::bigint AS cost_micros, coalesce(sum(input_tokens), 0)::bigint AS input_tokens,
       coalesce(sum(output_tokens), 0)::bigint AS output_tokens, count(*)::bigint AS requests
FROM ai_requests WHERE workspace_id = @workspace_id AND created_at >= @since;

-- name: AIUsageByFeature :many
SELECT feature, coalesce(sum(cost_micros), 0)::bigint AS cost_micros, coalesce(sum(input_tokens + output_tokens), 0)::bigint AS tokens, count(*)::bigint AS requests
FROM ai_requests WHERE workspace_id = @workspace_id AND created_at >= @since
GROUP BY feature;

-- name: AIUsageByMember :many
SELECT r.user_id, coalesce(u.name, '')::text AS user_name, coalesce(u.email, '')::text AS user_email,
       coalesce(sum(r.cost_micros), 0)::bigint AS cost_micros, coalesce(sum(r.input_tokens + r.output_tokens), 0)::bigint AS tokens, count(*)::bigint AS requests
FROM ai_requests r LEFT JOIN users u ON u.id = r.user_id
WHERE r.workspace_id = @workspace_id AND r.created_at >= @since
GROUP BY r.user_id, u.name, u.email
ORDER BY cost_micros DESC, user_name;

-- name: AIRequestsPurge :execrows
-- The activity log keeps 90 days.
DELETE FROM ai_requests WHERE created_at < @before;

-- name: AIReportSummary :one
SELECT s.*, coalesce(u.name, '')::text AS kept_by_name
FROM ai_report_summaries s LEFT JOIN users u ON u.id = s.kept_by
WHERE s.campaign_id = @campaign_id AND s.workspace_id = @workspace_id;

-- name: AIReportSummarySave :exec
INSERT INTO ai_report_summaries (campaign_id, workspace_id, summary, model, model_name, request_id, kept_by, kept_at)
VALUES (@campaign_id, @workspace_id, @summary, @model, @model_name, @request_id, @kept_by, @kept_at)
ON CONFLICT (campaign_id) DO UPDATE SET summary = EXCLUDED.summary, model = EXCLUDED.model, model_name = EXCLUDED.model_name,
    request_id = EXCLUDED.request_id, kept_by = EXCLUDED.kept_by, kept_at = EXCLUDED.kept_at;

-- name: AIReportSummaryDelete :execrows
DELETE FROM ai_report_summaries WHERE campaign_id = @campaign_id AND workspace_id = @workspace_id;

-- name: AITranslationMarks :many
-- A campaign's language versions an accepted AI translation wrote.
SELECT language, ai_model, ai_version, version
FROM content_translations
WHERE workspace_id = @workspace_id AND owner_kind = 'campaign' AND owner_id = @campaign_id AND owner_version = 0 AND ai_model <> '';

-- name: AITranslationMark :execrows
UPDATE content_translations SET ai_model = @ai_model, ai_version = version, ai_reviewed_by = NULL, ai_reviewed_at = NULL
WHERE workspace_id = @workspace_id AND owner_kind = 'campaign' AND owner_id = @campaign_id AND owner_version = 0 AND language = @language;

-- name: AITranslationReviewed :execrows
UPDATE content_translations SET ai_version = 0, ai_reviewed_by = @reviewed_by, ai_reviewed_at = @reviewed_at
WHERE workspace_id = @workspace_id AND owner_kind = 'campaign' AND owner_id = @campaign_id AND owner_version = 0 AND language = @language AND ai_version > 0;

-- name: AICampaignBrief :one
-- What the assistant needs about a campaign: its name, content and language.
SELECT c.id, c.name, c.subject, c.preheader, c.mjml, c.html, c.status
FROM campaigns c WHERE c.id = @id AND c.workspace_id = @workspace_id;

-- name: AIInstanceAdmins :many
-- The people who can allow private addresses for model servers.
SELECT coalesce(nullif(name, ''), email)::text AS name FROM users WHERE is_instance_admin AND status = 'active' ORDER BY created_at LIMIT 3;

-- name: AITagNames :many
-- The workspace's tag names, for the segment drafts' condition schema
-- (names only: no contact is named or counted).
SELECT DISTINCT t::text AS tag FROM contacts c, unnest(c.tags) AS t WHERE c.workspace_id = @workspace_id ORDER BY 1 LIMIT 200;

-- name: AISegmentName :one
SELECT name FROM segments WHERE id = @id AND workspace_id = @workspace_id;

-- name: AIMembersRoles :many
-- The members with their role, for "Members with ai:use: Owner, Admin…".
SELECT coalesce(nullif(u.name, ''), u.email)::text AS name, r.key AS role_key, r.name AS role_name, r.builtin, r.permissions
FROM workspace_members m
JOIN roles r ON r.id = m.role_id
JOIN users u ON u.id = m.user_id AND u.status = 'active'
WHERE m.workspace_id = @workspace_id AND m.status = 'active'
ORDER BY u.name;

-- name: AIMediaImage :one
-- An image of the workspace's media library, for alt text.
SELECT content_type, data FROM media_assets WHERE id = @id AND workspace_id = @workspace_id;
