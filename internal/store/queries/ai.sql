-- AI assistant (design expansion M22): the workspace's AI settings, its
-- model accounts and models (assistant brief D3), the AI activity log, kept
-- report summaries and AI translations.

-- name: AIBinding :one
-- A model with its account and the workspace's settings: what a request is
-- bound to. The column names are the single connection's before B1.
SELECT m.id AS model_id, a.id AS account_id, s.workspace_id,
    a.provider, a.label, a.base_url, m.deployment, a.api_version, m.model, m.model_name, m.image_input,
    a.key_sealed, a.key_added_by, a.key_added_at, a.connected_by, a.connected_at, a.disconnected_by, a.disconnected_at,
    a.status, a.status_detail, a.status_at, a.last_test_at, a.last_test_ok, a.last_test_ms,
    s.cap_kind, s.cap_usd, s.cap_tokens, s.cap_set_by, s.cap_set_at,
    m.input_price, m.output_price, m.prices_from, m.price_known,
    s.features_off, s.notice_month, s.cap_notice_month, a.demo,
    s.revision AS settings_revision, a.revision AS account_revision, m.revision AS model_revision,
    m.updated_at AS model_updated_at
FROM ai_models m
JOIN ai_accounts a ON a.id = m.account_id AND a.workspace_id = m.workspace_id
JOIN ai_workspace_settings s ON s.workspace_id = m.workspace_id
WHERE m.workspace_id = @workspace_id AND m.id = @model_id;

-- name: AIPrimaryModel :one
-- The workspace's first account and its first model: what the single-model
-- settings and operations act on.
SELECT m.id FROM ai_accounts a JOIN ai_models m ON m.account_id = a.id
WHERE a.workspace_id = @workspace_id
ORDER BY a.created_at, a.id, m.created_at, m.id
LIMIT 1;

-- name: AIFeatureModel :one
SELECT model_id FROM ai_feature_models WHERE workspace_id = @workspace_id AND feature = @feature;

-- name: AIFeatureModelsFill :exec
-- A first model is every feature's default; features with one keep it.
INSERT INTO ai_feature_models (workspace_id, feature, model_id)
SELECT @workspace_id::uuid, f, @model_id::uuid FROM unnest(@features::text[]) AS f
ON CONFLICT (workspace_id, feature) DO NOTHING;

-- name: AIFeatureModelsMove :exec
-- Features whose default is one of an account's other models move to its kept one.
UPDATE ai_feature_models f SET model_id = @keep_id
WHERE f.workspace_id = @workspace_id AND f.model_id IN (
    SELECT m.id FROM ai_models m WHERE m.workspace_id = @workspace_id AND m.account_id = @account_id AND m.id <> @keep_id);

-- name: AIModelsDeleteOthers :exec
DELETE FROM ai_models WHERE workspace_id = @workspace_id AND account_id = @account_id AND id <> @keep_id;

-- name: AIWorkspaceSettings :one
SELECT * FROM ai_workspace_settings WHERE workspace_id = $1;

-- name: AIWorkspaceSettingsLock :one
-- The AI configuration lock (see migration 03630).
SELECT * FROM ai_workspace_settings WHERE workspace_id = $1 FOR UPDATE;

-- name: AIWorkspaceSettingsEnsure :exec
INSERT INTO ai_workspace_settings (workspace_id) VALUES ($1) ON CONFLICT (workspace_id) DO NOTHING;

-- name: AIWorkspaceSettingsSave :exec
INSERT INTO ai_workspace_settings (workspace_id, cap_kind, cap_usd, cap_tokens, cap_set_by, cap_set_at, features_off,
    notice_month, cap_notice_month, updated_by, updated_at)
VALUES (@workspace_id, @cap_kind, @cap_usd, @cap_tokens, @cap_set_by, @cap_set_at, @features_off,
    @notice_month, @cap_notice_month, @updated_by, @updated_at)
ON CONFLICT (workspace_id) DO UPDATE SET cap_kind = EXCLUDED.cap_kind, cap_usd = EXCLUDED.cap_usd, cap_tokens = EXCLUDED.cap_tokens,
    cap_set_by = EXCLUDED.cap_set_by, cap_set_at = EXCLUDED.cap_set_at, features_off = EXCLUDED.features_off,
    notice_month = EXCLUDED.notice_month, cap_notice_month = EXCLUDED.cap_notice_month,
    revision = ai_workspace_settings.revision + 1, updated_by = EXCLUDED.updated_by, updated_at = EXCLUDED.updated_at;

-- name: AIAccountSave :exec
INSERT INTO ai_accounts (id, workspace_id, provider, label, base_url, api_version, key_sealed, key_added_by, key_added_at,
    connected_by, connected_at, disconnected_by, disconnected_at, status, status_detail, status_at,
    last_test_at, last_test_ok, last_test_ms, demo, created_at, updated_by, updated_at)
VALUES (@id, @workspace_id, @provider, @label, @base_url, @api_version, @key_sealed, @key_added_by, @key_added_at,
    @connected_by, @connected_at, @disconnected_by, @disconnected_at, @status, @status_detail, @status_at,
    @last_test_at, @last_test_ok, @last_test_ms, @demo, @updated_at, @updated_by, @updated_at)
ON CONFLICT (id) DO UPDATE SET provider = EXCLUDED.provider, label = EXCLUDED.label, base_url = EXCLUDED.base_url,
    api_version = EXCLUDED.api_version, key_sealed = EXCLUDED.key_sealed, key_added_by = EXCLUDED.key_added_by,
    key_added_at = EXCLUDED.key_added_at, connected_by = EXCLUDED.connected_by, connected_at = EXCLUDED.connected_at,
    disconnected_by = EXCLUDED.disconnected_by, disconnected_at = EXCLUDED.disconnected_at, status = EXCLUDED.status,
    status_detail = EXCLUDED.status_detail, status_at = EXCLUDED.status_at, last_test_at = EXCLUDED.last_test_at,
    last_test_ok = EXCLUDED.last_test_ok, last_test_ms = EXCLUDED.last_test_ms, demo = EXCLUDED.demo,
    revision = ai_accounts.revision + 1, updated_by = EXCLUDED.updated_by, updated_at = EXCLUDED.updated_at
WHERE ai_accounts.workspace_id = EXCLUDED.workspace_id;

-- name: AIModelSave :exec
INSERT INTO ai_models (id, workspace_id, account_id, model, model_name, deployment, image_input, input_price, output_price,
    prices_from, price_known, created_at, updated_by, updated_at)
VALUES (@id, @workspace_id, @account_id, @model, @model_name, @deployment, @image_input, @input_price, @output_price,
    @prices_from, @price_known, @updated_at, @updated_by, @updated_at)
ON CONFLICT (id) DO UPDATE SET model = EXCLUDED.model, model_name = EXCLUDED.model_name, deployment = EXCLUDED.deployment,
    image_input = EXCLUDED.image_input, input_price = EXCLUDED.input_price, output_price = EXCLUDED.output_price,
    prices_from = EXCLUDED.prices_from, price_known = EXCLUDED.price_known,
    revision = ai_models.revision + 1, updated_by = EXCLUDED.updated_by, updated_at = EXCLUDED.updated_at
WHERE ai_models.workspace_id = EXCLUDED.workspace_id AND ai_models.account_id = EXCLUDED.account_id;

-- name: AIAccountSetStatus :execrows
-- The provider rejected the saved key (or a passing request cleared it),
-- unless the account changed since the request was bound to it.
UPDATE ai_accounts SET status = @status, status_detail = @status_detail, status_at = @status_at
WHERE id = @id AND workspace_id = @workspace_id AND status <> @status AND revision = @expected_revision;

-- name: AIAccountSetTest :exec
UPDATE ai_accounts SET last_test_at = @last_test_at, last_test_ok = @last_test_ok, last_test_ms = @last_test_ms
WHERE id = @id AND workspace_id = @workspace_id;

-- name: AIWorkspaceSettingsClaimNotice :execrows
-- Claims this month's 80% notice, once.
UPDATE ai_workspace_settings SET notice_month = @month WHERE workspace_id = @workspace_id AND notice_month <> @month;

-- name: AIWorkspaceSettingsClaimCapNotice :execrows
-- Claims this month's cap-reached notice, once.
UPDATE ai_workspace_settings SET cap_notice_month = @month WHERE workspace_id = @workspace_id AND cap_notice_month <> @month;

-- name: AIRequestInsert :exec
INSERT INTO ai_requests (id, workspace_id, user_id, feature, action, target_kind, target_id, target_label, provider, model,
    input_tokens, output_tokens, cost_micros, outcome, outcome_detail, duration_ms, created_at, decided_at, voice_revision, usage_known,
    account_id, model_id)
VALUES (@id, @workspace_id, @user_id, @feature, @action, @target_kind, @target_id, @target_label, @provider, @model,
    @input_tokens, @output_tokens, @cost_micros, @outcome, @outcome_detail, @duration_ms, @created_at, @decided_at, @voice_revision, @usage_known,
    @account_id, @model_id)
ON CONFLICT (id) DO UPDATE SET input_tokens = EXCLUDED.input_tokens, output_tokens = EXCLUDED.output_tokens,
    cost_micros = EXCLUDED.cost_micros, usage_known = EXCLUDED.usage_known
WHERE ai_requests.workspace_id = EXCLUDED.workspace_id AND NOT ai_requests.usage_known;

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
