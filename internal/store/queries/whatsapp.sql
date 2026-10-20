-- WhatsApp (internal/whatsapp): the account, templates, the rate card,
-- WhatsApp campaigns and their sends, messages, what people send, and
-- Meta's raw webhook events.

-- name: WhatsAppAccount :one
SELECT a.*, coalesce(u.name, '')::text AS connected_by_name
FROM whatsapp_accounts a LEFT JOIN users u ON u.id = a.connected_by
WHERE a.workspace_id = @workspace_id AND a.removed_at IS NULL;

-- name: WhatsAppAccountByID :one
SELECT * FROM whatsapp_accounts WHERE id = @id;

-- name: WhatsAppAccountBySlug :one
-- The account a webhook request is for: /hooks/whatsapp/{workspace}.
SELECT a.* FROM whatsapp_accounts a JOIN workspaces w ON w.id = a.workspace_id AND w.deleted_at IS NULL
WHERE w.slug = @slug AND a.removed_at IS NULL;

-- name: WhatsAppAccountsAll :many
-- Every connected account, for the hourly sync.
SELECT a.* FROM whatsapp_accounts a JOIN workspaces w ON w.id = a.workspace_id AND w.deleted_at IS NULL
WHERE a.removed_at IS NULL ORDER BY a.id;

-- name: WhatsAppAccountInsert :exec
INSERT INTO whatsapp_accounts (id, workspace_id, credentials_enc, business_hint, verify_token, status, status_since, demo, connected_by, connected_at)
VALUES (@id, @workspace_id, @credentials_enc, @business_hint, @verify_token, 'waiting', @connected_at, @demo, sqlc.narg(connected_by), @connected_at);

-- name: WhatsAppAccountSetCredentials :exec
UPDATE whatsapp_accounts SET credentials_enc = @credentials_enc, business_hint = @business_hint WHERE id = @id;

-- name: WhatsAppAccountSetInfo :exec
-- What Meta reported for the number and the business account.
UPDATE whatsapp_accounts SET business_name = @business_name, phone_number = @phone_number, display_name = @display_name, name_status = @name_status,
    phone_status = @phone_status,
    previous_quality = CASE WHEN quality <> @quality::text THEN quality ELSE previous_quality END,
    quality_changed_at = CASE WHEN quality <> @quality::text THEN @at::timestamptz ELSE quality_changed_at END,
    quality = @quality,
    previous_limit = CASE WHEN messaging_limit <> @messaging_limit::int THEN messaging_limit ELSE previous_limit END,
    limit_changed_at = CASE WHEN messaging_limit <> @messaging_limit::int THEN @at::timestamptz ELSE limit_changed_at END,
    messaging_limit = @messaging_limit, checked_at = @at
WHERE id = @id;

-- name: WhatsAppAccountSetStatus :exec
UPDATE whatsapp_accounts SET status = @status, status_detail = @status_detail,
    status_since = CASE WHEN status = @status THEN status_since ELSE @at::timestamptz END
WHERE id = @id;

-- name: WhatsAppAccountVerified :exec
UPDATE whatsapp_accounts SET verified_at = coalesce(verified_at, @at::timestamptz),
    status = CASE WHEN status = 'waiting' THEN 'connected' ELSE status END,
    status_since = CASE WHEN status = 'waiting' THEN @at::timestamptz ELSE status_since END
WHERE id = @id;

-- name: WhatsAppAccountTouchEvent :exec
UPDATE whatsapp_accounts SET last_event_at = greatest(coalesce(last_event_at, @at::timestamptz), @at::timestamptz) WHERE id = @id;

-- name: WhatsAppAccountSynced :exec
UPDATE whatsapp_accounts SET templates_synced_at = @at WHERE id = @id;

-- name: WhatsAppAccountRemove :exec
UPDATE whatsapp_accounts SET removed_at = @at, removed_by = sqlc.narg(removed_by) WHERE id = @id AND removed_at IS NULL;

-- name: WhatsAppTemplates :many
SELECT * FROM whatsapp_templates WHERE account_id = @account_id AND removed_at IS NULL
ORDER BY CASE status WHEN 'approved' THEN 0 WHEN 'pending' THEN 1 WHEN 'rejected' THEN 3 ELSE 2 END, synced_at, name;

-- name: WhatsAppTemplate :one
SELECT * FROM whatsapp_templates WHERE id = @id AND workspace_id = @workspace_id;

-- name: WhatsAppTemplateByName :one
SELECT * FROM whatsapp_templates WHERE account_id = @account_id AND name = @name AND (@language::text = '' OR language = @language)
    AND removed_at IS NULL
ORDER BY language LIMIT 1;

-- name: WhatsAppTemplateUpsert :exec
-- A template as Meta lists it. Its status date moves only when the status
-- changes, and Meta's words for a rejection stay unless the code changed.
INSERT INTO whatsapp_templates (id, workspace_id, account_id, meta_id, name, language, category, status, status_since, reason_code, reason, body,
    variables, buttons, synced_at)
VALUES (@id, @workspace_id, @account_id, @meta_id, @name, @language, @category, @status, @at, @reason_code, @reason, @body, @variables, @buttons, @at)
ON CONFLICT (account_id, name, language) WHERE removed_at IS NULL DO UPDATE SET
    meta_id = EXCLUDED.meta_id, category = EXCLUDED.category,
    status_since = CASE WHEN whatsapp_templates.status <> EXCLUDED.status THEN EXCLUDED.status_since ELSE whatsapp_templates.status_since END,
    status = EXCLUDED.status,
    reason = CASE WHEN whatsapp_templates.reason_code = EXCLUDED.reason_code AND whatsapp_templates.reason <> '' AND EXCLUDED.reason_code <> ''
        THEN whatsapp_templates.reason ELSE EXCLUDED.reason END,
    reason_code = EXCLUDED.reason_code, body = EXCLUDED.body,
    variables = CASE WHEN jsonb_array_length(whatsapp_templates.variables) = jsonb_array_length(EXCLUDED.variables)
        THEN whatsapp_templates.variables ELSE EXCLUDED.variables END,
    buttons = EXCLUDED.buttons, synced_at = EXCLUDED.synced_at;

-- name: WhatsAppTemplateSetStatus :exec
-- A status update from Meta's webhook.
UPDATE whatsapp_templates SET
    status_since = CASE WHEN status <> @status THEN @at::timestamptz ELSE status_since END,
    status = @status, reason_code = @reason_code, reason = CASE WHEN @reason::text = '' THEN reason ELSE @reason END
WHERE account_id = @account_id AND name = @name AND (@language::text = '' OR language = @language) AND removed_at IS NULL;

-- name: WhatsAppTemplatesRemoveMissing :exec
-- Templates Meta no longer lists were deleted in WhatsApp Manager.
UPDATE whatsapp_templates SET removed_at = @at
WHERE account_id = @account_id AND removed_at IS NULL AND NOT (id = ANY(@keep::uuid[]));

-- name: WhatsAppSettings :one
SELECT * FROM whatsapp_settings WHERE workspace_id = @workspace_id;

-- name: WhatsAppSettingsEnsure :exec
INSERT INTO whatsapp_settings (workspace_id, rates_as_of, updated_at) VALUES (@workspace_id, @rates_as_of, @at)
ON CONFLICT (workspace_id) DO NOTHING;

-- name: WhatsAppSettingsSetQuiet :exec
UPDATE whatsapp_settings SET quiet_hours = @quiet_hours, updated_at = @at WHERE workspace_id = @workspace_id;

-- name: WhatsAppSettingsRatesEdited :exec
UPDATE whatsapp_settings SET rates_as_of = @rates_as_of, rates_edited_by = sqlc.narg(rates_edited_by), rates_edited_at = @at, updated_at = @at
WHERE workspace_id = @workspace_id;

-- name: WhatsAppRates :many
SELECT * FROM whatsapp_rates WHERE workspace_id = @workspace_id ORDER BY country, category;

-- name: WhatsAppRatesClear :exec
DELETE FROM whatsapp_rates WHERE workspace_id = @workspace_id;

-- name: WhatsAppRateInsert :exec
INSERT INTO whatsapp_rates (workspace_id, country, category, price) VALUES (@workspace_id, @country, @category, @price)
ON CONFLICT (workspace_id, country, category) DO UPDATE SET price = EXCLUDED.price;

-- name: WhatsAppCampaign :one
SELECT c.id, c.workspace_id, c.name, c.status, c.audience, c.version, c.content_hash, c.created_by, c.updated_by, c.created_at, c.updated_at,
    c.autosaved_at, c.scheduled_at, c.started_at, c.completed_at, c.started_by, c.channel, c.audience_count,
    w.template_id, w.template_name, w.template_language, w.variables, w.schedule_mode, w.send_at, w.local_date, w.local_minute, w.over_limit,
    coalesce(cu.name, '')::text AS created_by_name, coalesce(uu.name, '')::text AS updated_by_name, coalesce(su.name, '')::text AS started_by_name
FROM campaigns c
JOIN whatsapp_campaigns w ON w.campaign_id = c.id
LEFT JOIN users cu ON cu.id = c.created_by
LEFT JOIN users uu ON uu.id = c.updated_by
LEFT JOIN users su ON su.id = c.started_by
WHERE c.id = @id AND c.workspace_id = @workspace_id AND c.deleted_at IS NULL;

-- name: WhatsAppCampaignInsert :exec
INSERT INTO campaigns (id, workspace_id, name, status, subject, preheader, audience, editor_mode, mjml, html, plain_text, version, channel,
    content_hash, created_by, updated_by, created_at, updated_at)
VALUES (@id, @workspace_id, @name, 'draft', '', '', @audience, 'html', '', '', '', 1, 'whatsapp', @content_hash, sqlc.narg(by), sqlc.narg(by), @at, @at);

-- name: WhatsAppCampaignDetailInsert :exec
INSERT INTO whatsapp_campaigns (campaign_id, workspace_id, template_id, template_name, template_language, variables, schedule_mode, send_at,
    local_date, local_minute, over_limit)
VALUES (@campaign_id, @workspace_id, sqlc.narg(template_id), @template_name, @template_language, @variables, @schedule_mode, sqlc.narg(send_at),
    sqlc.narg(local_date), sqlc.narg(local_minute), @over_limit);

-- name: WhatsAppCampaignSave :exec
UPDATE campaigns SET name = @name, audience = @audience, content_hash = @content_hash, version = @version, updated_by = sqlc.narg(by),
    updated_at = @at, autosaved_at = @at
WHERE id = @id AND workspace_id = @workspace_id;

-- name: WhatsAppCampaignDetailSave :exec
UPDATE whatsapp_campaigns SET template_id = sqlc.narg(template_id), template_name = @template_name, template_language = @template_language,
    variables = @variables, schedule_mode = @schedule_mode, send_at = sqlc.narg(send_at), local_date = sqlc.narg(local_date),
    local_minute = sqlc.narg(local_minute), over_limit = @over_limit
WHERE campaign_id = @campaign_id;

-- name: WhatsAppCampaignSetStatus :exec
UPDATE campaigns SET status = @status, scheduled_at = sqlc.narg(scheduled_at), started_at = coalesce(sqlc.narg(started_at), started_at),
    started_by = coalesce(sqlc.narg(started_by), started_by), completed_at = sqlc.narg(completed_at), audience_count = coalesce(sqlc.narg(audience_count), audience_count),
    updated_at = @at
WHERE id = @id;

-- name: WhatsAppSend :one
SELECT * FROM whatsapp_sends WHERE campaign_id = @campaign_id;

-- name: WhatsAppSendUpsert :exec
INSERT INTO whatsapp_sends (campaign_id, workspace_id, phase, recipients, left_out, started_by, approved_by, scheduled_at)
VALUES (@campaign_id, @workspace_id, @phase, @recipients, @left_out, sqlc.narg(started_by), sqlc.narg(approved_by), @scheduled_at)
ON CONFLICT (campaign_id) DO UPDATE SET phase = EXCLUDED.phase, recipients = EXCLUDED.recipients, left_out = EXCLUDED.left_out,
    started_by = EXCLUDED.started_by, approved_by = EXCLUDED.approved_by, scheduled_at = EXCLUDED.scheduled_at, first_send_at = NULL,
    last_send_at = NULL, completed_at = NULL;

-- name: WhatsAppSendSetPhase :exec
UPDATE whatsapp_sends SET phase = @phase, completed_at = sqlc.narg(completed_at) WHERE campaign_id = @campaign_id;

-- name: WhatsAppMessageInsert :exec
INSERT INTO whatsapp_messages (message_id, workspace_id, account_id, kind, template, language, category, params)
VALUES (@message_id, @workspace_id, sqlc.narg(account_id), @kind, @template, @language, @category, @params);

-- name: WhatsAppMessageByMetaID :one
SELECT m.*, w.kind AS wa_kind, w.template, w.category, w.read_at, w.account_id
FROM channel_messages m JOIN whatsapp_messages w ON w.message_id = m.id
WHERE m.channel = 'whatsapp' AND m.provider_message_id = @provider_message_id AND m.workspace_id = @workspace_id;

-- name: WhatsAppMessageRead :execrows
UPDATE whatsapp_messages SET read_at = @at WHERE message_id = @message_id AND read_at IS NULL;

-- name: WhatsAppMessageBillable :exec
UPDATE whatsapp_messages SET billable = @billable WHERE message_id = @message_id;

-- name: WhatsAppInboundInsert :exec
INSERT INTO whatsapp_inbound (id, workspace_id, account_id, phone, name, body, kind, keyword, message_id, campaign_id, automation_id, contact_ids, effect,
    reply_message_id, meta_id, raw_event_id, received_at)
VALUES (@id, @workspace_id, sqlc.narg(account_id), @phone, @name, @body, @kind, @keyword, sqlc.narg(message_id), sqlc.narg(campaign_id),
    sqlc.narg(automation_id), @contact_ids, @effect, sqlc.narg(reply_message_id), @meta_id, sqlc.narg(raw_event_id), @received_at)
ON CONFLICT (workspace_id, meta_id) WHERE meta_id <> '' DO NOTHING;

-- name: WhatsAppLastInbound :one
-- The last message a number sent: the customer service window runs 24
-- hours from it.
SELECT * FROM whatsapp_inbound WHERE workspace_id = @workspace_id AND phone = @phone ORDER BY received_at DESC LIMIT 1;

-- name: WhatsAppRawEventInsert :exec
INSERT INTO whatsapp_raw_events (id, workspace_id, account_id, received_at, headers, body, signature_ok, summary)
VALUES (@id, @workspace_id, sqlc.narg(account_id), @received_at, @headers, @body, @signature_ok, @summary);

-- name: WhatsAppRawEvent :one
SELECT * FROM whatsapp_raw_events WHERE id = @id;

-- name: WhatsAppRawEventDone :exec
UPDATE whatsapp_raw_events SET normalized_at = @at, normalize_error = @normalize_error, events = @events WHERE id = @id;

-- name: WhatsAppRawEventsPending :many
SELECT id FROM whatsapp_raw_events WHERE normalized_at IS NULL AND signature_ok AND received_at < @before ORDER BY received_at LIMIT 500;
